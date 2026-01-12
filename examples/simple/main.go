package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nuradiyana/dim"
)

func main() {
	// Load configuration from environment
	cfg, err := dim.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Setup logger
	logLevel := slog.LevelInfo
	if os.Getenv("LOG_LEVEL") == "debug" {
		logLevel = slog.LevelDebug
	}
	logger := dim.NewLogger(logLevel)

	// Connect to database
	db, err := dim.NewPostgresDatabase(cfg.Database)
	if err != nil {
		logger.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Run migrations
	if err := dim.RunMigrations(db, getMigrations()); err != nil {
		logger.Error("Failed to run migrations", "error", err)
		os.Exit(1)
	}
	logger.Info("Migrations completed successfully")

	// Setup stores
	userStore := dim.NewPostgresUserStore(db)
	tokenStore := dim.NewPostgresTokenStore(db)

	// Setup auth service
	authService := dim.NewAuthService(userStore, tokenStore, &cfg.JWT)
	jwtManager := dim.NewJWTManager(&cfg.JWT)

	// Setup router
	router := dim.NewRouter()

	// Global middleware (in order)
	router.Use(dim.Recovery(logger))
	router.Use(dim.LoggerMiddleware(logger))
	router.Use(dim.CORS(cfg.CORS))

	// Setup handlers
	setupHandlers(router, authService, userStore, jwtManager)

	// Start server
	port := cfg.Server.Port
	logger.Info("Starting server", "port", port)
	if err := http.ListenAndServe(":"+port, router); err != nil && err != http.ErrServerClosed {
		logger.Error("Server error", "error", err)
		os.Exit(1)
	}
}

// getMigrations returns all database migrations
func getMigrations() []dim.Migration {
	return []dim.Migration{
		{
			Version: 1,
			Name:    "create_users_table",
			Up: func(pool *pgxpool.Pool) error {
				_, err := pool.Exec(context.Background(), `
					CREATE TABLE IF NOT EXISTS users (
						id BIGSERIAL PRIMARY KEY,
						email VARCHAR(255) UNIQUE NOT NULL,
						username VARCHAR(100) UNIQUE NOT NULL,
						password_hash VARCHAR(255) NOT NULL,
						created_at TIMESTAMP DEFAULT NOW(),
						updated_at TIMESTAMP DEFAULT NOW()
					)
				`)
				return err
			},
			Down: func(pool *pgxpool.Pool) error {
				_, err := pool.Exec(context.Background(), "DROP TABLE IF EXISTS users")
				return err
			},
		},
		{
			Version: 2,
			Name:    "create_refresh_tokens_table",
			Up: func(pool *pgxpool.Pool) error {
				_, err := pool.Exec(context.Background(), `
					CREATE TABLE IF NOT EXISTS refresh_tokens (
						id BIGSERIAL PRIMARY KEY,
						user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
						token_hash VARCHAR(255) UNIQUE NOT NULL,
						expires_at TIMESTAMP NOT NULL,
						created_at TIMESTAMP DEFAULT NOW(),
						revoked_at TIMESTAMP
					)
				`)
				return err
			},
			Down: func(pool *pgxpool.Pool) error {
				_, err := pool.Exec(context.Background(), "DROP TABLE IF EXISTS refresh_tokens")
				return err
			},
		},
		{
			Version: 3,
			Name:    "create_password_reset_tokens_table",
			Up: func(pool *pgxpool.Pool) error {
				_, err := pool.Exec(context.Background(), `
					CREATE TABLE IF NOT EXISTS password_reset_tokens (
						id BIGSERIAL PRIMARY KEY,
						user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
						token_hash VARCHAR(255) UNIQUE NOT NULL,
						expires_at TIMESTAMP NOT NULL,
						used_at TIMESTAMP,
						created_at TIMESTAMP DEFAULT NOW()
					)
				`)
				return err
			},
			Down: func(pool *pgxpool.Pool) error {
				_, err := pool.Exec(context.Background(), "DROP TABLE IF EXISTS password_reset_tokens")
				return err
			},
		},
	}
}
