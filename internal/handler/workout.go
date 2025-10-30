package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/MassBabyGeek/PumpPro-backend/internal/database"
	"github.com/MassBabyGeek/PumpPro-backend/internal/middleware"
	model "github.com/MassBabyGeek/PumpPro-backend/internal/models"
	"github.com/MassBabyGeek/PumpPro-backend/internal/scanner"
	"github.com/MassBabyGeek/PumpPro-backend/internal/utils"
	"github.com/gorilla/mux"
)

// validateWorkoutCompletion vérifie si les conditions du programme sont remplies
func validateWorkoutCompletion(program *model.WorkoutProgram, session *model.WorkoutSession) bool {
	switch program.Type {
	case "TARGET_REPS":
		// Vérifier si l'utilisateur a atteint le nombre de répétitions cible
		if program.TargetReps != nil {
			targetMet := session.TotalReps >= *program.TargetReps

			// Si un temps limite est défini, vérifier qu'il est respecté
			if program.TimeLimit != nil && targetMet {
				return session.TotalDuration <= *program.TimeLimit
			}
			return targetMet
		}
		return false

	case "MAX_TIME":
		// Pour MAX_TIME, l'objectif est de faire un maximum de reps dans le temps imparti
		// La session est "complétée" si la durée correspond au temps attendu (±5%)
		if program.Duration != nil {
			targetDuration := *program.Duration
			tolerance := float64(targetDuration) * 0.05 // 5% de tolérance
			return float64(session.TotalDuration) >= float64(targetDuration)-tolerance &&
				float64(session.TotalDuration) <= float64(targetDuration)+tolerance
		}
		return false

	case "SETS_REPS":
		// Vérifier que le nombre total de reps correspond aux attentes
		if program.Sets != nil && program.RepsPerSet != nil {
			expectedTotalReps := *program.Sets * *program.RepsPerSet
			minReps := int(float64(expectedTotalReps) * 0.9) // Tolérance de 90%
			return session.TotalReps >= minReps
		}
		return false

	case "PYRAMID":
		// Vérifier que le total de reps correspond à la séquence pyramide
		if len(program.RepsSequence) > 0 {
			expectedTotalReps := 0
			for _, reps := range program.RepsSequence {
				expectedTotalReps += reps
			}
			minReps := int(float64(expectedTotalReps) * 0.9) // Tolérance de 90%
			return session.TotalReps >= minReps
		}
		return false

	case "EMOM":
		// Every Minute On the Minute - vérifier le nombre de minutes et reps/minute
		if program.TotalMinutes != nil && program.RepsPerMinute != nil {
			expectedMinutes := *program.TotalMinutes
			expectedRepsPerMinute := *program.RepsPerMinute

			// Vérifier la durée totale (±10% de tolérance)
			expectedDuration := expectedMinutes * 60
			tolerance := float64(expectedDuration) * 0.1
			durationOk := float64(session.TotalDuration) >= float64(expectedDuration)-tolerance &&
				float64(session.TotalDuration) <= float64(expectedDuration)+tolerance

			// Vérifier le nombre total de reps
			expectedTotalReps := expectedMinutes * expectedRepsPerMinute
			repsOk := session.TotalReps >= int(float64(expectedTotalReps)*0.9)

			return durationOk && repsOk
		}
		return false

	case "AMRAP":
		// As Many Reps As Possible - vérifier que la durée est respectée
		if program.Duration != nil {
			targetDuration := *program.Duration
			tolerance := float64(targetDuration) * 0.05
			return float64(session.TotalDuration) >= float64(targetDuration)-tolerance &&
				float64(session.TotalDuration) <= float64(targetDuration)+tolerance
		}
		return false

	case "FREE_MODE":
		// En mode libre, la session est toujours considérée comme complétée
		return true

	default:
		// Type inconnu, considérer comme non complété
		return false
	}
}

