package dim

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// CreateRefreshTokensTable membuat refresh_tokens table untuk menyimpan refresh tokens.
// Table berisi referensi ke users dan metadata tentang token.
//
// Parameters:
//   - pool: pgxpool.Pool untuk execute query
//
// Returns:
//   - error: error jika CREATE TABLE query gagal
//
// Example:
//
//	err := CreateRefreshTokensTable(pool)
func CreateRefreshTokensTable(pool *pgxpool.Pool) error {
	_, err := pool.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS refresh_tokens (
			id BIGSERIAL PRIMARY KEY,
			user_id VARCHAR(36) NOT NULL,
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

// DropRefreshTokensTable menghapus refresh_tokens table dari database.
//
// Parameters:
//   - pool: pgxpool.Pool untuk execute query
//
// Returns:
//   - error: error jika DROP TABLE query gagal
//
// Example:
//
//	err := DropRefreshTokensTable(pool)
func DropRefreshTokensTable(pool *pgxpool.Pool) error {
	_, err := pool.Exec(context.Background(), "DROP TABLE IF EXISTS refresh_tokens CASCADE")
	return err
}

// CreatePasswordResetTokensTable membuat password_reset_tokens table untuk reset password flow.
// Table menyimpan token hash dengan expiry dan usage tracking.
//
// Parameters:
//   - pool: pgxpool.Pool untuk execute query
//
// Returns:
//   - error: error jika CREATE TABLE query gagal
//
// Example:
//
//	err := CreatePasswordResetTokensTable(pool)
func CreatePasswordResetTokensTable(pool *pgxpool.Pool) error {
	_, err := pool.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS password_reset_tokens (
			id BIGSERIAL PRIMARY KEY,
			user_id VARCHAR(36) NOT NULL,
			token_hash VARCHAR(255) UNIQUE NOT NULL,
			expires_at TIMESTAMP NOT NULL,
			used_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT NOW()
		)
	`)
	return err
}

// DropPasswordResetTokensTable menghapus password_reset_tokens table dari database.
//
// Parameters:
//   - pool: pgxpool.Pool untuk execute query
//
// Returns:
//   - error: error jika DROP TABLE query gagal
//
// Example:
//
//	err := DropPasswordResetTokensTable(pool)
func DropPasswordResetTokensTable(pool *pgxpool.Pool) error {
	_, err := pool.Exec(context.Background(), "DROP TABLE IF EXISTS password_reset_tokens CASCADE")
	return err
}
