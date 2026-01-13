package dim

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
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

// CreateRateLimitsTable membuat tabel UNLOGGED untuk rate limiting.
func CreateRateLimitsTable(pool *pgxpool.Pool) error {
	_, err := pool.Exec(context.Background(), `
		CREATE UNLOGGED TABLE IF NOT EXISTS rate_limits (
			key VARCHAR(255) PRIMARY KEY,
			count INT NOT NULL DEFAULT 0,
			expires_at TIMESTAMP NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_rate_limits_expires_at ON rate_limits(expires_at);
	`)
	return err
}

// DropRateLimitsTable menghapus tabel rate limits.
func DropRateLimitsTable(pool *pgxpool.Pool) error {
	_, err := pool.Exec(context.Background(), "DROP TABLE IF EXISTS rate_limits CASCADE")
	return err
}
