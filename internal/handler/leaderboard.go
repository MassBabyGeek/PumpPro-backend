package handler

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/MassBabyGeek/PumpPro-backend/internal/database"
	model "github.com/MassBabyGeek/PumpPro-backend/internal/models"
	"github.com/MassBabyGeek/PumpPro-backend/internal/utils"
	"github.com/gorilla/mux"
)

// GetLeaderboard récupère le classement général
func GetLeaderboard(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	period := query.Get("period") // daily, weekly, monthly, all-time
	limitStr := query.Get("limit")

	if period == "" {
		period = "all-time"
	}

	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	ctx := context.Background()

	// Calculer la date de début selon la période
	var startDate time.Time
	now := time.Now()

	switch period {
	case "daily":
		startDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	case "weekly":
		startDate = now.AddDate(0, 0, -7)
	case "monthly":
		startDate = now.AddDate(0, 0, -30)
	case "all-time":
		startDate = time.Time{} // Epoch time
	default:
		startDate = time.Time{}
	}

	// Requête pour calculer le classement
	var sqlQuery string
	var args []interface{}

	if period == "all-time" {
		sqlQuery = `
			WITH user_scores AS (
				SELECT
					ws.user_id,
					SUM(ws.total_reps) as score
				FROM workout_sessions ws
				GROUP BY ws.user_id
			),
			ranked_users AS (
				SELECT
					us.user_id,
					us.score,
					ROW_NUMBER() OVER (ORDER BY us.score DESC) as rank
				FROM user_scores us
			)
			SELECT
				ru.user_id,
				u.name as user_name,
				u.avatar,
				ru.rank,
				ru.score,
				0 as change
			FROM ranked_users ru
			INNER JOIN users u ON ru.user_id = u.id
			WHERE u.deleted_at IS NULL
			ORDER BY ru.rank
			LIMIT $1
		`
		args = []interface{}{limit}
	} else {
		sqlQuery = `
			WITH user_scores AS (
				SELECT
					ws.user_id,
					SUM(ws.total_reps) as score
				FROM workout_sessions ws
				WHERE ws.start_time >= $1
				GROUP BY ws.user_id
			),
			ranked_users AS (
				SELECT
					us.user_id,
					us.score,
					ROW_NUMBER() OVER (ORDER BY us.score DESC) as rank
				FROM user_scores us
			)
			SELECT
				ru.user_id,
				u.name as user_name,
				u.avatar,
				ru.rank,
				ru.score,
				0 as change
			FROM ranked_users ru
			INNER JOIN users u ON ru.user_id = u.id
			WHERE u.deleted_at IS NULL
			ORDER BY ru.rank
			LIMIT $2
		`
		args = []interface{}{startDate, limit}
	}

	rows, err := database.DB.Query(ctx, sqlQuery, args...)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not query leaderboard", err)
		return
	}
	defer rows.Close()

	var leaderboard []model.LeaderboardEntry
	for rows.Next() {
		var entry model.LeaderboardEntry
		if err := rows.Scan(
			&entry.UserID, &entry.UserName, &entry.Avatar,
			&entry.Rank, &entry.Score, &entry.Change,
		); err != nil {
			utils.Error(w, http.StatusInternalServerError, "could not scan leaderboard row", err)
			return
		}

		// Récupérer les badges de l'utilisateur (à implémenter selon votre logique)
		entry.Badges = []string{} // Placeholder

		leaderboard = append(leaderboard, entry)
	}

	utils.Success(w, leaderboard)
}

