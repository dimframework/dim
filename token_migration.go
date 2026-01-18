package dim

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// GetTokenMigrations mengembalikan daftar migrasi terkait token (refresh, reset, blocklist).
// Dimulai dari versi 2 (asumsi versi 1 adalah users).
func GetTokenMigrations() []Migration {
	return []Migration{
		{
			Version: 2,
			Name:    "create_refresh_tokens_table",
			Up:      CreateRefreshTokensTable,
			Down:    DropRefreshTokensTable,
		},
		{
			Version: 3,
			Name:    "create_password_reset_tokens_table",
			Up:      CreatePasswordResetTokensTable,
			Down:    DropPasswordResetTokensTable,
		},
		{
			Version: 4,
			Name:    "create_token_blocklist_table",
			Up:      CreateTokenBlocklistTable,
			Down:    DropTokenBlocklistTable,
		},
	}
}

// CreateRefreshTokensTable membuat refresh_tokens table.
func CreateRefreshTokensTable(pool *pgxpool.Pool) error {
	_, err := pool.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS refresh_tokens (
			id BIGSERIAL PRIMARY KEY,
			user_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			token_hash VARCHAR(255) UNIQUE NOT NULL,
			user_agent TEXT NOT NULL,
			ip_address VARCHAR(45) NOT NULL,
			expires_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP DEFAULT NOW(),
			revoked_at TIMESTAMP
		)
	`)
	return err
}

// DropRefreshTokensTable menghapus refresh_tokens table.
func DropRefreshTokensTable(pool *pgxpool.Pool) error {
	_, err := pool.Exec(context.Background(), "DROP TABLE IF EXISTS refresh_tokens CASCADE")
	return err
}

// CreatePasswordResetTokensTable membuat password_reset_tokens table.
func CreatePasswordResetTokensTable(pool *pgxpool.Pool) error {
	_, err := pool.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS password_reset_tokens (
			id BIGSERIAL PRIMARY KEY,
			user_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			token_hash VARCHAR(255) UNIQUE NOT NULL,
			expires_at TIMESTAMP NOT NULL,
			used_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT NOW()
		)
	`)
	return err
}

// DropPasswordResetTokensTable menghapus password_reset_tokens table.
func DropPasswordResetTokensTable(pool *pgxpool.Pool) error {
	_, err := pool.Exec(context.Background(), "DROP TABLE IF EXISTS password_reset_tokens CASCADE")
	return err
}

// CreateTokenBlocklistTable membuat tabel UNLOGGED untuk token blocklist.
func CreateTokenBlocklistTable(pool *pgxpool.Pool) error {
	_, err := pool.Exec(context.Background(), `
		CREATE UNLOGGED TABLE IF NOT EXISTS token_blocklist (
			identifier VARCHAR(255) PRIMARY KEY,
			expires_at TIMESTAMP NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_token_blocklist_expires_at ON token_blocklist(expires_at);
	`)
	return err
}

// DropTokenBlocklistTable menghapus tabel token blocklist.
func DropTokenBlocklistTable(pool *pgxpool.Pool) error {
	_, err := pool.Exec(context.Background(), "DROP TABLE IF EXISTS token_blocklist CASCADE")
	return err
}
