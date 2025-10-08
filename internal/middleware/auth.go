package middleware

import (
	"context"
	"fmt"
	"net/http"

	"github.com/MassBabyGeek/PumpPro-backend/internal/database"
	model "github.com/MassBabyGeek/PumpPro-backend/internal/models"
	"github.com/MassBabyGeek/PumpPro-backend/internal/utils"
)

// context keys (non exportées)
type contextKey string

const (
	userContextKey  = contextKey("user")
	tokenContextKey = contextKey("token")
)

// AuthMiddleware récupère le token, charge l'utilisateur et l'injecte dans le contexte
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Récupérer le token depuis l’en-tête Authorization
		token := r.Header.Get("Authorization")
		if token == "" {
			utils.ErrorSimple(w, http.StatusUnauthorized, "missing token")
			return
		}

		// Récupérer l'utilisateur lié au token
		var user model.UserProfile
		ctx := context.Background()
		err := database.DB.QueryRow(ctx, `
			SELECT
				u.id, u.name, u.email,
				COALESCE(u.avatar,'') as avatar,
				COALESCE(u.age,0) as age,
				COALESCE(u.weight,0) as weight,
				COALESCE(u.height,0) as height,
				COALESCE(u.goal,'') as goal,
				u.join_date, u.created_at, u.updated_at,
				u.created_by,
				u.updated_by,
				COALESCE(u.deleted_at, '1970-01-01'::timestamp) as deleted_at,
				u.deleted_by
			FROM users u
			INNER JOIN sessions s ON u.id = s.user_id
			WHERE s.token = $1 AND s.is_active = true;
		`, token).Scan(
			&user.ID, &user.Name, &user.Email, &user.Avatar, &user.Age, &user.Weight, &user.Height,
			&user.Goal, &user.JoinDate, &user.CreatedAt, &user.UpdatedAt,
			&user.CreatedBy, &user.UpdatedBy, &user.DeletedAt, &user.DeletedBy,
		)
		if err != nil {
			utils.ErrorSimple(w, http.StatusUnauthorized, fmt.Sprintf("invalid token: %v", err))
			return
		}

		if err != nil || user.ID == "" {
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		// Si on arrive ici, le token et l’utilisateur sont valides
		ctx = context.WithValue(ctx, userContextKey, user)
		ctx = context.WithValue(ctx, tokenContextKey, token)

		// Appeler le handler suivant avec ce contexte enrichi
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUserFromContext : renvoie le user injecté dans le contexte
func GetUserFromContext(r *http.Request) (model.UserProfile, error) {
	user, ok := r.Context().Value(userContextKey).(model.UserProfile)
	if !ok {
		return model.UserProfile{}, fmt.Errorf("user not found in context")
	}
	return user, nil
}

// GetTokenFromContext : renvoie le token injecté dans le contexte
func GetTokenFromContext(r *http.Request) (string, error) {
	token, ok := r.Context().Value(tokenContextKey).(string)
	if !ok || token == "" {
		return "", fmt.Errorf("token not found in context")
	}
	return token, nil
}
