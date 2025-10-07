package utils

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/MassBabyGeek/PumpPro-backend/internal/database"
	model "github.com/MassBabyGeek/PumpPro-backend/internal/models"
)

func DecodeJSON(r *http.Request, dest interface{}) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dest)
}

func GetToken(r *http.Request) string {
	token := r.Header.Get("Authorization")
	if token == "" {
		token = r.URL.Query().Get("token")
	}
	return token
}

func GetUserByToken(token string) (model.UserProfile, error) {
	var user model.UserProfile

	ctx := context.Background()
	err := database.DB.QueryRow(ctx, `
		SELECT
			id, name, email, COALESCE(avatar,'') as avatar, age, weight, height,
			COALESCE(goal,'') as goal, join_date, created_at, updated_at,
			created_by, updated_by, deleted_at, deleted_by
		FROM users
		WHERE id=$1 AND deleted_at IS NULL
	`, token).Scan(
		&user.ID, &user.Name, &user.Email, &user.Avatar, &user.Age, &user.Weight, &user.Height,
		&user.Goal, &user.JoinDate, &user.CreatedAt, &user.UpdatedAt,
		&user.CreatedBy, &user.UpdatedBy, &user.DeletedAt, &user.DeletedBy,
	)

	if err != nil {
		return user, err
	}

	return user, nil
}
