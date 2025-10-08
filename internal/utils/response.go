package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
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

// Success affiche le type et la valeur de data dans la console
func Success(w http.ResponseWriter, data interface{}) {
	fmt.Printf("[INFO][Success] Type: %s, Value: %#v\n", reflect.TypeOf(data), data)
	JSON(w, http.StatusOK, APIResponse{Success: true, Data: data})
}

// Error gère les erreurs HTTP avec logging automatique
func Error(w http.ResponseWriter, status int, message string, err error) {
	errMsg := message
	if err != nil {
		errMsg = fmt.Sprintf("%s: %v", message, err)
	}
	fmt.Printf("[ERROR][%d] %s\n", status, errMsg)
	JSON(w, status, APIResponse{Success: false, Error: errMsg})
}

// ErrorSimple pour les erreurs sans objet error (rétrocompatibilité)
func ErrorSimple(w http.ResponseWriter, status int, message string) {
	Error(w, status, message, nil)
}

func Message(w http.ResponseWriter, msg string) {
	fmt.Printf("[INFO][MESSAGE] %s\n", msg)
	JSON(w, http.StatusOK, APIResponse{Success: true, Message: msg})
}
