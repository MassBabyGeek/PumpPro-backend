package utils

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/MassBabyGeek/PumpPro-backend/internal/logger"
)

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

func JSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

// Success retourne un succès HTTP 200 (n'affiche plus les objets JSON)
func Success(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusOK, APIResponse{Success: true, Data: data})
}

// Error gère les erreurs HTTP avec logging via le logger unifié
func Error(w http.ResponseWriter, status int, message string, err error) {
	errMsg := message
	if err != nil {
		errMsg = fmt.Sprintf("%s: %v", message, err)
	}
	logger.Error("[%d] %s", status, errMsg)
	JSON(w, status, APIResponse{Success: false, Error: errMsg})
}

// ErrorSimple pour les erreurs sans objet error
func ErrorSimple(w http.ResponseWriter, status int, message string) {
	Error(w, status, message, nil)
}

// Message retourne un message de succès
func Message(w http.ResponseWriter, msg string) {
	JSON(w, http.StatusOK, APIResponse{Success: true, Message: msg})
}
