package utils

import (
	"context"
	"database/sql"

	"github.com/MassBabyGeek/PumpPro-backend/internal/database"
	model "github.com/MassBabyGeek/PumpPro-backend/internal/models"
)

// AddLike ajoute un like pour une entité
func AddLike(ctx context.Context, userID string, entityType model.EntityType, entityID string) error {
	_, err := database.DB.Exec(ctx, `
		INSERT INTO likes (user_id, entity_type, entity_id, created_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (user_id, entity_type, entity_id) DO NOTHING
	`, userID, entityType, entityID)

	return err
}

// RemoveLike retire un like pour une entité
func RemoveLike(ctx context.Context, userID string, entityType model.EntityType, entityID string) error {
	_, err := database.DB.Exec(ctx, `
		DELETE FROM likes
		WHERE user_id = $1 AND entity_type = $2 AND entity_id = $3
	`, userID, entityType, entityID)

	return err
}

// ToggleLike ajoute ou retire un like selon l'état actuel (retourne true si liked, false si unliked)
func ToggleLike(ctx context.Context, userID string, entityType model.EntityType, entityID string) (bool, error) {
	// Vérifier si le like existe déjà
	var exists bool
	err := database.DB.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM likes
			WHERE user_id = $1 AND entity_type = $2 AND entity_id = $3
		)
	`, userID, entityType, entityID).Scan(&exists)

	if err != nil {
		return false, err
	}

	if exists {
		// Unlike
		err = RemoveLike(ctx, userID, entityType, entityID)
		return false, err
	} else {
		// Like
		err = AddLike(ctx, userID, entityType, entityID)
		return true, err
	}
}

// GetLikeInfo récupère les informations de like pour une entité
func GetLikeInfo(ctx context.Context, userID *string, entityType model.EntityType, entityID string) (*model.LikeInfo, error) {
	var info model.LikeInfo

	// Compter le nombre total de likes
	err := database.DB.QueryRow(ctx, `
		SELECT COUNT(*) FROM likes
		WHERE entity_type = $1 AND entity_id = $2
	`, entityType, entityID).Scan(&info.TotalLikes)

	if err != nil {
		return nil, err
	}

	// Vérifier si l'utilisateur a liké (si un userID est fourni)
	if userID != nil && *userID != "" {
		var liked sql.NullBool
		err = database.DB.QueryRow(ctx, `
			SELECT TRUE FROM likes
			WHERE user_id = $1 AND entity_type = $2 AND entity_id = $3
			LIMIT 1
		`, *userID, entityType, entityID).Scan(&liked)

		if err != nil && err != sql.ErrNoRows {
			return nil, err
		}

		info.UserLiked = liked.Valid && liked.Bool
	}

	return &info, nil
}

// GetUserLikes récupère tous les likes d'un utilisateur pour un type d'entité
func GetUserLikes(ctx context.Context, userID string, entityType model.EntityType) ([]string, error) {
	rows, err := database.DB.Query(ctx, `
		SELECT entity_id FROM likes
		WHERE user_id = $1 AND entity_type = $2
		ORDER BY created_at DESC
	`, userID, entityType)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entityIDs []string
	for rows.Next() {
		var entityID string
		if err := rows.Scan(&entityID); err != nil {
			return nil, err
		}
		entityIDs = append(entityIDs, entityID)
	}

	return entityIDs, nil
}

// GetTopLikedEntities récupère les entités les plus likées d'un type donné
func GetTopLikedEntities(ctx context.Context, entityType model.EntityType, limit int) ([]model.LikesCount, error) {
	rows, err := database.DB.Query(ctx, `
		SELECT entity_type, entity_id, COUNT(*) as total_likes
		FROM likes
		WHERE entity_type = $1
		GROUP BY entity_type, entity_id
		ORDER BY total_likes DESC
		LIMIT $2
	`, entityType, limit)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []model.LikesCount
	for rows.Next() {
		var lc model.LikesCount
		if err := rows.Scan(&lc.EntityType, &lc.EntityID, &lc.TotalLikes); err != nil {
			return nil, err
		}
		results = append(results, lc)
	}

	return results, nil
}
