package dim

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"slices"
	"strings"
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

// DefaultMigrationsTable adalah nama tabel pencatat riwayat migrasi bawaan.
// Dipakai oleh RunMigrations dan oleh perintah migrasi yang dijalankan tanpa
// flag -table.
const DefaultMigrationsTable = "migrations"

var migrationRegistry []Migration
var includeFrameworkMigrations = true
var migrationSource func() []Migration

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
// Migration akan otomatis diurutkan berdasarkan Version, sehingga urutan jalannya
// tidak bergantung pada urutan init() antar package.
func GetRegisteredMigrations() []Migration {
	// Kopi slice untuk menghindari side effects modifikasi eksternal
	migrations := make([]Migration, len(migrationRegistry))
	copy(migrations, migrationRegistry)
	sortMigrations(migrations)
	return migrations
}

// GetAllMigrations mengembalikan gabungan migrasi framework dan migrasi aplikasi
// yang terdaftar via Register(), diurutkan berdasarkan Version.
//
// Gunakan fungsi ini sebagai sumber tunggal urutan migrasi agar `migrate`,
// `migrate:list`, dan `migrate:rollback` melihat urutan yang sama.
func GetAllMigrations() []Migration {
	migrations := GetFrameworkMigrations()
	migrations = append(migrations, migrationRegistry...)
	sortMigrations(migrations)
	return migrations
}

// SetMigrationSource mengganti sumber migrasi yang dibaca oleh perintah bawaan
// `migrate`, `migrate:list`, dan `migrate:rollback`.
//
// Registry global diisi dari init() lewat Register(), sehingga string SQL-nya
// beku sebelum program tahu ke schema mana ia akan bermigrasi. Aplikasi yang
// perlu merakit migrasinya saat runtime — misalnya menyuntikkan nama schema per
// modul atau prefix schema per test — dapat menyerahkan perakitnya ke sini,
// dan tetap memakai perkakas migrasi dim alih-alih menulis ulang ketiganya.
//
// Tidak dipanggil = perilaku sekarang, persis: sumbernya GetAllMigrations().
// Panggil dengan nil untuk mengembalikannya ke registry global.
//
// Slice yang dikembalikan fn selalu disalin dan diurutkan berdasarkan Version
// sebelum dipakai, sama seperti GetAllMigrations, sehingga urutan jalannya tidak
// bergantung pada urutan perakitan.
//
// Example:
//
//	dim.SetMigrationSource(func() []dim.Migration {
//	  return mymodule.Migrations(schema)
//	})
func SetMigrationSource(fn func() []Migration) {
	migrationSource = fn
}

// migrationsFromSource mengembalikan migrasi dari sumber yang dipasang lewat
// SetMigrationSource, atau GetAllMigrations() bila tidak ada.
func migrationsFromSource() []Migration {
	if migrationSource == nil {
		return GetAllMigrations()
	}

	// Salin agar pengurutan tidak menyentuh slice milik pemanggil
	source := migrationSource()
	migrations := make([]Migration, len(source))
	copy(migrations, source)
	sortMigrations(migrations)
	return migrations
}

// sortMigrations mengurutkan migrations berdasarkan Version secara menaik.
// Pengurutan bersifat stabil sehingga migrasi dengan Version yang sama
// tetap mengikuti urutan pendaftarannya.
func sortMigrations(migrations []Migration) {
	slices.SortStableFunc(migrations, func(a, b Migration) int {
		return cmp.Compare(a.Version, b.Version)
	})
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
	return RunMigrationsIn(db, DefaultMigrationsTable, migrations)
}

