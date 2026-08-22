package dim

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"text/template"
	"time"
)

// migrationConn returns the migration-specific DB if set, otherwise falls back to the main DB.
func migrationConn(ctx *CommandContext) Database {
	if ctx.MigrationDB != nil {
		return ctx.MigrationDB
	}
	return ctx.DB
}

// ============================================================================
// MigrateCommand - Run pending migrations
// ============================================================================

// MigrateCommand menjalankan semua pending database migrations.
type MigrateCommand struct {
	verbose bool
}

func (c *MigrateCommand) Name() string {
	return "migrate"
}

func (c *MigrateCommand) Description() string {
	return "Run pending database migrations"
}

func (c *MigrateCommand) DefineFlags(fs *flag.FlagSet) {
	fs.BoolVar(&c.verbose, "v", false, "Show detailed migration output")
}

func (c *MigrateCommand) Execute(ctx *CommandContext) error {
	if ctx.DB == nil {
		return fmt.Errorf("database connection required")
	}

	db := migrationConn(ctx)

	if c.verbose {
		if ctx.MigrationDB != nil {
			fmt.Println("Using dedicated migration database connection...")
		}
		fmt.Println("Running migrations in verbose mode...")
	}

	// Gabungan migrasi framework + migrasi terdaftar, terurut berdasarkan Version
	migrations := GetAllMigrations()

	if c.verbose {
		fmt.Printf("Found %d total migrations\n", len(migrations))
	}

	if err := RunMigrations(db, migrations); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	fmt.Println("✓ All migrations completed successfully")
	return nil
}

// ============================================================================
// MigrateRollbackCommand - Rollback migrations
// ============================================================================

// MigrateRollbackCommand membatalkan migration yang sudah dijalankan.
type MigrateRollbackCommand struct {
	steps        int
	force        bool
	allowMissing bool
}

func (c *MigrateRollbackCommand) Name() string {
	return "migrate:rollback"
}

func (c *MigrateRollbackCommand) Description() string {
	return "Rollback database migrations"
}

func (c *MigrateRollbackCommand) DefineFlags(fs *flag.FlagSet) {
	fs.IntVar(&c.steps, "step", 1, "Number of migrations to rollback")
	fs.BoolVar(&c.force, "force", false, "Skip confirmation prompt")
	fs.BoolVar(&c.allowMissing, "allow-missing", false, "Proceed even if some migrations are missing from the registry (skips them)")
}

func (c *MigrateRollbackCommand) Execute(ctx *CommandContext) error {
	if ctx.DB == nil {
		return fmt.Errorf("database connection required")
	}

	if c.steps <= 0 {
		return fmt.Errorf("steps must be greater than 0")
	}

	db := migrationConn(ctx)

	// Ambil N migrasi terakhir dari database, terbaru lebih dulu.
	// Urutan rollback ditentukan oleh query ini (menurun), bukan oleh urutan
	// registry — kebalikan dari `migrate` yang berjalan menaik.
	query := `SELECT version, name FROM migrations ORDER BY version DESC LIMIT $1`
	rows, err := db.Query(context.Background(), query, c.steps)
	if err != nil {
		return fmt.Errorf("failed to query migrations: %w", err)
	}
	defer rows.Close()

	// Kumpulkan migrasi yang akan di-rollback
	var migrationsToRollback []Migration
	var missing []MigrationHistory
	byVersion := make(map[int64]Migration)
	for _, migration := range GetAllMigrations() {
		byVersion[migration.Version] = migration
	}

	for rows.Next() {
		var version int64
		var name string
		if err := rows.Scan(&version, &name); err != nil {
			return err
		}

		migration, found := byVersion[version]
		if !found {
			// Kode migrasinya sudah tidak ada — fungsi Down-nya tidak bisa dijalankan
			missing = append(missing, MigrationHistory{Version: version, Name: name})
			continue
		}
		migrationsToRollback = append(migrationsToRollback, migration)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed to read migrations: %w", err)
	}

	// Migrasi yang tercatat di database tapi kodenya tidak terdaftar tidak punya
	// fungsi Down. Melewatinya diam-diam berbahaya: skema milik migrasi itu tetap
	// hidup sementara migrasi yang lebih tua di bawahnya ikut dibongkar.
	if len(missing) > 0 {
		fmt.Println("⚠ Migrasi berikut tercatat di database tapi tidak ada di registry:")
		for _, m := range missing {
			fmt.Printf("  - %s (version %d)\n", m.Name, m.Version)
		}
		fmt.Println()

		if !c.allowMissing {
			return fmt.Errorf(
				"%d dari %d migrasi yang ditargetkan tidak ada di registry sehingga fungsi Down-nya tidak dapat dijalankan; "+
					"tidak ada yang di-rollback. Pastikan package migrasinya ter-import, atau jalankan ulang dengan -allow-missing untuk melewatinya",
				len(missing), len(missing)+len(migrationsToRollback))
		}
		fmt.Println("-allow-missing aktif: migrasi di atas dilewati.")
	}

	if len(migrationsToRollback) == 0 {
		fmt.Println("No migrations to rollback")
		return nil
	}

	fmt.Printf("Rolling back %d migration(s)...\n", len(migrationsToRollback))

	// Display migrations that will be rolled back
	fmt.Println("\nThe following migrations will be rolled back:")
	for _, migration := range migrationsToRollback {
		fmt.Printf("  - %s (version %d)\n", migration.Name, migration.Version)
	}
	fmt.Println()

	// Confirmation prompt (unless -force flag is set)
	if !c.force {
		fmt.Print("Are you sure you want to proceed? (yes/no): ")
		var response string
		fmt.Scanln(&response)

		response = strings.ToLower(strings.TrimSpace(response))
		if response != "yes" && response != "y" {
			fmt.Println("Rollback cancelled")
			return nil
		}
		fmt.Println()
	}

	// Rollback each migration
	for _, migration := range migrationsToRollback {
		fmt.Printf("Rolling back: %s (version %d)\n", migration.Name, migration.Version)
		if err := RollbackMigration(db, migration); err != nil {
			return fmt.Errorf("rollback failed for %s: %w", migration.Name, err)
		}
		fmt.Printf("✓ Rolled back: %s\n", migration.Name)
	}

	fmt.Printf("\n✓ Successfully rolled back %d migration(s)\n", len(migrationsToRollback))
	if len(missing) > 0 {
		fmt.Printf("⚠ %d migrasi dilewati karena tidak ada di registry\n", len(missing))
	}
	return nil
}

