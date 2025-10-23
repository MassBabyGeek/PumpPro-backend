package handler

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/MassBabyGeek/PumpPro-backend/internal/database"
	"github.com/MassBabyGeek/PumpPro-backend/internal/logger"
	"github.com/MassBabyGeek/PumpPro-backend/internal/middleware"
	model "github.com/MassBabyGeek/PumpPro-backend/internal/models"
	"github.com/MassBabyGeek/PumpPro-backend/internal/scanner"
	"github.com/MassBabyGeek/PumpPro-backend/internal/utils"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5"
	"github.com/lib/pq"
)

// calculateOverallProgress calcule la progression globale d'un challenge pour un utilisateur
func calculateOverallProgress(tasks []model.ChallengeTask) *int {
	if len(tasks) == 0 {
		return nil
	}

	completedTasks := 0
	for _, task := range tasks {
		if task.UserProgress != nil && task.UserProgress.Completed {
			completedTasks++
		}
	}

	progress := (completedTasks * 100) / len(tasks)
	return &progress
}

// loadChallengeTasks charge les tasks d'un challenge avec la progression utilisateur optionnelle
func loadChallengeTasks(ctx context.Context, challengeID string, userID *string) ([]model.ChallengeTask, error) {
	rows, err := database.DB.Query(ctx, `
		SELECT
			id, challenge_id, day, title, description, type, variant,
			target_reps, duration, sets, reps_per_set, score,
			scheduled_date, is_locked, created_by, updated_by, deleted_by,
			created_at, updated_at, deleted_at
		FROM challenge_tasks
		WHERE challenge_id = $1 AND deleted_at IS NULL
		ORDER BY day ASC
	`, challengeID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []model.ChallengeTask
	for rows.Next() {
		task, err := scanner.ScanChallengeTask(rows)
		if err != nil {
			return nil, err
		}

		// Si un userID est fourni, charger la progression de l'utilisateur pour cette task
		if userID != nil && *userID != "" {
			progressRow := database.DB.QueryRow(ctx, `
				SELECT
					id, user_id, task_id, challenge_id, completed, completed_at,
					score, attempts, created_at, updated_at
				FROM user_challenge_task_progress
				WHERE task_id = $1 AND user_id = $2
			`, task.ID, *userID)

			progress, err := scanner.ScanUserChallengeTaskProgress(progressRow)
			if err == nil {
				task.UserProgress = progress
			}
		}

		// Load creator information
		utils.EnrichChallengeTaskWithCreator(ctx, task)

		tasks = append(tasks, *task)
	}

	return tasks, nil
}

func loadChallengeTask(ctx context.Context, challengeTaskId string, userID *string) (*model.ChallengeTask, error) {
	rows, err := database.DB.Query(ctx,
		`SELECT
			id, challenge_id, day, title, description, type, variant,
			target_reps, duration, sets, reps_per_set, score,
			scheduled_date, is_locked, created_by, updated_by, deleted_by,
			created_at, updated_at, deleted_at
		FROM challenge_tasks
		WHERE id=$1 AND deleted_at IS NULL`,
		challengeTaskId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var task *model.ChallengeTask

	if rows.Next() {
		task, err = scanner.ScanChallengeTask(rows)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("task not found")
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Charger la progression utilisateur si demandée
	if userID != nil && *userID != "" {
		progressRow := database.DB.QueryRow(ctx, `
			SELECT
				id, user_id, task_id, challenge_id, completed, completed_at,
				score, attempts, created_at, updated_at
			FROM user_challenge_task_progress
			WHERE task_id = $1 AND user_id = $2
		`, challengeTaskId, *userID)

		progress, err := scanner.ScanUserChallengeTaskProgress(progressRow)
		if err == nil {
			task.UserProgress = progress
		}
	}

	// Load creator information
	utils.EnrichChallengeTaskWithCreator(ctx, task)

	return task, nil
}

// GetChallenges récupère tous les challenges avec filtres optionnels
func GetChallenges(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	query := r.URL.Query()

	// Récupérer l'utilisateur depuis le contexte (OptionalAuth)
	user, _ := middleware.GetUserFromContext(r)
	var userID *string
	if user.ID != "" {
		userID = &user.ID
	}

	filters := map[string]string{
		"category":   query.Get("category"),
		"difficulty": query.Get("difficulty"),
		"type":       query.Get("type"),
		"variant":    query.Get("variant"),
		"status":     query.Get("status"),
	}

	searchQuery := query.Get("searchQuery")
	sortBy := query.Get("sortBy")
	limitStr := query.Get("limit")
	offsetStr := query.Get("offset")

	args := []interface{}{}
	argCount := 1

	// Base SELECT - Always include user columns (with FALSE if no user)
	sqlQuery := `
		SELECT
			c.id, c.title, c.description, c.category,
				c.type, c.variant, c.difficulty, c.target_reps, c.duration,
				c.sets, c.reps_per_set, c.image_url, c.icon_name, c.icon_color,
				c.participants, c.completions, c.likes, c.points, c.badge,
				c.start_date, c.end_date, c.status, c.tags, c.is_official,
				c.created_by, c.updated_by, c.created_at, c.updated_at,
				c.deleted_by, c.deleted_at,
	`

	// Add user columns dynamically based on authentication
	if userID != nil {
		sqlQuery += `
			COALESCE((
				SELECT TRUE
				FROM user_challenge_task_progress uctp
				WHERE uctp.challenge_id = c.id
				AND uctp.user_id = $` + strconv.Itoa(argCount) + `
				AND uctp.completed = TRUE
				LIMIT 1
			), FALSE) AS user_completed,
			COALESCE((
				SELECT TRUE
				FROM likes l
				WHERE l.entity_type = 'challenge'
				AND l.entity_id = c.id
				AND l.user_id = $` + strconv.Itoa(argCount) + `
			), FALSE) AS user_liked,
			COALESCE((
				SELECT TRUE
				FROM user_challenge_task_progress uctp
				WHERE uctp.challenge_id = c.id
				AND uctp.user_id = $` + strconv.Itoa(argCount) + `
				LIMIT 1
			), FALSE) AS user_participated
		`
		args = append(args, *userID)
		argCount++
	} else {
		// Add default FALSE values when no user is authenticated
		sqlQuery += `
			FALSE AS user_completed,
			FALSE AS user_liked,
			FALSE AS user_participated
		`
	}

	sqlQuery += `
		FROM challenges c
		WHERE deleted_at IS NULL
	`

	// Filtres dynamiques
	for col, val := range filters {
		if val != "" {
			sqlQuery += " AND " + col + " = $" + strconv.Itoa(argCount)
			args = append(args, val)
			argCount++
		}
	}

	// Recherche textuelle
	if searchQuery != "" {
		sqlQuery += " AND (LOWER(title) LIKE $" + strconv.Itoa(argCount) +
			" OR LOWER(description) LIKE $" + strconv.Itoa(argCount+1) + ")"
		searchPattern := "%" + strings.ToLower(searchQuery) + "%"
		args = append(args, searchPattern, searchPattern)
		argCount += 2
	}

	// Tri
	sortMap := map[string]string{
		"POPULAR":    "completions DESC",
		"LIKED":      "likes DESC",
		"RECENT":     "start_date DESC NULLS LAST",
		"DIFFICULTY": "CASE difficulty WHEN 'BEGINNER' THEN 1 WHEN 'INTERMEDIATE' THEN 2 WHEN 'ADVANCED' THEN 3 END",
		"POINTS":     "points DESC",
	}

	if order, ok := sortMap[strings.ToUpper(sortBy)]; ok {
		sqlQuery += " ORDER BY " + order
	} else {
		sqlQuery += " ORDER BY created_at DESC"
	}

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
		utils.Error(w, http.StatusInternalServerError, "could not query challenges", err)
		return
	}
	defer rows.Close()

	var challenges []model.Challenge
	for rows.Next() {
		challenge, err := scanner.ScanChallenge(rows)
		if err != nil {
			utils.Error(w, http.StatusInternalServerError, "could not scan challenge row", err)
			return
		}

		// Charger les tasks
		tasks, err := loadChallengeTasks(ctx, challenge.ID, userID)
		if err != nil {
			utils.Error(w, http.StatusInternalServerError, "could not load challenge tasks", err)
			return
		}
		challenge.Tasks = tasks

		// Calculer la progression globale si l'utilisateur est connecté
		if userID != nil {
			challenge.OverallProgress = calculateOverallProgress(tasks)
		}

		// Load creator information
		utils.EnrichChallengeWithCreator(ctx, challenge)

		challenges = append(challenges, *challenge)
	}

	utils.Success(w, challenges)
}

// GetChallengeById récupère un challenge par son ID
func GetChallengeById(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	challengeID := vars["id"]

	// Récupérer l'utilisateur courant depuis le contexte (si authentifié)
	user, _ := middleware.GetUserFromContext(r)
	ctx := context.Background()

	var row pgx.Row
	if user.ID != "" {
		// Utilisateur connecté : vérifier les valeurs user_completed / user_participated
		row = database.DB.QueryRow(ctx, `
			SELECT
				c.id, c.title, c.description, c.category,
				c.type, c.variant, c.difficulty, c.target_reps, c.duration,
				c.sets, c.reps_per_set, c.image_url, c.icon_name, c.icon_color,
				c.participants, c.completions, c.likes, c.points, c.badge,
				c.start_date, c.end_date, c.status, c.tags, c.is_official,
				c.created_by, c.updated_by, c.created_at, c.updated_at,
				c.deleted_by, c.deleted_at,

				COALESCE((
					SELECT TRUE
					FROM user_challenge_task_progress uctp
					WHERE uctp.challenge_id = c.id
					  AND uctp.user_id = $2
					  AND uctp.completed = TRUE
					LIMIT 1
				), FALSE) AS user_completed,

				COALESCE((
					SELECT TRUE
					FROM likes l
					WHERE l.entity_type = 'challenge'
					  AND l.entity_id = c.id
					  AND l.user_id = $2
				), FALSE) AS user_liked,

				COALESCE((
					SELECT TRUE
					FROM user_challenge_task_progress uctp
					WHERE uctp.challenge_id = c.id
					  AND uctp.user_id = $2
					LIMIT 1
				), FALSE) AS user_participated
			FROM challenges c
			WHERE c.id = $1
			  AND c.deleted_at IS NULL
		`, challengeID, user.ID)
	} else {
		// Utilisateur non connecté : valeurs par défaut FALSE
		row = database.DB.QueryRow(ctx, `
			SELECT
				c.id, c.title, c.description, c.category,
				c.type, c.variant, c.difficulty, c.target_reps, c.duration,
				c.sets, c.reps_per_set, c.image_url, c.icon_name, c.icon_color,
				c.participants, c.completions, c.likes, c.points, c.badge,
				c.start_date, c.end_date, c.status, c.tags, c.is_official,
				c.created_by, c.updated_by, c.created_at, c.updated_at,
				c.deleted_by, c.deleted_at,
				FALSE AS user_completed,
				FALSE AS user_liked,
				FALSE AS user_participated
			FROM challenges c
			WHERE c.id = $1
			  AND c.deleted_at IS NULL
		`, challengeID)
	}

	// Scanner le challenge
	challenge, err := scanner.ScanChallenge(row)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "challenge not found", err)
		return
	}

	// Charger les tasks du challenge avec progression utilisateur si connecté
	var userID *string
	if user.ID != "" {
		userID = &user.ID
	}
	tasks, err := loadChallengeTasks(ctx, challenge.ID, userID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not load challenge tasks", err)
		return
	}
	challenge.Tasks = tasks

	// Calculer la progression globale si l'utilisateur est connecté
	if userID != nil {
		challenge.OverallProgress = calculateOverallProgress(tasks)
	}

	// Load creator information
	utils.EnrichChallengeWithCreator(ctx, challenge)

	utils.Success(w, challenge)
}

