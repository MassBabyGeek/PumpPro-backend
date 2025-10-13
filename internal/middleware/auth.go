package middleware

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/MassBabyGeek/PumpPro-backend/internal/database"
	model "github.com/MassBabyGeek/PumpPro-backend/internal/models"
	"github.com/MassBabyGeek/PumpPro-backend/internal/utils"
)

// Context keys
type contextKey string

const (
	userContextKey  = contextKey("user")
	tokenContextKey = contextKey("token")
)

// AuthMiddleware valide le token et injecte l'utilisateur dans le contexte
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Récupérer le token depuis le header Authorization
		token := r.Header.Get("Authorization")
		if token == "" {
			utils.ErrorSimple(w, http.StatusUnauthorized, "missing authorization token")
			return
		}

		// Valider le token et récupérer l'utilisateur
		user, err := validateTokenAndGetUser(r.Context(), token)
		if err != nil {
			utils.ErrorSimple(w, http.StatusUnauthorized, fmt.Sprintf("invalid token: %v", err))
			return
		}

		fmt.Println("[DEBUG][AuthMiddleware] User found:", user.Name, "(ID:", user.ID, ")")

		// Injecter l'utilisateur et le token dans le contexte
		ctx := context.WithValue(r.Context(), userContextKey, *user)
		ctx = context.WithValue(ctx, tokenContextKey, token)

		// Appeler le handler suivant avec le contexte enrichi
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// validateTokenAndGetUser valide le token et retourne l'utilisateur associé
func validateTokenAndGetUser(ctx context.Context, token string) (*model.UserProfile, error) {
	var user model.UserProfile
	var avatar, goal, provider sql.NullString
	var age sql.NullInt64
	var weight, height sql.NullFloat64
	var updatedBy sql.NullString
	var isActive bool

	query := `
	SELECT
		u.id, u.name, u.email, u.avatar, u.age, u.weight, u.height, u.goal, u.provider,
		u.join_date, u.created_at, u.updated_at, u.created_by, u.updated_by,
		s.is_active
	FROM users u
	JOIN sessions s ON u.id = s.user_id
	WHERE s.token = $1
		AND s.is_active = true
		AND s.expires_at > NOW()
		AND u.deleted_at IS NULL
		AND s.deleted_at IS NULL`

	// Requête pour valider le token et récupérer l'utilisateur
	err := database.DB.QueryRow(ctx, query, token).Scan(
		&user.ID, &user.Name, &user.Email, &avatar, &age, &weight, &height, &goal, &provider,
		&user.JoinDate, &user.CreatedAt, &user.UpdatedAt, &user.CreatedBy, &updatedBy,
		&isActive,
	)

	fmt.Printf("[INFO][validateTokenAndGetUser] Utilisateur validé: %s (ID: %s)\n", user.Name, user.ID)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("token not found or expired")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Convertir les valeurs NULL
	user.Avatar = utils.NullStringToString(avatar)
	user.Goal = utils.NullStringToString(goal)
	user.Provider = utils.NullStringToString(provider)
	user.Age = utils.NullInt64ToInt(age)
	user.Weight = utils.NullFloat64ToFloat64(weight)
	user.Height = utils.NullFloat64ToFloat64(height)
	user.UpdatedBy = utils.NullStringToPointer(updatedBy)

	return &user, nil
}

// GetUserFromContext récupère l'utilisateur depuis le contexte de la requête
func GetUserFromContext(r *http.Request) (model.UserProfile, error) {
	user, ok := r.Context().Value(userContextKey).(model.UserProfile)
	if !ok {
		return model.UserProfile{}, fmt.Errorf("user not found in context")
	}
	return user, nil
}

// GetTokenFromContext récupère le token depuis le contexte de la requête
func GetTokenFromContext(r *http.Request) (string, error) {
	token, ok := r.Context().Value(tokenContextKey).(string)
	if !ok || token == "" {
		return "", fmt.Errorf("token not found in context")
	}
	return token, nil
}

// GetUserIDFromContext récupère l'ID de l'utilisateur depuis le contexte (helper)
func GetUserIDFromContext(r *http.Request) (string, error) {
	user, err := GetUserFromContext(r)
	if err != nil {
		return "", err
	}
	return user.ID, nil
}

// RequireAuth est un helper pour vérifier qu'un utilisateur est authentifié dans un handler
func RequireAuth(r *http.Request) (model.UserProfile, error) {
	return GetUserFromContext(r)
}

// ValidateToken valide un token sans passer par le middleware (utile pour des cas spécifiques)
func ValidateToken(ctx context.Context, token string) (*model.UserProfile, error) {
	return validateTokenAndGetUser(ctx, token)
}
