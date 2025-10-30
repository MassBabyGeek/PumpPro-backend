package handler

import (
	"context"
	"database/sql"
	"net/http"
	"strconv"

	"github.com/MassBabyGeek/PumpPro-backend/internal/database"
	"github.com/MassBabyGeek/PumpPro-backend/internal/middleware"
	model "github.com/MassBabyGeek/PumpPro-backend/internal/models"
	"github.com/MassBabyGeek/PumpPro-backend/internal/scanner"
	"github.com/MassBabyGeek/PumpPro-backend/internal/utils"
	"github.com/gorilla/mux"
)

// CreateBugReport crée un nouveau signalement
func CreateBugReport(w http.ResponseWriter, r *http.Request) {
	var req model.CreateBugReportRequest
	if err := utils.DecodeJSON(r, &req); err != nil {
		utils.Error(w, http.StatusBadRequest, "JSON invalide", err)
		return
	}

	// Validation
	if req.Title == "" || req.Description == "" || req.Category == "" {
		utils.ErrorSimple(w, http.StatusBadRequest, "titre, description et catégorie requis")
		return
	}

	// Récupérer l'utilisateur du contexte (optionnel)
	user, _ := middleware.GetUserFromContext(r)
	var userID *string
	if user.ID != "" {
		userID = &user.ID
	}

	// Définir la sévérité par défaut
	if req.Severity == "" {
		req.Severity = "medium"
	}

	ctx := context.Background()

	// Insérer le signalement
	row := database.DB.QueryRow(ctx, `
		INSERT INTO bug_reports(
			user_id, title, description, category, severity, status,
			device_info, app_version, page_url, error_stack, screenshot_url, user_email,
			created_at, updated_at
		) VALUES($1, $2, $3, $4, $5, 'open', $6, $7, $8, $9, $10, $11, NOW(), NOW())
		RETURNING id, user_id, title, description, category, severity, status,
				  device_info, app_version, page_url, error_stack, screenshot_url, user_email,
				  created_at, updated_at, resolved_at, resolved_by, admin_notes
	`,
		userID, req.Title, req.Description, req.Category, req.Severity,
		req.DeviceInfo, utils.StringToNullString(req.AppVersion),
		utils.StringToNullString(req.PageURL), utils.StringToNullString(req.ErrorStack),
		utils.StringToNullString(req.ScreenshotURL), utils.StringToNullString(req.UserEmail),
	)

	report, err := scanner.ScanBugReport(row)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "impossible de créer le signalement", err)
		return
	}

	utils.Success(w, report)
}

// GetBugReports récupère tous les signalements avec filtres
func GetBugReports(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	status := query.Get("status")
	category := query.Get("category")
	severity := query.Get("severity")
	limitStr := query.Get("limit")
	offsetStr := query.Get("offset")

	ctx := context.Background()

	sqlQuery := `
		SELECT
			id, user_id, title, description, category, severity, status,
			device_info, app_version, page_url, error_stack, screenshot_url, user_email,
			created_at, updated_at, resolved_at, resolved_by, admin_notes
		FROM bug_reports
		WHERE 1=1
	`

	args := []interface{}{}
	argCount := 1

	if status != "" {
		sqlQuery += " AND status = $" + strconv.Itoa(argCount)
		args = append(args, status)
		argCount++
	}

	if category != "" {
		sqlQuery += " AND category = $" + strconv.Itoa(argCount)
		args = append(args, category)
		argCount++
	}

	if severity != "" {
		sqlQuery += " AND severity = $" + strconv.Itoa(argCount)
		args = append(args, severity)
		argCount++
	}

	sqlQuery += " ORDER BY created_at DESC"

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
		}
	}

	rows, err := database.DB.Query(ctx, sqlQuery, args...)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "impossible de récupérer les signalements", err)
		return
	}
	defer rows.Close()

	var reports []model.BugReport
	for rows.Next() {
		report, err := scanner.ScanBugReport(rows)
		if err != nil {
			utils.Error(w, http.StatusInternalServerError, "erreur de lecture", err)
			return
		}
		reports = append(reports, *report)
	}

	utils.Success(w, reports)
}

// GetBugReportById récupère un signalement par son ID
func GetBugReportById(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	ctx := context.Background()

	row := database.DB.QueryRow(ctx, `
		SELECT
			id, user_id, title, description, category, severity, status,
			device_info, app_version, page_url, error_stack, screenshot_url, user_email,
			created_at, updated_at, resolved_at, resolved_by, admin_notes
		FROM bug_reports
		WHERE id = $1
	`, id)

	report, err := scanner.ScanBugReport(row)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "signalement introuvable", err)
		return
	}

	utils.Success(w, report)
}

