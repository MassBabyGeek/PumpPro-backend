package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/MassBabyGeek/PumpPro-backend/internal/middleware"
	model "github.com/MassBabyGeek/PumpPro-backend/internal/models"
	"github.com/MassBabyGeek/PumpPro-backend/internal/utils"
	"github.com/gorilla/mux"
)

// ToggleLike ajoute ou retire un like (endpoint générique)
func ToggleLike(w http.ResponseWriter, r *http.Request) {
	user, err := middleware.GetUserFromContext(r)
	if err != nil {
		utils.Error(w, http.StatusUnauthorized, "authentification requise", err)
		return
	}

	vars := mux.Vars(r)
	entityType := model.EntityType(vars["entityType"]) // challenge, program, workout, etc.
	entityID := vars["entityId"]

	// Validation du type d'entité
	validTypes := map[model.EntityType]bool{
		model.EntityTypeChallenge: true,
		model.EntityTypeProgram:   true,
		model.EntityTypeWorkout:   true,
		model.EntityTypeComment:   true,
	}

	if !validTypes[entityType] {
		utils.ErrorSimple(w, http.StatusBadRequest, "type d'entité invalide")
		return
	}

	ctx := context.Background()

	// Toggle le like
	liked, err := utils.ToggleLike(ctx, user.ID, entityType, entityID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "impossible de modifier le like", err)
		return
	}

	utils.Success(w, map[string]interface{}{
		"liked":      liked,
		"entityType": entityType,
		"entityId":   entityID,
	})
}

// GetLikeStatus récupère le statut de like pour une entité
func GetLikeStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	entityType := model.EntityType(vars["entityType"])
	entityID := vars["entityId"]

	// Récupérer l'utilisateur (optionnel)
	user, _ := middleware.GetUserFromContext(r)
	var userID *string
	if user.ID != "" {
		userID = &user.ID
	}

	ctx := context.Background()

	// Récupérer les infos de like
	info, err := utils.GetLikeInfo(ctx, userID, entityType, entityID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "impossible de récupérer les likes", err)
		return
	}

	utils.Success(w, info)
}

// GetUserLikedEntities récupère toutes les entités likées par un utilisateur
func GetUserLikedEntities(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]
	entityType := model.EntityType(r.URL.Query().Get("type"))

	if entityType == "" {
		entityType = model.EntityTypeChallenge // Par défaut
	}

	ctx := context.Background()

	// Récupérer les IDs des entités likées
	entityIDs, err := utils.GetUserLikes(ctx, userID, entityType)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "impossible de récupérer les likes", err)
		return
	}

	utils.Success(w, map[string]interface{}{
		"entityType": entityType,
		"entityIds":  entityIDs,
		"total":      len(entityIDs),
	})
}

// GetTopLiked récupère les entités les plus likées
func GetTopLiked(w http.ResponseWriter, r *http.Request) {
	entityType := model.EntityType(r.URL.Query().Get("type"))
	limitStr := r.URL.Query().Get("limit")

	if entityType == "" {
		entityType = model.EntityTypeChallenge
	}

	limit := 10
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	ctx := context.Background()

	// Récupérer les entités les plus likées
	topLiked, err := utils.GetTopLikedEntities(ctx, entityType, limit)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "impossible de récupérer les top likes", err)
		return
	}

	utils.Success(w, topLiked)
}
