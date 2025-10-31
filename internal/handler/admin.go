package handler

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/MassBabyGeek/PumpPro-backend/internal/database"
	"github.com/MassBabyGeek/PumpPro-backend/internal/middleware"
	model "github.com/MassBabyGeek/PumpPro-backend/internal/models"
	"github.com/MassBabyGeek/PumpPro-backend/internal/utils"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
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
			  AND LENGTH(avatar) > 0
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
			var createdAt time.Time
			err := rows.Scan(&photo.URL, &photo.Type, &photo.EntityID, &photo.EntityName, &createdAt)
			if err != nil {
				continue
			}
			photo.CreatedAt = createdAt.Format(time.RFC3339)
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
			  AND LENGTH(image_url) > 0
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
			var createdAt time.Time
			err := rows.Scan(&photo.URL, &photo.Type, &photo.EntityID, &photo.EntityName, &createdAt)
			if err != nil {
				continue
			}
			photo.CreatedAt = createdAt.Format(time.RFC3339)
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
			  AND LENGTH(screenshot_url) > 0
		`

		rows, err := database.DB.Query(ctx, bugReportQuery)
		if err != nil {
			utils.Error(w, http.StatusInternalServerError, "could not query bug report screenshots", err)
			return
		}
		defer rows.Close()

		for rows.Next() {
			var photo Photo
			var createdAt time.Time
			err := rows.Scan(&photo.URL, &photo.Type, &photo.EntityID, &photo.EntityName, &createdAt)
			if err != nil {
				continue
			}
			photo.CreatedAt = createdAt.Format(time.RFC3339)
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

// GetAdminDashboard retourne toutes les statistiques pour le dashboard admin
func GetAdminDashboard(w http.ResponseWriter, r *http.Request) {
	if !middleware.IsAdmin(r) {
		utils.ErrorSimple(w, http.StatusForbidden, "admin privileges required")
		return
	}

	ctx := context.Background()
	stats := model.AdminDashboardStats{
		GeneratedAt: time.Now(),
	}

	// Total des utilisateurs
	database.DB.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE deleted_at IS NULL").Scan(&stats.TotalUsers)

	// Utilisateurs actifs (dernières 24h)
	database.DB.QueryRow(ctx, `
		SELECT COUNT(DISTINCT user_id)
		FROM workout_sessions
		WHERE created_at > NOW() - INTERVAL '24 hours'
	`).Scan(&stats.ActiveUsers)

	// Nouveaux utilisateurs
	database.DB.QueryRow(ctx, `
		SELECT COUNT(*) FROM users
		WHERE created_at::date = CURRENT_DATE AND deleted_at IS NULL
	`).Scan(&stats.NewUsersToday)

	database.DB.QueryRow(ctx, `
		SELECT COUNT(*) FROM users
		WHERE created_at > NOW() - INTERVAL '7 days' AND deleted_at IS NULL
	`).Scan(&stats.NewUsersThisWeek)

	database.DB.QueryRow(ctx, `
		SELECT COUNT(*) FROM users
		WHERE created_at > NOW() - INTERVAL '30 days' AND deleted_at IS NULL
	`).Scan(&stats.NewUsersThisMonth)

	// Total challenges
	database.DB.QueryRow(ctx, "SELECT COUNT(*) FROM challenges WHERE deleted_at IS NULL").Scan(&stats.TotalChallenges)

	// Challenges actifs
	database.DB.QueryRow(ctx, `
		SELECT COUNT(*) FROM challenges
		WHERE status = 'active' AND deleted_at IS NULL
	`).Scan(&stats.ActiveChallenges)

	// Total programs
	database.DB.QueryRow(ctx, "SELECT COUNT(*) FROM workout_programs WHERE deleted_at IS NULL").Scan(&stats.TotalPrograms)

	// Total workouts
	database.DB.QueryRow(ctx, "SELECT COUNT(*) FROM workout_sessions").Scan(&stats.TotalWorkouts)

	// Total pushups
	database.DB.QueryRow(ctx, "SELECT COALESCE(SUM(total_reps), 0) FROM workout_sessions").Scan(&stats.TotalPushups)

	// Workouts aujourd'hui
	database.DB.QueryRow(ctx, `
		SELECT COUNT(*) FROM workout_sessions
		WHERE created_at::date = CURRENT_DATE
	`).Scan(&stats.WorkoutsToday)

	// Workouts cette semaine
	database.DB.QueryRow(ctx, `
		SELECT COUNT(*) FROM workout_sessions
		WHERE created_at > NOW() - INTERVAL '7 days'
	`).Scan(&stats.WorkoutsThisWeek)

	// Workouts ce mois
	database.DB.QueryRow(ctx, `
		SELECT COUNT(*) FROM workout_sessions
		WHERE created_at > NOW() - INTERVAL '30 days'
	`).Scan(&stats.WorkoutsThisMonth)

	// Bug reports
	database.DB.QueryRow(ctx, "SELECT COUNT(*) FROM bug_reports").Scan(&stats.TotalBugReports)
	database.DB.QueryRow(ctx, "SELECT COUNT(*) FROM bug_reports WHERE status = 'pending'").Scan(&stats.PendingBugReports)

	// Moyennes
	database.DB.QueryRow(ctx, `
		SELECT COALESCE(AVG(total_reps), 0)
		FROM (
			SELECT user_id, SUM(total_reps) as total_reps
			FROM workout_sessions
			GROUP BY user_id
		) as user_stats
	`).Scan(&stats.AvgPushupsPerUser)

	database.DB.QueryRow(ctx, `
		SELECT COALESCE(AVG(workout_count), 0)
		FROM (
			SELECT user_id, COUNT(*) as workout_count
			FROM workout_sessions
			GROUP BY user_id
		) as user_stats
	`).Scan(&stats.AvgWorkoutsPerUser)

	// Compter les photos
	var avatarCount, challengeCount, bugReportCount int
	database.DB.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE avatar IS NOT NULL AND avatar != '' AND deleted_at IS NULL").Scan(&avatarCount)
	database.DB.QueryRow(ctx, "SELECT COUNT(*) FROM challenges WHERE image_url IS NOT NULL AND image_url != '' AND deleted_at IS NULL").Scan(&challengeCount)
	database.DB.QueryRow(ctx, "SELECT COUNT(*) FROM bug_reports WHERE screenshot_url IS NOT NULL AND screenshot_url != ''").Scan(&bugReportCount)
	stats.TotalPhotos = avatarCount + challengeCount + bugReportCount

	// Calculer l'utilisation du stockage (uploads folder)
	uploadsPath := "./uploads"
	var totalSize int64
	filepath.Walk(uploadsPath, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})
	sizeMB := float64(totalSize) / (1024 * 1024)
	if sizeMB > 1024 {
		stats.StorageUsed = fmt.Sprintf("%.2f GB", sizeMB/1024)
	} else {
		stats.StorageUsed = fmt.Sprintf("%.2f MB", sizeMB)
	}

	utils.Success(w, stats)
}

// GetAdminRecentActivity retourne l'activité récente des utilisateurs
func GetAdminRecentActivity(w http.ResponseWriter, r *http.Request) {
	if !middleware.IsAdmin(r) {
		utils.ErrorSimple(w, http.StatusForbidden, "admin privileges required")
		return
	}

	ctx := context.Background()
	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 200 {
			limit = l
		}
	}

	query := `
		-- Workouts récents
		SELECT
			ws.user_id,
			u.name,
			u.avatar,
			'workout' as action,
			'program' as entity_type,
			CAST(ws.program_id AS TEXT) as entity_id,
			wp.name as entity_name,
			ws.created_at as timestamp
		FROM workout_sessions ws
		JOIN users u ON ws.user_id = u.id
		LEFT JOIN workout_programs wp ON ws.program_id = wp.id
		WHERE ws.created_at > NOW() - INTERVAL '7 days'

		UNION ALL

		-- Challenges complétés
		SELECT
			uctp.user_id,
			u.name,
			u.avatar,
			'challenge_completed' as action,
			'challenge' as entity_type,
			CAST(c.id AS TEXT) as entity_id,
			c.title as entity_name,
			uctp.completed_at as timestamp
		FROM user_challenge_task_progress uctp
		JOIN users u ON uctp.user_id = u.id
		JOIN challenges c ON uctp.challenge_id = c.id
		WHERE uctp.completed = true
		  AND uctp.completed_at > NOW() - INTERVAL '7 days'

		UNION ALL

		-- Nouveaux utilisateurs
		SELECT
			u.id,
			u.name,
			u.avatar,
			'signup' as action,
			NULL as entity_type,
			NULL as entity_id,
			NULL as entity_name,
			u.created_at as timestamp
		FROM users u
		WHERE u.created_at > NOW() - INTERVAL '7 days'
		  AND u.deleted_at IS NULL

		ORDER BY timestamp DESC
		LIMIT $1
	`

	rows, err := database.DB.Query(ctx, query, limit)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not fetch recent activity", err)
		return
	}
	defer rows.Close()

	var activities []model.AdminUserActivity
	for rows.Next() {
		var activity model.AdminUserActivity
		err := rows.Scan(
			&activity.UserID,
			&activity.UserName,
			&activity.UserAvatar,
			&activity.Action,
			&activity.EntityType,
			&activity.EntityID,
			&activity.EntityName,
			&activity.Timestamp,
		)
		if err != nil {
			continue
		}
		activities = append(activities, activity)
	}

	utils.Success(w, activities)
}

// GetAdminSystemHealth retourne l'état de santé du système
func GetAdminSystemHealth(w http.ResponseWriter, r *http.Request) {
	if !middleware.IsAdmin(r) {
		utils.ErrorSimple(w, http.StatusForbidden, "admin privileges required")
		return
	}

	ctx := context.Background()
	health := model.AdminSystemHealth{
		Status:    "healthy",
		CheckedAt: time.Now(),
	}

	// Vérifier la connexion à la base de données
	start := time.Now()
	err := database.DB.Ping(ctx)
	responseTime := time.Since(start).Milliseconds()
	health.ResponseTime = float64(responseTime)

	if err != nil {
		health.Status = "critical"
		health.DatabaseStatus = "disconnected"
	} else {
		health.DatabaseStatus = "connected"
	}

	// Taille de la base de données
	var dbSize string
	database.DB.QueryRow(ctx, "SELECT pg_size_pretty(pg_database_size(current_database()))").Scan(&dbSize)
	health.DatabaseSize = dbSize

	// Sessions actives (approximation via workouts récents)
	var activeSessions int
	database.DB.QueryRow(ctx, `
		SELECT COUNT(DISTINCT user_id)
		FROM workout_sessions
		WHERE created_at > NOW() - INTERVAL '5 minutes'
	`).Scan(&activeSessions)
	health.ActiveSessions = activeSessions

	// Uptime (depuis le démarrage du serveur - approximatif)
	health.Uptime = "N/A" // Nécessiterait une variable globale pour tracker le démarrage

	utils.Success(w, health)
}

// GetAdminTopContent retourne les contenus les plus populaires
func GetAdminTopContent(w http.ResponseWriter, r *http.Request) {
	if !middleware.IsAdmin(r) {
		utils.ErrorSimple(w, http.StatusForbidden, "admin privileges required")
		return
	}

	ctx := context.Background()
	topContent := model.AdminTopContent{}

	// Top challenges par likes
	challengeRows, err := database.DB.Query(ctx, `
		SELECT id, title, image_url, likes, completions
		FROM challenges
		WHERE deleted_at IS NULL
		ORDER BY likes DESC
		LIMIT 10
	`)
	if err == nil {
		defer challengeRows.Close()
		for challengeRows.Next() {
			var item model.TopItem
			var imageURL sql.NullString
			var completions int
			challengeRows.Scan(&item.ID, &item.Name, &imageURL, &item.Count, &completions)
			item.ImageURL = utils.NullStringToPointer(imageURL)
			item.Metric = "likes"
			item.MetricValue = item.Count
			topContent.Challenges = append(topContent.Challenges, item)
		}
	}

	// Top programs par utilisation
	programRows, err := database.DB.Query(ctx, `
		SELECT id, name, usage_count, likes
		FROM workout_programs
		WHERE deleted_at IS NULL
		ORDER BY usage_count DESC
		LIMIT 10
	`)
	if err == nil {
		defer programRows.Close()
		for programRows.Next() {
			var item model.TopItem
			var likes int
			programRows.Scan(&item.ID, &item.Name, &item.Count, &likes)
			item.Metric = "uses"
			item.MetricValue = item.Count
			topContent.Programs = append(topContent.Programs, item)
		}
	}

	// Top users par workouts
	userRows, err := database.DB.Query(ctx, `
		SELECT u.id, u.name, u.avatar, COUNT(ws.id) as workout_count
		FROM users u
		JOIN workout_sessions ws ON u.id = ws.user_id
		WHERE u.deleted_at IS NULL
		GROUP BY u.id, u.name, u.avatar
		ORDER BY workout_count DESC
		LIMIT 10
	`)
	if err == nil {
		defer userRows.Close()
		for userRows.Next() {
			var item model.TopItem
			var avatar sql.NullString
			userRows.Scan(&item.ID, &item.Name, &avatar, &item.Count)
			item.Avatar = utils.NullStringToPointer(avatar)
			item.Metric = "workouts"
			item.MetricValue = item.Count
			topContent.Users = append(topContent.Users, item)
		}
	}

	utils.Success(w, topContent)
}

// GetAdminAnalytics retourne les données analytiques pour les graphiques
func GetAdminAnalytics(w http.ResponseWriter, r *http.Request) {
	if !middleware.IsAdmin(r) {
		utils.ErrorSimple(w, http.StatusForbidden, "admin privileges required")
		return
	}

	ctx := context.Background()
	period := r.URL.Query().Get("period") // "7d", "30d", "90d", "1y"
	if period == "" {
		period = "30d"
	}

	var days int
	switch period {
	case "7d":
		days = 7
	case "30d":
		days = 30
	case "90d":
		days = 90
	case "1y":
		days = 365
	default:
		days = 30
	}

	analytics := model.AdminAnalytics{}

	// Croissance des utilisateurs
	userGrowthQuery := `
		SELECT
			DATE(created_at) as date,
			COUNT(*) as count
		FROM users
		WHERE created_at > NOW() - INTERVAL '1 day' * $1
		  AND deleted_at IS NULL
		GROUP BY DATE(created_at)
		ORDER BY date ASC
	`
	rows, err := database.DB.Query(ctx, userGrowthQuery, days)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var dp model.DataPoint
			var date time.Time
			rows.Scan(&date, &dp.Value)
			dp.Date = date.Format("2006-01-02")
			analytics.UserGrowth = append(analytics.UserGrowth, dp)
		}
	}

	// Activité des workouts
	workoutActivityQuery := `
		SELECT
			DATE(created_at) as date,
			COUNT(*) as count
		FROM workout_sessions
		WHERE created_at > NOW() - INTERVAL '1 day' * $1
		GROUP BY DATE(created_at)
		ORDER BY date ASC
	`
	rows, err = database.DB.Query(ctx, workoutActivityQuery, days)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var dp model.DataPoint
			var date time.Time
			rows.Scan(&date, &dp.Value)
			dp.Date = date.Format("2006-01-02")
			analytics.WorkoutActivity = append(analytics.WorkoutActivity, dp)
		}
	}

	// Stats des challenges (complétions)
	challengeStatsQuery := `
		SELECT
			DATE(uctp.completed_at) as date,
			COUNT(*) as count
		FROM user_challenge_task_progress uctp
		WHERE uctp.completed = true
		  AND uctp.completed_at > NOW() - INTERVAL '1 day' * $1
		GROUP BY DATE(uctp.completed_at)
		ORDER BY date ASC
	`
	rows, err = database.DB.Query(ctx, challengeStatsQuery, days)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var dp model.DataPoint
			var date time.Time
			rows.Scan(&date, &dp.Value)
			dp.Date = date.Format("2006-01-02")
			analytics.ChallengeStats = append(analytics.ChallengeStats, dp)
		}
	}

	utils.Success(w, analytics)
}

// GetAdminUsers retourne la liste des utilisateurs avec options de filtrage
func GetAdminUsers(w http.ResponseWriter, r *http.Request) {
	if !middleware.IsAdmin(r) {
		utils.ErrorSimple(w, http.StatusForbidden, "admin privileges required")
		return
	}

	ctx := context.Background()
	query := r.URL.Query()

	// Pagination
	limit := 50
	offset := 0
	if limitStr := query.Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	if offsetStr := query.Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Filtres
	search := query.Get("search")
	sortBy := query.Get("sort") // "name", "email", "score", "workouts", "joined"
	if sortBy == "" {
		sortBy = "joined"
	}

	sqlQuery := `
		SELECT
			u.id, u.name, u.email, u.avatar, u.is_admin,
			COALESCE(u.score, 0) as score,
			COUNT(DISTINCT ws.id) as total_workouts,
			COALESCE(SUM(ws.total_reps), 0) as total_pushups,
			u.created_at as join_date,
			MAX(ws.created_at) as last_active,
			CASE
				WHEN u.deleted_at IS NOT NULL THEN 'deleted'
				ELSE 'active'
			END as status
		FROM users u
		LEFT JOIN workout_sessions ws ON u.id = ws.user_id
		WHERE 1=1
	`

	args := []interface{}{}
	argCount := 1

	if search != "" {
		sqlQuery += fmt.Sprintf(" AND (u.name ILIKE $%d OR u.email ILIKE $%d)", argCount, argCount)
		args = append(args, "%"+search+"%")
		argCount++
	}

	sqlQuery += " GROUP BY u.id, u.name, u.email, u.avatar, u.is_admin, u.score, u.created_at, u.deleted_at"

	// Tri
	switch sortBy {
	case "name":
		sqlQuery += " ORDER BY u.name ASC"
	case "email":
		sqlQuery += " ORDER BY u.email ASC"
	case "score":
		sqlQuery += " ORDER BY score DESC"
	case "workouts":
		sqlQuery += " ORDER BY total_workouts DESC"
	default:
		sqlQuery += " ORDER BY join_date DESC"
	}

	sqlQuery += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argCount, argCount+1)
	args = append(args, limit, offset)

	rows, err := database.DB.Query(ctx, sqlQuery, args...)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not fetch users", err)
		return
	}
	defer rows.Close()

	var users []model.AdminUserListItem
	for rows.Next() {
		var user model.AdminUserListItem
		var avatar sql.NullString
		var lastActive sql.NullTime
		err := rows.Scan(
			&user.ID, &user.Name, &user.Email, &avatar, &user.IsAdmin,
			&user.Score, &user.TotalWorkouts, &user.TotalPushups,
			&user.JoinDate, &lastActive, &user.Status,
		)
		if err != nil {
			continue
		}
		user.Avatar = utils.NullStringToPointer(avatar)
		user.LastActive = utils.NullTimeToPointer(lastActive)
		users = append(users, user)
	}

	// Compter le total
	var total int
	countQuery := "SELECT COUNT(*) FROM users WHERE 1=1"
	if search != "" {
		countQuery += " AND (name ILIKE $1 OR email ILIKE $1)"
		database.DB.QueryRow(ctx, countQuery, "%"+search+"%").Scan(&total)
	} else {
		database.DB.QueryRow(ctx, countQuery).Scan(&total)
	}

	result := map[string]interface{}{
		"users": users,
		"pagination": map[string]interface{}{
			"total":  total,
			"limit":  limit,
			"offset": offset,
			"count":  len(users),
		},
	}

	utils.Success(w, result)
}

// PromoteUserToAdmin promeut un utilisateur en admin
func PromoteUserToAdmin(w http.ResponseWriter, r *http.Request) {
	if !middleware.IsAdmin(r) {
		utils.ErrorSimple(w, http.StatusForbidden, "admin privileges required")
		return
	}

	vars := mux.Vars(r)
	userID := vars["userId"]

	if userID == "" {
		utils.ErrorSimple(w, http.StatusBadRequest, "user ID required")
		return
	}

	ctx := context.Background()
	_, err := database.DB.Exec(ctx, "UPDATE users SET is_admin = true WHERE id = $1", userID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not promote user", err)
		return
	}

	utils.Message(w, "user promoted to admin successfully")
}

// DemoteUserFromAdmin retire les privilèges admin d'un utilisateur
func DemoteUserFromAdmin(w http.ResponseWriter, r *http.Request) {
	if !middleware.IsAdmin(r) {
		utils.ErrorSimple(w, http.StatusForbidden, "admin privileges required")
		return
	}

	vars := mux.Vars(r)
	userID := vars["userId"]

	if userID == "" {
		utils.ErrorSimple(w, http.StatusBadRequest, "user ID required")
		return
	}

	ctx := context.Background()
	_, err := database.DB.Exec(ctx, "UPDATE users SET is_admin = false WHERE id = $1", userID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not demote user", err)
		return
	}

	utils.Message(w, "admin privileges removed successfully")
}

// DeleteUserPermanently supprime définitivement un utilisateur (soft delete)
func DeleteUserPermanently(w http.ResponseWriter, r *http.Request) {
	if !middleware.IsAdmin(r) {
		utils.ErrorSimple(w, http.StatusForbidden, "admin privileges required")
		return
	}

	vars := mux.Vars(r)
	userID := vars["userId"]

	if userID == "" {
		utils.ErrorSimple(w, http.StatusBadRequest, "user ID required")
		return
	}

	// Récupérer l'ID de l'admin qui effectue la suppression
	adminID, err := middleware.GetUserIDFromContext(r)
	if err != nil {
		utils.ErrorSimple(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	ctx := context.Background()
	_, err = database.DB.Exec(ctx,
		"UPDATE users SET deleted_at = NOW(), deleted_by = $1 WHERE id = $2",
		adminID, userID,
	)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not delete user", err)
		return
	}

	utils.Message(w, "user deleted successfully")
}

// AdminUpdateUser permet à un admin de modifier n'importe quel utilisateur
func AdminUpdateUser(w http.ResponseWriter, r *http.Request) {
	if !middleware.IsAdmin(r) {
		utils.ErrorSimple(w, http.StatusForbidden, "admin privileges required")
		return
	}

	vars := mux.Vars(r)
	userID := vars["userId"]

	if userID == "" {
		utils.ErrorSimple(w, http.StatusBadRequest, "user ID required")
		return
	}

	// Structure pour recevoir les données de mise à jour (avec champs optionnels)
	type UpdateUserRequest struct {
		Name     *string  `json:"name"`
		Email    *string  `json:"email"`
		Avatar   *string  `json:"avatar"`
		Age      *int     `json:"age"`
		Weight   *float64 `json:"weight"`
		Height   *float64 `json:"height"`
		Goal     *string  `json:"goal"`
		Score    *int     `json:"score"`
		IsAdmin  *bool    `json:"isAdmin"`
		Password *string  `json:"password"` // Optionnel, ignoré si non fourni
	}

	var req UpdateUserRequest
	if err := utils.DecodeJSON(r, &req); err != nil {
		utils.Error(w, http.StatusBadRequest, "JSON invalide", err)
		return
	}

	adminID, err := middleware.GetUserIDFromContext(r)
	if err != nil {
		utils.ErrorSimple(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	ctx := context.Background()

	// Construire la requête SQL dynamiquement en fonction des champs fournis
	updateFields := []string{}
	args := []interface{}{}
	argCount := 1

	if req.Name != nil {
		updateFields = append(updateFields, fmt.Sprintf("name = $%d", argCount))
		args = append(args, *req.Name)
		argCount++
	}
	if req.Email != nil {
		updateFields = append(updateFields, fmt.Sprintf("email = $%d", argCount))
		args = append(args, *req.Email)
		argCount++
	}
	if req.Avatar != nil {
		updateFields = append(updateFields, fmt.Sprintf("avatar = $%d", argCount))
		args = append(args, *req.Avatar)
		argCount++
	}
	if req.Age != nil {
		updateFields = append(updateFields, fmt.Sprintf("age = $%d", argCount))
		args = append(args, *req.Age)
		argCount++
	}
	if req.Weight != nil {
		updateFields = append(updateFields, fmt.Sprintf("weight = $%d", argCount))
		args = append(args, *req.Weight)
		argCount++
	}
	if req.Height != nil {
		updateFields = append(updateFields, fmt.Sprintf("height = $%d", argCount))
		args = append(args, *req.Height)
		argCount++
	}
	if req.Goal != nil {
		updateFields = append(updateFields, fmt.Sprintf("goal = $%d", argCount))
		args = append(args, *req.Goal)
		argCount++
	}
	if req.Score != nil {
		updateFields = append(updateFields, fmt.Sprintf("score = $%d", argCount))
		args = append(args, *req.Score)
		argCount++
	}
	if req.IsAdmin != nil {
		updateFields = append(updateFields, fmt.Sprintf("is_admin = $%d", argCount))
		args = append(args, *req.IsAdmin)
		argCount++
	}
	if req.Password != nil && *req.Password != "" {
		// Hash le mot de passe si fourni
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if err != nil {
			utils.Error(w, http.StatusInternalServerError, "could not hash password", err)
			return
		}
		updateFields = append(updateFields, fmt.Sprintf("password = $%d", argCount))
		args = append(args, string(hashedPassword))
		argCount++
	}

	if len(updateFields) == 0 {
		utils.ErrorSimple(w, http.StatusBadRequest, "no fields to update")
		return
	}

	// Ajouter les champs updated_at et updated_by
	updateFields = append(updateFields, fmt.Sprintf("updated_at = NOW()"))
	updateFields = append(updateFields, fmt.Sprintf("updated_by = $%d", argCount))
	args = append(args, adminID)
	argCount++

	// Ajouter l'ID de l'utilisateur à modifier
	args = append(args, userID)

	query := fmt.Sprintf(
		"UPDATE users SET %s WHERE id = $%d AND deleted_at IS NULL",
		strings.Join(updateFields, ", "),
		argCount,
	)

	_, err = database.DB.Exec(ctx, query, args...)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not update user", err)
		return
	}

	// Récupérer l'utilisateur mis à jour
	var updatedUser model.UserProfile
	err = database.DB.QueryRow(ctx, `
		SELECT id, name, email, avatar, age, weight, height, goal, score, is_admin,
		       join_date, created_at, updated_at, created_by, updated_by
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`, userID).Scan(
		&updatedUser.ID, &updatedUser.Name, &updatedUser.Email, &updatedUser.Avatar,
		&updatedUser.Age, &updatedUser.Weight, &updatedUser.Height, &updatedUser.Goal,
		&updatedUser.Score, &updatedUser.IsAdmin, &updatedUser.JoinDate,
		&updatedUser.CreatedAt, &updatedUser.UpdatedAt, &updatedUser.CreatedBy, &updatedUser.UpdatedBy,
	)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not fetch updated user", err)
		return
	}

	utils.Success(w, updatedUser)
}

// AdminDeleteUser permet à un admin de supprimer n'importe quel utilisateur
func AdminDeleteUser(w http.ResponseWriter, r *http.Request) {
	if !middleware.IsAdmin(r) {
		utils.ErrorSimple(w, http.StatusForbidden, "admin privileges required")
		return
	}

	vars := mux.Vars(r)
	userID := vars["userId"]

	if userID == "" {
		utils.ErrorSimple(w, http.StatusBadRequest, "user ID required")
		return
	}

	adminID, err := middleware.GetUserIDFromContext(r)
	if err != nil {
		utils.ErrorSimple(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Empêcher un admin de se supprimer lui-même
	if adminID == userID {
		utils.ErrorSimple(w, http.StatusBadRequest, "cannot delete yourself")
		return
	}

	ctx := context.Background()
	_, err = database.DB.Exec(ctx,
		"UPDATE users SET deleted_at = NOW(), deleted_by = $1 WHERE id = $2 AND deleted_at IS NULL",
		adminID, userID,
	)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not delete user", err)
		return
	}

	utils.Message(w, "user deleted successfully")
}
