package utils

import (
	"context"
	"fmt"

	"github.com/MassBabyGeek/PumpPro-backend/internal/database"
)

// IncrementUserScore incrémente le score d'un utilisateur
func IncrementUserScore(ctx context.Context, userID string, points int) error {

	_, err := database.DB.Exec(ctx,
		`UPDATE users SET score = score + $1 WHERE id = $2 AND deleted_at IS NULL`,
		points, userID,
	)
	if err != nil {
		return fmt.Errorf("impossible d'incrémenter le score: %w", err)
	}

	return nil
}
