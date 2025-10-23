package database

import (
	"context"
	"fmt"
	"time"

	"github.com/MassBabyGeek/PumpPro-backend/internal/config"
	"github.com/MassBabyGeek/PumpPro-backend/internal/logger"
	"github.com/jackc/pgx/v5/pgxpool"
)

var DB *pgxpool.Pool

func ConnectPostgres(cfg *config.Config) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?pool_max_conns=25&pool_min_conns=5&pool_max_conn_lifetime=5m&pool_max_conn_idle_time=1m",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Configuration du pool
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("unable to parse pool config: %w", err)
	}

	// Paramètres du pool pour éviter les context deadline exceeded
	poolConfig.MaxConns = 25                            // Max 25 connexions simultanées
	poolConfig.MinConns = 5                             // Min 5 connexions toujours ouvertes
	poolConfig.MaxConnLifetime = 5 * time.Minute        // Durée de vie max d'une connexion
	poolConfig.MaxConnIdleTime = 1 * time.Minute        // Temps d'inactivité max avant fermeture
	poolConfig.HealthCheckPeriod = 30 * time.Second     // Vérification santé des connexions
	poolConfig.ConnConfig.ConnectTimeout = 5 * time.Second // Timeout de connexion

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	if err = pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	logger.Success("Connected to PostgreSQL (pool: %d-%d conns)", poolConfig.MinConns, poolConfig.MaxConns)

	DB = pool

	return pool, nil
}
