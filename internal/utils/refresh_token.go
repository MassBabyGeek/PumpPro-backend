package utils

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/MassBabyGeek/PumpPro-backend/internal/database"
	"github.com/google/uuid"
)

// RefreshTokenDuration durée de validité d'un refresh token (30 jours)
const RefreshTokenDuration = 30 * 24 * time.Hour

// AccessTokenDuration durée de validité d'un access token (1 heure)
const AccessTokenDuration = 1 * time.Hour

// CreateRefreshToken crée un nouveau refresh token pour un utilisateur
func CreateRefreshToken(ctx context.Context, userID, ipAddress, userAgent string) (string, error) {
	fmt.Printf("[INFO][CreateRefreshToken] Création d'un refresh token pour l'utilisateur: %s\n", userID)

	// Générer un token unique
	token := uuid.NewString()

	// Hasher le token avant de le stocker
	tokenHash := hashToken(token)

	now := time.Now()
	expiresAt := now.Add(RefreshTokenDuration)

	// Insérer en base de données
	var refreshTokenID string
	err := database.DB.QueryRow(ctx,
		`INSERT INTO refresh_tokens(user_id, token_hash, ip_address, user_agent, expires_at, created_at, created_by)
		 VALUES($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id`,
		userID, tokenHash, ipAddress, userAgent, expiresAt, now, userID,
	).Scan(&refreshTokenID)

	if err != nil {
		return "", fmt.Errorf("erreur lors de la création du refresh token: %w", err)
	}

	fmt.Printf("[INFO][CreateRefreshToken] Refresh token créé avec succès: ID=%s\n", refreshTokenID)
	return token, nil
}

// ValidateRefreshToken valide un refresh token et retourne l'ID utilisateur
func ValidateRefreshToken(ctx context.Context, token string) (string, error) {
	fmt.Printf("[INFO][ValidateRefreshToken] Validation du refresh token\n")

	tokenHash := hashToken(token)

	var userID string
	var expiresAt time.Time
	var revokedAt *time.Time

	err := database.DB.QueryRow(ctx,
		`SELECT user_id, expires_at, revoked_at
		 FROM refresh_tokens
		 WHERE token_hash=$1 AND deleted_at IS NULL`,
		tokenHash,
	).Scan(&userID, &expiresAt, &revokedAt)

	if err != nil {
		return "", fmt.Errorf("refresh token invalide ou introuvable")
	}

	// Vérifier si le token est révoqué
	if revokedAt != nil {
		return "", fmt.Errorf("refresh token révoqué")
	}

	// Vérifier si le token est expiré
	if time.Now().After(expiresAt) {
		return "", fmt.Errorf("refresh token expiré")
	}

	fmt.Printf("[INFO][ValidateRefreshToken] Refresh token valide pour l'utilisateur: %s\n", userID)
	return userID, nil
}

// RevokeRefreshToken révoque un refresh token
func RevokeRefreshToken(ctx context.Context, token string) error {
	fmt.Printf("[INFO][RevokeRefreshToken] Révocation du refresh token\n")

	tokenHash := hashToken(token)
	now := time.Now()

	// Récupérer l'ID de l'utilisateur pour le champ revoked_by
	var userID string
	err := database.DB.QueryRow(ctx,
		`SELECT user_id FROM refresh_tokens WHERE token_hash=$1 AND deleted_at IS NULL`,
		tokenHash,
	).Scan(&userID)


	if err != nil {
		return fmt.Errorf("refresh token introuvable")
	}

	// Révoquer le token
	res, err := database.DB.Exec(ctx,
		`UPDATE refresh_tokens
		 SET revoked_at=$2, updated_at=$3, updated_by=$4
		 WHERE token_hash=$1 AND deleted_at IS NULL AND revoked_at IS NULL`,
		tokenHash, now, now, userID,
	)

	if err != nil {
		return fmt.Errorf("erreur lors de la révocation: %w", err)
	}

	if res.RowsAffected() == 0 {
		return fmt.Errorf("refresh token déjà révoqué ou introuvable")
	}

	fmt.Printf("[INFO][RevokeRefreshToken] Refresh token révoqué avec succès\n")
	return nil
}

// RevokeAllUserRefreshTokens révoque tous les refresh tokens d'un utilisateur
func RevokeAllUserRefreshTokens(ctx context.Context, userID string) error {
	fmt.Printf("[INFO][RevokeAllUserRefreshTokens] Révocation de tous les refresh tokens pour l'utilisateur: %s\n", userID)

	now := time.Now()

	res, err := database.DB.Exec(ctx,
		`UPDATE refresh_tokens
		 SET revoked_at=$2, updated_at=$3, updated_by=$4
		 WHERE user_id=$1 AND deleted_at IS NULL AND revoked_at IS NULL`,
		userID, now, now, userID,
	)

	if err != nil {
		return fmt.Errorf("erreur lors de la révocation: %w", err)
	}

	fmt.Printf("[INFO][RevokeAllUserRefreshTokens] %d refresh tokens révoqués\n", res.RowsAffected())
	return nil
}

// CreateAccessToken crée un access token avec une durée de vie de 1h
func CreateAccessToken(ctx context.Context, userID, ipAddress, userAgent string) (string, error) {
	fmt.Printf("[INFO][CreateAccessToken] Création d'un access token pour l'utilisateur: %s\n", userID)

	token := uuid.NewString()
	now := time.Now()
	expiresAt := now.Add(AccessTokenDuration)

	var sessionID string
	err := database.DB.QueryRow(ctx,
		`INSERT INTO sessions(user_id, token, ip_address, user_agent, is_active, created_at, expires_at, created_by)
		 VALUES($1, $2, $3, $4, true, $5, $6, $7)
		 RETURNING id`,
		userID, token, ipAddress, userAgent, now, expiresAt, userID,
	).Scan(&sessionID)

	if err != nil {
		return "", err
	}

	fmt.Printf("[INFO][CreateAccessToken] Access token créé avec succès: ID=%s, Token=%s\n", sessionID, token)
	return token, nil
}

// hashToken génère un hash SHA-256 du token
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
