package utils

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/MassBabyGeek/PumpPro-backend/internal/database"
	model "github.com/MassBabyGeek/PumpPro-backend/internal/models"
)

// FindUserByEmail recherche un utilisateur par son email
func FindUserByEmail(ctx context.Context, email string) (*model.UserProfile, error) {
	fmt.Printf("[INFO][FindUserByEmail] Recherche de l'utilisateur avec email: %s\n", email)

	var user model.UserProfile
	var avatar, goal, provider sql.NullString
	var age sql.NullInt64
	var weight, height sql.NullFloat64

	err := database.DB.QueryRow(ctx,
		`SELECT id, name, email, avatar, age, weight, height, goal, provider,
		 join_date, created_at, updated_at
		 FROM users WHERE email=$1 AND deleted_at IS NULL`,
		email,
	).Scan(&user.ID, &user.Name, &user.Email, &avatar, &age, &weight, &height,
		&goal, &provider, &user.JoinDate, &user.CreatedAt, &user.UpdatedAt)

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

	fmt.Printf("[INFO][FindUserByEmail] Utilisateur trouvé: %s (ID: %s)\n", user.Name, user.ID)
	return &user, nil
}

// FindUserByEmailWithPassword recherche un utilisateur par email et retourne aussi le hash du mot de passe
func FindUserByEmailWithPassword(ctx context.Context, email string) (*model.UserProfile, string, error) {
	fmt.Printf("[INFO][FindUserByEmailWithPassword] Recherche de l'utilisateur avec email: %s\n", email)

	var user model.UserProfile
	var passwordHash string
	var avatar, goal sql.NullString
	var age sql.NullInt64
	var weight, height sql.NullFloat64

	err := database.DB.QueryRow(ctx,
		`SELECT id, name, email, avatar, age, weight, height, goal,
		 join_date, created_at, updated_at, password_hash
		 FROM users WHERE email=$1 AND deleted_at IS NULL`,
		email,
	).Scan(&user.ID, &user.Name, &user.Email, &avatar, &age, &weight, &height,
		&goal, &user.JoinDate, &user.CreatedAt, &user.UpdatedAt, &passwordHash)

	if err != nil {
		return nil, "", err
	}

	user.Avatar = NullStringToString(avatar)
	user.Goal = NullStringToString(goal)
	user.Age = NullInt64ToInt(age)
	user.Weight = NullFloat64ToFloat64(weight)
	user.Height = NullFloat64ToFloat64(height)

	fmt.Printf("[INFO][FindUserByEmailWithPassword] Utilisateur trouvé: %s (ID: %s)\n", user.Name, user.ID)
	return &user, passwordHash, nil
}

// CreateUser crée un nouvel utilisateur
func CreateUser(ctx context.Context, name, email, passwordHash, avatar, provider string) (*model.UserProfile, error) {
	fmt.Printf("[INFO][CreateUser] Création de l'utilisateur: %s (%s) via %s\n", name, email, provider)

	var user model.UserProfile
	err := database.DB.QueryRow(ctx,
		`INSERT INTO users(name, email, password_hash, avatar, provider, age, weight, height, goal, join_date, created_at, updated_at)
		 VALUES($1, $2, $3, $4, $5, 0, 0, 0, '', NOW(), NOW(), NOW())
		 RETURNING id, name, email, avatar, age, weight, height, goal, join_date, created_at, updated_at`,
		name, email, passwordHash, avatar, provider,
	).Scan(&user.ID, &user.Name, &user.Email, &user.Avatar, &user.Age, &user.Weight, &user.Height,
		&user.Goal, &user.JoinDate, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return nil, err
	}

	// Mise à jour de created_by
	_, _ = database.DB.Exec(ctx, `UPDATE users SET created_by=$1 WHERE id=$1`, user.ID)

	user.Provider = provider
	fmt.Printf("[INFO][CreateUser] Utilisateur créé avec succès: ID=%s\n", user.ID)
	return &user, nil
}

// FindOrCreateOAuthUser trouve ou crée un utilisateur OAuth
func FindOrCreateOAuthUser(ctx context.Context, email, name, avatar, provider string) (*model.UserProfile, error) {
	fmt.Printf("[INFO][FindOrCreateOAuthUser] Tentative de connexion/création via %s pour %s\n", provider, email)

	// Essayer de trouver l'utilisateur
	user, err := FindUserByEmail(ctx, email)
	if err == nil {
		fmt.Printf("[INFO][FindOrCreateOAuthUser] Utilisateur existant trouvé\n")
		return user, nil
	}

	// Créer l'utilisateur s'il n'existe pas
	fmt.Printf("[INFO][FindOrCreateOAuthUser] Création d'un nouvel utilisateur\n")
	return CreateUser(ctx, name, email, "", avatar, provider)
}
