package main

import (
	"log"
	"net/http"

	"github.com/MassBabyGeek/PumpPro-backend/internal/api"
	"github.com/MassBabyGeek/PumpPro-backend/internal/config"
	"github.com/MassBabyGeek/PumpPro-backend/internal/database"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("could not load config: %v", err)
	}

	// Connect to PostgreSQL
	db, err := database.ConnectPostgres(cfg)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer db.Close()

	// Initialize routes
	router := api.SetupRouter()

	// Start server
	log.Printf("Starting server on port %s...", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, router); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