// GetUserRank récupère le rang d'un utilisateur dans le classement
func GetUserRank(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]
	period := r.URL.Query().Get("period")

	if period == "" {
		period = "all-time"
	}

	ctx := context.Background()

	// Calculer la date de début selon la période
	var startDate time.Time
	now := time.Now()

	switch period {
	case "daily":
		startDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	case "weekly":
		startDate = now.AddDate(0, 0, -7)
	case "monthly":
		startDate = now.AddDate(0, 0, -30)
	case "all-time":
		startDate = time.Time{}
	default:
		startDate = time.Time{}
	}

	var userRank model.UserRank
	var sqlQuery string
	var args []interface{}

	if period == "all-time" {
		sqlQuery = `
			WITH user_scores AS (
				SELECT
					ws.user_id,
					SUM(ws.total_reps) as score
				FROM workout_sessions ws
				GROUP BY ws.user_id
			),
			ranked_users AS (
				SELECT
					us.user_id,
					us.score,
					ROW_NUMBER() OVER (ORDER BY us.score DESC) as rank
				FROM user_scores us
			),
			total_count AS (
				SELECT COUNT(*) as total FROM ranked_users
			)
			SELECT
				COALESCE(ru.rank, (SELECT total FROM total_count) + 1) as rank,
				ru.score as score,
				(SELECT total FROM total_count) as total_users
			FROM ranked_users ru
			RIGHT JOIN (SELECT $1::uuid as uid) u ON ru.user_id = u.uid
		`
		args = []interface{}{userID}
	} else {
		sqlQuery = `
			WITH user_scores AS (
				SELECT
					ws.user_id,
					SUM(ws.total_reps) as score
				FROM workout_sessions ws
				WHERE ws.start_time >= $1
				GROUP BY ws.user_id
			),
			ranked_users AS (
				SELECT
					us.user_id,
					us.score,
					ROW_NUMBER() OVER (ORDER BY us.score DESC) as rank
				FROM user_scores us
			),
			total_count AS (
				SELECT COUNT(*) as total FROM ranked_users
			)
			SELECT
				COALESCE(ru.rank, (SELECT total FROM total_count) + 1) as rank,
				ru.score as score,
				(SELECT total FROM total_count) as total_users
			FROM ranked_users ru
			RIGHT JOIN (SELECT $2::uuid as uid) u ON ru.user_id = u.uid
		`
		args = []interface{}{startDate, userID}
	}

	err := database.DB.QueryRow(ctx, sqlQuery, args...).Scan(
		&userRank.Rank,
		&userRank.Score,
		&userRank.TotalUsers,
	)

	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not fetch user rank", err)
		return
	}

	userRank.UserID = userID

	// Calculer le percentile
	if userRank.TotalUsers > 0 {
		userRank.Percentile = float64(userRank.Rank) / float64(userRank.TotalUsers) * 100
	} else {
		userRank.Percentile = 100
	}

	utils.Success(w, userRank)
}

// GetNearbyUsers récupère les utilisateurs proches dans le classement
func GetNearbyUsers(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]

	query := r.URL.Query()
	period := query.Get("period")
	rangeStr := query.Get("range")

	if period == "" {
		period = "all-time"
	}

	rangeVal := 5
	if rangeStr != "" {
		if r, err := strconv.Atoi(rangeStr); err == nil {
			rangeVal = r
		}
	}

	ctx := context.Background()

	// Calculer la date de début selon la période
	var startDate time.Time
	now := time.Now()

	switch period {
	case "daily":
		startDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	case "weekly":
		startDate = now.AddDate(0, 0, -7)
	case "monthly":
		startDate = now.AddDate(0, 0, -30)
	case "all-time":
		startDate = time.Time{}
	default:
		startDate = time.Time{}
	}

	var sqlQuery string
	var args []interface{}

	if period == "all-time" {
		sqlQuery = `
			WITH user_scores AS (
				SELECT
					ws.user_id,
					SUM(ws.total_reps) as score
				FROM workout_sessions ws
				GROUP BY ws.user_id
			),
			ranked_users AS (
				SELECT
					us.user_id,
					us.score,
					ROW_NUMBER() OVER (ORDER BY us.score DESC) as rank
				FROM user_scores us
			),
			target_rank AS (
				SELECT rank FROM ranked_users WHERE user_id = $1
			)
			SELECT
				ru.user_id,
				u.name as user_name,
				u.avatar,
				ru.rank,
				ru.score,
				0 as change
			FROM ranked_users ru
			INNER JOIN users u ON ru.user_id = u.id
			WHERE u.deleted_at IS NULL
			AND ru.rank BETWEEN (SELECT rank FROM target_rank) - $2 AND (SELECT rank FROM target_rank) + $2
			ORDER BY ru.rank
		`
		args = []interface{}{userID, rangeVal}
	} else {
		sqlQuery = `
			WITH user_scores AS (
				SELECT
					ws.user_id,
					SUM(ws.total_reps) as score
				FROM workout_sessions ws
				WHERE ws.start_time >= $1
				GROUP BY ws.user_id
			),
			ranked_users AS (
				SELECT
					us.user_id,
					us.score,
					ROW_NUMBER() OVER (ORDER BY us.score DESC) as rank
				FROM user_scores us
			),
			target_rank AS (
				SELECT rank FROM ranked_users WHERE user_id = $2
			)
			SELECT
				ru.user_id,
				u.name as user_name,
				u.avatar,
				ru.rank,
				ru.score,
				0 as change
			FROM ranked_users ru
			INNER JOIN users u ON ru.user_id = u.id
			WHERE u.deleted_at IS NULL
			AND ru.rank BETWEEN (SELECT rank FROM target_rank) - $3 AND (SELECT rank FROM target_rank) + $3
			ORDER BY ru.rank
		`
		args = []interface{}{startDate, userID, rangeVal}
	}

	rows, err := database.DB.Query(ctx, sqlQuery, args...)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not query nearby users", err)
		return
	}
	defer rows.Close()

	var nearby []model.LeaderboardEntry
	for rows.Next() {
		var entry model.LeaderboardEntry
		if err := rows.Scan(
			&entry.UserID, &entry.UserName, &entry.Avatar,
			&entry.Rank, &entry.Score, &entry.Change,
		); err != nil {
			utils.Error(w, http.StatusInternalServerError, "could not scan nearby user row", err)
			return
		}

		entry.Badges = []string{} // Placeholder
		nearby = append(nearby, entry)
	}

	utils.Success(w, nearby)
}

