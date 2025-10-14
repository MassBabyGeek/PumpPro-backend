package handler

import (
	"context"
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/MassBabyGeek/PumpPro-backend/internal/database"
	"github.com/MassBabyGeek/PumpPro-backend/internal/middleware"
	model "github.com/MassBabyGeek/PumpPro-backend/internal/models"
	"github.com/MassBabyGeek/PumpPro-backend/internal/scanner"
	"github.com/MassBabyGeek/PumpPro-backend/internal/utils"
	"github.com/gorilla/mux"

	"golang.org/x/crypto/bcrypt"
)

func CreateUser(w http.ResponseWriter, r *http.Request) {
	var user model.UserProfile
	if err := utils.DecodeJSON(r, &user); err != nil {
		utils.Error(w, http.StatusBadRequest, "JSON invalide", err)
		return
	}

	// Récupérer l'ID de l'utilisateur qui crée (à adapter selon votre système d'auth)
	// Pour l'instant, on utilise l'ID de l'utilisateur lui-même lors de la création
	ctx := context.Background()
	password := "password"
	hashed, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	err := database.DB.QueryRow(ctx,
		`INSERT INTO users(name,email,password_hash,avatar,age,weight,height,goal,join_date,created_at,updated_at,created_by)
		 VALUES($1,$2,$3,$4,$5,$6,$7,$8,NOW(),NOW(),NOW(),$9)
		 RETURNING id, join_date, created_at, updated_at, created_by`,
		user.Name, user.Email, string(hashed), user.Avatar, user.Age,
		user.Weight, user.Height, user.Goal, user.CreatedBy,
	).Scan(&user.ID, &user.JoinDate, &user.CreatedAt, &user.UpdatedAt, &user.CreatedBy)

	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not create user", err)
		return
	}

	utils.Success(w, user)
}

func UpdateUser(w http.ResponseWriter, r *http.Request) {
	var user model.UserProfile
	if err := utils.DecodeJSON(r, &user); err != nil {
		utils.Error(w, http.StatusBadRequest, "JSON invalide", err)
		return
	}

	vars := mux.Vars(r)
	userId := vars["id"]
	if userId == "" {
		utils.ErrorSimple(w, http.StatusBadRequest, "ID utilisateur manquant")
		return
	}

	user, err := middleware.GetUserFromContext(r)
	if err != nil {
		utils.Error(w, http.StatusUnauthorized, "impossible de récupérer l'utilisateur", err)
		return
	}

	if user.ID != userId {
		utils.ErrorSimple(w, http.StatusUnauthorized, "impossible de modifier l'utilisateur")
		return
	}

	ctx := context.Background()
	_, err = database.DB.Exec(ctx,
		`UPDATE users
		 SET name = COALESCE(NULLIF($1, ''), name),
		     avatar = COALESCE(NULLIF($2, ''), avatar),
		     age = COALESCE($3, age),
		     weight = COALESCE($4, weight),
		     height = COALESCE($5, height),
		     goal = COALESCE(NULLIF($6, ''), goal),
		     updated_at = NOW(),
		     updated_by = $7
		 WHERE id = $8 AND deleted_at IS NULL`,
		user.Name, user.Avatar, user.Age, user.Weight, user.Height, user.Goal,
		user.ID, // ici UpdatedBy = userID pour l'instant
		user.ID,
	)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not update user", err)
		return
	}

	utils.Success(w, user)
}

func GetUsers(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	rows, err := database.DB.Query(ctx, `
		SELECT
			id, name, email, avatar, age, weight, height, goal,
			join_date, created_at, updated_at,
			created_by, updated_by
		FROM users
		WHERE deleted_at IS NULL
	`)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not query users", err)
		return
	}
	defer rows.Close()

	var users []model.UserProfile
	for rows.Next() {
		user, err := scanner.ScanUserProfile(rows)
		if err != nil {
			utils.Error(w, http.StatusInternalServerError, "could not scan user row", err)
			return
		}
		users = append(users, *user)
	}

	utils.Success(w, users)
}

func GetUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	ctx := context.Background()

	row := database.DB.QueryRow(ctx,
		`SELECT id, name, email, avatar, age, weight, height, goal,
			 join_date, created_at, updated_at,
			 created_by, updated_by
		 FROM users WHERE id=$1 AND deleted_at IS NULL`,
		id,
	)

	user, err := scanner.ScanUserProfile(row)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not get user", err)
		return
	}

	utils.Success(w, user)
}

