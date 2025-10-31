package handler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/MassBabyGeek/PumpPro-backend/internal/config"
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
		`INSERT INTO users(name,email,password_hash,avatar,age,weight,height,goal,score,join_date,created_at,updated_at,created_by)
		 VALUES($1,$2,$3,$4,$5,$6,$7,$8,0,NOW(),NOW(),NOW(),$9)
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

	userFromContext, err := middleware.GetUserFromContext(r)
	if err != nil {
		utils.Error(w, http.StatusUnauthorized, "impossible de récupérer l'utilisateur", err)
		return
	}

	if userFromContext.ID != userId {
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
		     email = COALESCE(NULLIF($7, ''), email),
		     updated_at = NOW(),
		     updated_by = $8
		 WHERE id = $9 AND deleted_at IS NULL`,
		user.Name, user.Avatar, user.Age, user.Weight, user.Height, user.Goal, user.Email,
		userFromContext.ID, userFromContext.ID,
	)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not update user", err)
		return
	}

	utils.Success(w, user)
}

func GetUsers(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	query := r.URL.Query()
	limitStr := query.Get("limit")
	offsetStr := query.Get("offset")

	sqlQuery := `
		SELECT
			id, name, email, avatar, age, weight, height, goal, score, is_admin,
			join_date, created_at, updated_at,
			created_by, updated_by
		FROM users
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC
	`

	args := []interface{}{}
	argCount := 1

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
		`SELECT id, name, email, avatar, age, weight, height, goal, score, is_admin,
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

	// Vérifier si l'utilisateur est admin OU s'il supprime son propre compte
	if !middleware.IsOwnerOrAdmin(r, userId) {
		utils.ErrorSimple(w, http.StatusForbidden, "impossible de supprimer l'utilisateur")
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

	// Vérifier que l'utilisateur est admin OU consulte ses propres stats
	if !middleware.IsOwnerOrAdmin(r, userId) {
		utils.ErrorSimple(w, http.StatusForbidden, "you can only view your own stats unless you are an admin")
		return
	}

	var startDate, endDate time.Time
	now := time.Now()

	switch period {
	case "daily", "today":
		startDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		endDate = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, now.Location())

	case "weekly", "week":
		weekday := int(now.Weekday())
		if weekday == 0 { // dimanche
			weekday = 7
		}
		startDate = time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, now.Location())
		endDate = startDate.AddDate(0, 0, 6).Add(time.Hour*23 + time.Minute*59 + time.Second*59)

	case "monthly", "month":
		startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		endDate = startDate.AddDate(0, 1, 0).Add(-time.Nanosecond)

	case "yearly", "year":
		startDate = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		endDate = time.Date(now.Year(), 12, 31, 23, 59, 59, 999999999, now.Location())

	case "all-time":
		startDate = time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC)
		endDate = time.Date(9999, 12, 31, 23, 59, 59, 999999999, time.UTC)

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

// GetChartData récupère les données des pompes par période : semaine, mois, année ou total
func GetChartData(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]
	period := vars["period"] // "week", "month", "year", "total"

	ctx := context.Background()

	var query string
	var args []interface{}

	switch period {
	case "week":
		query = `
			WITH date_range AS (
				SELECT generate_series(
					date_trunc('week', CURRENT_DATE),          -- lundi
					date_trunc('week', CURRENT_DATE) + interval '6 days', -- dimanche
					interval '1 day'
				)::date AS date
			)
			SELECT
				dr.date,
				COALESCE(SUM(ws.total_reps), 0) AS total_reps,
				COALESCE(SUM(ws.total_duration), 0) AS total_duration,
				COALESCE(SUM(ws.total_reps) * 0.29, 0) AS calories
			FROM date_range dr
			LEFT JOIN workout_sessions ws
				ON DATE(ws.start_time) = dr.date AND ws.user_id = $1
			GROUP BY dr.date
			ORDER BY dr.date;
		`
		args = append(args, userID)

	case "month":
		query = `
			WITH date_range AS (
				SELECT generate_series(
					date_trunc('month', CURRENT_DATE),
					date_trunc('month', CURRENT_DATE) + interval '1 month' - interval '1 day',
					interval '1 day'
				)::date AS date
			)
			SELECT
				dr.date,
				COALESCE(SUM(ws.total_reps), 0) AS total_reps,
				COALESCE(SUM(ws.total_duration), 0) AS total_duration,
				COALESCE(SUM(ws.total_reps) * 0.29, 0) AS calories
			FROM date_range dr
			LEFT JOIN workout_sessions ws
				ON DATE(ws.start_time) = dr.date AND ws.user_id = $1
			GROUP BY dr.date
			ORDER BY dr.date;

		`
		args = append(args, userID)

	case "year":
		query = `
			WITH month_range AS (
				SELECT generate_series(
					date_trunc('year', CURRENT_DATE),
					date_trunc('year', CURRENT_DATE) + interval '11 months',
					interval '1 month'
				)::date AS month_start
			)
			SELECT
				TO_CHAR(mr.month_start, 'YYYY-MM') AS date,
				COALESCE(SUM(ws.total_reps), 0) AS total_reps,
				COALESCE(SUM(ws.total_duration), 0) AS total_duration,
				COALESCE(SUM(ws.total_reps) * 0.29, 0) AS calories
			FROM month_range mr
			LEFT JOIN workout_sessions ws
				ON DATE_TRUNC('month', ws.start_time) = mr.month_start AND ws.user_id = $1
			GROUP BY mr.month_start
			ORDER BY mr.month_start;

		`
		args = append(args, userID)

	default:
		utils.Error(w, http.StatusBadRequest, "invalid period (use week, month, year, total)", nil)
		return
	}

	// Exécution de la requête
	rows, err := database.DB.Query(ctx, query, args...)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not fetch chart data", err)
		return
	}
	defer rows.Close()

	var chartDatas []model.ChartData

	for rows.Next() {
		chartData, err := scanner.ScanChartData(rows)
		if err != nil {
			utils.Error(w, http.StatusInternalServerError, "could not scan chart data", err)
			return
		}
		chartDatas = append(chartDatas, chartData)
	}

	utils.Success(w, chartDatas)
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

	// Vérifier si l'utilisateur est admin OU s'il modifie son propre profil
	if !middleware.IsOwnerOrAdmin(r, userId) {
		utils.ErrorSimple(w, http.StatusForbidden, "impossible de modifier l'utilisateur")
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

	// Déterminer l'extension du fichier
	ext := ".jpg"
	if contentType == "image/png" {
		ext = ".png"
	}

	// Créer le nom du fichier
	filename := user.ID + ext
	uploadDir := "uploads/avatars"

	// Créer le dossier s'il n'existe pas
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		utils.Error(w, http.StatusInternalServerError, "impossible de créer le dossier d'upload", err)
		return
	}

	// Supprimer les anciennes photos de cet utilisateur (jpg, png, jpeg, svg)
	oldExtensions := []string{".jpg", ".jpeg", ".png", ".svg"}
	for _, oldExt := range oldExtensions {
		oldFilePath := filepath.Join(uploadDir, user.ID+oldExt)
		if _, err := os.Stat(oldFilePath); err == nil {
			// Le fichier existe, on le supprime
			os.Remove(oldFilePath)
		}
	}

	// Créer le fichier de destination
	filepath := filepath.Join(uploadDir, filename)
	dst, err := os.Create(filepath)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "impossible de créer le fichier", err)
		return
	}
	defer dst.Close()

	// Copier le fichier uploadé vers la destination
	if _, err := io.Copy(dst, file); err != nil {
		utils.Error(w, http.StatusInternalServerError, "impossible de sauvegarder le fichier", err)
		return
	}

	// Charger la config pour récupérer l'URL de base
	cfg, err := config.LoadConfig()
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "impossible de charger la configuration", err)
		return
	}

	// Construire l'URL complète de l'avatar
	avatarURL := fmt.Sprintf("%s/avatars/%s", cfg.URL, filename)

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
		SELECT id, name, email, avatar, age, weight, height, goal, score,
		       is_admin, join_date, created_at, updated_at, created_by, updated_by
		FROM users WHERE id=$1 AND deleted_at IS NULL
	`, user.ID)

	updatedUser, err := scanner.ScanUserProfile(row)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not fetch updated user", err)
		return
	}

	utils.Success(w, updatedUser)
}

