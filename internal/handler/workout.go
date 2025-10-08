package handler

import (
	"context"
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/MassBabyGeek/PumpPro-backend/internal/database"
	model "github.com/MassBabyGeek/PumpPro-backend/internal/models"
	"github.com/MassBabyGeek/PumpPro-backend/internal/utils"
	"github.com/gorilla/mux"
)

// SaveWorkoutSession enregistre une nouvelle session d'entraînement
func SaveWorkoutSession(w http.ResponseWriter, r *http.Request) {
	var session model.WorkoutSession
	if err := utils.DecodeJSON(r, &session); err != nil {
		utils.ErrorSimple(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	ctx := context.Background()

	// Insérer la session
	err := database.DB.QueryRow(ctx, `
		INSERT INTO workout_sessions(
			program_id, user_id, start_time, end_time, total_reps, total_duration, completed, notes, created_at, updated_at
		) VALUES($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`,
		session.ProgramID, session.UserID, session.StartTime, session.EndTime,
		session.TotalReps, session.TotalDuration, session.Completed, session.Notes,
	).Scan(&session.ID, &session.CreatedAt, &session.UpdatedAt)

	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not save workout session", err)
		return
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

	utils.Success(w, session)
}

// GetWorkoutSessions récupère les sessions d'entraînement d'un utilisateur avec filtres
func GetWorkoutSessions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]

	query := r.URL.Query()
	startDate := query.Get("startDate")
	endDate := query.Get("endDate")
	programType := query.Get("programType")
	limitStr := query.Get("limit")
	offsetStr := query.Get("offset")

	ctx := context.Background()

	sqlQuery := `
		SELECT
			ws.id, ws.program_id, ws.user_id, ws.start_time, ws.end_time,
			ws.total_reps, ws.total_duration, ws.completed, ws.notes,
			ws.created_at, ws.updated_at
		FROM workout_sessions ws
		WHERE ws.user_id = $1
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

	if limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			sqlQuery += " LIMIT $" + strconv.Itoa(argCount)
			args = append(args, limit)
			argCount++
		}
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
		var s model.WorkoutSession
		if err := rows.Scan(
			&s.ID, &s.ProgramID, &s.UserID, &s.StartTime, &s.EndTime,
			&s.TotalReps, &s.TotalDuration, &s.Completed, &s.Notes,
			&s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			utils.Error(w, http.StatusInternalServerError, "could not scan session row", err)
			return
		}
		sessions = append(sessions, s)
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

	var session model.WorkoutSession
	err := database.DB.QueryRow(ctx, `
		SELECT
			id, program_id, user_id, start_time, end_time,
			total_reps, total_duration, completed, notes,
			created_at, updated_at
		FROM workout_sessions
		WHERE id = $1
	`, sessionID).Scan(
		&session.ID, &session.ProgramID, &session.UserID, &session.StartTime, &session.EndTime,
		&session.TotalReps, &session.TotalDuration, &session.Completed, &session.Notes,
		&session.CreatedAt, &session.UpdatedAt,
	)

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

	_, err := database.DB.Exec(ctx, query, args...)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not update session", err)
		return
	}

	// Récupérer la session mise à jour
	var session model.WorkoutSession
	err = database.DB.QueryRow(ctx, `
		SELECT
			id, program_id, user_id, start_time, end_time,
			total_reps, total_duration, completed, notes,
			created_at, updated_at
		FROM workout_sessions
		WHERE id = $1
	`, sessionID).Scan(
		&session.ID, &session.ProgramID, &session.UserID, &session.StartTime, &session.EndTime,
		&session.TotalReps, &session.TotalDuration, &session.Completed, &session.Notes,
		&session.CreatedAt, &session.UpdatedAt,
	)

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

	// Record de répétitions dans une série
	err = database.DB.QueryRow(ctx, `
		SELECT COALESCE(MAX(sr.completed_reps), 0)
		FROM set_results sr
		INNER JOIN workout_sessions ws ON sr.session_id = ws.id
		WHERE ws.user_id = $1
	`, userID).Scan(&records.MaxRepsInSet)

	if err != nil && err != sql.ErrNoRows {
		utils.Error(w, http.StatusInternalServerError, "could not fetch set records", err)
		return
	}

	utils.Success(w, records)
}

// SaveSetResults enregistre les résultats d'une série
func SaveSetResults(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["sessionId"]

	var setResults []model.SetResult
	if err := utils.DecodeJSON(r, &setResults); err != nil {
		utils.ErrorSimple(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	ctx := context.Background()

	// Insérer tous les résultats de séries
	for _, set := range setResults {
		_, err := database.DB.Exec(ctx, `
			INSERT INTO set_results(session_id, set_number, target_reps, completed_reps, duration, timestamp)
			VALUES($1, $2, $3, $4, $5, $6)
		`,
			sessionID, set.SetNumber, set.TargetReps, set.CompletedReps, set.Duration, set.Timestamp,
		)

		if err != nil {
			utils.Error(w, http.StatusInternalServerError, "could not save set result", err)
			return
		}
	}

	utils.Success(w, map[string]bool{"success": true})
}

// GetSetResults récupère les résultats des séries d'une session
func GetSetResults(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["sessionId"]

	ctx := context.Background()

	rows, err := database.DB.Query(ctx, `
		SELECT id, session_id, set_number, target_reps, completed_reps, duration, timestamp
		FROM set_results
		WHERE session_id = $1
		ORDER BY set_number ASC
	`, sessionID)

	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not query set results", err)
		return
	}
	defer rows.Close()

	var results []model.SetResult
	for rows.Next() {
		var r model.SetResult
		if err := rows.Scan(
			&r.ID, &r.SessionID, &r.SetNumber, &r.TargetReps, &r.CompletedReps, &r.Duration, &r.Timestamp,
		); err != nil {
			utils.Error(w, http.StatusInternalServerError, "could not scan set result", err)
			return
		}
		results = append(results, r)
	}

	utils.Success(w, results)
}
