package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/MassBabyGeek/PumpPro-backend/internal/database"
	model "github.com/MassBabyGeek/PumpPro-backend/internal/models"
)

func DecodeJSON(r *http.Request, dest interface{}) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dest)
}

func GetToken(r *http.Request) (string, error) {
	token := r.Header.Get("Authorization")
	if token == "" {
		return "", fmt.Errorf("missing token")
	}
	return token, nil
}

func GetUserByToken(token string) (model.UserProfile, error) {
	var user model.UserProfile

	if token == "" {
		return user, fmt.Errorf("empty token")
	}

	ctx := context.Background()
	err := database.DB.QueryRow(ctx, `
		SELECT
			u.id,
			u.name,
			u.email,
			COALESCE(u.avatar,'') as avatar,
			COALESCE(u.age,0) as age,
			COALESCE(u.weight,0) as weight,
			COALESCE(u.height,0) as height,
			COALESCE(u.goal,'') as goal,
			u.join_date,
			u.created_at,
			u.updated_at,
			COALESCE(u.created_by,'') as created_by,
			COALESCE(u.updated_by,'') as updated_by,
			COALESCE(u.deleted_at, '1970-01-01') as deleted_at,  -- <-- default value
			COALESCE(u.deleted_by,'') as deleted_by
		FROM users u
		INNER JOIN sessions s ON u.id = s.user_id
		WHERE s.token = $1 AND s.is_active = true AND u.deleted_at IS NULL
	`, token).Scan(
		&user.ID, &user.Name, &user.Email, &user.Avatar, &user.Age, &user.Weight, &user.Height,
		&user.Goal, &user.JoinDate, &user.CreatedAt, &user.UpdatedAt,
		&user.CreatedBy, &user.UpdatedBy, &user.DeletedAt, &user.DeletedBy,
	)
	if err != nil {
		return user, fmt.Errorf("user not found or invalid token: %w", err)
	}

	if user.ID == "" {
		return user, fmt.Errorf("invalid user ID retrieved from token")
	}

	return user, nil
}
