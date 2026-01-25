package dim

import (
	"context"
)

// GetRateLimitMigrations mengembalikan daftar migrasi terkait rate limit.
// Dimulai dari versi 5 (melanjutkan token migrations).
func GetRateLimitMigrations() []Migration {
	return []Migration{
		{
			Version: 5,
			Name:    "create_rate_limits_table",
			Up:      CreateRateLimitsTable,
			Down:    DropRateLimitsTable,
		},
	}
}

// CreateRateLimitsTable membuat tabel untuk rate limiting.
func CreateRateLimitsTable(db Database) error {
	var query string
	if db.DriverName() == "sqlite" {
		query = `
			CREATE TABLE IF NOT EXISTS rate_limits (
				key TEXT PRIMARY KEY,
				count INT NOT NULL DEFAULT 0,
				expires_at TIMESTAMP NOT NULL
			);
			CREATE INDEX IF NOT EXISTS idx_rate_limits_expires_at ON rate_limits(expires_at);
		`
	} else {
		query = `
			CREATE UNLOGGED TABLE IF NOT EXISTS rate_limits (
				key VARCHAR(255) PRIMARY KEY,
				count INT NOT NULL DEFAULT 0,
				expires_at TIMESTAMP NOT NULL
			);
			CREATE INDEX IF NOT EXISTS idx_rate_limits_expires_at ON rate_limits(expires_at);
		`
	}
	return db.Exec(context.Background(), query)
}

// DropRateLimitsTable menghapus tabel rate limits.
func DropRateLimitsTable(db Database) error {
	query := "DROP TABLE IF EXISTS rate_limits CASCADE"
	if db.DriverName() == "sqlite" {
		query = "DROP TABLE IF EXISTS rate_limits"
	}
	return db.Exec(context.Background(), query)
}