// SaveWorkoutSession enregistre une nouvelle session d'entraînement
func SaveWorkoutSession(w http.ResponseWriter, r *http.Request) {
	var session model.WorkoutSession
	if err := utils.DecodeJSON(r, &session); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid JSON body", err)
		return
	}

	user, err := middleware.GetUserFromContext(r)
	if err != nil {
		utils.Error(w, http.StatusUnauthorized, "user not found in context", err)
		return
	}

	ctx := context.Background()

	// Récupérer le programme pour valider la complétion
	var program model.WorkoutProgram
	var repsSequenceJSON []byte
	err = database.DB.QueryRow(ctx, `
		SELECT
			id, name, type, variant, difficulty, rest_between_sets,
			target_reps, time_limit, duration, allow_rest, sets, reps_per_set,
			reps_sequence, reps_per_minute, total_minutes
		FROM workout_programs
		WHERE id=$1 AND deleted_at IS NULL
	`, session.ProgramID).Scan(
		&program.ID, &program.Name, &program.Type, &program.Variant,
		&program.Difficulty, &program.RestBetweenSets,
		&program.TargetReps, &program.TimeLimit, &program.Duration, &program.AllowRest,
		&program.Sets, &program.RepsPerSet, &repsSequenceJSON, &program.RepsPerMinute,
		&program.TotalMinutes,
	)

	if err != nil {
		utils.Error(w, http.StatusNotFound, "workout program not found", err)
		return
	}

	// Décoder la séquence de reps si présente
	if repsSequenceJSON != nil {
		json.Unmarshal(repsSequenceJSON, &program.RepsSequence)
	}

	// Valider si la session est complétée selon les critères du programme
	isCompleted := validateWorkoutCompletion(&program, &session)

	// Insérer la session avec le statut de complétion validé
	err = database.DB.QueryRow(ctx, `
		INSERT INTO workout_sessions(
			program_id, user_id, start_time, end_time, total_reps, total_duration, completed, notes, created_at, created_by
		) VALUES($1, $2, $3, NOW(), $4, $5, $6, $7, NOW(), $8)
		RETURNING id, created_at, created_by
	`,
		session.ProgramID, user.ID, session.StartTime,
		session.TotalReps, session.TotalDuration, isCompleted, session.Notes, user.ID,
	).Scan(&session.ID, &session.CreatedAt, &session.CreatedBy)

	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not save workout session", err)
		return
	}

	// Mettre à jour le champ completed de la session pour le retour
	session.Completed = isCompleted

	// Si la session est liée à une tâche de challenge, mettre à jour la progression
	if session.ChallengeID != nil && session.ChallengeTaskID != nil {
		// Récupérer les infos de la tâche pour obtenir le score
		var taskScore int
		err := database.DB.QueryRow(ctx, `
			SELECT score FROM challenge_tasks WHERE id = $1
		`, *session.ChallengeTaskID).Scan(&taskScore)

		if err != nil {
			// Log l'erreur mais ne pas bloquer la création de la session
			utils.Error(w, http.StatusInternalServerError, "could not fetch task score", err)
			return
		}

		// Vérifier si la task était déjà complétée pour ne pas donner les points plusieurs fois
		var alreadyCompleted bool
		err = database.DB.QueryRow(ctx, `
			SELECT completed FROM user_challenge_task_progress
			WHERE user_id = $1 AND task_id = $2
		`, user.ID, *session.ChallengeTaskID).Scan(&alreadyCompleted)

		// Si la requête ne retourne rien (task pas encore dans la table), err != nil et alreadyCompleted = false
		wasNotCompleted := (err != nil || !alreadyCompleted)

		// Insérer ou mettre à jour la progression de la tâche
		_, err = database.DB.Exec(ctx, `
			INSERT INTO user_challenge_task_progress(
				user_id, task_id, challenge_id, completed, completed_at,
				score, attempts, created_at, updated_at
			)
			VALUES($1, $2, $3, TRUE, NOW(), $4, 1, NOW(), NOW())
			ON CONFLICT (user_id, task_id)
			DO UPDATE SET
				completed = TRUE,
				completed_at = NOW(),
				attempts = user_challenge_task_progress.attempts + 1,
				updated_at = NOW()
		`, user.ID, *session.ChallengeTaskID, *session.ChallengeID, taskScore)

		if err != nil {
			// Log l'erreur mais ne pas bloquer la création de la session
			utils.Error(w, http.StatusInternalServerError, "could not update challenge task progress", err)
			return
		}

		// Ajouter les points au score de l'utilisateur seulement si la task n'était pas déjà complétée ET que la session est complétée
		if wasNotCompleted && taskScore > 0 && isCompleted {
			if err := utils.IncrementUserScore(ctx, user.ID, taskScore); err != nil {
				// Log l'erreur mais ne pas bloquer la création de la session
				utils.Error(w, http.StatusInternalServerError, "could not update user score for task", err)
				return
			}
		}
	}

	// Incrémenter le usage_count du programme
	_, err = database.DB.Exec(ctx,
		`UPDATE workout_programs SET usage_count = usage_count + 1 WHERE id = $1`,
		session.ProgramID,
	)
	if err != nil {
		// Log l'erreur mais ne pas bloquer la création de la session
		utils.Error(w, http.StatusInternalServerError, "could not update program usage count", err)
		return
	}

	// Incrémenter le score de l'utilisateur si la session est complétée
	if isCompleted {
		// Points basés sur la difficulté: BEGINNER=5, INTERMEDIATE=10, ADVANCED=15
		points := 5
		switch program.Difficulty {
		case "INTERMEDIATE":
			points = 10
		case "ADVANCED":
			points = 15
		}

		// Incrémenter le score de l'utilisateur
		if err := utils.IncrementUserScore(ctx, user.ID, points); err != nil {
			// Log l'erreur mais ne pas bloquer la création de la session
			utils.Error(w, http.StatusInternalServerError, "could not update user score", err)
			return
		}
	}

	utils.Success(w, session)
}

