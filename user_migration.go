package dim

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// GetUserMigrations mengembalikan daftar migrasi terkait tabel users.
// Mencakup pembuatan tabel users dasar.
func GetUserMigrations() []Migration {
	return []Migration{
		{
			Version: 1,
			Name:    "create_users_table",
			Up: func(pool *pgxpool.Pool) error {
				_, err := pool.Exec(context.Background(), `
					CREATE TABLE IF NOT EXISTS users (
						id VARCHAR(36) PRIMARY KEY,
						email VARCHAR(255) UNIQUE NOT NULL,
						username VARCHAR(100),
						password_hash VARCHAR(255) NOT NULL,
						created_at TIMESTAMP DEFAULT NOW(),
						updated_at TIMESTAMP DEFAULT NOW()
					);
					CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
				`)
				return err
			},
			Down: func(pool *pgxpool.Pool) error {
				_, err := pool.Exec(context.Background(), "DROP TABLE IF EXISTS users CASCADE")
				return err
			},
		},
	}
}
