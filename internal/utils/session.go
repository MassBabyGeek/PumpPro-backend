package utils

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/MassBabyGeek/PumpPro-backend/internal/database"
	"github.com/google/uuid"
)

// SessionDuration durée de validité d'une session (24h)
const SessionDuration = 24 * time.Hour

// CreateSession crée une nouvelle session pour un utilisateur
func CreateSession(ctx context.Context, userID, ipAddress, userAgent string) (string, error) {
	fmt.Printf("[INFO][CreateSession] Création d'une session pour l'utilisateur: %s\n", userID)

	token := uuid.NewString()
	now := time.Now()

	var sessionID string
	err := database.DB.QueryRow(ctx,
		`INSERT INTO sessions(user_id, token, ip_address, user_agent, is_active, created_at, expires_at, created_by)
		 VALUES($1, $2, $3, $4, true, $5, $6, $7)
		 RETURNING id`,
		userID, token, ipAddress, userAgent, now, now.Add(SessionDuration), userID,
	).Scan(&sessionID)

	if err != nil {
		return "", err
	}

	fmt.Printf("[INFO][CreateSession] Session créée avec succès: ID=%s, Token=%s\n", sessionID, token)
	return token, nil
}

// InvalidateSession invalide une session (soft delete)
func InvalidateSession(ctx context.Context, token string) error {
	fmt.Printf("[INFO][InvalidateSession] Invalidation de la session avec token: %s\n", token)

	// Récupérer l'ID de l'utilisateur
	var userID string
	err := database.DB.QueryRow(ctx,
		`SELECT user_id FROM sessions WHERE token=$1 AND is_active=true AND deleted_at IS NULL`,
		token,
	).Scan(&userID)

	if err != nil {
		return fmt.Errorf("session introuvable ou déjà invalide")
	}

	// Soft delete de la session
	res, err := database.DB.Exec(ctx,
		`UPDATE sessions
		 SET is_active=false, expires_at=$2, deleted_at=NOW(), deleted_by=$3
		 WHERE token=$1 AND is_active=true AND deleted_at IS NULL`,
		token, time.Now(), userID,
	)

	if err != nil {
		return err
	}

	if res.RowsAffected() == 0 {
		return fmt.Errorf("aucune session invalide")
	}

	fmt.Printf("[INFO][InvalidateSession] Session invalidée avec succès\n")
	return nil
}

// ExtractIPAndUserAgent extrait l'IP et le User-Agent depuis une requête HTTP
func ExtractIPAndUserAgent(r *http.Request) (string, string) {
	ip := r.RemoteAddr
	userAgent := r.UserAgent()
	fmt.Printf("[INFO][ExtractIPAndUserAgent] IP=%s, UserAgent=%s\n", ip, userAgent)
	return ip, userAgent
}
