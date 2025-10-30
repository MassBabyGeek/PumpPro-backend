package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/MassBabyGeek/PumpPro-backend/internal/database"
	"github.com/MassBabyGeek/PumpPro-backend/internal/middleware"
	"github.com/MassBabyGeek/PumpPro-backend/internal/utils"
)

type Photo struct {
	URL        string  `json:"url"`
	Type       string  `json:"type"`       // "avatar", "challenge", "bug_report"
	EntityID   string  `json:"entityId"`   // ID of the user, challenge, or bug report
	EntityName *string `json:"entityName"` // Name of the user or title of challenge/bug report
	CreatedAt  string  `json:"createdAt"`
}

// GetAllPhotos récupère toutes les photos de l'application (admin only)
func GetAllPhotos(w http.ResponseWriter, r *http.Request) {
	// Vérifier que l'utilisateur est admin
	if !middleware.IsAdmin(r) {
		utils.ErrorSimple(w, http.StatusForbidden, "admin privileges required")
		return
	}

	ctx := context.Background()

	query := r.URL.Query()
	limitStr := query.Get("limit")
	offsetStr := query.Get("offset")
	photoType := query.Get("type") // "avatar", "challenge", "bug_report", or "all"

	// Pagination par défaut
	limit := 50
	offset := 0

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	var photos []Photo
	var totalCount int

	// Construction de la requête selon le type demandé
	if photoType == "" || photoType == "all" || photoType == "avatar" {
		// Récupérer les avatars des utilisateurs
		avatarQuery := `
			SELECT
				avatar,
				'avatar' as type,
				id as entity_id,
				name as entity_name,
				created_at
			FROM users
			WHERE avatar IS NOT NULL
			  AND avatar != ''
			  AND deleted_at IS NULL
		`

		rows, err := database.DB.Query(ctx, avatarQuery)
		if err != nil {
			utils.Error(w, http.StatusInternalServerError, "could not query user avatars", err)
			return
		}
		defer rows.Close()

		for rows.Next() {
			var photo Photo
			err := rows.Scan(&photo.URL, &photo.Type, &photo.EntityID, &photo.EntityName, &photo.CreatedAt)
			if err != nil {
				continue
			}
			photos = append(photos, photo)
		}
	}

	if photoType == "" || photoType == "all" || photoType == "challenge" {
		// Récupérer les images des challenges
		challengeQuery := `
			SELECT
				image_url,
				'challenge' as type,
				id as entity_id,
				title as entity_name,
				created_at
			FROM challenges
			WHERE image_url IS NOT NULL
			  AND image_url != ''
			  AND deleted_at IS NULL
		`

		rows, err := database.DB.Query(ctx, challengeQuery)
		if err != nil {
			utils.Error(w, http.StatusInternalServerError, "could not query challenge images", err)
			return
		}
		defer rows.Close()

		for rows.Next() {
			var photo Photo
			err := rows.Scan(&photo.URL, &photo.Type, &photo.EntityID, &photo.EntityName, &photo.CreatedAt)
			if err != nil {
				continue
			}
			photos = append(photos, photo)
		}
	}

	if photoType == "" || photoType == "all" || photoType == "bug_report" {
		// Récupérer les screenshots des bug reports
		bugReportQuery := `
			SELECT
				screenshot_url,
				'bug_report' as type,
				id as entity_id,
				title as entity_name,
				created_at
			FROM bug_reports
			WHERE screenshot_url IS NOT NULL
			  AND screenshot_url != ''
		`

		rows, err := database.DB.Query(ctx, bugReportQuery)
		if err != nil {
			utils.Error(w, http.StatusInternalServerError, "could not query bug report screenshots", err)
			return
		}
		defer rows.Close()

		for rows.Next() {
			var photo Photo
			err := rows.Scan(&photo.URL, &photo.Type, &photo.EntityID, &photo.EntityName, &photo.CreatedAt)
			if err != nil {
				continue
			}
			photos = append(photos, photo)
		}
	}

	// Appliquer la pagination sur le tableau de résultats
	totalCount = len(photos)

	start := offset
	end := offset + limit

	if start > len(photos) {
		start = len(photos)
	}
	if end > len(photos) {
		end = len(photos)
	}

	paginatedPhotos := photos[start:end]

	// Retourner le résultat avec métadonnées de pagination
	result := map[string]interface{}{
		"photos": paginatedPhotos,
		"pagination": map[string]interface{}{
			"total":  totalCount,
			"limit":  limit,
			"offset": offset,
			"count":  len(paginatedPhotos),
		},
	}

	utils.Success(w, result)
}
