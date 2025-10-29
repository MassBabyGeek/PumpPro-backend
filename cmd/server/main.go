package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/MassBabyGeek/PumpPro-backend/internal/api"
	"github.com/MassBabyGeek/PumpPro-backend/internal/config"
	"github.com/MassBabyGeek/PumpPro-backend/internal/database"
	"github.com/MassBabyGeek/PumpPro-backend/internal/logger"
	"github.com/MassBabyGeek/PumpPro-backend/internal/middleware"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Error("Could not load config: %v", err)
		os.Exit(1)
	}

	// Connect to PostgreSQL
	db, err := database.ConnectPostgres(cfg)
	if err != nil {
		logger.Error("Database connection failed: %v", err)
		os.Exit(1)
	}
	defer db.Close()

	// Initialize routes
	router := api.SetupRouter()

	// Wrap router with CORS middleware
	handler := middleware.CORSMiddleware(router)

	// Start server
	logger.Success("Server starting on port %s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, handler); err != nil {
		logger.Error("Server failed: %v", err)
		os.Exit(1)
	}

	// Start server
	logger.Success("Server starting on port %s", cfg.Port)
	fmt.Printf("\n")
	if err := http.ListenAndServe(":"+cfg.Port, router); err != nil {
		logger.Error("Server failed: %v", err)
		os.Exit(1)
	}
}
