package handler

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/MassBabyGeek/PumpPro-backend/internal/database"
	"github.com/MassBabyGeek/PumpPro-backend/internal/middleware"
	model "github.com/MassBabyGeek/PumpPro-backend/internal/models"
	"github.com/MassBabyGeek/PumpPro-backend/internal/utils"
	"github.com/gorilla/mux"

	"golang.org/x/crypto/bcrypt"
)

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type Session struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
	IsActive  bool      `json:"isActive"`
	IP        string    `json:"ipAddress"`
	UserAgent string    `json:"userAgent"`
	model.DateFields
}

func Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := utils.DecodeJSON(r, &req); err != nil {
		utils.Error(w, http.StatusBadRequest, "JSON invalide", err)
		return
	}

	ctx := context.Background()

	// Rechercher l'utilisateur avec son mot de passe
	user, hashedPassword, err := utils.FindUserByEmailWithPassword(ctx, req.Email)
	if err != nil {
		utils.ErrorSimple(w, http.StatusUnauthorized, "identifiants invalides")
		return
	}

	// Vérifier le mot de passe
	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(req.Password)); err != nil {
		utils.ErrorSimple(w, http.StatusUnauthorized, "identifiants invalides")
		return
	}

	// Créer un access token (1h) et un refresh token (30 jours)
	ip, userAgent := utils.ExtractIPAndUserAgent(r)
	accessToken, err := utils.CreateAccessToken(ctx, user.ID, ip, userAgent)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "impossible de créer l'access token", err)
		return
	}

	refreshToken, err := utils.CreateRefreshToken(ctx, user.ID, ip, userAgent)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "impossible de créer le refresh token", err)
		return
	}

	utils.Success(w, map[string]interface{}{
		"user":         user,
		"token":        accessToken,
		"refreshToken": refreshToken,
	})
}

func Logout(w http.ResponseWriter, r *http.Request) {
	token, err := middleware.GetTokenFromContext(r)
	if err != nil {
		utils.Error(w, http.StatusBadRequest, "token manquant", err)
		return
	}

	ctx := context.Background()

	// Invalider la session
	if err := utils.InvalidateSession(ctx, token); err != nil {
		utils.Error(w, http.StatusNotFound, "session introuvable ou déjà déconnectée", err)
		return
	}

	utils.Success(w, map[string]bool{"success": true})
}

// Register (alias de Signup pour correspondre à l'API TypeScript)
func Register(w http.ResponseWriter, r *http.Request) {
	Signup(w, r)
}

func Signup(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := utils.DecodeJSON(r, &payload); err != nil {
		utils.Error(w, http.StatusBadRequest, "JSON invalide", err)
		return
	}

	ctx := context.Background()

	// Hasher le mot de passe
	hashed, err := bcrypt.GenerateFromPassword([]byte(payload.Password), bcrypt.DefaultCost)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "impossible de hasher le mot de passe", err)
		return
	}

	// Créer l'utilisateur
	user, err := utils.CreateUser(ctx, payload.Name, payload.Email, string(hashed), "", "email")
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "impossible de créer l'utilisateur", err)
		return
	}

	// Créer un access token (1h) et un refresh token (30 jours) pour l'auto-login
	ip, userAgent := utils.ExtractIPAndUserAgent(r)
	accessToken, err := utils.CreateAccessToken(ctx, user.ID, ip, userAgent)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "impossible de créer l'access token", err)
		return
	}

	refreshToken, err := utils.CreateRefreshToken(ctx, user.ID, ip, userAgent)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "impossible de créer le refresh token", err)
		return
	}

	utils.Success(w, map[string]interface{}{
		"user":         user,
		"token":        accessToken,
		"refreshToken": refreshToken,
	})
}