// GetTopPerformers récupère les 3 meilleurs utilisateurs
func GetTopPerformers(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")

	if period == "" {
		period = "all-time"
	}

	ctx := context.Background()

	// Calculer la date de début selon la période
	var startDate time.Time
	now := time.Now()

	switch period {
	case "daily":
		startDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	case "weekly":
		startDate = now.AddDate(0, 0, -7)
	case "monthly":
		startDate = now.AddDate(0, 0, -30)
	case "all-time":
		startDate = time.Time{}
	default:
		startDate = time.Time{}
	}

	var sqlQuery string
	var args []interface{}

	if period == "all-time" {
		sqlQuery = `
			WITH user_scores AS (
				SELECT
					ws.user_id,
					SUM(ws.total_reps) as score
				FROM workout_sessions ws
				GROUP BY ws.user_id
			),
			ranked_users AS (
				SELECT
					us.user_id,
					us.score,
					ROW_NUMBER() OVER (ORDER BY us.score DESC) as rank
				FROM user_scores us
			)
			SELECT
				ru.user_id,
				u.name as user_name,
				u.avatar,
				ru.rank,
				ru.score,
				0 as change
			FROM ranked_users ru
			INNER JOIN users u ON ru.user_id = u.id
			WHERE u.deleted_at IS NULL
			ORDER BY ru.rank
			LIMIT 3
		`
		args = []interface{}{}
	} else {
		sqlQuery = `
			WITH user_scores AS (
				SELECT
					ws.user_id,
					SUM(ws.total_reps) as score
				FROM workout_sessions ws
				WHERE ws.start_time >= $1
				GROUP BY ws.user_id
			),
			ranked_users AS (
				SELECT
					us.user_id,
					us.score,
					ROW_NUMBER() OVER (ORDER BY us.score DESC) as rank
				FROM user_scores us
			)
			SELECT
				ru.user_id,
				u.name as user_name,
				u.avatar,
				ru.rank,
				ru.score,
				0 as change
			FROM ranked_users ru
			INNER JOIN users u ON ru.user_id = u.id
			WHERE u.deleted_at IS NULL
			ORDER BY ru.rank
			LIMIT 3
		`
		args = []interface{}{startDate}
	}

	rows, err := database.DB.Query(ctx, sqlQuery, args...)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not query top performers", err)
		return
	}
	defer rows.Close()

	var topPerformers []model.LeaderboardEntry
	for rows.Next() {
		var entry model.LeaderboardEntry
		if err := rows.Scan(
			&entry.UserID, &entry.UserName, &entry.Avatar,
			&entry.Rank, &entry.Score, &entry.Change,
		); err != nil {
			utils.Error(w, http.StatusInternalServerError, "could not scan top performer row", err)
			return
		}

		// Badges spéciaux pour le podium
		switch entry.Rank {
		case 1:
			entry.Badges = []string{"👑", "🔥", "💎"}
		case 2:
			entry.Badges = []string{"🔥", "💪"}
		case 3:
			entry.Badges = []string{"💎", "⚡"}
		}

		topPerformers = append(topPerformers, entry)
	}

	utils.Success(w, topPerformers)
}