// GetWorkoutSessions récupère toutes les sessions d'entraînement avec filtres
func GetWorkoutSessions(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	startDate := query.Get("startDate")
	endDate := query.Get("endDate")
	programType := query.Get("programType")
	limitStr := query.Get("limit")
	offsetStr := query.Get("offset")

	ctx := context.Background()

	// Get optional authenticated user
	user, _ := middleware.GetUserFromContext(r)
	var userID *string
	if user.ID != "" {
		userID = &user.ID
	}

	sqlQuery := `
		SELECT
			ws.id, ws.program_id, ws.user_id, ws.start_time, ws.end_time,
			ws.total_reps, ws.total_duration, ws.completed, ws.notes,
			COALESCE(ws.likes, 0) as likes,
			COALESCE((
				SELECT TRUE
				FROM likes l
				WHERE l.entity_type = 'workout'
				AND l.entity_id = ws.id
				AND l.user_id = $1
			), FALSE) AS user_liked,
			ws.created_at, ws.updated_at,
			creator.id as creator_id, creator.name as creator_name, creator.avatar as creator_avatar,
			u.id as user_id, u.name as user_name, u.avatar as user_avatar
		FROM workout_sessions ws
		LEFT JOIN users creator ON ws.created_by = creator.id AND creator.deleted_at IS NULL
		LEFT JOIN users u ON ws.user_id = u.id AND u.deleted_at IS NULL
		WHERE 1=1
	`

	args := []interface{}{userID}
	argCount := 2

	if startDate != "" {
		sqlQuery += " AND ws.start_time >= $" + strconv.Itoa(argCount)
		args = append(args, startDate)
		argCount++
	}

	if endDate != "" {
		sqlQuery += " AND ws.start_time <= $" + strconv.Itoa(argCount)
		args = append(args, endDate)
		argCount++
	}

	if programType != "" {
		sqlQuery += ` AND ws.program_id IN (
			SELECT id FROM workout_programs WHERE type = $` + strconv.Itoa(argCount) + `
		)`
		args = append(args, programType)
		argCount++
	}

	sqlQuery += " ORDER BY ws.start_time DESC"

	// Pagination
	if limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			sqlQuery += " LIMIT $" + strconv.Itoa(argCount)
			args = append(args, limit)
			argCount++
		}
	} else {
		// Limite par défaut de 50 pour éviter de retourner trop de données
		sqlQuery += " LIMIT 50"
	}

	if offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			sqlQuery += " OFFSET $" + strconv.Itoa(argCount)
			args = append(args, offset)
			argCount++
		}
	}

	rows, err := database.DB.Query(ctx, sqlQuery, args...)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not query workout sessions", err)
		return
	}
	defer rows.Close()

	var sessions []model.WorkoutSession
	for rows.Next() {
		session, err := scanner.ScanWorkoutSessionWithCreatorAndUser(rows)
		if err != nil {
			utils.Error(w, http.StatusInternalServerError, "could not scan session row", err)
			return
		}

		sessions = append(sessions, *session)
	}

	utils.Success(w, sessions)
}