// CreateChallenge crée un nouveau challenge
func CreateChallenge(w http.ResponseWriter, r *http.Request) {
	var challenge model.Challenge
	if err := utils.DecodeJSON(r, &challenge); err != nil {
		utils.ErrorSimple(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	ctx := context.Background()

	err := database.DB.QueryRow(ctx, `
		INSERT INTO challenges(
			title, description, category, type, variant, difficulty,
			target_reps, duration, sets, reps_per_set, image_url,
			icon_name, icon_color, participants, completions, likes, points,
			badge, start_date, end_date, status, tags, is_official,
			created_by, created_at, updated_at
		) VALUES(
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, NOW(), NOW()
		)
		RETURNING id, created_at, updated_at
	`,
		challenge.Title, challenge.Description, challenge.Category, challenge.Type,
		challenge.Variant, challenge.Difficulty, challenge.TargetReps, challenge.Duration,
		challenge.Sets, challenge.RepsPerSet, challenge.ImageURL, challenge.IconName,
		challenge.IconColor, challenge.Participants, challenge.Completions, challenge.Likes,
		challenge.Points, challenge.Badge, challenge.StartDate, challenge.EndDate,
		challenge.Status, pq.Array(challenge.Tags), challenge.IsOfficial, challenge.CreatedBy,
	).Scan(&challenge.ID, &challenge.CreatedAt, &challenge.UpdatedAt)

	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not create challenge", err)
		return
	}

	utils.Success(w, challenge)
}

// UpdateChallenge met à jour un challenge existant
func UpdateChallenge(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var challenge model.Challenge
	if err := utils.DecodeJSON(r, &challenge); err != nil {
		utils.ErrorSimple(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	ctx := context.Background()

	_, err := database.DB.Exec(ctx, `
		UPDATE challenges SET
			title=$1, description=$2, category=$3, type=$4, variant=$5, difficulty=$6,
			target_reps=$7, duration=$8, sets=$9, reps_per_set=$10, image_url=$11,
			icon_name=$12, icon_color=$13, badge=$14, start_date=$15, end_date=$16,
			status=$17, tags=$18, is_official=$19, updated_by=$20, updated_at=NOW()
		WHERE id=$21 AND deleted_at IS NULL
	`,
		challenge.Title, challenge.Description, challenge.Category, challenge.Type,
		challenge.Variant, challenge.Difficulty, challenge.TargetReps, challenge.Duration,
		challenge.Sets, challenge.RepsPerSet, challenge.ImageURL, challenge.IconName,
		challenge.IconColor, challenge.Badge, challenge.StartDate, challenge.EndDate,
		challenge.Status, pq.Array(challenge.Tags), challenge.IsOfficial, challenge.UpdatedBy, id,
	)

	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not update challenge", err)
		return
	}

	challenge.ID = id
	utils.Success(w, challenge)
}

// DeleteChallenge soft delete un challenge
func DeleteChallenge(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var payload struct {
		DeletedBy *string `json:"deletedBy"`
	}
	if err := utils.DecodeJSON(r, &payload); err != nil {
		utils.ErrorSimple(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	ctx := context.Background()
	res, err := database.DB.Exec(ctx, `
		UPDATE challenges SET deleted_at=NOW(), deleted_by=$2
		WHERE id=$1 AND deleted_at IS NULL
	`, id, payload.DeletedBy)

	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not delete challenge", err)
		return
	}

	if res.RowsAffected() == 0 {
		utils.ErrorSimple(w, http.StatusNotFound, "challenge not found or already deleted")
		return
	}

	utils.Message(w, "challenge deleted successfully")
}

// LikeChallenge ajoute un like à un challenge
func LikeChallenge(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	challengeID := vars["id"]

	user, err := middleware.GetUserFromContext(r)
	if err != nil {
		utils.Error(w, http.StatusUnauthorized, "impossible de récupérer l'utilisateur", err)
		return
	}

	ctx := context.Background()

	// Utiliser le système unifié de likes
	err = utils.AddLike(ctx, user.ID, model.EntityTypeChallenge, challengeID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not add like", err)
		return
	}

	// Retourner le challenge mis à jour
	row := database.DB.QueryRow(ctx, `
		SELECT
			id, title, description, category, type, variant, difficulty,
			target_reps, duration, sets, reps_per_set, image_url,
			icon_name, icon_color, participants, completions, likes, points,
			badge, start_date, end_date, status, tags, is_official,
			created_by, updated_by, deleted_by, created_at, updated_at, deleted_at,
			COALESCE((
				SELECT TRUE
				FROM user_challenge_task_progress uctp
				WHERE uctp.challenge_id = id
				  AND uctp.user_id = $2
				  AND uctp.completed = TRUE
				LIMIT 1
			), FALSE) AS user_completed,
			TRUE AS user_liked,
			COALESCE((
				SELECT TRUE
				FROM user_challenge_task_progress uctp
				WHERE uctp.challenge_id = id
				  AND uctp.user_id = $2
				LIMIT 1
			), FALSE) AS user_participated
		FROM challenges
		WHERE id=$1
	`, challengeID, user.ID)

	challenge, err := scanner.ScanChallenge(row)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not fetch challenge", err)
		return
	}

	// Charger les tasks du challenge avec la progression utilisateur
	tasks, err := loadChallengeTasks(ctx, challenge.ID, &user.ID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not load challenge tasks", err)
		return
	}
	challenge.Tasks = tasks

	utils.Success(w, challenge)
}

// UnlikeChallenge retire un like d'un challenge
func UnlikeChallenge(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	challengeID := vars["id"]

	user, err := middleware.GetUserFromContext(r)
	if err != nil {
		utils.Error(w, http.StatusUnauthorized, "impossible de récupérer l'utilisateur", err)
		return
	}
	ctx := context.Background()

	// Utiliser le système unifié de likes
	err = utils.RemoveLike(ctx, user.ID, model.EntityTypeChallenge, challengeID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not remove like", err)
		return
	}

	// Retourner le challenge mis à jour
	row := database.DB.QueryRow(ctx, `
		SELECT
			id, title, description, category, type, variant, difficulty,
			target_reps, duration, sets, reps_per_set, image_url,
			icon_name, icon_color, participants, completions, likes, points,
			badge, start_date, end_date, status, tags, is_official,
			created_by, updated_by, deleted_by, created_at, updated_at, deleted_at,
			COALESCE((
				SELECT TRUE
				FROM user_challenge_task_progress uctp
				WHERE uctp.challenge_id = id
				  AND uctp.user_id = $2
				  AND uctp.completed = TRUE
				LIMIT 1
			), FALSE) AS user_completed,
			FALSE AS user_liked,
			COALESCE((
				SELECT TRUE
				FROM user_challenge_task_progress uctp
				WHERE uctp.challenge_id = id
				  AND uctp.user_id = $2
				LIMIT 1
			), FALSE) AS user_participated
		FROM challenges
		WHERE id=$1
	`, challengeID, user.ID)

	challenge, err := scanner.ScanChallenge(row)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not fetch challenge", err)
		return
	}

	// Charger les tasks du challenge avec la progression utilisateur
	tasks, err := loadChallengeTasks(ctx, challenge.ID, &user.ID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not load challenge tasks", err)
		return
	}
	challenge.Tasks = tasks

	utils.Success(w, challenge)
}

// StartChallenge démarre un challenge pour un utilisateur
func StartChallenge(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	challengeID := vars["id"]

	var payload struct {
		UserID string `json:"userId"`
	}
	if err := utils.DecodeJSON(r, &payload); err != nil {
		utils.ErrorSimple(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	ctx := context.Background()

	// Récupérer les infos du challenge
	var targetReps int
	err := database.DB.QueryRow(ctx,
		`SELECT target_reps FROM challenges WHERE id=$1 AND deleted_at IS NULL`,
		challengeID,
	).Scan(&targetReps)

	if err != nil {
		utils.Error(w, http.StatusNotFound, "challenge not found", err)
		return
	}

	// Vérifier si l'utilisateur a déjà commencé ce challenge
	var exists bool
	err = database.DB.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM user_challenge_progress WHERE challenge_id=$1 AND user_id=$2)`,
		challengeID, payload.UserID,
	).Scan(&exists)

	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not check progress", err)
		return
	}

	if exists {
		utils.ErrorSimple(w, http.StatusBadRequest, "challenge already started")
		return
	}

	// Créer la progression
	row := database.DB.QueryRow(ctx, `
		INSERT INTO user_challenge_progress(challenge_id, user_id, progress, current_reps, target_reps, attempts, completed_at, created_at, updated_at)
		VALUES($1, $2, 0, 0, $3, 0, NULL, NOW(), NOW())
		RETURNING id, challenge_id, user_id, progress, current_reps, target_reps, attempts, completed_at, created_at, updated_at
	`, challengeID, payload.UserID, targetReps)

	progress, err := scanner.ScanUserChallengeProgress(row)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not create progress", err)
		return
	}

	// Incrémenter le nombre de participants
	_, err = database.DB.Exec(ctx,
		`UPDATE challenges SET participants = participants + 1 WHERE id=$1`,
		challengeID,
	)

	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not increment participants", err)
		return
	}

	utils.Success(w, progress)
}

// CompleteChallenge marque un challenge comme complété
func CompleteChallenge(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	challengeID := vars["id"]

	var payload struct {
		UserID string `json:"userId"`
	}
	if err := utils.DecodeJSON(r, &payload); err != nil {
		utils.ErrorSimple(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	ctx := context.Background()

	// Mettre à jour la progression
	row := database.DB.QueryRow(ctx, `
		UPDATE user_challenge_progress
		SET progress=100, completed_at=NOW(), updated_at=NOW()
		WHERE challenge_id=$1 AND user_id=$2
		RETURNING id, challenge_id, user_id, progress, current_reps, target_reps, attempts, completed_at, created_at, updated_at
	`, challengeID, payload.UserID)

	progress, err := scanner.ScanUserChallengeProgress(row)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "progress not found", err)
		return
	}

	// Incrémenter le nombre de complétions
	_, err = database.DB.Exec(ctx,
		`UPDATE challenges SET completions = completions + 1 WHERE id=$1`,
		challengeID,
	)

	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not increment completions", err)
		return
	}

	// Récupérer les points du challenge et les ajouter au score de l'utilisateur
	var challengePoints int
	err = database.DB.QueryRow(ctx, `
		SELECT points FROM challenges WHERE id = $1
	`, challengeID).Scan(&challengePoints)

	if err == nil && challengePoints > 0 {
		// Incrémenter le score de l'utilisateur
		if err := utils.IncrementUserScore(ctx, payload.UserID, challengePoints); err != nil {
			// Log l'erreur mais ne pas bloquer la complétion du challenge
			utils.Error(w, http.StatusInternalServerError, "could not update user score", err)
			return
		}
	}

	utils.Success(w, progress)
}

// GetUserChallengeProgress récupère la progression d'un utilisateur sur un challenge
func GetUserChallengeProgress(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	challengeID := vars["id"]
	userID := r.URL.Query().Get("userId")

	ctx := context.Background()

	row := database.DB.QueryRow(ctx, `
		SELECT id, challenge_id, user_id, progress, current_reps, target_reps, attempts, completed_at, created_at, updated_at
		FROM user_challenge_progress
		WHERE challenge_id=$1 AND user_id=$2
	`, challengeID, userID)

	progress, err := scanner.ScanUserChallengeProgress(row)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "progress not found", err)
		return
	}

	utils.Success(w, progress)
}

// GetUserActiveChallenges récupère tous les challenges actifs d'un utilisateur
func GetUserActiveChallenges(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]

	ctx := context.Background()

	rows, err := database.DB.Query(ctx, `
		SELECT
			c.id, c.title, c.description, c.category, c.type, c.variant, c.difficulty,
			c.target_reps, c.duration, c.sets, c.reps_per_set, c.image_url,
			c.icon_name, c.icon_color, c.participants, c.completions, c.likes, c.points,
			c.badge, c.start_date, c.end_date, c.status, c.tags, c.is_official,
			c.created_by, c.updated_by, c.deleted_by, c.created_at, c.updated_at, c.deleted_at,
			TRUE AS user_completed,
			COALESCE((
				SELECT TRUE
				FROM likes l
				WHERE l.entity_type = 'challenge'
				AND l.entity_id = c.id
				AND l.user_id = $1
			), FALSE) AS user_liked,
			TRUE AS user_participated
		FROM challenges c
		INNER JOIN user_challenge_progress ucp ON c.id = ucp.challenge_id
		WHERE ucp.user_id=$1 AND ucp.progress < 100 AND c.deleted_at IS NULL
		ORDER BY ucp.updated_at DESC
	`, userID)

	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not query challenges", err)
		return
	}
	defer rows.Close()

	var challenges []model.Challenge
	for rows.Next() {
		challenge, err := scanner.ScanChallenge(rows)
		if err != nil {
			utils.Error(w, http.StatusInternalServerError, "could not scan challenge row", err)
			return
		}

		// Charger les tasks du challenge avec la progression utilisateur
		tasks, err := loadChallengeTasks(ctx, challenge.ID, &userID)
		if err != nil {
			utils.Error(w, http.StatusInternalServerError, "could not load challenge tasks", err)
			return
		}
		challenge.Tasks = tasks

		// Load creator information
		utils.EnrichChallengeWithCreator(ctx, challenge)

		challenges = append(challenges, *challenge)
	}

	utils.Success(w, challenges)
}

// GetUserCompletedChallenges récupère tous les challenges complétés d'un utilisateur
func GetUserCompletedChallenges(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]

	ctx := context.Background()

	rows, err := database.DB.Query(ctx, `
		SELECT
			c.id, c.title, c.description, c.category, c.type, c.variant, c.difficulty,
			c.target_reps, c.duration, c.sets, c.reps_per_set, c.image_url,
			c.icon_name, c.icon_color, c.participants, c.completions, c.likes, c.points,
			c.badge, c.start_date, c.end_date, c.status, c.tags, c.is_official,
			c.created_by, c.updated_by, c.deleted_by, c.created_at, c.updated_at, c.deleted_at,
			TRUE AS user_completed,
			COALESCE((
				SELECT TRUE
				FROM likes l
				WHERE l.entity_type = 'challenge'
				AND l.entity_id = c.id
				AND l.user_id = $1
			), FALSE) AS user_liked,
			TRUE AS user_participated
		FROM challenges c
		INNER JOIN user_challenge_progress ucp ON c.id = ucp.challenge_id
		WHERE ucp.user_id=$1 AND ucp.progress = 100 AND c.deleted_at IS NULL
		ORDER BY ucp.completed_at DESC
	`, userID)

	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not query challenges", err)
		return
	}
	defer rows.Close()

	var challenges []model.Challenge
	for rows.Next() {
		challenge, err := scanner.ScanChallenge(rows)
		if err != nil {
			utils.Error(w, http.StatusInternalServerError, "could not scan challenge row", err)
			return
		}

		// Charger les tasks du challenge avec la progression utilisateur
		tasks, err := loadChallengeTasks(ctx, challenge.ID, &userID)
		if err != nil {
			utils.Error(w, http.StatusInternalServerError, "could not load challenge tasks", err)
			return
		}
		challenge.Tasks = tasks

		// Load creator information
		utils.EnrichChallengeWithCreator(ctx, challenge)

		challenges = append(challenges, *challenge)
	}

	utils.Success(w, challenges)
}

// CompleteTask marque une task comme complétée
func CompleteTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["taskId"]

	user, err := middleware.GetUserFromContext(r)
	if err != nil {
		utils.Error(w, http.StatusUnauthorized, "impossible de récupérer l'utilisateur", err)
		return
	}

	logger.Info("CompleteTask: User=%s, Task=%s", user.ID, taskID)

	ctx := context.Background()
	task, err := loadChallengeTask(ctx, taskID, &user.ID)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "task not found", err)
		return
	}

	// Vérifier si c'est la première task que l'utilisateur complète pour ce challenge
	var isFirstTask bool
	err = database.DB.QueryRow(ctx, `
		SELECT NOT EXISTS(
			SELECT 1 FROM user_challenge_task_progress
			WHERE user_id = $1 AND challenge_id = $2
		)
	`, user.ID, task.ChallengeID).Scan(&isFirstTask)

	if err != nil {
		logger.Error("Could not check if first task: %v", err)
		isFirstTask = false
	}

	_, err = database.DB.Exec(ctx, `
		INSERT INTO user_challenge_task_progress(user_id, task_id, challenge_id, completed, completed_at, score, attempts, created_at, updated_at)
		VALUES($1, $2, $3, true, NOW(), $4, 1, NOW(), NOW())
		ON CONFLICT (user_id, task_id)
		DO UPDATE SET
			completed = true,
			completed_at = NOW(),
			attempts = user_challenge_task_progress.attempts + 1,
			updated_at = NOW()
	`, user.ID, task.ID, task.ChallengeID, task.Score)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not create progress", err)
		return
	}

	// Si c'est la première task de ce challenge pour cet utilisateur, incrémenter participants
	if isFirstTask {
		_, err = database.DB.Exec(ctx, `
			UPDATE challenges SET participants = participants + 1 WHERE id = $1
		`, task.ChallengeID)

		if err != nil {
			logger.Error("Could not increment participants for challenge %s: %v", task.ChallengeID, err)
		}
	}

	// Vérifier si toutes les tasks du challenge sont complétées par cet utilisateur
	var totalTasks, completedTasks int
	err = database.DB.QueryRow(ctx, `
		SELECT
			COUNT(*) as total_tasks,
			COUNT(uctp.id) FILTER (WHERE uctp.completed = true) as completed_tasks
		FROM challenge_tasks ct
		LEFT JOIN user_challenge_task_progress uctp
			ON ct.id = uctp.task_id AND uctp.user_id = $1
		WHERE ct.challenge_id = $2 AND ct.deleted_at IS NULL
	`, user.ID, task.ChallengeID).Scan(&totalTasks, &completedTasks)

	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not check challenge completion", err)
		return
	}

	// Si toutes les tasks sont complétées, incrémenter les completions
	if totalTasks > 0 && completedTasks == totalTasks {
		// Vérifier si on n'a pas déjà incrémenté les completions pour cet utilisateur
		var alreadyCounted bool
		err = database.DB.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM user_challenge_progress
				WHERE challenge_id = $1 AND user_id = $2 AND progress = 100
			)
		`, task.ChallengeID, user.ID).Scan(&alreadyCounted)

		if err == nil && !alreadyCounted {
			// Incrémenter les completions du challenge
			_, err = database.DB.Exec(ctx, `
				UPDATE challenges SET completions = completions + 1 WHERE id = $1
			`, task.ChallengeID)

			if err != nil {
				logger.Error("Could not increment completions for challenge %s: %v", task.ChallengeID, err)
			}

			// Mettre à jour la progression de l'utilisateur à 100%
			_, err = database.DB.Exec(ctx, `
				INSERT INTO user_challenge_progress(challenge_id, user_id, progress, current_reps, target_reps, attempts, completed_at, created_at, updated_at)
				VALUES($1, $2, 100, 0, 0, 1, NOW(), NOW(), NOW())
				ON CONFLICT (challenge_id, user_id)
				DO UPDATE SET
					progress = 100,
					completed_at = NOW(),
					updated_at = NOW()
			`, task.ChallengeID, user.ID)

			if err != nil {
				logger.Error("Could not update user challenge progress: %v", err)
			}
		}
	}

	utils.Success(w, task)
}
