package utils

import (
	"context"
	"database/sql"

	"github.com/MassBabyGeek/PumpPro-backend/internal/database"
	model "github.com/MassBabyGeek/PumpPro-backend/internal/models"
)

// FindUserByID recherche un utilisateur par son ID
func FindUserByID(ctx context.Context, userID string) (*model.UserProfile, string, error) {

	var user model.UserProfile
	var passwordHash sql.NullString
	var avatar, goal, provider sql.NullString
	var age, score sql.NullInt64
	var weight, height sql.NullFloat64

	err := database.DB.QueryRow(ctx,
		`SELECT id, name, email, avatar, age, weight, height, goal, score, provider, password_hash,
		 join_date, created_at, updated_at
		 FROM users WHERE id=$1 AND deleted_at IS NULL`,
		userID,
	).Scan(&user.ID, &user.Name, &user.Email, &avatar, &age, &weight, &height,
		&goal, &score, &provider, &passwordHash, &user.JoinDate, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return nil, "", err
	}

	user.Avatar = NullStringToString(avatar)
	user.Goal = NullStringToString(goal)
	user.Provider = NullStringToString(provider)
	if user.Provider == "" {
		user.Provider = "email"
	}
	user.Age = NullInt64ToInt(age)
	user.Weight = NullFloat64ToFloat64(weight)
	user.Height = NullFloat64ToFloat64(height)
	user.Score = NullInt64ToInt(score)

	return &user, NullStringToString(passwordHash), nil
}

// FindUserByEmail recherche un utilisateur par son email
func FindUserByEmail(ctx context.Context, email string) (*model.UserProfile, error) {

	var user model.UserProfile
	var avatar, goal, provider sql.NullString
	var age, score sql.NullInt64
	var weight, height sql.NullFloat64

	err := database.DB.QueryRow(ctx,
		`SELECT id, name, email, avatar, age, weight, height, goal, score, provider,
		 join_date, created_at, updated_at
		 FROM users WHERE email=$1 AND deleted_at IS NULL`,
		email,
	).Scan(&user.ID, &user.Name, &user.Email, &avatar, &age, &weight, &height,
		&goal, &score, &provider, &user.JoinDate, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return nil, err
	}

	user.Avatar = NullStringToString(avatar)
	user.Goal = NullStringToString(goal)
	user.Provider = NullStringToString(provider)
	if user.Provider == "" {
		user.Provider = "email"
	}
	user.Age = NullInt64ToInt(age)
	user.Weight = NullFloat64ToFloat64(weight)
	user.Height = NullFloat64ToFloat64(height)
	user.Score = NullInt64ToInt(score)

	return &user, nil
}

// FindUserByEmailWithPassword recherche un utilisateur par email et retourne aussi le hash du mot de passe
func FindUserByEmailWithPassword(ctx context.Context, email string) (*model.UserProfile, string, error) {

	var user model.UserProfile
	var passwordHash string
	var avatar, goal sql.NullString
	var age, score sql.NullInt64
	var weight, height sql.NullFloat64

	err := database.DB.QueryRow(ctx,
		`SELECT id, name, email, avatar, age, weight, height, goal, score,
		 join_date, created_at, updated_at, password_hash
		 FROM users WHERE email=$1 AND deleted_at IS NULL`,
		email,
	).Scan(&user.ID, &user.Name, &user.Email, &avatar, &age, &weight, &height,
		&goal, &score, &user.JoinDate, &user.CreatedAt, &user.UpdatedAt, &passwordHash)

	if err != nil {
		return nil, "", err
	}

	user.Avatar = NullStringToString(avatar)
	user.Goal = NullStringToString(goal)
	user.Age = NullInt64ToInt(age)
	user.Weight = NullFloat64ToFloat64(weight)
	user.Height = NullFloat64ToFloat64(height)
	user.Score = NullInt64ToInt(score)

	return &user, passwordHash, nil
}

// CreateUser crée un nouvel utilisateur
func CreateUser(ctx context.Context, name, email, passwordHash, avatar, provider string) (*model.UserProfile, error) {

	var user model.UserProfile
	err := database.DB.QueryRow(ctx,
		`INSERT INTO users(name, email, password_hash, avatar, provider, age, weight, height, goal, score, join_date, created_at, updated_at)
		 VALUES($1, $2, $3, $4, $5, 0, 0, 0, '', 0, NOW(), NOW(), NOW())
		 RETURNING id, name, email, avatar, age, weight, height, goal, score, join_date, created_at, updated_at`,
		name, email, passwordHash, avatar, provider,
	).Scan(&user.ID, &user.Name, &user.Email, &user.Avatar, &user.Age, &user.Weight, &user.Height,
		&user.Goal, &user.Score, &user.JoinDate, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return nil, err
	}

	// Mise à jour de created_by
	_, _ = database.DB.Exec(ctx, `UPDATE users SET created_by=$1 WHERE id=$1`, user.ID)

	// Si aucun avatar n'a été fourni, générer un avatar par défaut
	if avatar == "" {
		defaultAvatar, err := GenerateDefaultAvatar(user.ID, name)
		if err == nil {
			// Mettre à jour l'avatar dans la base de données
			_, _ = database.DB.Exec(ctx, `UPDATE users SET avatar=$1 WHERE id=$2`, defaultAvatar, user.ID)
			user.Avatar = defaultAvatar
		}
	}

	user.Provider = provider
	return &user, nil
}

// FindOrCreateOAuthUser trouve ou crée un utilisateur OAuth
func FindOrCreateOAuthUser(ctx context.Context, email, name, avatar, provider string) (*model.UserProfile, error) {

	// Essayer de trouver l'utilisateur
	user, err := FindUserByEmail(ctx, email)
	if err == nil {
		return user, nil
	}

	// Créer l'utilisateur s'il n'existe pas
	return CreateUser(ctx, name, email, "", avatar, provider)
}
