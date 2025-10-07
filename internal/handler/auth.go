package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/MassBabyGeek/PumpPro-backend/internal/database"
	model "github.com/MassBabyGeek/PumpPro-backend/internal/models"
	"github.com/MassBabyGeek/PumpPro-backend/internal/utils"
	"github.com/google/uuid"
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
	model.AuditFields
}

func Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := utils.DecodeJSON(r, &req); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	ctx := context.Background()
	var user model.UserProfile
	var hashedPassword string

	err := database.DB.QueryRow(ctx,
		`SELECT id, name, email, COALESCE(avatar,'') as avatar, age, weight, height, COALESCE(goal,'') as goal,
		 join_date, created_at, updated_at, password_hash
		 FROM users WHERE email=$1 AND deleted_at IS NULL`,
		req.Email,
	).Scan(&user.ID, &user.Name, &user.Email, &user.Avatar, &user.Age, &user.Weight, &user.Height,
		&user.Goal, &user.JoinDate, &user.CreatedAt, &user.UpdatedAt, &hashedPassword)

	if err != nil {
		utils.Error(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(req.Password)); err != nil {
		utils.Error(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	// Génération token UUID
	token := uuid.NewString()
	now := time.Now()

	// Création session avec created_by
	var sessionID string
	err = database.DB.QueryRow(ctx,
		`INSERT INTO sessions(user_id, token, ip_address, user_agent, is_active, created_at, expires_at, created_by)
		 VALUES($1,$2,$3,$4,true,$5,$6,$7)
		 RETURNING id`,
		user.ID, token, r.RemoteAddr, r.UserAgent(), now, now.Add(24*time.Hour), user.ID,
	).Scan(&sessionID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not create session: "+err.Error())
		return
	}

	utils.Success(w, map[string]interface{}{
		"user":  user,
		"token": token,
	})
}

func Logout(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	if token == "" {
		utils.Error(w, http.StatusBadRequest, "missing token")
		return
	}

	ctx := context.Background()

	// Récupérer l'ID de l'utilisateur de la session avant de la soft delete
	var userID string
	err := database.DB.QueryRow(ctx,
		`SELECT user_id FROM sessions WHERE token=$1 AND is_active=true AND deleted_at IS NULL`,
		token,
	).Scan(&userID)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "session not found or already logged out")
		return
	}

	// Soft delete de la session
	res, err := database.DB.Exec(ctx,
		`UPDATE sessions
		 SET is_active=false, expires_at=$2, deleted_at=NOW(), deleted_by=$3
		 WHERE token=$1 AND is_active=true AND deleted_at IS NULL`,
		token, time.Now(), userID,
	)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not logout: "+err.Error())
		return
	}

	if res.RowsAffected() == 0 {
		utils.Error(w, http.StatusNotFound, "session not found or already logged out")
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
		utils.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	ctx := context.Background()
	hashed, _ := bcrypt.GenerateFromPassword([]byte(payload.Password), bcrypt.DefaultCost)

	var user model.UserProfile
	// Lors du signup, l'utilisateur se crée lui-même, donc created_by sera l'ID retourné
	err := database.DB.QueryRow(ctx,
		`INSERT INTO users(name,email,password_hash,avatar,age,weight,height,goal,join_date,created_at,updated_at)
		 VALUES($1,$2,$3,$4,$5,$6,$7,$8,NOW(),NOW(),NOW())
		 RETURNING id, name, email, avatar, age, weight, height, goal, join_date, created_at, updated_at`,
		payload.Name, payload.Email, string(hashed), "", 0, 0, 0, "",
	).Scan(&user.ID, &user.Name, &user.Email, &user.Avatar,
		&user.Age, &user.Weight, &user.Height, &user.Goal,
		&user.JoinDate, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not create user: "+err.Error())
		return
	}

	// Mise à jour de created_by avec l'ID de l'utilisateur créé
	_, err = database.DB.Exec(ctx,
		`UPDATE users SET created_by=$1 WHERE id=$1`,
		user.ID,
	)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not update created_by: "+err.Error())
		return
	}

	// Créer un token pour l'auto-login après inscription
	token := uuid.NewString()
	now := time.Now()

	_, err = database.DB.Exec(ctx,
		`INSERT INTO sessions(user_id, token, ip_address, user_agent, is_active, created_at, expires_at, created_by)
		 VALUES($1,$2,$3,$4,true,$5,$6,$7)`,
		user.ID, token, r.RemoteAddr, r.UserAgent(), now, now.Add(24*time.Hour), user.ID,
	)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not create session: "+err.Error())
		return
	}

	utils.Success(w, map[string]interface{}{
		"user":  user,
		"token": token,
	})
}

// ResetPassword envoie un email de réinitialisation de mot de passe
func ResetPassword(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Email string `json:"email"`
	}
	if err := utils.DecodeJSON(r, &payload); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid JSON body")
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
		utils.Error(w, http.StatusInternalServerError, "could not check user: "+err.Error())
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
		utils.Error(w, http.StatusBadRequest, "invalid JSON body")
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
			created_at, updated_at, created_by, updated_by, deleted_at, deleted_by
		FROM sessions
		WHERE deleted_at IS NULL
	`)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not query sessions: "+err.Error())
		return
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var s Session
		if err := rows.Scan(
			&s.ID, &s.UserID, &s.Token, &s.ExpiresAt, &s.IsActive,
			&s.IP, &s.UserAgent,
			&s.CreatedAt, &s.UpdatedAt, &s.CreatedBy, &s.UpdatedBy, &s.DeletedAt, &s.DeletedBy,
		); err != nil {
			utils.Error(w, http.StatusInternalServerError, "could not scan session row: "+err.Error())
			return
		}
		sessions = append(sessions, s)
	}

	utils.Success(w, sessions)
}

func GetSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	ctx := context.Background()
	var session Session
	err := database.DB.QueryRow(ctx,
		`SELECT id, user_id, token, expires_at, is_active,
			ip_address, user_agent,
			created_at, updated_at, created_by, updated_by, deleted_at, deleted_by
		 FROM sessions WHERE id=$1 AND deleted_at IS NULL`,
		id,
	).Scan(&session.ID, &session.UserID, &session.Token, &session.ExpiresAt, &session.IsActive,
		&session.IP, &session.UserAgent,
		&session.CreatedAt, &session.UpdatedAt, &session.CreatedBy, &session.UpdatedBy, &session.DeletedAt, &session.DeletedBy,
	)

	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not get session: "+err.Error())
		return
	}

	utils.Success(w, session)
}
