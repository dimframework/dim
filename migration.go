package dim

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
)

// Migration represents a single migration
type Migration struct {
	Version int64
	Name    string
	Up      func(Database) error
	Down    func(Database) error
}

// MigrationHistory represents the migration history table
type MigrationHistory struct {
	Version int64
	Name    string
}

var migrationRegistry []Migration
var includeFrameworkMigrations = true

// Register mendaftarkan migration ke global registry.
// Fungsi ini biasanya dipanggil di dalam fungsi init() pada file migration.
func Register(m Migration) {
	migrationRegistry = append(migrationRegistry, m)
}

// DisableFrameworkMigrations menonaktifkan migrasi bawaan framework (User, Token, RateLimit).
// Panggil fungsi ini di init() aplikasi jika Anda ingin mendefinisikan skema tabel inti Anda sendiri.
// Gunakan ini untuk kustomisasi penuh (misal: ID int64, tambah kolom, ganti nama tabel).
func DisableFrameworkMigrations() {
	includeFrameworkMigrations = false
}

// GetRegisteredMigrations mengembalikan semua migration yang terdaftar via Register().
// Migration akan otomatis diurutkan berdasarkan Version.
func GetRegisteredMigrations() []Migration {
	// Kopi slice untuk menghindari side effects modifikasi eksternal
	migrations := make([]Migration, len(migrationRegistry))
	copy(migrations, migrationRegistry)
	return migrations
}

// GetFrameworkMigrations mengembalikan semua migrasi bawaan framework dim (User, Token, RateLimit).
// Migrasi ini mencakup tabel-tabel inti yang diperlukan oleh fitur-fitur framework.
// Jika DisableFrameworkMigrations() telah dipanggil, fungsi ini mengembalikan slice kosong.
// Urutan versi:
// 1. Users
// 2. Refresh Tokens
// 3. Password Reset Tokens
// 4. Token Blocklist
// 5. Rate Limits
func GetFrameworkMigrations() []Migration {
	if !includeFrameworkMigrations {
		return []Migration{}
	}

	var migrations []Migration
	migrations = append(migrations, GetUserMigrations()...)
	migrations = append(migrations, GetTokenMigrations()...)
	migrations = append(migrations, GetRateLimitMigrations()...)
	return migrations
}

// RunMigrations menjalankan semua pending migrations yang belum pernah dijalankan.
// Membuat migrations table jika belum ada, kemudian menjalankan migrations yang baru.
// Semua migrations di-log menggunakan slog.
//
// Parameters:
//   - db: Database instance untuk execute migration queries
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
func RunMigrations(db Database, migrations []Migration) error {
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

		if err := migration.Up(db); err != nil {
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
//   - db: Database instance untuk execute rollback queries
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
func RollbackMigration(db Database, migration Migration) error {
	if err := migration.Down(db); err != nil {
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
func ensureMigrationsTable(db Database) error {
	var query string
	if db.DriverName() == "sqlite" {
		query = `
			CREATE TABLE IF NOT EXISTS migrations (
				version INTEGER PRIMARY KEY,
				name TEXT NOT NULL,
				applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)
		`
	} else {
		query = `
			CREATE TABLE IF NOT EXISTS migrations (
				version BIGINT PRIMARY KEY,
				name VARCHAR(255) NOT NULL,
				applied_at TIMESTAMP DEFAULT NOW()
			)
		`
	}
	return db.Exec(context.Background(), query)
}

// getAppliedMigrations retrieves all applied migrations
func getAppliedMigrations(db Database) (map[int64]MigrationHistory, error) {
	rows, err := db.Query(context.Background(), "SELECT version, name FROM migrations ORDER BY version")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[int64]MigrationHistory)
	for rows.Next() {
		var version int64
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
func recordMigration(db Database, migration Migration) error {
	query := "INSERT INTO migrations (version, name) VALUES ($1, $2)"
	if db.DriverName() == "sqlite" {
		query = rebind(query)
	}
	return db.Exec(context.Background(), query, migration.Version, migration.Name)
}

// removeMigration removes a migration record
func removeMigration(db Database, migration Migration) error {
	query := "DELETE FROM migrations WHERE version = $1"
	if db.DriverName() == "sqlite" {
		query = rebind(query)
	}
	return db.Exec(context.Background(), query, migration.Version)
}

// rebind replaces $1, $2, etc with ? for SQLite compatibility
func rebind(query string) string {
	re := regexp.MustCompile(`\$[0-9]+`)
	return re.ReplaceAllString(query, "?")
}