// ResetPassword envoie un email de réinitialisation de mot de passe
func ResetPassword(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Email string `json:"email"`
	}
	if err := utils.DecodeJSON(r, &payload); err != nil {
		utils.Error(w, http.StatusBadRequest, "JSON invalide", err)
		return
	}

	ctx := context.Background()

	// Vérifier si l'utilisateur existe
	var userExists bool
	err := database.DB.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM users WHERE email=$1 AND deleted_at IS NULL)`,
		payload.Email,
	).Scan(&userExists)

	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "impossible de vérifier l'utilisateur", err)
		return
	}

	if !userExists {
		// Pour la sécurité, on ne révèle pas si l'email existe ou non
		utils.Success(w, map[string]bool{"success": true})
		return
	}

	// TODO: Générer un token de réinitialisation et envoyer l'email
	// Pour l'instant, on retourne simplement success
	// En production, créer une table password_reset_tokens et envoyer l'email

	utils.Success(w, map[string]bool{"success": true})
}

// VerifyEmail vérifie l'email d'un utilisateur
func VerifyEmail(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Token string `json:"token"`
	}
	if err := utils.DecodeJSON(r, &payload); err != nil {
		utils.Error(w, http.StatusBadRequest, "JSON invalide", err)
		return
	}

	// TODO: Implémenter la vérification d'email
	// Pour l'instant, on retourne simplement success
	// En production, vérifier le token et marquer l'email comme vérifié

	utils.Success(w, map[string]bool{"success": true})
}

func GetSessions(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	rows, err := database.DB.Query(ctx, `
		SELECT
			id, user_id, token, expires_at, is_active,
			ip_address, user_agent,
			created_at, updated_at, created_by, updated_by
		FROM sessions
		WHERE deleted_at IS NULL
	`)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "impossible de récupérer les sessions", err)
		return
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var s Session
		var createdBy, updatedBy sql.NullString

		if err := rows.Scan(
			&s.ID, &s.UserID, &s.Token, &s.ExpiresAt, &s.IsActive,
			&s.IP, &s.UserAgent,
			&s.CreatedAt, &s.UpdatedAt, &createdBy, &updatedBy,
		); err != nil {
			utils.Error(w, http.StatusInternalServerError, "erreur de lecture des sessions", err)
			return
		}

		s.CreatedBy = utils.NullStringToPointer(createdBy)
		s.UpdatedBy = utils.NullStringToPointer(updatedBy)
		sessions = append(sessions, s)
	}

	utils.Success(w, sessions)
}

func GetSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	ctx := context.Background()
	var session Session
	var createdBy, updatedBy sql.NullString

	err := database.DB.QueryRow(ctx,
		`SELECT id, user_id, token, expires_at, is_active,
			ip_address, user_agent,
			created_at, updated_at, created_by, updated_by
		 FROM sessions WHERE id=$1 AND deleted_at IS NULL`,
		id,
	).Scan(&session.ID, &session.UserID, &session.Token, &session.ExpiresAt, &session.IsActive,
		&session.IP, &session.UserAgent,
		&session.CreatedAt, &session.UpdatedAt, &createdBy, &updatedBy,
	)

	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "impossible de récupérer la session", err)
		return
	}

	session.CreatedBy = utils.NullStringToPointer(createdBy)
	session.UpdatedBy = utils.NullStringToPointer(updatedBy)

	utils.Success(w, session)
}

// GoogleAuth gère l'authentification via Google OAuth
func GoogleAuth(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		IDToken string `json:"idToken"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Avatar  string `json:"avatar"`
	}
	if err := utils.DecodeJSON(r, &payload); err != nil {
		utils.Error(w, http.StatusBadRequest, "JSON invalide", err)
		return
	}

	// Validation basique
	if payload.Email == "" {
		utils.ErrorSimple(w, http.StatusBadRequest, "email requis")
		return
	}

	ctx := context.Background()

	// Trouver ou créer l'utilisateur OAuth
	user, err := utils.FindOrCreateOAuthUser(ctx, payload.Email, payload.Name, payload.Avatar, "google")
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "impossible de créer/trouver l'utilisateur", err)
		return
	}

	// Créer un access token (1h) et un refresh token (30 jours)
	ip, userAgent := utils.ExtractIPAndUserAgent(r)
	accessToken, err := utils.CreateAccessToken(ctx, user.ID, ip, userAgent)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "impossible de créer l'access token", err)
		return
	}

	refreshToken, err := utils.CreateRefreshToken(ctx, user.ID, ip, userAgent)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "impossible de créer le refresh token", err)
		return
	}

	utils.Success(w, map[string]interface{}{
		"user":         user,
		"token":        accessToken,
		"refreshToken": refreshToken,
	})
}

