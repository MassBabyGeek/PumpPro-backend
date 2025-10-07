package handler

import (
	"context"
	"net/http"

	"github.com/MassBabyGeek/PumpPro-backend/internal/database"
	model "github.com/MassBabyGeek/PumpPro-backend/internal/models"
	"github.com/MassBabyGeek/PumpPro-backend/internal/utils"
	"github.com/gorilla/mux"

	"golang.org/x/crypto/bcrypt"
)

func CreateUser(w http.ResponseWriter, r *http.Request) {
	var user model.UserProfile
	if err := utils.DecodeJSON(r, &user); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid JSON body")
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
		utils.Error(w, http.StatusInternalServerError, "could not create user: "+err.Error())
		return
	}

	utils.Success(w, user)
}

func UpdateUser(w http.ResponseWriter, r *http.Request) {
	var user model.UserProfile
	if err := utils.DecodeJSON(r, &user); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	// Récupérer l'ID de l'utilisateur qui modifie (à adapter selon votre système d'auth)
	ctx := context.Background()
	_, err := database.DB.Exec(ctx,
		`UPDATE users SET name=$1, avatar=$2, age=$3, weight=$4, height=$5, goal=$6, updated_at=NOW(), updated_by=$8
		 WHERE id=$7 AND deleted_at IS NULL`,
		user.Name, user.Avatar, user.Age, user.Weight, user.Height, user.Goal, user.ID, user.UpdatedBy,
	)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not update user: "+err.Error())
		return
	}

	utils.Success(w, user)
}

func GetUsers(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	rows, err := database.DB.Query(ctx, `
		SELECT
			id, name, email,
			COALESCE(avatar,'') AS avatar,
			age, weight, height,
			COALESCE(goal,'') AS goal,
			join_date, created_at, updated_at,
			created_by, updated_by, deleted_at, deleted_by
		FROM users
		WHERE deleted_at IS NULL
	`)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not query users: "+err.Error())
		return
	}
	defer rows.Close()

	var users []model.UserProfile
	for rows.Next() {
		var u model.UserProfile
		if err := rows.Scan(
			&u.ID, &u.Name, &u.Email, &u.Avatar,
			&u.Age, &u.Weight, &u.Height, &u.Goal,
			&u.JoinDate, &u.CreatedAt, &u.UpdatedAt,
			&u.CreatedBy, &u.UpdatedBy, &u.DeletedAt, &u.DeletedBy,
		); err != nil {
			utils.Error(w, http.StatusInternalServerError, "could not scan user row: "+err.Error())
			return
		}
		users = append(users, u)
	}

	utils.Success(w, users)
}

func GetUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	ctx := context.Background()
	var user model.UserProfile
	err := database.DB.QueryRow(ctx,
		`SELECT id, name, email,
			 COALESCE(avatar,'') AS avatar,
			 age, weight, height,
			 COALESCE(goal,'') AS goal,
			 join_date, created_at, updated_at,
			 created_by, updated_by, deleted_at, deleted_by
		 FROM users WHERE id=$1 AND deleted_at IS NULL`,
		id,
	).Scan(&user.ID, &user.Name, &user.Email, &user.Avatar,
		&user.Age, &user.Weight, &user.Height, &user.Goal,
		&user.JoinDate, &user.CreatedAt, &user.UpdatedAt,
		&user.CreatedBy, &user.UpdatedBy, &user.DeletedAt, &user.DeletedBy,
	)

	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not get user: "+err.Error())
		return
	}

	utils.Success(w, user)
}

func DeleteUser(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		ID        string  `json:"id"`
		DeletedBy *string `json:"deletedBy"`
	}
	if err := utils.DecodeJSON(r, &payload); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	// Soft delete: on met à jour deleted_at et deleted_by au lieu de supprimer
	ctx := context.Background()
	res, err := database.DB.Exec(ctx,
		`UPDATE users SET deleted_at=NOW(), deleted_by=$2
		 WHERE id=$1 AND deleted_at IS NULL`,
		payload.ID, payload.DeletedBy,
	)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not delete user: "+err.Error())
		return
	}

	if res.RowsAffected() == 0 {
		utils.Error(w, http.StatusNotFound, "user not found or already deleted")
		return
	}

	utils.Success(w, map[string]bool{"success": true})
}

