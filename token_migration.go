package dim

import (
	"context"
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
func CreateRefreshTokensTable(db Database) error {
	var query string
	if db.DriverName() == "sqlite" {
		query = `
			CREATE TABLE IF NOT EXISTS refresh_tokens (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
				token_hash TEXT UNIQUE NOT NULL,
				user_agent TEXT NOT NULL,
				ip_address TEXT NOT NULL,
				expires_at TIMESTAMP NOT NULL,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				revoked_at TIMESTAMP
			)
		`
	} else {
		query = `
			CREATE TABLE IF NOT EXISTS refresh_tokens (
				id BIGSERIAL PRIMARY KEY,
				user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
				token_hash VARCHAR(255) UNIQUE NOT NULL,
				user_agent TEXT NOT NULL,
				ip_address VARCHAR(45) NOT NULL,
				expires_at TIMESTAMP NOT NULL,
				created_at TIMESTAMP DEFAULT NOW(),
				revoked_at TIMESTAMP
			)
		`
	}
	return db.Exec(context.Background(), query)
}

// DropRefreshTokensTable menghapus refresh_tokens table.
func DropRefreshTokensTable(db Database) error {
	query := "DROP TABLE IF EXISTS refresh_tokens CASCADE"
	if db.DriverName() == "sqlite" {
		query = "DROP TABLE IF EXISTS refresh_tokens"
	}
	return db.Exec(context.Background(), query)
}

// CreatePasswordResetTokensTable membuat password_reset_tokens table.
func CreatePasswordResetTokensTable(db Database) error {
	var query string
	if db.DriverName() == "sqlite" {
		query = `
			CREATE TABLE IF NOT EXISTS password_reset_tokens (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
				token_hash TEXT UNIQUE NOT NULL,
				expires_at TIMESTAMP NOT NULL,
				used_at TIMESTAMP,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)
		`
	} else {
		query = `
			CREATE TABLE IF NOT EXISTS password_reset_tokens (
				id BIGSERIAL PRIMARY KEY,
				user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
				token_hash VARCHAR(255) UNIQUE NOT NULL,
				expires_at TIMESTAMP NOT NULL,
				used_at TIMESTAMP,
				created_at TIMESTAMP DEFAULT NOW()
			)
		`
	}
	return db.Exec(context.Background(), query)
}

// DropPasswordResetTokensTable menghapus password_reset_tokens table.
func DropPasswordResetTokensTable(db Database) error {
	query := "DROP TABLE IF EXISTS password_reset_tokens CASCADE"
	if db.DriverName() == "sqlite" {
		query = "DROP TABLE IF EXISTS password_reset_tokens"
	}
	return db.Exec(context.Background(), query)
}

// CreateTokenBlocklistTable membuat tabel untuk token blocklist.
func CreateTokenBlocklistTable(db Database) error {
	var query string
	if db.DriverName() == "sqlite" {
		query = `
			CREATE TABLE IF NOT EXISTS token_blocklist (
				identifier TEXT PRIMARY KEY,
				expires_at TIMESTAMP NOT NULL
			);
			CREATE INDEX IF NOT EXISTS idx_token_blocklist_expires_at ON token_blocklist(expires_at);
		`
	} else {
		query = `
			CREATE UNLOGGED TABLE IF NOT EXISTS token_blocklist (
				identifier VARCHAR(255) PRIMARY KEY,
				expires_at TIMESTAMP NOT NULL
			);
			CREATE INDEX IF NOT EXISTS idx_token_blocklist_expires_at ON token_blocklist(expires_at);
		`
	}
	return db.Exec(context.Background(), query)
}

// DropTokenBlocklistTable menghapus tabel token blocklist.
func DropTokenBlocklistTable(db Database) error {
	query := "DROP TABLE IF EXISTS token_blocklist CASCADE"
	if db.DriverName() == "sqlite" {
		query = "DROP TABLE IF EXISTS token_blocklist"
	}
	return db.Exec(context.Background(), query)
}