// UpdateBugReport met à jour un signalement (admin seulement)
func UpdateBugReport(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var req model.UpdateBugReportRequest
	if err := utils.DecodeJSON(r, &req); err != nil {
		utils.Error(w, http.StatusBadRequest, "JSON invalide", err)
		return
	}

	user, err := middleware.GetUserFromContext(r)
	if err != nil {
		utils.Error(w, http.StatusUnauthorized, "authentification requise", err)
		return
	}

	ctx := context.Background()

	// Récupérer le user_id du bug report pour vérifier la propriété
	var reportUserID sql.NullString
	err = database.DB.QueryRow(ctx,
		`SELECT user_id FROM bug_reports WHERE id=$1`,
		id,
	).Scan(&reportUserID)

	if err != nil {
		utils.ErrorSimple(w, http.StatusNotFound, "bug report not found")
		return
	}

	// Vérifier que l'utilisateur est admin OU propriétaire du bug report
	var ownerID string
	if reportUserID.Valid {
		ownerID = reportUserID.String
	}
	if !middleware.IsOwnerOrAdmin(r, ownerID) {
		utils.ErrorSimple(w, http.StatusForbidden, "you are not authorized to update this bug report")
		return
	}

	// Construction dynamique de la requête UPDATE
	query := "UPDATE bug_reports SET updated_at = NOW()"
	args := []interface{}{}
	argCount := 1

	if req.Status != "" {
		query += ", status = $" + strconv.Itoa(argCount)
		args = append(args, req.Status)
		argCount++

		// Si le statut est "resolved" ou "closed", ajouter resolved_at et resolved_by
		if req.Status == "resolved" || req.Status == "closed" {
			query += ", resolved_at = NOW(), resolved_by = $" + strconv.Itoa(argCount)
			args = append(args, user.ID)
			argCount++
		}
	}

	if req.AdminNotes != "" {
		query += ", admin_notes = $" + strconv.Itoa(argCount)
		args = append(args, req.AdminNotes)
		argCount++
	}

	query += " WHERE id = $" + strconv.Itoa(argCount)
	args = append(args, id)
	query += " RETURNING id, user_id, title, description, category, severity, status, device_info, app_version, page_url, error_stack, screenshot_url, user_email, created_at, updated_at, resolved_at, resolved_by, admin_notes"

	row := database.DB.QueryRow(ctx, query, args...)
	report, err := scanner.ScanBugReport(row)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "impossible de mettre à jour le signalement", err)
		return
	}

	utils.Success(w, report)
}

// DeleteBugReport supprime un signalement
func DeleteBugReport(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	ctx := context.Background()

	// Récupérer le user_id du bug report pour vérifier la propriété
	var reportUserID sql.NullString
	err := database.DB.QueryRow(ctx,
		`SELECT user_id FROM bug_reports WHERE id=$1`,
		id,
	).Scan(&reportUserID)

	if err != nil {
		utils.ErrorSimple(w, http.StatusNotFound, "bug report not found")
		return
	}

	// Vérifier que l'utilisateur est admin OU propriétaire du bug report
	var ownerID string
	if reportUserID.Valid {
		ownerID = reportUserID.String
	}
	if !middleware.IsOwnerOrAdmin(r, ownerID) {
		utils.ErrorSimple(w, http.StatusForbidden, "you are not authorized to delete this bug report")
		return
	}

	res, err := database.DB.Exec(ctx, "DELETE FROM bug_reports WHERE id = $1", id)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "impossible de supprimer le signalement", err)
		return
	}

	if res.RowsAffected() == 0 {
		utils.ErrorSimple(w, http.StatusNotFound, "signalement introuvable")
		return
	}

	utils.Success(w, map[string]bool{"success": true})
}

// GetBugReportStats récupère des statistiques sur les signalements
func GetBugReportStats(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	var stats struct {
		Total      int `json:"total"`
		Open       int `json:"open"`
		InProgress int `json:"inProgress"`
		Resolved   int `json:"resolved"`
		Closed     int `json:"closed"`
		Critical   int `json:"critical"`
		High       int `json:"high"`
		Medium     int `json:"medium"`
		Low        int `json:"low"`
	}

	// Compter par statut
	err := database.DB.QueryRow(ctx, `
		SELECT
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE status = 'open') as open,
			COUNT(*) FILTER (WHERE status = 'in-progress') as in_progress,
			COUNT(*) FILTER (WHERE status = 'resolved') as resolved,
			COUNT(*) FILTER (WHERE status = 'closed') as closed,
			COUNT(*) FILTER (WHERE severity = 'critical') as critical,
			COUNT(*) FILTER (WHERE severity = 'high') as high,
			COUNT(*) FILTER (WHERE severity = 'medium') as medium,
			COUNT(*) FILTER (WHERE severity = 'low') as low
		FROM bug_reports
	`).Scan(&stats.Total, &stats.Open, &stats.InProgress, &stats.Resolved, &stats.Closed,
		&stats.Critical, &stats.High, &stats.Medium, &stats.Low)

	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "impossible de récupérer les statistiques", err)
		return
	}

	utils.Success(w, stats)
}
