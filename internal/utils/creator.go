package utils

import (
	"context"
	"database/sql"

	"github.com/MassBabyGeek/PumpPro-backend/internal/database"
	model "github.com/MassBabyGeek/PumpPro-backend/internal/models"
)

// LoadCreator charge les informations du créateur depuis son ID
func LoadCreator(ctx context.Context, creatorID *string) (*model.UserCreator, error) {
	if creatorID == nil || *creatorID == "" {
		return nil, nil
	}

	var creator model.UserCreator
	var avatar sql.NullString

	err := database.DB.QueryRow(ctx, `
		SELECT id, name, avatar
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`, *creatorID).Scan(&creator.ID, &creator.Name, &avatar)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Utilisateur non trouvé, retourner nil sans erreur
		}
		return nil, err
	}

	creator.Avatar = NullStringToString(avatar)
	return &creator, nil
}

// LoadUser charge les informations de l'utilisateur (similaire à LoadCreator mais avec un nom différent)
func LoadUser(ctx context.Context, userID *string) (*model.UserCreator, error) {
	return LoadCreator(ctx, userID)
}

// EnrichWorkoutProgramWithCreator ajoute les informations du créateur à un programme
func EnrichWorkoutProgramWithCreator(ctx context.Context, program *model.WorkoutProgram) {
	if program == nil {
		return
	}
	creator, err := LoadCreator(ctx, program.CreatedBy)
	if err == nil {
		program.Creator = creator
	}
}

// EnrichChallengeWithCreator ajoute les informations du créateur à un challenge
func EnrichChallengeWithCreator(ctx context.Context, challenge *model.Challenge) {
	if challenge == nil {
		return
	}
	creator, err := LoadCreator(ctx, challenge.CreatedBy)
	if err == nil {
		challenge.Creator = creator
	}
}

// EnrichChallengeTaskWithCreator ajoute les informations du créateur à une challenge task
func EnrichChallengeTaskWithCreator(ctx context.Context, task *model.ChallengeTask) {
	if task == nil {
		return
	}
	creator, err := LoadCreator(ctx, task.CreatedBy)
	if err == nil {
		task.Creator = creator
	}
}

// EnrichWorkoutSessionWithCreatorAndUser ajoute les informations du créateur et de l'utilisateur à une session
func EnrichWorkoutSessionWithCreatorAndUser(ctx context.Context, session *model.WorkoutSession) {
	if session == nil {
		return
	}
	// Charger le créateur (celui qui a créé la session)
	creator, err := LoadCreator(ctx, session.CreatedBy)
	if err == nil {
		session.Creator = creator
	}

	// Charger l'utilisateur qui a fait la session
	user, err := LoadUser(ctx, &session.UserID)
	if err == nil {
		session.User = user
	}
}
