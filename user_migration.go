package dim

import (
	"context"
)

// GetUserMigrations mengembalikan daftar migrasi terkait tabel users.
// Mencakup pembuatan tabel users dasar.
func GetUserMigrations() []Migration {
	return []Migration{
		{
			Version: 1,
			Name:    "create_users_table",
			Up: func(db Database) error {
				var query string
				if db.DriverName() == "sqlite" {
					query = `
						CREATE TABLE IF NOT EXISTS users (
							id TEXT PRIMARY KEY,
							email TEXT UNIQUE NOT NULL,
							name TEXT,
							password TEXT NOT NULL,
							created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
							updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
						);
						CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
					`
				} else {
					query = `
						CREATE TABLE IF NOT EXISTS users (
							id UUID PRIMARY KEY,
							email VARCHAR(255) UNIQUE NOT NULL,
							name VARCHAR(100),
							password VARCHAR(255) NOT NULL,
							created_at TIMESTAMP DEFAULT NOW(),
							updated_at TIMESTAMP DEFAULT NOW()
						);
						CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
					`
				}
				return db.Exec(context.Background(), query)
			},
			Down: func(db Database) error {
				query := "DROP TABLE IF EXISTS users CASCADE"
				if db.DriverName() == "sqlite" {
					query = "DROP TABLE IF EXISTS users" // SQLite doesn't strictly need CASCADE for drop table, usually implicit or pragma controlled
				}
				return db.Exec(context.Background(), query)
			},
		},
	}
}
