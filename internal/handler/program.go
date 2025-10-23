package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/MassBabyGeek/PumpPro-backend/internal/database"
	"github.com/MassBabyGeek/PumpPro-backend/internal/middleware"
	model "github.com/MassBabyGeek/PumpPro-backend/internal/models"
	"github.com/MassBabyGeek/PumpPro-backend/internal/scanner"
	"github.com/MassBabyGeek/PumpPro-backend/internal/utils"
	"github.com/gorilla/mux"
)

// GetPrograms récupère tous les programmes avec filtres optionnels
func GetPrograms(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	// Récupérer l'utilisateur optionnel depuis le contexte
	user, _ := middleware.GetUserFromContext(r)
	var userID *string
	if user.ID != "" {
		userID = &user.ID
	}

	// Récupérer les paramètres de filtrage
	query := r.URL.Query()
	difficulty := query.Get("difficulty")
	programType := query.Get("type")
	variant := query.Get("variant")
	isCustomStr := query.Get("isCustom")
	searchQuery := query.Get("searchQuery")
	limitStr := query.Get("limit")
	offsetStr := query.Get("offset")

	sqlQuery := `
		SELECT
			wp.id, wp.name, wp.description, wp.type, wp.variant, wp.difficulty, wp.rest_between_sets,
			wp.target_reps, wp.time_limit, wp.duration, wp.allow_rest, wp.sets, wp.reps_per_set,
			wp.reps_sequence, wp.reps_per_minute, wp.total_minutes,
			wp.is_custom, wp.is_featured, wp.usage_count, COALESCE(wp.likes, 0) as likes,
			wp.created_by, wp.updated_by, wp.deleted_by, wp.created_at, wp.updated_at, wp.deleted_at,
			u.id as creator_id, u.name as creator_name, u.avatar as creator_avatar
		FROM workout_programs wp
		LEFT JOIN users u ON wp.created_by = u.id AND u.deleted_at IS NULL
		WHERE wp.deleted_at IS NULL
	`

	args := []interface{}{}
	argCount := 1

	if difficulty != "" {
		sqlQuery += " AND difficulty = $" + strconv.Itoa(argCount)
		args = append(args, difficulty)
		argCount++
	}

	if programType != "" {
		sqlQuery += " AND type = $" + strconv.Itoa(argCount)
		args = append(args, programType)
		argCount++
	}

	if variant != "" {
		sqlQuery += " AND variant = $" + strconv.Itoa(argCount)
		args = append(args, variant)
		argCount++
	}

	if isCustomStr != "" {
		isCustom := isCustomStr == "true"
		sqlQuery += " AND is_custom = $" + strconv.Itoa(argCount)
		args = append(args, isCustom)
		argCount++
	}

	if searchQuery != "" {
		sqlQuery += " AND (LOWER(name) LIKE $" + strconv.Itoa(argCount) +
			" OR LOWER(description) LIKE $" + strconv.Itoa(argCount) + ")"
		searchPattern := "%" + searchQuery + "%"
		args = append(args, searchPattern)
		argCount++
	}

	sqlQuery += " ORDER BY is_featured DESC, usage_count DESC, created_at DESC"

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
		utils.Error(w, http.StatusInternalServerError, "could not query programs", err)
		return
	}
	defer rows.Close()

	var programs []model.WorkoutProgram
	for rows.Next() {
		program, err := scanner.ScanWorkoutProgramWithCreator(rows, json.Unmarshal)
		if err != nil {
			utils.Error(w, http.StatusInternalServerError, "could not scan program row", err)
			return
		}

		// Populate UserLiked field if user is authenticated
		if userID != nil {
			likeInfo, err := utils.GetLikeInfo(ctx, userID, model.EntityTypeProgram, program.ID)
			if err == nil {
				program.UserLiked = likeInfo.UserLiked
			}
		}

		programs = append(programs, *program)
	}

	utils.Success(w, programs)
}