func DeleteUser(w http.ResponseWriter, r *http.Request) {
	user, err := middleware.GetUserFromContext(r)
	if err != nil {
		utils.Error(w, http.StatusUnauthorized, "impossible de récupérer l'utilisateur", err)
		return
	}

	vars := mux.Vars(r)
	userId := vars["id"]
	if userId == "" {
		utils.ErrorSimple(w, http.StatusBadRequest, "ID utilisateur manquant")
		return
	}

	if user.ID != userId {
		utils.ErrorSimple(w, http.StatusUnauthorized, "impossible de supprimer l'utilisateur")
		return
	}

	// Soft delete: on met à jour deleted_at et deleted_by au lieu de supprimer
	ctx := context.Background()
	res, err := database.DB.Exec(ctx,
		`UPDATE users SET deleted_at=NOW(), deleted_by=$2
		 WHERE id=$1 AND deleted_at IS NULL`,
		user.ID, user.ID,
	)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "impossible de supprimer l'utilisateur", err)
		return
	}

	if res.RowsAffected() == 0 {
		utils.ErrorSimple(w, http.StatusNotFound, "utilisateur introuvable ou déjà supprimé")
		return
	}

	utils.Success(w, map[string]bool{"success": true})
}

// GetUserStats récupère les statistiques d'un utilisateur
func GetUserStats(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userId := vars["userId"]
	period := vars["period"]

	if userId == "" {
		utils.ErrorSimple(w, http.StatusBadRequest, "ID utilisateur manquant")
		return
	}

	if period == "" {
		utils.ErrorSimple(w, http.StatusBadRequest, "période manquante")
		return
	}

	var startDate, endDate time.Time
	now := time.Now()

	switch period {
	case "daily", "today":
		// Début et fin de la journée
		startDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		endDate = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, now.Location())

	case "weekly", "week":
		// Début de la semaine (lundi) et fin (dimanche)
		weekday := int(now.Weekday())
		if weekday == 0 { // dimanche
			weekday = 7
		}
		startDate = time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, now.Location())
		endDate = startDate.AddDate(0, 0, 6).Add(time.Hour*23 + time.Minute*59 + time.Second*59)

	case "monthly", "month":
		// Début et fin du mois
		startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		endDate = startDate.AddDate(0, 1, 0).Add(-time.Nanosecond)

	case "yearly", "year":
		// Début et fin de l'année
		startDate = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		endDate = time.Date(now.Year(), 12, 31, 23, 59, 59, 999999999, now.Location())

	default:
		utils.ErrorSimple(w, http.StatusBadRequest, "période invalide")
		return
	}

	ctx := context.Background()

	row := database.DB.QueryRow(ctx, `
		SELECT
			COUNT(*) as totalWorkouts,
			COALESCE(SUM(total_reps), 0) as totalPushUps,
			COALESCE(SUM(total_duration), 0) as totalTime,
			COALESCE(MAX(total_reps), 0) as bestSession,
			0 as totalCalories,
			COALESCE(AVG(total_reps), 0) as averagePushUps
		FROM workout_sessions
		WHERE user_id = $1 AND start_time >= $2 AND start_time <= $3
	`, userId, startDate, endDate)

	stats, err := scanner.ScanStats(row)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not fetch stats", err)
		return
	}

	// Calculs dérivés
	if stats.TotalWorkouts > 0 {
		stats.AveragePushUps = float64(stats.TotalPushUps) / float64(stats.TotalWorkouts)
	}
	stats.TotalCalories = float64(stats.TotalPushUps) * 0.29

	utils.Success(w, stats)
}

