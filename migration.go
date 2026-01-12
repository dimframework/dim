package dim

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Migration represents a single migration
type Migration struct {
	Version int
	Name    string
	Up      func(*pgxpool.Pool) error
	Down    func(*pgxpool.Pool) error
}

// MigrationHistory represents the migration history table
type MigrationHistory struct {
	Version int
	Name    string
}

// RunMigrations menjalankan semua pending migrations yang belum pernah dijalankan.
// Membuat migrations table jika belum ada, kemudian menjalankan migrations yang baru.
// Semua migrations di-log menggunakan slog.
//
// Parameters:
//   - db: PostgresDatabase instance untuk execute migration queries
//   - migrations: slice dari Migration structs yang berisi Up dan Down functions
//
// Returns:
//   - error: error jika pembuatan migrations table gagal atau ada migration yang error
//
// Example:
//
//	err := RunMigrations(db, migrations)
//	if err != nil {
//	  log.Fatal(err)
//	}
func RunMigrations(db *PostgresDatabase, migrations []Migration) error {
	// Create migrations table if it doesn't exist
	if err := ensureMigrationsTable(db); err != nil {
		return fmt.Errorf("failed to ensure migrations table: %w", err)
	}

	// Get applied migrations
	applied, err := getAppliedMigrations(db)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Apply pending migrations
	for _, migration := range migrations {
		if _, exists := applied[migration.Version]; exists {
			slog.Info("migration already applied", "version", migration.Version, "name", migration.Name)
			continue
		}

		slog.Info("running migration", "version", migration.Version, "name", migration.Name)

		if err := migration.Up(db.writePool); err != nil {
			return fmt.Errorf("migration %d (%s) failed: %w", migration.Version, migration.Name, err)
		}

		// Record migration
		if err := recordMigration(db, migration); err != nil {
			return fmt.Errorf("failed to record migration %d: %w", migration.Version, err)
		}

		slog.Info("migration completed", "version", migration.Version, "name", migration.Name)
	}

	return nil
}

// RollbackMigration membatalkan/rollback migration tertentu dengan menjalankan Down function.
// Menghapus record migration dari migrations table.
//
// Parameters:
//   - db: PostgresDatabase instance untuk execute rollback queries
//   - migration: Migration struct yang akan di-rollback
//
// Returns:
//   - error: error jika Down function gagal atau gagal menghapus migration record
//
// Example:
//
//	err := RollbackMigration(db, migration)
//	if err != nil {
//	  log.Fatal(err)
//	}
func RollbackMigration(db *PostgresDatabase, migration Migration) error {
	if err := migration.Down(db.writePool); err != nil {
		return fmt.Errorf("rollback failed for migration %d: %w", migration.Version, err)
	}

	// Remove migration record
	if err := removeMigration(db, migration); err != nil {
		return fmt.Errorf("failed to remove migration record: %w", err)
	}

	slog.Info("migration rolled back", "version", migration.Version, "name", migration.Name)
	return nil
}

// ensureMigrationsTable creates the migrations history table
func ensureMigrationsTable(db *PostgresDatabase) error {
	_, err := db.writePool.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS migrations (
			version INTEGER PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			applied_at TIMESTAMP DEFAULT NOW()
		)
	`)
	return err
}

// getAppliedMigrations retrieves all applied migrations
func getAppliedMigrations(db *PostgresDatabase) (map[int]MigrationHistory, error) {
	rows, err := db.writePool.Query(context.Background(), "SELECT version, name FROM migrations ORDER BY version")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[int]MigrationHistory)
	for rows.Next() {
		var version int
		var name string

		if err := rows.Scan(&version, &name); err != nil {
			return nil, err
		}

		applied[version] = MigrationHistory{
			Version: version,
			Name:    name,
		}
	}

	return applied, rows.Err()
}

// recordMigration records a migration as applied
func recordMigration(db *PostgresDatabase, migration Migration) error {
	_, err := db.writePool.Exec(
		context.Background(),
		"INSERT INTO migrations (version, name) VALUES ($1, $2)",
		migration.Version,
		migration.Name,
	)
	return err
}

// removeMigration removes a migration record
func removeMigration(db *PostgresDatabase, migration Migration) error {
	_, err := db.writePool.Exec(
		context.Background(),
		"DELETE FROM migrations WHERE version = $1",
		migration.Version,
	)
	return err
}
