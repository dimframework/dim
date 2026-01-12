package dim

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// GetMigrations mengembalikan semua available migrations untuk aplikasi.
// Ini adalah placeholder yang harus di-override dalam aplikasi actual.
// Override function ini dan return slice dari Migration structs yang sudah di-define.
//
// Returns:
//   - []Migration: slice dari migration structs yang berisi Up dan Down functions
//
// Example:
//
//	func GetMigrations() []Migration {
//	  return []Migration{
//	    {
//	      Version: 1,
//	      Name: "create_users_table",
//	      Up: CreateUsersTable,
//	      Down: DropUsersTable,
//	    },
//	  }
//	}
func GetMigrations() []Migration {
	return []Migration{}
}

// Example migrations for reference

// CreateUsersTable membuat users table dengan schema untuk user data.
// Table berisi id, email, name, password, dan timestamp columns.
//
// Parameters:
//   - pool: pgxpool.Pool untuk execute query
//
// Returns:
//   - error: error jika CREATE TABLE query gagal
//
// Example:
//
//	err := CreateUsersTable(pool)
func CreateUsersTable(pool *pgxpool.Pool) error {
	_, err := pool.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS users (
			id BIGSERIAL PRIMARY KEY,
			email VARCHAR(255) UNIQUE NOT NULL,
			name VARCHAR(255) NOT NULL,
			password VARCHAR(255) NOT NULL,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)
	`)
	return err
}

// DropUsersTable menghapus users table dari database.
//
// Parameters:
//   - pool: pgxpool.Pool untuk execute query
//
// Returns:
//   - error: error jika DROP TABLE query gagal
//
// Example:
//
//	err := DropUsersTable(pool)
func DropUsersTable(pool *pgxpool.Pool) error {
	_, err := pool.Exec(context.Background(), "DROP TABLE IF EXISTS users CASCADE")
	return err
}

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
			user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
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
			user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
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
