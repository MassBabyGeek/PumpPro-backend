package handler

import (
	"context"
	"net/http"

	"github.com/MassBabyGeek/PumpPro-backend/internal/database"
	"github.com/MassBabyGeek/PumpPro-backend/internal/middleware"
	model "github.com/MassBabyGeek/PumpPro-backend/internal/models"
	"github.com/MassBabyGeek/PumpPro-backend/internal/scanner"
	"github.com/MassBabyGeek/PumpPro-backend/internal/utils"
	"github.com/gorilla/mux"
)

// GetUserChallenges récupère tous les challenges d'un utilisateur (en cours, complétés, etc.)
func GetUserChallenges(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]

	if userID == "" {
		utils.ErrorSimple(w, http.StatusBadRequest, "user ID manquant")
		return
	}

	// Vérifier que l'utilisateur est admin OU consulte ses propres challenges
	if !middleware.IsOwnerOrAdmin(r, userID) {
		utils.ErrorSimple(w, http.StatusForbidden, "you can only view your own challenges unless you are an admin")
		return
	}

	ctx := context.Background()

	// Query parameter pour filtrer
	status := r.URL.Query().Get("status") // "active", "completed", "all"
	if status == "" {
		status = "all"
	}

	// Récupérer tous les challenges auxquels l'utilisateur participe ou a participé
	query := `
		SELECT DISTINCT
			c.id, c.title, c.description, c.category, c.type, c.variant, c.difficulty,
			c.target_reps, c.duration, c.sets, c.reps_per_set, c.image_url,
			c.icon_name, c.icon_color, c.participants, c.completions, c.likes, c.points,
			c.badge, c.start_date, c.end_date, c.status, c.tags, c.is_official,
			c.created_by, c.updated_by, c.deleted_by, c.created_at, c.updated_at, c.deleted_at,
			-- Vérifier si l'utilisateur a complété le challenge
			COALESCE((
				SELECT COUNT(*) = COUNT(CASE WHEN uctp.completed THEN 1 END)
				FROM challenge_tasks ct
				LEFT JOIN user_challenge_task_progress uctp
					ON ct.id = uctp.task_id AND uctp.user_id = $1
				WHERE ct.challenge_id = c.id
				GROUP BY ct.challenge_id
			), FALSE) AS user_completed,
			-- Vérifier si l'utilisateur a liké
			COALESCE((
				SELECT TRUE
				FROM likes l
				WHERE l.entity_type = 'challenge'
				AND l.entity_id = c.id
				AND l.user_id = $1
			), FALSE) AS user_liked,
			TRUE AS user_participated
		FROM challenges c
		INNER JOIN user_challenge_task_progress uctp ON c.id = uctp.challenge_id
		WHERE uctp.user_id = $1
			AND c.deleted_at IS NULL
	`

	// Ajouter un filtre selon le statut
	if status == "active" {
		query += ` AND c.status = 'active'`
	} else if status == "completed" {
		query += ` AND EXISTS (
			SELECT 1
			FROM challenge_tasks ct
			LEFT JOIN user_challenge_task_progress uctp2
				ON ct.id = uctp2.task_id AND uctp2.user_id = $1
			WHERE ct.challenge_id = c.id
			GROUP BY ct.challenge_id
			HAVING COUNT(*) = COUNT(CASE WHEN uctp2.completed THEN 1 END)
		)`
	}

	query += ` ORDER BY c.created_at DESC`

	rows, err := database.DB.Query(ctx, query, userID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not query user challenges", err)
		return
	}
	defer rows.Close()

	var challenges []model.Challenge
	for rows.Next() {
		challenge, err := scanner.ScanChallengeWithPqArray(rows)
		if err != nil {
			utils.Error(w, http.StatusInternalServerError, "could not scan challenge", err)
			return
		}
		challenges = append(challenges, *challenge)
	}

	// Charger les tasks pour chaque challenge avec la progression de l'utilisateur
	for i := range challenges {
		var userIDPtr *string = &userID
		tasks, err := loadChallengeTasks(ctx, challenges[i].ID, userIDPtr)
		if err != nil {
			utils.Error(w, http.StatusInternalServerError, "could not load challenge tasks", err)
			return
		}
		challenges[i].Tasks = tasks

		// Calculer la progression globale
		if len(tasks) > 0 {
			challenges[i].OverallProgress = calculateOverallProgress(tasks)
		}
	}

	utils.Success(w, challenges)
}