// ============================================================================
// MakeMigrationCommand - Create a new migration file
// ============================================================================

// MakeMigrationCommand generates a new migration file
type MakeMigrationCommand struct {
	dir string
	pkg string
}

func (c *MakeMigrationCommand) Name() string {
	return "make:migration"
}

func (c *MakeMigrationCommand) Description() string {
	return "Create a new migration file"
}

func (c *MakeMigrationCommand) DefineFlags(fs *flag.FlagSet) {
	fs.StringVar(&c.dir, "dir", "migrations", "Directory to store migration files")
	fs.StringVar(&c.pkg, "pkg", "", "Go package name (default: directory name)")
}

func (c *MakeMigrationCommand) Execute(ctx *CommandContext) error {
	if len(ctx.Args) < 1 {
		return fmt.Errorf("migration name is required\nUsage: make:migration <name>")
	}

	name := ctx.Args[0]
	name = strings.ToLower(name)

	// Create directory if not exists
	if err := os.MkdirAll(c.dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Determine package name
	pkgName := c.pkg
	if pkgName == "" {
		pkgName = filepath.Base(c.dir)
		if pkgName == "." || pkgName == "/" {
			pkgName = "migrations"
		}
	}

	// Generate timestamp version
	timestamp := time.Now()
	version := timestamp.Format("20060102150405")

	// Construct filename: YYYYMMDDHHMMSS_name.go
	filename := fmt.Sprintf("%s_%s.go", version, name)
	filepath := filepath.Join(c.dir, filename)

	// CamelCase name for Go functions (create_users -> CreateUsers)
	funcName := ToCamelCase(name)

	data := migrationTemplateData{
		Package:   pkgName,
		Version:   version,
		Name:      name,
		FuncName:  funcName,
		Timestamp: timestamp.Format(time.RFC3339),
	}

	// Create file
	f, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	// Execute template
	tmpl, err := template.New("migration").Parse(migrationTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("failed to write template: %w", err)
	}

	fmt.Printf("✓ Migration created: %s\n", filepath)
	fmt.Printf("  Version: %s\n", version)
	fmt.Println("\nDon't forget to import this package in your main.go to register it:")
	fmt.Printf("  import _ \"github.com/dimframework/dimulai/%s\"\n", c.dir)

	return nil
}

type migrationTemplateData struct {
	Package   string
	Version   string
	Name      string
	FuncName  string
	Timestamp string
}

const migrationTemplate = `package {{.Package}}

import (
	"context"

	"github.com/dimframework/dim"
)

func init() {
	dim.Register(dim.Migration{
		Version: {{.Version}},
		Name:    "{{.Name}}",
		Up:      Up{{.FuncName}},
		Down:    Down{{.FuncName}},
	})
}

// Up{{.FuncName}} executes the migration (add tables, columns, etc)
func Up{{.FuncName}}(db dim.Database) error {
	// TODO: Write your migration SQL here
	query := ` + "`" + `
		CREATE TABLE IF NOT EXISTS example (
			id BIGSERIAL PRIMARY KEY,
			created_at TIMESTAMP DEFAULT NOW()
		);
	` + "`" + `
	err := db.Exec(context.Background(), query)
	return err
}

// Down{{.FuncName}} rolls back the migration (drop tables, columns, etc)
func Down{{.FuncName}}(db dim.Database) error {
	// TODO: Write your rollback SQL here
	query := "DROP TABLE IF EXISTS example CASCADE"
	err := db.Exec(context.Background(), query)
	return err
}
`

// ============================================================================
// MigrateListCommand - List migration status
// ============================================================================

// MigrateListCommand menampilkan status semua migrations (applied dan pending).
type MigrateListCommand struct{}

func (c *MigrateListCommand) Name() string {
	return "migrate:list"
}

func (c *MigrateListCommand) Description() string {
	return "Show migration status"
}

func (c *MigrateListCommand) Execute(ctx *CommandContext) error {
	if ctx.DB == nil {
		return fmt.Errorf("database connection required")
	}

	db := migrationConn(ctx)

	// Gabungan migrasi framework + migrasi terdaftar, terurut berdasarkan Version
	allMigrations := GetAllMigrations()

	// Ambil migrasi yang sudah tercatat di database
	type appliedRecord struct {
		name      string
		appliedAt time.Time
	}
	appliedMap := make(map[int64]appliedRecord)
	query := `SELECT version, name, applied_at FROM migrations ORDER BY version`
	rows, err := db.Query(context.Background(), query)
	if err != nil {
		// Table might not exist yet
		fmt.Println("⚠ Migrations table does not exist yet. Run 'migrate' first.")
	} else {
		defer rows.Close()
		for rows.Next() {
			var version int64
			var name string
			var appliedAt time.Time
			if err := rows.Scan(&version, &name, &appliedAt); err != nil {
				return err
			}
			appliedMap[version] = appliedRecord{name: name, appliedAt: appliedAt}
		}
		if err := rows.Err(); err != nil {
			return err
		}
	}

	// Display header with 80-column friendly layout
	fmt.Println("Migration Status:")
	fmt.Println()

	// Column widths (total ~78 chars with spacing)
	const (
		versionWidth = 8
		nameWidth    = 32
		statusWidth  = 10
		dateWidth    = 19
	)

	// Calculate separator width
	separatorWidth := versionWidth + nameWidth + statusWidth + dateWidth + 6 // 6 for spacing

	fmt.Printf("%-*s %-*s %-*s %s\n", versionWidth, "Version", nameWidth, "Name", statusWidth, "Status", "Applied At")
	fmt.Println(strings.Repeat("-", separatorWidth))

	truncate := func(name string) string {
		if len(name) > nameWidth {
			return name[:nameWidth-3] + "..."
		}
		return name
	}

	// Tampilkan setiap migrasi yang terdaftar
	appliedCount := 0
	pendingCount := 0
	known := make(map[int64]bool, len(allMigrations))
	for _, migration := range allMigrations {
		known[migration.Version] = true

		status := "Pending"
		appliedAt := "-"

		if rec, applied := appliedMap[migration.Version]; applied {
			status = "Applied"
			appliedAt = rec.appliedAt.Format("2006-01-02 15:04:05")
			appliedCount++
		} else {
			pendingCount++
		}

		fmt.Printf("%-*d %-*s %-*s %s\n", versionWidth, migration.Version, nameWidth, truncate(migration.Name), statusWidth, status, appliedAt)
	}

	// Migrasi yang tercatat di database tapi kodenya sudah tidak terdaftar.
	// Ditampilkan terpisah, bukan disembunyikan: inilah yang membuat
	// `migrate:rollback` menolak berjalan.
	var orphans []int64
	for version := range appliedMap {
		if !known[version] {
			orphans = append(orphans, version)
		}
	}
	slices.Sort(orphans)

	for _, version := range orphans {
		rec := appliedMap[version]
		fmt.Printf("%-*d %-*s %-*s %s\n", versionWidth, version, nameWidth, truncate(rec.name), statusWidth, "Orphan", rec.appliedAt.Format("2006-01-02 15:04:05"))
	}

	// Summary
	fmt.Println()
	fmt.Printf("Total: %d | Applied: %d | Pending: %d\n", len(allMigrations), appliedCount, pendingCount)
	if len(orphans) > 0 {
		fmt.Printf("⚠ Orphan: %d (tercatat di database tapi tidak ada di registry)\n", len(orphans))
	}

	return nil
}
