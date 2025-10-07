package handler

import (
	"net/http"

	"github.com/MassBabyGeek/PumpPro-backend/internal/utils"
)

func HealthCheck(w http.ResponseWriter, r *http.Request) {
	utils.Message(w, "ok")
}