// GetAvatar sert l'image de profil d'un utilisateur
func GetAvatar(w http.ResponseWriter, r *http.Request) {
	// Ajouter les headers CORS explicitement pour les images
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// Gérer les requêtes preflight OPTIONS
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	vars := mux.Vars(r)
	filename := vars["filename"]

	if filename == "" {
		utils.ErrorSimple(w, http.StatusBadRequest, "nom de fichier manquant")
		return
	}

	// Nettoyer le filename pour extraire juste le nom du fichier (au cas où c'est une URL)
	// Exemple: si filename contient une URL complète, on extrait juste le nom du fichier
	if strings.Contains(filename, "/") {
		parts := strings.Split(filename, "/")
		filename = parts[len(parts)-1]
	}

	// Récupérer le chemin absolu vers le répertoire d'uploads
	cwd, err := os.Getwd()
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "impossible de déterminer le répertoire courant", err)
		return
	}

	// Construire le chemin complet du fichier
	filePath := filepath.Join(cwd, "uploads", "avatars", filename)

	// Vérifier que le fichier existe
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		utils.ErrorSimple(w, http.StatusNotFound, "image non trouvée")
		return
	}

	// Ouvrir le fichier
	file, err := os.Open(filePath)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "impossible d'ouvrir l'image", err)
		return
	}
	defer file.Close()

	// Déterminer le type MIME en fonction de l'extension
	ext := strings.ToLower(filepath.Ext(filename))
	var contentType string

	switch ext {
	case ".png":
		contentType = "image/png"
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".svg":
		contentType = "image/svg+xml"
	case ".webp":
		contentType = "image/webp"
	case ".gif":
		contentType = "image/gif"
	default:
		utils.ErrorSimple(w, http.StatusBadRequest, "format d'image non supporté")
		return
	}

	// Définir les en-têtes HTTP
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=86400") // Cache de 24h
	w.Header().Set("Access-Control-Expose-Headers", "Content-Type, Cache-Control")

	// Envoyer le contenu de l'image
	if _, err := io.Copy(w, file); err != nil {
		utils.Error(w, http.StatusInternalServerError, "erreur lors de l'envoi de l'image", err)
		return
	}
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

	// Get optional authenticated user
	user, _ := middleware.GetUserFromContext(r)
	var authenticatedUserID *string
	if user.ID != "" {
		authenticatedUserID = &user.ID
	}

	sqlQuery := `
		SELECT
			ws.id, ws.program_id, ws.user_id, ws.start_time, ws.end_time,
			ws.total_reps, ws.total_duration, ws.completed, ws.notes,
			COALESCE(ws.likes, 0) AS likes,
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
		WHERE ws.user_id = $2
	`

	args := []interface{}{authenticatedUserID, userID}
	argCount := 3

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