// RunMigrationsIn sama dengan RunMigrations, tetapi riwayatnya dicatat di tabel
// bernama `table` alih-alih `migrations`.
//
// Nama tabelnya boleh dikualifikasi schema (`myschema.migrations`), yang diperlukan
// bila koneksi berjalan tanpa `search_path` yang menunjuk ke schema modulnya —
// tanpa kualifikasi, tabel pencatat mendarat di schema bawaan koneksi, bukan di
// schema yang sedang dimigrasi. Berguna pula untuk isolasi test berbasis prefix
// schema, yang tiap test-nya membutuhkan pencatatnya sendiri.
//
// Schema-nya harus sudah ada; dim tidak membuatkannya.
//
// Parameters:
//   - db: Database instance untuk execute migration queries
//   - table: nama tabel pencatat, boleh `schema.tabel`. Kosong = "migrations"
//   - migrations: slice dari Migration structs yang berisi Up dan Down functions
//
// Returns:
//   - error: error jika nama tabelnya tidak valid, pembuatan tabel pencatat
//     gagal, atau ada migration yang error
//
// Example:
//
//	err := RunMigrationsIn(db, "myschema.migrations", migrations)
//	if err != nil {
//	  log.Fatal(err)
//	}
func RunMigrationsIn(db Database, table string, migrations []Migration) error {
	table, err := resolveMigrationsTable(table)
	if err != nil {
		return err
	}

	// Create migrations table if it doesn't exist
	if err := ensureMigrationsTable(db, table); err != nil {
		return fmt.Errorf("failed to ensure migrations table: %w", err)
	}

	// Get applied migrations
	applied, err := getAppliedMigrations(db, table)
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
		if err := recordMigration(db, table, migration); err != nil {
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
	return RollbackMigrationIn(db, DefaultMigrationsTable, migration)
}

// RollbackMigrationIn sama dengan RollbackMigration, tetapi record-nya dihapus
// dari tabel pencatat bernama `table` alih-alih `migrations`.
//
// Pasangan dari RunMigrationsIn: rollback harus menghapus catatannya dari tabel
// yang sama dengan yang mencatatnya.
//
// Parameters:
//   - db: Database instance untuk execute rollback queries
//   - table: nama tabel pencatat, boleh `schema.tabel`. Kosong = "migrations"
//   - migration: Migration struct yang akan di-rollback
//
// Returns:
//   - error: error jika nama tabelnya tidak valid, Down function gagal, atau
//     gagal menghapus migration record
//
// Example:
//
//	err := RollbackMigrationIn(db, "myschema.migrations", migration)
//	if err != nil {
//	  log.Fatal(err)
//	}
func RollbackMigrationIn(db Database, table string, migration Migration) error {
	table, err := resolveMigrationsTable(table)
	if err != nil {
		return err
	}

	if err := migration.Down(db); err != nil {
		return fmt.Errorf("rollback failed for migration %d: %w", migration.Version, err)
	}

	// Remove migration record
	if err := removeMigration(db, table, migration); err != nil {
		return fmt.Errorf("failed to remove migration record: %w", err)
	}

	slog.Info("migration rolled back", "version", migration.Version, "name", migration.Name)
	return nil
}

// migrationsTableIdentifier mencocokkan satu identifier SQL tak-terkutip.
var migrationsTableIdentifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_$]*$`)

// resolveMigrationsTable memvalidasi nama tabel pencatat dan mengembalikan
// bentuk yang aman disisipkan ke SQL. Nama tabel tidak dapat dikirim sebagai
// parameter query, sehingga ia harus disisipkan sebagai teks — dan karena itu
// wajib divalidasi di sini, bukan dipercaya.
//
// Yang diterima: satu identifier (`migrations`) atau identifier berkualifikasi
// schema (`myschema.migrations`), keduanya tak-terkutip. String kosong berarti
// DefaultMigrationsTable.
func resolveMigrationsTable(table string) (string, error) {
	table = strings.TrimSpace(table)
	if table == "" {
		return DefaultMigrationsTable, nil
	}

	parts := strings.Split(table, ".")
	if len(parts) > 2 {
		return "", fmt.Errorf("invalid migrations table %q: expected \"table\" or \"schema.table\"", table)
	}

	for _, part := range parts {
		if !migrationsTableIdentifier.MatchString(part) {
			return "", fmt.Errorf(
				"invalid migrations table %q: %q is not a valid unquoted SQL identifier",
				table, part)
		}
	}

	return table, nil
}

// ensureMigrationsTable creates the migrations history table
func ensureMigrationsTable(db Database, table string) error {
	var query string
	if db.DriverName() == "sqlite" {
		query = fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				version INTEGER PRIMARY KEY,
				name TEXT NOT NULL,
				applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)
		`, table)
	} else {
		query = fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				version BIGINT PRIMARY KEY,
				name VARCHAR(255) NOT NULL,
				applied_at TIMESTAMP DEFAULT NOW()
			)
		`, table)
	}
	return db.Exec(context.Background(), query)
}

// getAppliedMigrations retrieves all applied migrations
func getAppliedMigrations(db Database, table string) (map[int64]MigrationHistory, error) {
	rows, err := db.Query(context.Background(), fmt.Sprintf("SELECT version, name FROM %s ORDER BY version", table))
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
func recordMigration(db Database, table string, migration Migration) error {
	query := fmt.Sprintf("INSERT INTO %s (version, name) VALUES ($1, $2)", table)
	if db.DriverName() == "sqlite" {
		query = rebind(query)
	}
	return db.Exec(context.Background(), query, migration.Version, migration.Name)
}

// removeMigration removes a migration record
func removeMigration(db Database, table string, migration Migration) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE version = $1", table)
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