// GetProgramById récupère un programme par son ID
func GetProgramById(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	ctx := context.Background()

	// Récupérer l'utilisateur optionnel depuis le contexte
	user, _ := middleware.GetUserFromContext(r)
	var userID *string
	if user.ID != "" {
		userID = &user.ID
	}

	row := database.DB.QueryRow(ctx, `
		SELECT
			wp.id, wp.name, wp.description, wp.type, wp.variant, wp.difficulty, wp.rest_between_sets,
			wp.target_reps, wp.time_limit, wp.duration, wp.allow_rest, wp.sets, wp.reps_per_set,
			wp.reps_sequence, wp.reps_per_minute, wp.total_minutes,
			wp.is_custom, wp.is_featured, wp.usage_count, COALESCE(wp.likes, 0) as likes,
			wp.created_by, wp.updated_by, wp.deleted_by, wp.created_at, wp.updated_at, wp.deleted_at,
			u.id as creator_id, u.name as creator_name, u.avatar as creator_avatar
		FROM workout_programs wp
		LEFT JOIN users u ON wp.created_by = u.id AND u.deleted_at IS NULL
		WHERE wp.id=$1 AND wp.deleted_at IS NULL
	`, id)

	program, err := scanner.ScanWorkoutProgramWithCreator(row, json.Unmarshal)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "program not found", err)
		return
	}

	// Populate UserLiked field if user is authenticated
	if userID != nil {
		likeInfo, err := utils.GetLikeInfo(ctx, userID, model.EntityTypeProgram, program.ID)
		if err == nil {
			program.UserLiked = likeInfo.UserLiked
		}
	}

	utils.Success(w, program)
}