// GetChartData récupère les données pour les graphiques
func GetChartData(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]
	period := vars["period"] // week, month, year

	ctx := context.Background()

	var days int
	switch period {
	case "week":
		days = 7
	case "month":
		days = 30
	case "year":
		days = 365
	default:
		days = 7
	}

	// Générer les dates pour la période
	rows, err := database.DB.Query(ctx, `
		WITH date_range AS (
			SELECT generate_series(
				CURRENT_DATE - INTERVAL '1 day' * $1,
				CURRENT_DATE,
				INTERVAL '1 day'
			)::date as date
		)
		SELECT
			dr.date,
			COALESCE(SUM(ws.total_reps), 0) as total_reps,
			COALESCE(SUM(ws.total_duration), 0) as total_duration,
			COALESCE(SUM(ws.total_reps) * 0.29, 0) as calories
		FROM date_range dr
		LEFT JOIN workout_sessions ws ON DATE(ws.start_time) = dr.date AND ws.user_id = $2
		GROUP BY dr.date
		ORDER BY dr.date
	`, days, userID)

	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not fetch chart data", err)
		return
	}
	defer rows.Close()

	type DayData struct {
		Date     string  `json:"date"`
		PushUps  int     `json:"pushUps"`
		Duration int     `json:"duration"`
		Calories float64 `json:"calories"`
	}

	var chartData []DayData
	for rows.Next() {
		var date string
		var data DayData
		var pushUps, duration sql.NullInt64
		var calories sql.NullFloat64

		if err := rows.Scan(&date, &pushUps, &duration, &calories); err != nil {
			utils.Error(w, http.StatusInternalServerError, "could not scan chart data", err)
			return
		}

		data.Date = date
		data.PushUps = utils.NullInt64ToInt(pushUps)
		data.Duration = utils.NullInt64ToInt(duration)
		data.Calories = utils.NullFloat64ToFloat64(calories)
		chartData = append(chartData, data)
	}

	utils.Success(w, chartData)
}

// UploadAvatar gère l'upload d'avatar utilisateur
func UploadAvatar(w http.ResponseWriter, r *http.Request) {
	user, err := middleware.GetUserFromContext(r)
	if err != nil {
		utils.Error(w, http.StatusUnauthorized, "impossible de récupérer l'utilisateur", err)
		return
	}

	vars := mux.Vars(r)
	userId := vars["id"]
	if userId == "" {
		utils.ErrorSimple(w, http.StatusBadRequest, "ID utilisateur manquant")
		return
	}

	if user.ID != userId {
		utils.ErrorSimple(w, http.StatusUnauthorized, "impossible de modifier l'utilisateur")
		return
	}

	// Limiter la taille du fichier à 10MB
	r.ParseMultipartForm(10 << 20)

	file, handler, err := r.FormFile("avatar")
	if err != nil {
		utils.Error(w, http.StatusBadRequest, "aucun fichier uploadé", err)
		return
	}
	defer file.Close()

	// Vérifier le type de fichier
	contentType := handler.Header.Get("Content-Type")
	if contentType != "image/jpeg" && contentType != "image/png" && contentType != "image/jpg" {
		utils.ErrorSimple(w, http.StatusBadRequest, "seules les images JPEG et PNG sont autorisées")
		return
	}

	// TODO: En production, uploader le fichier vers un service de stockage (S3, Cloud Storage, etc.)
	// Pour l'instant, on simule l'URL
	avatarURL := "https://api.pompeurpro.com/avatars/" + user.ID + ".jpg"

	ctx := context.Background()

	// Mettre à jour l'avatar dans la base de données
	_, err = database.DB.Exec(ctx,
		`UPDATE users SET avatar=$1, updated_at=NOW() WHERE id=$2 AND deleted_at IS NULL`,
		avatarURL, user.ID,
	)

	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not update avatar", err)
		return
	}

	// Récupérer le profil mis à jour
	row := database.DB.QueryRow(ctx, `
		SELECT id, name, email, avatar, age, weight, height, goal,
		       join_date, created_at, updated_at, created_by, updated_by
		FROM users WHERE id=$1 AND deleted_at IS NULL
	`, user.ID)

	updatedUser, err := scanner.ScanUserProfile(row)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not fetch updated user", err)
		return
	}

	utils.Success(w, updatedUser)
}

func GetUsersWorkoutSessions(w http.ResponseWriter, r *http.Request) {
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
		session, err := scanner.ScanWorkoutSession(rows)
		if err != nil {
			utils.Error(w, http.StatusInternalServerError, "could not scan session row", err)
			return
		}
		sessions = append(sessions, *session)
	}

	// Charger les sets pour chaque session
	for i := range sessions {
		setRows, err := database.DB.Query(ctx, `
			SELECT id, session_id, set_number, target_reps, completed_reps, duration, timestamp
			FROM set_results
			WHERE session_id = $1
			ORDER BY set_number ASC
		`, sessions[i].ID)

		if err != nil {
			utils.Error(w, http.StatusInternalServerError, "could not query set results", err)
			return
		}

		var sets []model.SetResult
		for setRows.Next() {
			set, err := scanner.ScanSetResult(setRows)
			if err != nil {
				setRows.Close()
				utils.Error(w, http.StatusInternalServerError, "could not scan set result", err)
				return
			}
			sets = append(sets, *set)
		}
		setRows.Close()
		sessions[i].Sets = sets
	}

	utils.Success(w, sessions)
}