// GetWorkoutStats récupère les statistiques d'entraînement pour un utilisateur
func GetWorkoutStats(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]
	period := r.URL.Query().Get("period") // today, week, month, year

	ctx := context.Background()

	// Calculer la date de début selon la période
	var startDate time.Time
	now := time.Now()

	switch period {
	case "today":
		startDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	case "week":
		startDate = now.AddDate(0, 0, -7)
	case "month":
		startDate = now.AddDate(0, 0, -30)
	case "year":
		startDate = now.AddDate(0, 0, -365)
	default:
		startDate = now.AddDate(0, 0, -7) // Par défaut: semaine
	}

	var stats struct {
		TotalPushUps   int     `json:"totalPushUps"`
		TotalWorkouts  int     `json:"totalWorkouts"`
		TotalTime      int     `json:"totalTime"`
		BestSession    int     `json:"bestSession"`
		AveragePushUps float64 `json:"averagePushUps"`
		TotalCalories  float64 `json:"totalCalories"`
	}

	err := database.DB.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(total_reps), 0) as total_push_ups,
			COUNT(*) as total_workouts,
			COALESCE(SUM(total_duration), 0) as total_time,
			COALESCE(MAX(total_reps), 0) as best_session
		FROM workout_sessions
		WHERE user_id = $1 AND start_time >= $2
	`, userID, startDate).Scan(
		&stats.TotalPushUps,
		&stats.TotalWorkouts,
		&stats.TotalTime,
		&stats.BestSession,
	)

	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not fetch stats", err)
		return
	}

	// Calculs dérivés
	if stats.TotalWorkouts > 0 {
		stats.AveragePushUps = float64(stats.TotalPushUps) / float64(stats.TotalWorkouts)
	}
	stats.TotalCalories = float64(stats.TotalPushUps) * 0.29 // ~0.29 calories par pompe

	utils.Success(w, stats)
}

// DeleteWorkoutSession supprime une session d'entraînement
func DeleteWorkoutSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	ctx := context.Background()

	// Récupérer le user_id de la session pour vérifier la propriété
	var sessionUserID string
	var err error
	err = database.DB.QueryRow(ctx,
		`SELECT user_id FROM workout_sessions WHERE id = $1`,
		sessionID,
	).Scan(&sessionUserID)

	if err != nil {
		utils.ErrorSimple(w, http.StatusNotFound, "session not found")
		return
	}

	// Vérifier que l'utilisateur est admin OU propriétaire de la session
	if !middleware.IsOwnerOrAdmin(r, sessionUserID) {
		utils.ErrorSimple(w, http.StatusForbidden, "you are not authorized to delete this session")
		return
	}

	res, err := database.DB.Exec(ctx,
		`DELETE FROM workout_sessions WHERE id = $1`,
		sessionID,
	)

	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not delete session", err)
		return
	}

	if res.RowsAffected() == 0 {
		utils.ErrorSimple(w, http.StatusNotFound, "session not found")
		return
	}

	utils.Success(w, map[string]bool{"success": true})
}

// GetWorkoutSession récupère une session d'entraînement par son ID
func GetWorkoutSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	ctx := context.Background()

	// Get optional authenticated user
	user, _ := middleware.GetUserFromContext(r)
	var userID *string
	if user.ID != "" {
		userID = &user.ID
	}

	rows, err := database.DB.Query(ctx, `
		SELECT
			ws.id, ws.program_id, ws.user_id, ws.start_time, ws.end_time,
			ws.total_reps, ws.total_duration, ws.completed, ws.notes,
			COALESCE(ws.likes, 0) as likes,
			COALESCE((
				SELECT TRUE
				FROM likes l
				WHERE l.entity_type = 'workout'
				AND l.entity_id = ws.id
				AND l.user_id = $1
			), FALSE) AS user_liked,
			ws.created_at, ws.updated_at,
			creator.id as creator_id, creator.name as creator_name, creator.avatar as creator_avatar,
			u.id as user_id, u.name as user_name, u.avatar as user_avatar
		FROM workout_sessions ws
		LEFT JOIN users creator ON ws.created_by = creator.id AND creator.deleted_at IS NULL
		LEFT JOIN users u ON ws.user_id = u.id AND u.deleted_at IS NULL
		WHERE ws.id = $2
	`, userID, sessionID)

	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not query workout session", err)
		return
	}

	session, err := scanner.ScanWorkoutSessionWithCreatorAndUser(rows)
	rows.Close() // Fermer immédiatement après le scan
	if err != nil {
		utils.Error(w, http.StatusNotFound, "session not found", err)
		return
	}

	utils.Success(w, session)
}

// UpdateWorkoutSession met à jour une session d'entraînement
func UpdateWorkoutSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	var updates map[string]interface{}
	if err := utils.DecodeJSON(r, &updates); err != nil {
		utils.ErrorSimple(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	ctx := context.Background()

	// Récupérer le user_id de la session pour vérifier la propriété
	var sessionUserID string
	var err error
	err = database.DB.QueryRow(ctx,
		`SELECT user_id FROM workout_sessions WHERE id = $1`,
		sessionID,
	).Scan(&sessionUserID)

	if err != nil {
		utils.ErrorSimple(w, http.StatusNotFound, "session not found")
		return
	}

	// Vérifier que l'utilisateur est admin OU propriétaire de la session
	if !middleware.IsOwnerOrAdmin(r, sessionUserID) {
		utils.ErrorSimple(w, http.StatusForbidden, "you are not authorized to update this session")
		return
	}

	// Construction dynamique de la requête UPDATE
	query := "UPDATE workout_sessions SET updated_at = NOW()"
	args := []interface{}{}
	argCount := 1

	if endTime, ok := updates["endTime"]; ok {
		query += ", end_time = $" + strconv.Itoa(argCount)
		args = append(args, endTime)
		argCount++
	}

	if totalReps, ok := updates["totalReps"]; ok {
		query += ", total_reps = $" + strconv.Itoa(argCount)
		args = append(args, totalReps)
		argCount++
	}

	if totalDuration, ok := updates["totalDuration"]; ok {
		query += ", total_duration = $" + strconv.Itoa(argCount)
		args = append(args, totalDuration)
		argCount++
	}

	if completed, ok := updates["completed"]; ok {
		query += ", completed = $" + strconv.Itoa(argCount)
		args = append(args, completed)
		argCount++
	}

	if notes, ok := updates["notes"]; ok {
		query += ", notes = $" + strconv.Itoa(argCount)
		args = append(args, notes)
		argCount++
	}

	query += " WHERE id = $" + strconv.Itoa(argCount)
	args = append(args, sessionID)

	_, err = database.DB.Exec(ctx, query, args...)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not update session", err)
		return
	}

	// Get optional authenticated user
	user, _ := middleware.GetUserFromContext(r)
	var authenticatedUserID *string
	if user.ID != "" {
		authenticatedUserID = &user.ID
	}

	// Récupérer la session mise à jour
	sessionRows, err := database.DB.Query(ctx, `
		SELECT
			ws.id, ws.program_id, ws.user_id, ws.start_time, ws.end_time,
			ws.total_reps, ws.total_duration, ws.completed, ws.notes,
			COALESCE(ws.likes, 0) as likes,
			COALESCE((
				SELECT TRUE
				FROM likes l
				WHERE l.entity_type = 'workout'
				AND l.entity_id = ws.id
				AND l.user_id = $1
			), FALSE) AS user_liked,
			ws.created_at, ws.updated_at
		FROM workout_sessions ws
		WHERE ws.id = $2
	`, authenticatedUserID, sessionID)

	session, err := scanner.ScanWorkoutSession(sessionRows)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not fetch updated session", err)
		return
	}

	utils.Success(w, session)
}

// GetWorkoutSummary récupère un résumé des entraînements pour une période donnée
func GetWorkoutSummary(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]

	query := r.URL.Query()
	startDate := query.Get("startDate")
	endDate := query.Get("endDate")

	ctx := context.Background()

	var summary struct {
		TotalSessions int     `json:"totalSessions"`
		TotalReps     int     `json:"totalReps"`
		TotalDuration int     `json:"totalDuration"`
		AverageReps   float64 `json:"averageReps"`
		BestSession   int     `json:"bestSession"`
		TotalCalories float64 `json:"totalCalories"`
	}

	err := database.DB.QueryRow(ctx, `
		SELECT
			COUNT(*) as total_sessions,
			COALESCE(SUM(total_reps), 0) as total_reps,
			COALESCE(SUM(total_duration), 0) as total_duration,
			COALESCE(MAX(total_reps), 0) as best_session
		FROM workout_sessions
		WHERE user_id = $1 AND start_time >= $2 AND start_time <= $3
	`, userID, startDate, endDate).Scan(
		&summary.TotalSessions,
		&summary.TotalReps,
		&summary.TotalDuration,
		&summary.BestSession,
	)

	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not fetch summary", err)
		return
	}

	// Calculs dérivés
	if summary.TotalSessions > 0 {
		summary.AverageReps = float64(summary.TotalReps) / float64(summary.TotalSessions)
	}
	summary.TotalCalories = float64(summary.TotalReps) * 0.29

	utils.Success(w, summary)
}

// GetPersonalRecords récupère les records personnels d'un utilisateur
func GetPersonalRecords(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]

	ctx := context.Background()

	var records struct {
		MaxRepsInSession  int `json:"maxRepsInSession"`
		MaxRepsInSet      int `json:"maxRepsInSet"`
		LongestSession    int `json:"longestSession"`
		TotalLifetimeReps int `json:"totalLifetimeReps"`
	}

	// Records de sessions
	err := database.DB.QueryRow(ctx, `
		SELECT
			COALESCE(MAX(total_reps), 0) as max_reps_in_session,
			COALESCE(MAX(total_duration), 0) as longest_session,
			COALESCE(SUM(total_reps), 0) as total_lifetime_reps
		FROM workout_sessions
		WHERE user_id = $1
	`, userID).Scan(
		&records.MaxRepsInSession,
		&records.LongestSession,
		&records.TotalLifetimeReps,
	)

	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not fetch session records", err)
		return
	}

	utils.Success(w, records)
}

// LikeWorkout ajoute un like à une session de travail
func LikeWorkout(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	user, err := middleware.GetUserFromContext(r)
	if err != nil {
		utils.Error(w, http.StatusUnauthorized, "impossible de récupérer l'utilisateur", err)
		return
	}

	ctx := context.Background()

	// Utiliser le système unifié de likes
	err = utils.AddLike(ctx, user.ID, model.EntityTypeWorkout, sessionID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not add like", err)
		return
	}

	// Incrémenter le compteur de likes
	_, err = database.DB.Exec(ctx, `
		UPDATE workout_sessions SET likes = likes + 1 WHERE id=$1`,
		sessionID,
	)

	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not increment likes", err)
		return
	}

	// Retourner le session mis à jour
	row := database.DB.QueryRow(ctx, `
		SELECT
			ws.id, ws.program_id, ws.user_id, ws.start_time, ws.end_time,
			ws.total_reps, ws.total_duration, ws.completed, ws.notes,
			COALESCE(ws.likes, 0) as likes,
			COALESCE((
				SELECT TRUE
				FROM likes l
				WHERE l.entity_type = 'workout'
				AND l.entity_id = ws.id
				AND l.user_id = $1
			), FALSE) AS user_liked,
			ws.created_at, ws.updated_at
		FROM workout_sessions ws
		WHERE ws.id = $2
	`, user.ID, sessionID)

	session, err := scanner.ScanWorkoutSession(row)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not fetch session", err)
		return
	}

	utils.Success(w, session)
}

// UnlikeWorkout retire un like à une session de travail
func UnlikeWorkout(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	user, err := middleware.GetUserFromContext(r)
	if err != nil {
		utils.Error(w, http.StatusUnauthorized, "impossible de récupérer l'utilisateur", err)
		return
	}

	ctx := context.Background()

	// Utiliser le système unifié de likes
	err = utils.RemoveLike(ctx, user.ID, model.EntityTypeWorkout, sessionID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not remove like", err)
		return
	}

	// Décrémenter le compteur de likes
	_, err = database.DB.Exec(ctx, `
		UPDATE workout_sessions SET likes = GREATEST(likes - 1, 0) WHERE id=$1`,
		sessionID,
	)

	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not decrement likes", err)
		return
	}

	// Retourner le session mis à jour
	row := database.DB.QueryRow(ctx, `
		SELECT
			ws.id, ws.program_id, ws.user_id, ws.start_time, ws.end_time,
			ws.total_reps, ws.total_duration, ws.completed, ws.notes,
			COALESCE(ws.likes, 0) as likes,
			COALESCE((
				SELECT TRUE
				FROM likes l
				WHERE l.entity_type = 'workout'
				AND l.entity_id = ws.id
				AND l.user_id = $1
			), FALSE) AS user_liked,
			ws.created_at, ws.updated_at
		FROM workout_sessions ws
		WHERE ws.id = $2
	`, user.ID, sessionID)

	session, err := scanner.ScanWorkoutSession(row)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not fetch session", err)
		return
	}

	utils.Success(w, session)
}