// CreateProgram crée un nouveau programme
func CreateProgram(w http.ResponseWriter, r *http.Request) {
	var program model.WorkoutProgram
	if err := utils.DecodeJSON(r, &program); err != nil {
		utils.ErrorSimple(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	ctx := context.Background()

	// Encoder reps_sequence en JSON
	repsSequenceJSON, _ := json.Marshal(program.RepsSequence)

	err := database.DB.QueryRow(ctx, `
		INSERT INTO workout_programs(
			name, description, type, variant, difficulty, rest_between_sets,
			target_reps, time_limit, duration, allow_rest, sets, reps_per_set,
			reps_sequence, reps_per_minute, total_minutes,
			is_custom, is_featured, usage_count,
			created_by, created_at, updated_at
		) VALUES(
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, NOW(), NOW()
		)
		RETURNING id, created_at, updated_at
	`,
		program.Name, program.Description, program.Type, program.Variant, program.Difficulty,
		program.RestBetweenSets, program.TargetReps, program.TimeLimit, program.Duration,
		program.AllowRest, program.Sets, program.RepsPerSet, repsSequenceJSON,
		program.RepsPerMinute, program.TotalMinutes, program.IsCustom, program.IsFeatured,
		program.UsageCount, program.CreatedBy,
	).Scan(&program.ID, &program.CreatedAt, &program.UpdatedAt)

	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not create program", err)
		return
	}

	utils.Success(w, program)
}

// UpdateProgram met à jour un programme existant
func UpdateProgram(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var program model.WorkoutProgram
	if err := utils.DecodeJSON(r, &program); err != nil {
		utils.ErrorSimple(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	ctx := context.Background()

	// Vérifier que c'est un programme custom (seuls les custom peuvent être modifiés)
	var isCustom bool
	err := database.DB.QueryRow(ctx,
		`SELECT is_custom FROM workout_programs WHERE id=$1 AND deleted_at IS NULL`,
		id,
	).Scan(&isCustom)

	if err != nil {
		utils.ErrorSimple(w, http.StatusNotFound, "program not found")
		return
	}

	if !isCustom {
		utils.ErrorSimple(w, http.StatusForbidden, "cannot modify non-custom program")
		return
	}

	// Encoder reps_sequence en JSON
	repsSequenceJSON, _ := json.Marshal(program.RepsSequence)

	_, err = database.DB.Exec(ctx, `
		UPDATE workout_programs SET
			name=$1, description=$2, type=$3, variant=$4, difficulty=$5, rest_between_sets=$6,
			target_reps=$7, time_limit=$8, duration=$9, allow_rest=$10, sets=$11, reps_per_set=$12,
			reps_sequence=$13, reps_per_minute=$14, total_minutes=$15,
			updated_by=$16, updated_at=NOW()
		WHERE id=$17 AND deleted_at IS NULL
	`,
		program.Name, program.Description, program.Type, program.Variant, program.Difficulty,
		program.RestBetweenSets, program.TargetReps, program.TimeLimit, program.Duration,
		program.AllowRest, program.Sets, program.RepsPerSet, repsSequenceJSON,
		program.RepsPerMinute, program.TotalMinutes, program.UpdatedBy, id,
	)

	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not update program", err)
		return
	}

	program.ID = id
	utils.Success(w, program)
}

// DeleteProgram soft delete un programme
func DeleteProgram(w http.ResponseWriter, r *http.Request) {
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

	// Vérifier que c'est un programme custom
	var isCustom bool
	err := database.DB.QueryRow(ctx,
		`SELECT is_custom FROM workout_programs WHERE id=$1 AND deleted_at IS NULL`,
		id,
	).Scan(&isCustom)

	if err != nil {
		utils.ErrorSimple(w, http.StatusNotFound, "program not found")
		return
	}

	if !isCustom {
		utils.ErrorSimple(w, http.StatusForbidden, "cannot delete non-custom program")
		return
	}

	res, err := database.DB.Exec(ctx, `
		UPDATE workout_programs SET deleted_at=NOW(), deleted_by=$2
		WHERE id=$1 AND deleted_at IS NULL
	`, id, payload.DeletedBy)

	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not delete program", err)
		return
	}

	if res.RowsAffected() == 0 {
		utils.ErrorSimple(w, http.StatusNotFound, "program not found or already deleted")
		return
	}

	utils.Success(w, map[string]bool{"success": true})
}

// GetRecommendedPrograms récupère des programmes recommandés pour un utilisateur
func GetRecommendedPrograms(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]

	ctx := context.Background()

	query := r.URL.Query()
	limitStr := query.Get("limit")
	offsetStr := query.Get("offset")

	sqlQuery := `
		SELECT
			id, name, description, type, variant, difficulty, rest_between_sets,
			target_reps, time_limit, duration, allow_rest, sets, reps_per_set,
			reps_sequence, reps_per_minute, total_minutes,
			is_custom, is_featured, usage_count, COALESCE(likes, 0) as likes,
			created_by, updated_by, deleted_by, created_at, updated_at, deleted_at
		FROM workout_programs
		WHERE deleted_at IS NULL AND difficulty='INTERMEDIATE'
		ORDER BY usage_count DESC, is_featured DESC
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
		// Limite par défaut de 20 pour les recommandations
		sqlQuery += " LIMIT 20"
	}

	if offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			sqlQuery += " OFFSET $" + strconv.Itoa(argCount)
			args = append(args, offset)
			argCount++
		}
	}

	// Pour l'instant, on retourne les programmes les plus populaires de niveau intermédiaire
	// En production, cela devrait être basé sur l'historique et le niveau de l'utilisateur
	rows, err := database.DB.Query(ctx, sqlQuery, args...)

	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not query programs", err)
		return
	}
	defer rows.Close()

	var programs []model.WorkoutProgram
	for rows.Next() {
		var p model.WorkoutProgram
		var repsSequenceJSON []byte

		if err := rows.Scan(
			&p.ID, &p.Name, &p.Description, &p.Type, &p.Variant, &p.Difficulty, &p.RestBetweenSets,
			&p.TargetReps, &p.TimeLimit, &p.Duration, &p.AllowRest, &p.Sets, &p.RepsPerSet,
			&repsSequenceJSON, &p.RepsPerMinute, &p.TotalMinutes,
			&p.IsCustom, &p.IsFeatured, &p.UsageCount, &p.Likes,
			&p.CreatedBy, &p.UpdatedBy, &p.DeletedBy, &p.CreatedAt, &p.UpdatedAt, &p.DeletedAt,
		); err != nil {
			utils.Error(w, http.StatusInternalServerError, "could not scan program row", err)
			return
		}

		if repsSequenceJSON != nil {
			json.Unmarshal(repsSequenceJSON, &p.RepsSequence)
		}

		// Populate UserLiked field
		likeInfo, err := utils.GetLikeInfo(ctx, &userID, model.EntityTypeProgram, p.ID)
		if err == nil {
			p.UserLiked = likeInfo.UserLiked
		}

		// Load creator information
		utils.EnrichWorkoutProgramWithCreator(ctx, &p)

		programs = append(programs, p)
	}

	utils.Success(w, programs)
}

// GetProgramsByDifficulty récupère les programmes par niveau de difficulté
func GetProgramsByDifficulty(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	difficulty := vars["difficulty"]

	ctx := context.Background()

	// Get optional authenticated user
	user, _ := middleware.GetUserFromContext(r)
	var userID *string
	if user.ID != "" {
		userID = &user.ID
	}

	rows, err := database.DB.Query(ctx, `
		SELECT
			id, name, description, type, variant, difficulty, rest_between_sets,
			target_reps, time_limit, duration, allow_rest, sets, reps_per_set,
			reps_sequence, reps_per_minute, total_minutes,
			is_custom, is_featured, usage_count, COALESCE(likes, 0) as likes,
			created_by, updated_by, deleted_by, created_at, updated_at, deleted_at
		FROM workout_programs
		WHERE deleted_at IS NULL AND difficulty=$1
		ORDER BY is_featured DESC, usage_count DESC
	`, difficulty)

	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not query programs", err)
		return
	}
	defer rows.Close()

	var programs []model.WorkoutProgram
	for rows.Next() {
		var p model.WorkoutProgram
		var repsSequenceJSON []byte

		if err := rows.Scan(
			&p.ID, &p.Name, &p.Description, &p.Type, &p.Variant, &p.Difficulty, &p.RestBetweenSets,
			&p.TargetReps, &p.TimeLimit, &p.Duration, &p.AllowRest, &p.Sets, &p.RepsPerSet,
			&repsSequenceJSON, &p.RepsPerMinute, &p.TotalMinutes,
			&p.IsCustom, &p.IsFeatured, &p.UsageCount, &p.Likes,
			&p.CreatedBy, &p.UpdatedBy, &p.DeletedBy, &p.CreatedAt, &p.UpdatedAt, &p.DeletedAt,
		); err != nil {
			utils.Error(w, http.StatusInternalServerError, "could not scan program row", err)
			return
		}

		if repsSequenceJSON != nil {
			json.Unmarshal(repsSequenceJSON, &p.RepsSequence)
		}

		// Populate UserLiked field if user is authenticated
		if userID != nil {
			likeInfo, err := utils.GetLikeInfo(ctx, userID, model.EntityTypeProgram, p.ID)
			if err == nil {
				p.UserLiked = likeInfo.UserLiked
			}
		}

		// Load creator information
		utils.EnrichWorkoutProgramWithCreator(ctx, &p)

		programs = append(programs, p)
	}

	utils.Success(w, programs)
}

// GetUserCustomPrograms récupère les programmes personnalisés d'un utilisateur
func GetUserCustomPrograms(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]

	ctx := context.Background()

	query := r.URL.Query()
	limitStr := query.Get("limit")
	offsetStr := query.Get("offset")

	sqlQuery := `
		SELECT
			id, name, description, type, variant, difficulty, rest_between_sets,
			target_reps, time_limit, duration, allow_rest, sets, reps_per_set,
			reps_sequence, reps_per_minute, total_minutes,
			is_custom, is_featured, usage_count, COALESCE(likes, 0) as likes,
			created_by, updated_by, deleted_by, created_at, updated_at, deleted_at
		FROM workout_programs
		WHERE deleted_at IS NULL AND is_custom=true AND created_by=$1
		ORDER BY updated_at DESC
	`

	args := []interface{}{userID}
	argCount := 2

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
		utils.Error(w, http.StatusInternalServerError, "could not query programs", err)
		return
	}
	defer rows.Close()

	var programs []model.WorkoutProgram
	for rows.Next() {
		var p model.WorkoutProgram
		var repsSequenceJSON []byte

		if err := rows.Scan(
			&p.ID, &p.Name, &p.Description, &p.Type, &p.Variant, &p.Difficulty, &p.RestBetweenSets,
			&p.TargetReps, &p.TimeLimit, &p.Duration, &p.AllowRest, &p.Sets, &p.RepsPerSet,
			&repsSequenceJSON, &p.RepsPerMinute, &p.TotalMinutes,
			&p.IsCustom, &p.IsFeatured, &p.UsageCount, &p.Likes,
			&p.CreatedBy, &p.UpdatedBy, &p.DeletedBy, &p.CreatedAt, &p.UpdatedAt, &p.DeletedAt,
		); err != nil {
			utils.Error(w, http.StatusInternalServerError, "could not scan program row", err)
			return
		}

		if repsSequenceJSON != nil {
			json.Unmarshal(repsSequenceJSON, &p.RepsSequence)
		}

		// Populate UserLiked field for the requesting user
		likeInfo, err := utils.GetLikeInfo(ctx, &userID, model.EntityTypeProgram, p.ID)
		if err == nil {
			p.UserLiked = likeInfo.UserLiked
		}

		programs = append(programs, p)
	}

	utils.Success(w, programs)
}

// DuplicateProgram duplique un programme existant
func DuplicateProgram(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	programID := vars["id"]

	var payload struct {
		UserID string `json:"userId"`
	}
	if err := utils.DecodeJSON(r, &payload); err != nil {
		utils.ErrorSimple(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	ctx := context.Background()

	// Récupérer le programme source
	var program model.WorkoutProgram
	var repsSequenceJSON []byte

	err := database.DB.QueryRow(ctx, `
		SELECT
			name, description, type, variant, difficulty, rest_between_sets,
			target_reps, time_limit, duration, allow_rest, sets, reps_per_set,
			reps_sequence, reps_per_minute, total_minutes
		FROM workout_programs
		WHERE id=$1 AND deleted_at IS NULL
	`, programID).Scan(
		&program.Name, &program.Description, &program.Type, &program.Variant,
		&program.Difficulty, &program.RestBetweenSets,
		&program.TargetReps, &program.TimeLimit, &program.Duration, &program.AllowRest,
		&program.Sets, &program.RepsPerSet, &repsSequenceJSON, &program.RepsPerMinute,
		&program.TotalMinutes,
	)

	if err != nil {
		utils.Error(w, http.StatusNotFound, "program not found", err)
		return
	}

	if repsSequenceJSON != nil {
		json.Unmarshal(repsSequenceJSON, &program.RepsSequence)
	}

	// Créer la copie
	program.Name = program.Name + " (Copie)"
	program.IsCustom = true
	program.IsFeatured = false
	program.UsageCount = 0

	repsSequenceJSONCopy, _ := json.Marshal(program.RepsSequence)

	var newProgram model.WorkoutProgram
	err = database.DB.QueryRow(ctx, `
		INSERT INTO workout_programs(
			name, description, type, variant, difficulty, rest_between_sets,
			target_reps, time_limit, duration, allow_rest, sets, reps_per_set,
			reps_sequence, reps_per_minute, total_minutes,
			is_custom, is_featured, usage_count,
			created_by, created_at, updated_at
		) VALUES(
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, true, false, 0, $16, NOW(), NOW()
		)
		RETURNING id, created_at, updated_at
	`,
		program.Name, program.Description, program.Type, program.Variant, program.Difficulty,
		program.RestBetweenSets, program.TargetReps, program.TimeLimit, program.Duration,
		program.AllowRest, program.Sets, program.RepsPerSet, repsSequenceJSONCopy,
		program.RepsPerMinute, program.TotalMinutes, payload.UserID,
	).Scan(&newProgram.ID, &newProgram.CreatedAt, &newProgram.UpdatedAt)

	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not duplicate program", err)
		return
	}

	// Copier les autres champs
	newProgram.Name = program.Name
	newProgram.Description = program.Description
	newProgram.Type = program.Type
	newProgram.Variant = program.Variant
	newProgram.Difficulty = program.Difficulty
	newProgram.RestBetweenSets = program.RestBetweenSets
	newProgram.TargetReps = program.TargetReps
	newProgram.TimeLimit = program.TimeLimit
	newProgram.Duration = program.Duration
	newProgram.AllowRest = program.AllowRest
	newProgram.Sets = program.Sets
	newProgram.RepsPerSet = program.RepsPerSet
	newProgram.RepsSequence = program.RepsSequence
	newProgram.RepsPerMinute = program.RepsPerMinute
	newProgram.TotalMinutes = program.TotalMinutes
	newProgram.IsCustom = true
	newProgram.IsFeatured = false
	newProgram.UsageCount = 0

	utils.Success(w, newProgram)
}

// GetFeaturedPrograms récupère les programmes mis en avant
func GetFeaturedPrograms(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	// Get optional authenticated user
	user, _ := middleware.GetUserFromContext(r)
	var userID *string
	if user.ID != "" {
		userID = &user.ID
	}

	query := r.URL.Query()
	limitStr := query.Get("limit")
	offsetStr := query.Get("offset")

	sqlQuery := `
		SELECT
			id, name, description, type, variant, difficulty, rest_between_sets,
			target_reps, time_limit, duration, allow_rest, sets, reps_per_set,
			reps_sequence, reps_per_minute, total_minutes,
			is_custom, is_featured, usage_count, COALESCE(likes, 0) as likes,
			created_by, updated_by, deleted_by, created_at, updated_at, deleted_at
		FROM workout_programs
		WHERE deleted_at IS NULL AND is_featured=true
		ORDER BY usage_count DESC
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
		// Limite par défaut de 20 pour les programmes featured
		sqlQuery += " LIMIT 20"
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
		utils.Error(w, http.StatusInternalServerError, "could not query programs", err)
		return
	}
	defer rows.Close()

	var programs []model.WorkoutProgram
	for rows.Next() {
		var p model.WorkoutProgram
		var repsSequenceJSON []byte

		if err := rows.Scan(
			&p.ID, &p.Name, &p.Description, &p.Type, &p.Variant, &p.Difficulty, &p.RestBetweenSets,
			&p.TargetReps, &p.TimeLimit, &p.Duration, &p.AllowRest, &p.Sets, &p.RepsPerSet,
			&repsSequenceJSON, &p.RepsPerMinute, &p.TotalMinutes,
			&p.IsCustom, &p.IsFeatured, &p.UsageCount, &p.Likes,
			&p.CreatedBy, &p.UpdatedBy, &p.DeletedBy, &p.CreatedAt, &p.UpdatedAt, &p.DeletedAt,
		); err != nil {
			utils.Error(w, http.StatusInternalServerError, "could not scan program row", err)
			return
		}

		if repsSequenceJSON != nil {
			json.Unmarshal(repsSequenceJSON, &p.RepsSequence)
		}

		// Populate UserLiked field if user is authenticated
		if userID != nil {
			likeInfo, err := utils.GetLikeInfo(ctx, userID, model.EntityTypeProgram, p.ID)
			if err == nil {
				p.UserLiked = likeInfo.UserLiked
			}
		}

		// Load creator information
		utils.EnrichWorkoutProgramWithCreator(ctx, &p)

		programs = append(programs, p)
	}

	utils.Success(w, programs)
}