// GetUserStats récupère les statistiques d'un utilisateur
func GetUserStats(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["id"]

	ctx := context.Background()

	// Calculer les stats globales
	var stats struct {
		TotalWorkouts   int     `json:"totalWorkouts"`
		TotalPushUps    int     `json:"totalPushUps"`
		TotalCalories   float64 `json:"totalCalories"`
		TotalTime       int     `json:"totalTime"`
		BestSession     int     `json:"bestSession"`
		AveragePushUps  float64 `json:"averagePushUps"`
		CurrentStreak   int     `json:"currentStreak"`
		LongestStreak   int     `json:"longestStreak"`
	}

	err := database.DB.QueryRow(ctx, `
		SELECT
			COUNT(*) as total_workouts,
			COALESCE(SUM(total_reps), 0) as total_push_ups,
			COALESCE(SUM(total_duration), 0) as total_time,
			COALESCE(MAX(total_reps), 0) as best_session
		FROM workout_sessions
		WHERE user_id = $1
	`, userID).Scan(
		&stats.TotalWorkouts,
		&stats.TotalPushUps,
		&stats.TotalTime,
		&stats.BestSession,
	)

	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not fetch stats: "+err.Error())
		return
	}

	// Calculs dérivés
	if stats.TotalWorkouts > 0 {
		stats.AveragePushUps = float64(stats.TotalPushUps) / float64(stats.TotalWorkouts)
	}
	stats.TotalCalories = float64(stats.TotalPushUps) * 0.29

	// TODO: Calculer les streaks (série de jours consécutifs)
	stats.CurrentStreak = 0
	stats.LongestStreak = 0

	utils.Success(w, stats)
}

// GetChartData récupère les données pour les graphiques
func GetChartData(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["id"]
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
		utils.Error(w, http.StatusInternalServerError, "could not fetch chart data: "+err.Error())
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
		if err := rows.Scan(&date, &data.PushUps, &data.Duration, &data.Calories); err != nil {
			utils.Error(w, http.StatusInternalServerError, "could not scan chart data: "+err.Error())
			return
		}
		data.Date = date
		chartData = append(chartData, data)
	}

	utils.Success(w, chartData)
}

// UploadAvatar gère l'upload d'avatar utilisateur
func UploadAvatar(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["id"]

	// Limiter la taille du fichier à 10MB
	r.ParseMultipartForm(10 << 20)

	file, handler, err := r.FormFile("avatar")
	if err != nil {
		utils.Error(w, http.StatusBadRequest, "no file uploaded")
		return
	}
	defer file.Close()

	// Vérifier le type de fichier
	contentType := handler.Header.Get("Content-Type")
	if contentType != "image/jpeg" && contentType != "image/png" && contentType != "image/jpg" {
		utils.Error(w, http.StatusBadRequest, "only JPEG and PNG images are allowed")
		return
	}

	// TODO: En production, uploader le fichier vers un service de stockage (S3, Cloud Storage, etc.)
	// Pour l'instant, on simule l'URL
	avatarURL := "https://api.pompeurpro.com/avatars/" + userID + ".jpg"

	ctx := context.Background()

	// Mettre à jour l'avatar dans la base de données
	_, err = database.DB.Exec(ctx,
		`UPDATE users SET avatar=$1, updated_at=NOW() WHERE id=$2 AND deleted_at IS NULL`,
		avatarURL, userID,
	)

	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not update avatar: "+err.Error())
		return
	}

	// Récupérer le profil mis à jour
	var user model.UserProfile
	err = database.DB.QueryRow(ctx, `
		SELECT id, name, email, COALESCE(avatar,'') as avatar, age, weight, height, COALESCE(goal,'') as goal,
		       join_date, created_at, updated_at
		FROM users WHERE id=$1 AND deleted_at IS NULL
	`, userID).Scan(
		&user.ID, &user.Name, &user.Email, &user.Avatar, &user.Age, &user.Weight, &user.Height,
		&user.Goal, &user.JoinDate, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not fetch updated user: "+err.Error())
		return
	}

	utils.Success(w, user)
}
