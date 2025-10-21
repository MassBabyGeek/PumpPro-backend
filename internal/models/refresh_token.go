package model

import "time"

// RefreshToken représente un refresh token en base de données
type RefreshToken struct {
	ID        string     `json:"id"`
	UserID    string     `json:"userId"`
	TokenHash string     `json:"-"` // Ne jamais exposer le hash dans l'API
	ExpiresAt time.Time  `json:"expiresAt"`
	RevokedAt *time.Time `json:"revokedAt,omitempty"`
	IPAddress string     `json:"ipAddress,omitempty"`
	UserAgent string     `json:"userAgent,omitempty"`
	DateFields
}

// AuthResponse représente la réponse complète lors de l'authentification
type AuthResponse struct {
	User         *UserProfile `json:"user"`
	Token        string       `json:"token"`        // Access token (expire 1h)
	RefreshToken string       `json:"refreshToken"` // Refresh token (expire 30 jours)
}