// GetChallengeLeaderboard récupère le classement d'un challenge spécifique
func GetChallengeLeaderboard(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	challengeID := vars["challengeId"]

	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	ctx := context.Background()

	// Classement basé sur la progression du challenge
	rows, err := database.DB.Query(ctx, `
		WITH user_progress AS (
			SELECT
				ucp.user_id,
				ucp.progress as score
			FROM user_challenge_progress ucp
			WHERE ucp.challenge_id = $1
		),
		ranked_users AS (
			SELECT
				up.user_id,
				up.score,
				ROW_NUMBER() OVER (ORDER BY up.score DESC) as rank
			FROM user_progress up
		)
		SELECT
			ru.user_id,
			u.name as user_name,
			u.avatar,
			ru.rank,
			ru.score,
			0 as change
		FROM ranked_users ru
		INNER JOIN users u ON ru.user_id = u.id
		WHERE u.deleted_at IS NULL
		ORDER BY ru.rank
		LIMIT $2
	`, challengeID, limit)

	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not query challenge leaderboard", err)
		return
	}
	defer rows.Close()

	var leaderboard []model.LeaderboardEntry
	for rows.Next() {
		var entry model.LeaderboardEntry
		if err := rows.Scan(
			&entry.UserID, &entry.UserName, &entry.Avatar,
			&entry.Rank, &entry.Score, &entry.Change,
		); err != nil {
			utils.Error(w, http.StatusInternalServerError, "could not scan challenge leaderboard row", err)
			return
		}

		entry.Badges = []string{} // Placeholder
		leaderboard = append(leaderboard, entry)
	}

	utils.Success(w, leaderboard)
}

// GetFriendsLeaderboard récupère le classement des amis (si fonctionnalité sociale implémentée)
func GetFriendsLeaderboard(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]
	period := r.URL.Query().Get("period")

	if period == "" {
		period = "all-time"
	}

	ctx := context.Background()

	// Pour l'instant, retourner le top 10 global comme placeholder
	// En production, filtrer par les amis de l'utilisateur
	var startDate time.Time
	now := time.Now()

	switch period {
	case "daily":
		startDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	case "weekly":
		startDate = now.AddDate(0, 0, -7)
	case "monthly":
		startDate = now.AddDate(0, 0, -30)
	case "all-time":
		startDate = time.Time{}
	default:
		startDate = time.Time{}
	}

	var sqlQuery string
	var args []interface{}

	if period == "all-time" {
		sqlQuery = `
			WITH user_scores AS (
				SELECT
					ws.user_id,
					SUM(ws.total_reps) as score
				FROM workout_sessions ws
				GROUP BY ws.user_id
			),
			ranked_users AS (
				SELECT
					us.user_id,
					us.score,
					ROW_NUMBER() OVER (ORDER BY us.score DESC) as rank
				FROM user_scores us
			)
			SELECT
				ru.user_id,
				u.name as user_name,
				u.avatar,
				ru.rank,
				ru.score,
				0 as change
			FROM ranked_users ru
			INNER JOIN users u ON ru.user_id = u.id
			WHERE u.deleted_at IS NULL
			ORDER BY ru.rank
			LIMIT 10
		`
		args = []interface{}{}
	} else {
		sqlQuery = `
			WITH user_scores AS (
				SELECT
					ws.user_id,
					SUM(ws.total_reps) as score
				FROM workout_sessions ws
				WHERE ws.start_time >= $1
				GROUP BY ws.user_id
			),
			ranked_users AS (
				SELECT
					us.user_id,
					us.score,
					ROW_NUMBER() OVER (ORDER BY us.score DESC) as rank
				FROM user_scores us
			)
			SELECT
				ru.user_id,
				u.name as user_name,
				u.avatar,
				ru.rank,
				ru.score,
				0 as change
			FROM ranked_users ru
			INNER JOIN users u ON ru.user_id = u.id
			WHERE u.deleted_at IS NULL
			ORDER BY ru.rank
			LIMIT 10
		`
		args = []interface{}{startDate}
	}

	rows, err := database.DB.Query(ctx, sqlQuery, args...)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not query friends leaderboard", err)
		return
	}
	defer rows.Close()

	var leaderboard []model.LeaderboardEntry
	for rows.Next() {
		var entry model.LeaderboardEntry
		if err := rows.Scan(
			&entry.UserID, &entry.UserName, &entry.Avatar,
			&entry.Rank, &entry.Score, &entry.Change,
		); err != nil {
			utils.Error(w, http.StatusInternalServerError, "could not scan friends leaderboard row", err)
			return
		}

		entry.Badges = []string{} // Placeholder
		leaderboard = append(leaderboard, entry)
	}

	// Éviter erreur unused variable
	_ = userID

	utils.Success(w, leaderboard)
}