func GetUserStreak(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]

	ctx := context.Background()

	// Récupérer toutes les dates de workout distinctes, triées par ordre décroissant
	rows, err := database.DB.Query(ctx, `
		SELECT DISTINCT DATE(start_time) as workout_date
		FROM workout_sessions
		WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY workout_date DESC
	`, userID)

	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not query workout dates", err)
		return
	}
	defer rows.Close()

	var dates []time.Time
	for rows.Next() {
		var date time.Time
		if err := rows.Scan(&date); err != nil {
			utils.Error(w, http.StatusInternalServerError, "could not scan date", err)
			return
		}
		dates = append(dates, date)
	}

	// Calculer le streak
	currentStreak := 0
	maxStreak := 0
	var lastWorkoutDate *string

	if len(dates) > 0 {
		lastWorkoutDate = new(string)
		*lastWorkoutDate = dates[0].Format("2006-01-02")

		// Calculer le current streak
		today := time.Now().UTC().Truncate(24 * time.Hour)
		yesterday := today.AddDate(0, 0, -1)

		// Le streak commence si le dernier workout était aujourd'hui ou hier
		latestWorkout := dates[0].UTC().Truncate(24 * time.Hour)
		if latestWorkout.Equal(today) || latestWorkout.Equal(yesterday) {
			currentStreak = 1
			expectedDate := latestWorkout.AddDate(0, 0, -1)

			for i := 1; i < len(dates); i++ {
				currentDate := dates[i].UTC().Truncate(24 * time.Hour)
				if currentDate.Equal(expectedDate) {
					currentStreak++
					expectedDate = expectedDate.AddDate(0, 0, -1)
				} else {
					break
				}
			}
		}

		// Calculer le max streak
		tempStreak := 1
		maxStreak = 1
		expectedDate := dates[0].UTC().Truncate(24*time.Hour).AddDate(0, 0, -1)

		for i := 1; i < len(dates); i++ {
			currentDate := dates[i].UTC().Truncate(24 * time.Hour)
			if currentDate.Equal(expectedDate) {
				tempStreak++
				if tempStreak > maxStreak {
					maxStreak = tempStreak
				}
				expectedDate = expectedDate.AddDate(0, 0, -1)
			} else {
				tempStreak = 1
				expectedDate = currentDate.AddDate(0, 0, -1)
			}
		}
	}

	response := map[string]interface{}{
		"currentStreak": currentStreak,
		"maxStreak":     maxStreak,
	}

	if lastWorkoutDate != nil {
		response["lastWorkoutDate"] = *lastWorkoutDate
	}

	utils.Success(w, response)
}