// RefreshToken génère un nouveau access token et refresh token à partir d'un refresh token valide
func RefreshToken(w http.ResponseWriter, r *http.Request) {
	// Récupérer le refresh token depuis le body
	var payload struct {
		RefreshToken string `json:"refreshToken"`
	}
	if err := utils.DecodeJSON(r, &payload); err != nil {
		utils.Error(w, http.StatusBadRequest, "JSON invalide", err)
		return
	}

	if payload.RefreshToken == "" {
		utils.ErrorSimple(w, http.StatusBadRequest, "refresh token manquant")
		return
	}

	ctx := context.Background()

	// Valider le refresh token et récupérer l'ID utilisateur
	userID, err := utils.ValidateRefreshToken(ctx, payload.RefreshToken)
	if err != nil {
		utils.ErrorSimple(w, http.StatusUnauthorized, "refresh token invalide ou expiré")
		return
	}

	// Révoquer l'ancien refresh token (rotation)
	if err := utils.RevokeRefreshToken(ctx, payload.RefreshToken); err != nil {
		// Log l'erreur mais continue quand même
		// (le token pourrait déjà être révoqué, ce qui n'est pas critique)
	}

	// Récupérer les informations de l'utilisateur
	user, _, err := utils.FindUserByID(ctx, userID)
	if err != nil {
		utils.ErrorSimple(w, http.StatusUnauthorized, "utilisateur introuvable")
		return
	}

	// Créer un nouveau access token et refresh token
	ip, userAgent := utils.ExtractIPAndUserAgent(r)
	newAccessToken, err := utils.CreateAccessToken(ctx, userID, ip, userAgent)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "impossible de créer l'access token", err)
		return
	}

	newRefreshToken, err := utils.CreateRefreshToken(ctx, userID, ip, userAgent)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "impossible de créer le refresh token", err)
		return
	}

	utils.Success(w, map[string]interface{}{
		"user":         user,
		"token":        newAccessToken,
		"refreshToken": newRefreshToken,
	})
}

// AppleAuth gère l'authentification via Apple Sign In
func AppleAuth(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		IDToken      string `json:"idToken"`
		Email        string `json:"email"`
		Name         string `json:"name"`
		UserIdentity string `json:"userIdentity"`
	}
	if err := utils.DecodeJSON(r, &payload); err != nil {
		utils.Error(w, http.StatusBadRequest, "JSON invalide", err)
		return
	}

	// Validation basique
	if payload.Email == "" && payload.UserIdentity == "" {
		utils.ErrorSimple(w, http.StatusBadRequest, "email ou userIdentity requis")
		return
	}

	ctx := context.Background()

	// Déterminer les valeurs pour l'utilisateur
	userName := payload.Name
	if userName == "" {
		userName = "Apple User"
	}

	userEmail := payload.Email
	if userEmail == "" {
		userEmail = payload.UserIdentity + "@appleid.private"
	}

	// Trouver ou créer l'utilisateur OAuth
	user, err := utils.FindOrCreateOAuthUser(ctx, userEmail, userName, "", "apple")
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "impossible de créer/trouver l'utilisateur", err)
		return
	}

	// Créer un access token (1h) et un refresh token (30 jours)
	ip, userAgent := utils.ExtractIPAndUserAgent(r)
	accessToken, err := utils.CreateAccessToken(ctx, user.ID, ip, userAgent)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "impossible de créer l'access token", err)
		return
	}

	refreshToken, err := utils.CreateRefreshToken(ctx, user.ID, ip, userAgent)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "impossible de créer le refresh token", err)
		return
	}

	utils.Success(w, map[string]interface{}{
		"user":         user,
		"token":        accessToken,
		"refreshToken": refreshToken,
	})
}
