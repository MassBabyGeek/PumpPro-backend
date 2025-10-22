package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/MassBabyGeek/PumpPro-backend/internal/database"
	model "github.com/MassBabyGeek/PumpPro-backend/internal/models"
	"github.com/MassBabyGeek/PumpPro-backend/internal/scanner"
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
		token, err := GetToken(r)
		if err != nil {
			utils.Error(w, http.StatusUnauthorized, "missing authorization token", err)
			return
		}

		// Valider le token et récupérer l'utilisateur
		user, err := validateTokenAndGetUser(r.Context(), token)
		if err != nil {
			utils.ErrorSimple(w, http.StatusUnauthorized, fmt.Sprintf("invalid token: %v", err))
			return
		}

		// Injecter l'utilisateur et le token dans le contexte
		ctx := context.WithValue(r.Context(), userContextKey, *user)
		ctx = context.WithValue(ctx, tokenContextKey, token)

		// Appeler le handler suivant avec le contexte enrichi
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		token, err := GetToken(r)
		if err != nil || token == "" {
			next.ServeHTTP(w, r)
			return
		}

		ctx = context.WithValue(ctx, tokenContextKey, token)

		user, err := validateTokenAndGetUser(ctx, token)
		if err != nil || user == nil {
			next.ServeHTTP(w, r)
			return
		}

		ctx = context.WithValue(ctx, userContextKey, *user)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetToken(r *http.Request) (string, error) {
	token := r.Header.Get("Authorization")
	if token == "" {
		return "", fmt.Errorf("token not found in context")
	}
	return token, nil
}

// validateTokenAndGetUser valide le token et retourne l'utilisateur associé
func validateTokenAndGetUser(ctx context.Context, token string) (*model.UserProfile, error) {
	// Créer un contexte avec timeout pour éviter les "context canceled" en prod
	// Si le contexte parent est déjà annulé, on utilise context.Background()
	queryCtx := ctx
	if ctx.Err() != nil {
		queryCtx = context.Background()
	}

	// Appliquer un timeout de 5 secondes pour la requête DB
	queryCtx, cancel := context.WithTimeout(queryCtx, 5*time.Second)
	defer cancel()

	query := `
	SELECT
		u.id,
		u.name,
		u.email,
		u.avatar,
		u.age,
		u.weight,
		u.height,
		u.goal,
		u.score,
		u.join_date,
		u.created_at,
		u.updated_at,
		u.created_by,
		u.updated_by
	FROM users u
	JOIN sessions s ON u.id = s.user_id
	WHERE s.token = $1
		AND s.is_active = true
		AND s.expires_at > NOW()
		AND u.deleted_at IS NULL
		AND s.deleted_at IS NULL
	LIMIT 1
	`

	row := database.DB.QueryRow(queryCtx, query, token)

	user, err := scanner.ScanUserProfile(row)
	if err != nil {
		fmt.Printf("[INFO][validateTokenAndGetUser] Erreur de scan: %v\n", err)
		return nil, err
	}

	return user, nil
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