// GetPopularPrograms récupère les programmes les plus utilisés
func GetPopularPrograms(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	query := r.URL.Query()
	limitStr := query.Get("limit")
	offsetStr := query.Get("offset")

	// Get optional authenticated user
	user, _ := middleware.GetUserFromContext(r)
	var userID *string
	if user.ID != "" {
		userID = &user.ID
	}

	sqlQuery := `
		SELECT
			id, name, description, type, variant, difficulty, rest_between_sets,
			target_reps, time_limit, duration, allow_rest, sets, reps_per_set,
			reps_sequence, reps_per_minute, total_minutes,
			is_custom, is_featured, usage_count, COALESCE(likes, 0) as likes,
			created_by, updated_by, deleted_by, created_at, updated_at, deleted_at
		FROM workout_programs
		WHERE deleted_at IS NULL
		ORDER BY usage_count DESC
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
		// Limite par défaut de 20 pour les programmes populaires
		sqlQuery += " LIMIT 20"
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
		utils.Error(w, http.StatusInternalServerError, "could not query programs", err)
		return
	}
	defer rows.Close()

	var programs []model.WorkoutProgram
	for rows.Next() {
		var p model.WorkoutProgram
		var repsSequenceJSON []byte

		if err := rows.Scan(
			&p.ID, &p.Name, &p.Description, &p.Type, &p.Variant, &p.Difficulty, &p.RestBetweenSets,
			&p.TargetReps, &p.TimeLimit, &p.Duration, &p.AllowRest, &p.Sets, &p.RepsPerSet,
			&repsSequenceJSON, &p.RepsPerMinute, &p.TotalMinutes,
			&p.IsCustom, &p.IsFeatured, &p.UsageCount, &p.Likes,
			&p.CreatedBy, &p.UpdatedBy, &p.DeletedBy, &p.CreatedAt, &p.UpdatedAt, &p.DeletedAt,
		); err != nil {
			utils.Error(w, http.StatusInternalServerError, "could not scan program row", err)
			return
		}

		if repsSequenceJSON != nil {
			json.Unmarshal(repsSequenceJSON, &p.RepsSequence)
		}

		// Populate UserLiked field if user is authenticated
		if userID != nil {
			likeInfo, err := utils.GetLikeInfo(ctx, userID, model.EntityTypeProgram, p.ID)
			if err == nil {
				p.UserLiked = likeInfo.UserLiked
			}
		}

		// Load creator information
		utils.EnrichWorkoutProgramWithCreator(ctx, &p)

		programs = append(programs, p)
	}

	utils.Success(w, programs)
}

// LikeProgram ajoute un like à un programme
func LikeProgram(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	programID := vars["id"]

	user, err := middleware.GetUserFromContext(r)
	if err != nil {
		utils.Error(w, http.StatusUnauthorized, "impossible de récupérer l'utilisateur", err)
		return
	}

	ctx := context.Background()

	// Utiliser le système unifié de likes
	err = utils.AddLike(ctx, user.ID, model.EntityTypeProgram, programID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not add like", err)
		return
	}

	// Incrémenter le compteur de likes
	_, err = database.DB.Exec(ctx, `
		UPDATE workout_programs SET likes = likes + 1 WHERE id=$1`,
		programID,
	)

	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not increment likes", err)
		return
	}

	// Retourner le programme mis à jour
	row := database.DB.QueryRow(ctx, `
		SELECT
			id, name, description, type, variant, difficulty, rest_between_sets,
			target_reps, time_limit, duration, allow_rest, sets, reps_per_set,
			reps_sequence, reps_per_minute, total_minutes,
			is_custom, is_featured, usage_count, COALESCE(likes, 0) as likes,
			created_by, updated_by, deleted_by, created_at, updated_at, deleted_at
		FROM workout_programs
		WHERE id=$1
	`, programID)

	program, err := scanner.ScanWorkoutProgram(row)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not fetch program", err)
		return
	}

	utils.Success(w, program)
}

// UnlikeProgram retire un like à un programme
func UnlikeProgram(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	programID := vars["id"]

	user, err := middleware.GetUserFromContext(r)
	if err != nil {
		utils.Error(w, http.StatusUnauthorized, "impossible de récupérer l'utilisateur", err)
		return
	}

	ctx := context.Background()

	// Utiliser le système unifié de likes
	err = utils.RemoveLike(ctx, user.ID, model.EntityTypeProgram, programID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not remove like", err)
		return
	}

	// Décrémenter le compteur de likes
	_, err = database.DB.Exec(ctx, `
		UPDATE workout_programs SET likes = GREATEST(likes - 1, 0) WHERE id=$1`,
		programID,
	)

	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not decrement likes", err)
		return
	}

	// Retourner le programme mis à jour
	row := database.DB.QueryRow(ctx, `
		SELECT
			id, name, description, type, variant, difficulty, rest_between_sets,
			target_reps, time_limit, duration, allow_rest, sets, reps_per_set,
			reps_sequence, reps_per_minute, total_minutes,
			is_custom, is_featured, usage_count, COALESCE(likes, 0) as likes,
			created_by, updated_by, deleted_by, created_at, updated_at, deleted_at
		FROM workout_programs
		WHERE id=$1
	`, programID)

	program, err := scanner.ScanWorkoutProgram(row)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not fetch program", err)
		return
	}
	utils.Success(w, program)
}
