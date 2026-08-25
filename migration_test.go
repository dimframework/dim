package dim

import (
	"bytes"
	"context"
	"io"
	"os"
	"slices"
	"strings"
	"testing"
)

func TestFrameworkMigrationsToggle(t *testing.T) {
	// Simpan state awal agar tidak mengganggu test lain
	originalState := includeFrameworkMigrations
	defer func() {
		includeFrameworkMigrations = originalState
	}()

	t.Run("DefaultEnabled", func(t *testing.T) {
		// Reset ke true (default)
		includeFrameworkMigrations = true
		migrations := GetFrameworkMigrations()
		if len(migrations) == 0 {
			t.Error("Expected framework migrations to be present by default")
		}

		// Verifikasi keberadaan migrasi spesifik (misal: users)
		foundUsers := false
		for _, m := range migrations {
			if m.Name == "create_users_table" {
				foundUsers = true
				break
			}
		}
		if !foundUsers {
			t.Error("Expected create_users_table migration to be present")
		}
	})

	t.Run("CanDisable", func(t *testing.T) {
		// Disable migrations
		DisableFrameworkMigrations()

		migrations := GetFrameworkMigrations()
		if len(migrations) != 0 {
			t.Errorf("Expected 0 migrations after disable, got %d", len(migrations))
		}
	})
}

// TestGetRegisteredMigrationsSorted memastikan urutan jalan mengikuti Version,
// bukan urutan pemanggilan Register() (yang di dunia nyata = urutan init() package).
//
// Skenario: aplikasi modular dengan migrasi tersebar di beberapa package yang
// ter-inisialisasi terbalik terhadap urutan versinya.
func TestGetRegisteredMigrationsSorted(t *testing.T) {
	originalRegistry := migrationRegistry
	defer func() { migrationRegistry = originalRegistry }()
	migrationRegistry = nil

	noop := func(Database) error { return nil }

	// package "orders" ter-init lebih dulu, padahal versinya lebih besar
	Register(Migration{Version: 20260802100200, Name: "create_orders_table", Up: noop, Down: noop})
	// package "partners" ter-init belakangan, versinya lebih kecil dan dirujuk oleh orders
	Register(Migration{Version: 20260802100100, Name: "create_partners_table", Up: noop, Down: noop})

	migrations := GetRegisteredMigrations()
	if len(migrations) != 2 {
		t.Fatalf("expected 2 migrations, got %d", len(migrations))
	}

	if migrations[0].Version >= migrations[1].Version {
		t.Errorf("migrations not sorted by version: got %d before %d",
			migrations[0].Version, migrations[1].Version)
	}
	if migrations[0].Name != "create_partners_table" {
		t.Errorf("expected create_partners_table to run first, got %s", migrations[0].Name)
	}
}

// TestGetRegisteredMigrationsDoesNotMutateRegistry memastikan pengurutan bekerja
// pada salinan, bukan pada registry global.
func TestGetRegisteredMigrationsDoesNotMutateRegistry(t *testing.T) {
	originalRegistry := migrationRegistry
	defer func() { migrationRegistry = originalRegistry }()
	migrationRegistry = nil

	noop := func(Database) error { return nil }
	Register(Migration{Version: 20260802100200, Name: "second", Up: noop, Down: noop})
	Register(Migration{Version: 20260802100100, Name: "first", Up: noop, Down: noop})

	_ = GetRegisteredMigrations()

	if migrationRegistry[0].Name != "second" || migrationRegistry[1].Name != "first" {
		t.Errorf("registry global termodifikasi: %v", []string{migrationRegistry[0].Name, migrationRegistry[1].Name})
	}
}

// TestGetAllMigrationsSorted memastikan slice gabungan (framework + aplikasi)
// juga terurut, tidak hanya kebetulan karena versi framework selalu lebih kecil.
func TestGetAllMigrationsSorted(t *testing.T) {
	originalRegistry := migrationRegistry
	originalState := includeFrameworkMigrations
	defer func() {
		migrationRegistry = originalRegistry
		includeFrameworkMigrations = originalState
	}()
	migrationRegistry = nil
	includeFrameworkMigrations = true

	noop := func(Database) error { return nil }
	Register(Migration{Version: 20260802100200, Name: "create_orders_table", Up: noop, Down: noop})
	Register(Migration{Version: 20260802100100, Name: "create_partners_table", Up: noop, Down: noop})

	migrations := GetAllMigrations()

	frameworkCount := len(GetFrameworkMigrations())
	if len(migrations) != frameworkCount+2 {
		t.Fatalf("expected %d migrations, got %d", frameworkCount+2, len(migrations))
	}

	for i := 1; i < len(migrations); i++ {
		if migrations[i-1].Version > migrations[i].Version {
			t.Fatalf("migrations not sorted at index %d: %d before %d",
				i, migrations[i-1].Version, migrations[i].Version)
		}
	}
}

// TestMigrateAndRollbackOrder membuktikan arah urutan keduanya secara end-to-end
// terhadap SQLite in-memory:
//   - migrate  : menaik  (versi terlama lebih dulu)
//   - rollback : menurun (versi terbaru lebih dulu)
//
// Migrasi didaftarkan dengan urutan terbalik terhadap versinya, meniru dua
// package yang urutan init()-nya kebalikan dari urutan versi.
func TestMigrateAndRollbackOrder(t *testing.T) {
	originalRegistry := migrationRegistry
	originalState := includeFrameworkMigrations
	defer func() {
		migrationRegistry = originalRegistry
		includeFrameworkMigrations = originalState
	}()
	migrationRegistry = nil
	includeFrameworkMigrations = false

	db, err := NewSQLiteDatabase(DatabaseConfig{Database: ":memory:"})
	if err != nil {
		t.Fatalf("NewSQLiteDatabase: %v", err)
	}
	defer db.Close()

	var upOrder, downOrder []int64
	track := func(dst *[]int64, v int64) func(Database) error {
		return func(Database) error {
			*dst = append(*dst, v)
			return nil
		}
	}

	// Sengaja didaftarkan terbalik: versi besar lebih dulu
	Register(Migration{Version: 300, Name: "third", Up: track(&upOrder, 300), Down: track(&downOrder, 300)})
	Register(Migration{Version: 100, Name: "first", Up: track(&upOrder, 100), Down: track(&downOrder, 100)})
	Register(Migration{Version: 200, Name: "second", Up: track(&upOrder, 200), Down: track(&downOrder, 200)})

	ctx := &CommandContext{DB: db}

	if err := (&MigrateCommand{}).Execute(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	wantUp := []int64{100, 200, 300}
	if !slices.Equal(upOrder, wantUp) {
		t.Errorf("urutan migrate salah: got %v, want %v (menaik)", upOrder, wantUp)
	}

	// Rollback seluruhnya; -force agar tidak menunggu konfirmasi stdin
	if err := (&MigrateRollbackCommand{steps: 3, force: true}).Execute(ctx); err != nil {
		t.Fatalf("rollback: %v", err)
	}

	wantDown := []int64{300, 200, 100}
	if !slices.Equal(downOrder, wantDown) {
		t.Errorf("urutan rollback salah: got %v, want %v (menurun)", downOrder, wantDown)
	}
}

// setupMigratedDB menyiapkan SQLite in-memory dengan tiga migrasi terpakai
// (100, 200, 300) dan mengembalikan db beserta pencatat urutan Down.
func setupMigratedDB(t *testing.T) (Database, *[]int64) {
	t.Helper()

	db, err := NewSQLiteDatabase(DatabaseConfig{Database: ":memory:"})
	if err != nil {
		t.Fatalf("NewSQLiteDatabase: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	downOrder := &[]int64{}
	noop := func(Database) error { return nil }
	track := func(v int64) func(Database) error {
		return func(Database) error {
			*downOrder = append(*downOrder, v)
			return nil
		}
	}

	Register(Migration{Version: 100, Name: "first", Up: noop, Down: track(100)})
	Register(Migration{Version: 200, Name: "second", Up: noop, Down: track(200)})
	Register(Migration{Version: 300, Name: "third", Up: noop, Down: track(300)})

	if err := (&MigrateCommand{}).Execute(&CommandContext{DB: db}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db, downOrder
}

// isolateRegistry mengosongkan registry global untuk durasi satu test.
func isolateRegistry(t *testing.T) {
	t.Helper()
	originalRegistry := migrationRegistry
	originalState := includeFrameworkMigrations
	t.Cleanup(func() {
		migrationRegistry = originalRegistry
		includeFrameworkMigrations = originalState
	})
	migrationRegistry = nil
	includeFrameworkMigrations = false
}

// orphanRegistry mensimulasikan kode migrasi yang dihapus: versi 300 tetap
// tercatat di database, tapi tidak lagi ada di registry.
func orphanRegistry(t *testing.T) {
	t.Helper()
	kept := make([]Migration, 0, len(migrationRegistry))
	for _, m := range migrationRegistry {
		if m.Version != 300 {
			kept = append(kept, m)
		}
	}
	migrationRegistry = kept
}

// TestRollbackRefusesWhenMigrationMissing memastikan rollback menolak berjalan
// ketika migrasi yang ditargetkan tidak punya fungsi Down, dan tidak
// membongkar migrasi yang lebih tua di bawahnya.
func TestRollbackRefusesWhenMigrationMissing(t *testing.T) {
	isolateRegistry(t)
	db, downOrder := setupMigratedDB(t)
	orphanRegistry(t)

	// -step 3 menargetkan 300 (hilang), 200, dan 100
	err := (&MigrateRollbackCommand{steps: 3, force: true}).Execute(&CommandContext{DB: db})
	if err == nil {
		t.Fatal("expected error ketika migrasi target tidak ada di registry, got nil")
	}

	if len(*downOrder) != 0 {
		t.Errorf("tidak boleh ada yang di-rollback, tapi Down dijalankan untuk %v", *downOrder)
	}

	// Database harus tetap utuh
	var count int64
	row := db.QueryRow(context.Background(), "SELECT COUNT(*) FROM migrations")
	if err := row.Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3 migrasi masih tercatat, got %d", count)
	}
}

// TestRollbackAllowMissingSkips memastikan -allow-missing memberi jalan keluar
// dan melewati migrasi yang hilang, bukan menggagalkan seluruh perintah.
func TestRollbackAllowMissingSkips(t *testing.T) {
	isolateRegistry(t)
	db, downOrder := setupMigratedDB(t)
	orphanRegistry(t)

	cmd := &MigrateRollbackCommand{steps: 3, force: true, allowMissing: true}
	if err := cmd.Execute(&CommandContext{DB: db}); err != nil {
		t.Fatalf("rollback dengan -allow-missing: %v", err)
	}

	want := []int64{200, 100}
	if !slices.Equal(*downOrder, want) {
		t.Errorf("urutan rollback salah: got %v, want %v", *downOrder, want)
	}

	// Versi 300 yang dilewati harus tetap tercatat
	var remaining int64
	row := db.QueryRow(context.Background(), "SELECT version FROM migrations")
	if err := row.Scan(&remaining); err != nil {
		t.Fatalf("scan sisa: %v", err)
	}
	if remaining != 300 {
		t.Errorf("expected versi 300 tetap tercatat, got %d", remaining)
	}
}

// captureStdout menangkap tulisan ke os.Stdout selama fn berjalan.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	original := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		done <- buf.String()
	}()

	fn()

	w.Close()
	os.Stdout = original
	return <-done
}

// TestMigrateListCountsWithOrphan menutup lubang hitungan: sebelumnya
// pendingCount = len(allMigrations) - len(appliedMap), yang ikut menghitung
// versi database yang tidak ada di registry dan bisa jadi negatif.
func TestMigrateListCountsWithOrphan(t *testing.T) {
	isolateRegistry(t)
	db, _ := setupMigratedDB(t)
	orphanRegistry(t)

	// Tambah satu migrasi yang belum pernah dijalankan
	noop := func(Database) error { return nil }
	Register(Migration{Version: 400, Name: "fourth", Up: noop, Down: noop})

	// Registry kini: 100 (applied), 200 (applied), 400 (pending)
	// Database kini: 100, 200, 300 — versi 300 adalah orphan
	out := captureStdout(t, func() {
		if err := (&MigrateListCommand{}).Execute(&CommandContext{DB: db}); err != nil {
			t.Errorf("migrate:list: %v", err)
		}
	})

	if !strings.Contains(out, "Total: 3 | Applied: 2 | Pending: 1") {
		t.Errorf("ringkasan salah.\ngot:\n%s", out)
	}

	// Orphan harus terlihat, bukan tersembunyi
	if !strings.Contains(out, "Orphan: 1") {
		t.Errorf("orphan tidak dilaporkan di ringkasan.\ngot:\n%s", out)
	}
	if !strings.Contains(out, "third") {
		t.Errorf("baris orphan 'third' tidak ditampilkan.\ngot:\n%s", out)
	}
}

// TestMigrateListNoOrphan memastikan jalur normal tetap bersih: tanpa drift,
// tidak ada baris atau peringatan orphan sama sekali.
func TestMigrateListNoOrphan(t *testing.T) {
	isolateRegistry(t)
	db, _ := setupMigratedDB(t)

	out := captureStdout(t, func() {
		if err := (&MigrateListCommand{}).Execute(&CommandContext{DB: db}); err != nil {
			t.Errorf("migrate:list: %v", err)
		}
	})

	if !strings.Contains(out, "Total: 3 | Applied: 3 | Pending: 0") {
		t.Errorf("ringkasan salah.\ngot:\n%s", out)
	}
	if strings.Contains(out, "Orphan") {
		t.Errorf("tidak boleh ada orphan.\ngot:\n%s", out)
	}
}

// isolateMigrationSource mengembalikan sumber migrasi ke registry global
// setelah satu test selesai.
func isolateMigrationSource(t *testing.T) {
	t.Helper()
	original := migrationSource
	t.Cleanup(func() { migrationSource = original })
}

// tableExists melaporkan apakah sebuah tabel ada di SQLite in-memory.
func tableExists(t *testing.T, db Database, name string) bool {
	t.Helper()
	var count int64
	row := db.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?", name)
	if err := row.Scan(&count); err != nil {
		t.Fatalf("sqlite_master: %v", err)
	}
	return count > 0
}

// recordedVersions mengembalikan versi yang tercatat di sebuah tabel pencatat.
func recordedVersions(t *testing.T, db Database, table string) []int64 {
	t.Helper()
	rows, err := db.Query(context.Background(), "SELECT version FROM "+table+" ORDER BY version")
	if err != nil {
		t.Fatalf("query %s: %v", table, err)
	}
	defer rows.Close()

	var versions []int64
	for rows.Next() {
		var v int64
		if err := rows.Scan(&v); err != nil {
			t.Fatalf("scan: %v", err)
		}
		versions = append(versions, v)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	return versions
}

// newMemoryDB membuka SQLite in-memory yang ditutup saat test selesai.
func newMemoryDB(t *testing.T) Database {
	t.Helper()
	db, err := NewSQLiteDatabase(DatabaseConfig{Database: ":memory:"})
	if err != nil {
		t.Fatalf("NewSQLiteDatabase: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// TestResolveMigrationsTable mengunci apa yang boleh disisipkan ke SQL.
// Nama tabel tidak dapat dikirim sebagai parameter query, sehingga validasinya
// di sini adalah satu-satunya penjaga.
func TestResolveMigrationsTable(t *testing.T) {
	valid := map[string]string{
		"":                              DefaultMigrationsTable,
		"  ":                            DefaultMigrationsTable,
		"migrations":                    "migrations",
		"myschema.migrations":           "myschema.migrations",
		"test_9f2c_module_a.migrations": "test_9f2c_module_a.migrations",
		"_private.t$bl":                 "_private.t$bl",
		"  myschema.migrations  ":       "myschema.migrations",
	}
	for input, want := range valid {
		got, err := resolveMigrationsTable(input)
		if err != nil {
			t.Errorf("resolveMigrationsTable(%q): unexpected error %v", input, err)
			continue
		}
		if got != want {
			t.Errorf("resolveMigrationsTable(%q) = %q, want %q", input, got, want)
		}
	}

	invalid := []string{
		"migrations; DROP TABLE users",
		"migrations--",
		"public.myschema.migrations",
		"1migrations",
		"mig rations",
		"\"quoted\"",
		"myschema.",
		".migrations",
		"migrations)",
	}
	for _, input := range invalid {
		if got, err := resolveMigrationsTable(input); err == nil {
			t.Errorf("resolveMigrationsTable(%q) = %q, want error", input, got)
		}
	}
}

// TestRunMigrationsInCustomTable memastikan tabel pencatat benar-benar pindah:
// riwayatnya masuk ke tabel yang diminta, dan tabel `migrations` bawaan tidak
// ikut dibuat.
func TestRunMigrationsInCustomTable(t *testing.T) {
	db := newMemoryDB(t)

	noop := func(Database) error { return nil }
	migrations := []Migration{
		{Version: 100, Name: "first", Up: noop, Down: noop},
		{Version: 200, Name: "second", Up: noop, Down: noop},
	}

	if err := RunMigrationsIn(db, "module_b_migrations", migrations); err != nil {
		t.Fatalf("RunMigrationsIn: %v", err)
	}

	if got := recordedVersions(t, db, "module_b_migrations"); !slices.Equal(got, []int64{100, 200}) {
		t.Errorf("versi tercatat: got %v, want [100 200]", got)
	}
	if tableExists(t, db, DefaultMigrationsTable) {
		t.Error("tabel `migrations` bawaan tidak boleh ikut dibuat")
	}
}

// TestRunMigrationsInQualifiedTable memastikan nama berkualifikasi benar-benar
// sampai ke SQL, bukan hanya lolos validasi. SQLite mengenal `main` sebagai
// kualifikasi database, cukup untuk membuktikan bentuk `x.y` tersisip utuh.
func TestRunMigrationsInQualifiedTable(t *testing.T) {
	db := newMemoryDB(t)

	noop := func(Database) error { return nil }
	migrations := []Migration{{Version: 100, Name: "first", Up: noop, Down: noop}}

	if err := RunMigrationsIn(db, "main.module_a_migrations", migrations); err != nil {
		t.Fatalf("RunMigrationsIn: %v", err)
	}

	if got := recordedVersions(t, db, "module_a_migrations"); !slices.Equal(got, []int64{100}) {
		t.Errorf("versi tercatat: got %v, want [100]", got)
	}
}

// TestRunMigrationsInRejectsBadTable memastikan nama tabel yang tidak valid
// ditolak sebelum apa pun dijalankan — bukan diteruskan ke SQL.
func TestRunMigrationsInRejectsBadTable(t *testing.T) {
	db := newMemoryDB(t)

	ran := false
	migrations := []Migration{{
		Version: 100,
		Name:    "first",
		Up:      func(Database) error { ran = true; return nil },
		Down:    func(Database) error { return nil },
	}}

	err := RunMigrationsIn(db, "migrations; DROP TABLE users", migrations)
	if err == nil {
		t.Fatal("expected error untuk nama tabel yang tidak valid")
	}
	if ran {
		t.Error("tidak boleh ada migrasi yang berjalan saat nama tabelnya ditolak")
	}
}

// TestRunMigrationsUsesDefaultTable mengunci kompatibilitas mundur: pembungkus
// lamanya tetap mencatat di tabel `migrations`.
func TestRunMigrationsUsesDefaultTable(t *testing.T) {
	db := newMemoryDB(t)

	noop := func(Database) error { return nil }
	if err := RunMigrations(db, []Migration{{Version: 100, Name: "first", Up: noop, Down: noop}}); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}

	if got := recordedVersions(t, db, DefaultMigrationsTable); !slices.Equal(got, []int64{100}) {
		t.Errorf("versi tercatat di `migrations`: got %v, want [100]", got)
	}
}

// TestRollbackMigrationInRemovesFromScopedTable memastikan rollback menghapus
// catatannya dari tabel yang sama dengan yang mencatatnya.
func TestRollbackMigrationInRemovesFromScopedTable(t *testing.T) {
	db := newMemoryDB(t)

	noop := func(Database) error { return nil }
	m := Migration{Version: 100, Name: "first", Up: noop, Down: noop}

	if err := RunMigrationsIn(db, "module_b_migrations", []Migration{m}); err != nil {
		t.Fatalf("RunMigrationsIn: %v", err)
	}
	if err := RollbackMigrationIn(db, "module_b_migrations", m); err != nil {
		t.Fatalf("RollbackMigrationIn: %v", err)
	}

	if got := recordedVersions(t, db, "module_b_migrations"); len(got) != 0 {
		t.Errorf("catatan harusnya terhapus, tersisa %v", got)
	}
}

// TestSetMigrationSourceUsedByCommands memastikan perintah bawaan membaca
// sumber yang dipasang, bukan registry global — inti dari kasus aplikasi yang
// merakit migrasinya saat runtime karena butuh nama schema.
func TestSetMigrationSourceUsedByCommands(t *testing.T) {
	isolateRegistry(t)
	isolateMigrationSource(t)
	db := newMemoryDB(t)

	// Registry global sengaja diisi hal lain; ia tidak boleh terpakai
	noop := func(Database) error { return nil }
	Register(Migration{Version: 900, Name: "dari_registry", Up: noop, Down: noop})

	var upOrder []int64
	track := func(v int64) func(Database) error {
		return func(Database) error {
			upOrder = append(upOrder, v)
			return nil
		}
	}

	SetMigrationSource(func() []Migration {
		// Sengaja tidak terurut: dim yang mengurutkannya
		return []Migration{
			{Version: 200, Name: "dari_sumber_kedua", Up: track(200), Down: noop},
			{Version: 100, Name: "dari_sumber_pertama", Up: track(100), Down: noop},
		}
	})

	if err := (&MigrateCommand{}).Execute(&CommandContext{DB: db}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	if !slices.Equal(upOrder, []int64{100, 200}) {
		t.Errorf("urutan migrate salah: got %v, want [100 200] (menaik)", upOrder)
	}
	if got := recordedVersions(t, db, DefaultMigrationsTable); !slices.Equal(got, []int64{100, 200}) {
		t.Errorf("versi tercatat: got %v, want [100 200] — registry global ikut terpakai?", got)
	}
}

// TestMigrationSourceUnsetKeepsRegistry memastikan tanpa SetMigrationSource
// perilakunya persis seperti sebelumnya.
func TestMigrationSourceUnsetKeepsRegistry(t *testing.T) {
	isolateRegistry(t)
	isolateMigrationSource(t)

	noop := func(Database) error { return nil }
	Register(Migration{Version: 100, Name: "first", Up: noop, Down: noop})

	got := migrationsFromSource()
	want := GetAllMigrations()
	if len(got) != len(want) {
		t.Fatalf("got %d migrasi, want %d", len(got), len(want))
	}
	for i := range got {
		if got[i].Version != want[i].Version {
			t.Errorf("index %d: got versi %d, want %d", i, got[i].Version, want[i].Version)
		}
	}

	// Dipasang lalu dilepas kembali
	SetMigrationSource(func() []Migration { return nil })
	if len(migrationsFromSource()) != 0 {
		t.Error("sumber terpasang tidak terpakai")
	}
	SetMigrationSource(nil)
	if len(migrationsFromSource()) != len(want) {
		t.Error("SetMigrationSource(nil) tidak mengembalikan ke registry global")
	}
}

// TestMigrateCommandTableFlagScopesHistory adalah alasan utama flag -table ada:
// dua modul dengan pencatatnya masing-masing tidak saling melihat, sehingga
// `-step` punya makna domain — "dua terakhir milik modul ini", bukan "dua baris
// terakhir dari riwayat gabungan".
func TestMigrateCommandTableFlagScopesHistory(t *testing.T) {
	isolateRegistry(t)
	isolateMigrationSource(t)
	db := newMemoryDB(t)

	var downOrder []int64
	noop := func(Database) error { return nil }
	track := func(v int64) func(Database) error {
		return func(Database) error {
			downOrder = append(downOrder, v)
			return nil
		}
	}

	moduleA := []Migration{
		{Version: 100, Name: "module_a_first", Up: noop, Down: track(100)},
		{Version: 200, Name: "module_a_second", Up: noop, Down: track(200)},
	}
	moduleB := []Migration{
		{Version: 300, Name: "module_b_first", Up: noop, Down: track(300)},
	}

	SetMigrationSource(func() []Migration { return moduleA })
	if err := (&MigrateCommand{table: "module_a_migrations"}).Execute(&CommandContext{DB: db}); err != nil {
		t.Fatalf("migrate module_a: %v", err)
	}

	SetMigrationSource(func() []Migration { return moduleB })
	if err := (&MigrateCommand{table: "module_b_migrations"}).Execute(&CommandContext{DB: db}); err != nil {
		t.Fatalf("migrate module_b: %v", err)
	}

	if got := recordedVersions(t, db, "module_a_migrations"); !slices.Equal(got, []int64{100, 200}) {
		t.Errorf("riwayat module_a: got %v, want [100 200]", got)
	}
	if got := recordedVersions(t, db, "module_b_migrations"); !slices.Equal(got, []int64{300}) {
		t.Errorf("riwayat module_b: got %v, want [300]", got)
	}

	// Rollback satu langkah pada module_a harus membatalkan 200 — migrasi
	// terakhir miliknya — bukan 300 yang versinya lebih tinggi tapi milik modul
	// lain.
	SetMigrationSource(func() []Migration { return moduleA })
	rollback := &MigrateRollbackCommand{steps: 1, force: true, table: "module_a_migrations"}
	if err := rollback.Execute(&CommandContext{DB: db}); err != nil {
		t.Fatalf("rollback module_a: %v", err)
	}

	if !slices.Equal(downOrder, []int64{200}) {
		t.Errorf("rollback bocor lintas modul: got %v, want [200]", downOrder)
	}
	if got := recordedVersions(t, db, "module_a_migrations"); !slices.Equal(got, []int64{100}) {
		t.Errorf("riwayat module_a setelah rollback: got %v, want [100]", got)
	}
	if got := recordedVersions(t, db, "module_b_migrations"); !slices.Equal(got, []int64{300}) {
		t.Errorf("riwayat module_b tersentuh rollback module_a: got %v", got)
	}
}

// TestRollbackMissingCheckIsScopedToTable memastikan penolakan `-allow-missing`
// dari v0.11.0 ikut bercakupan: migrasi modul lain yang tidak ada di sumber
// saat ini tidak lagi mengunci rollback modul ini.
func TestRollbackMissingCheckIsScopedToTable(t *testing.T) {
	isolateRegistry(t)
	isolateMigrationSource(t)
	db := newMemoryDB(t)

	noop := func(Database) error { return nil }
	moduleA := []Migration{{Version: 100, Name: "module_a_first", Up: noop, Down: noop}}
	moduleB := []Migration{{Version: 300, Name: "module_b_first", Up: noop, Down: noop}}

	SetMigrationSource(func() []Migration { return moduleA })
	if err := (&MigrateCommand{table: "module_a_migrations"}).Execute(&CommandContext{DB: db}); err != nil {
		t.Fatalf("migrate module_a: %v", err)
	}
	SetMigrationSource(func() []Migration { return moduleB })
	if err := (&MigrateCommand{table: "module_b_migrations"}).Execute(&CommandContext{DB: db}); err != nil {
		t.Fatalf("migrate module_b: %v", err)
	}

	// module_b dicabut: sumbernya kini hanya module_a. Riwayat module_b masih
	// ada, tapi di tabelnya sendiri — sehingga rollback module_a tidak melihatnya.
	SetMigrationSource(func() []Migration { return moduleA })
	rollback := &MigrateRollbackCommand{steps: 1, force: true, table: "module_a_migrations"}
	if err := rollback.Execute(&CommandContext{DB: db}); err != nil {
		t.Fatalf("rollback module_a seharusnya tidak terhalang riwayat modul lain: %v", err)
	}
}

// TestMigrateListTableFlag memastikan `migrate:list` membaca tabel yang diminta.
func TestMigrateListTableFlag(t *testing.T) {
	isolateRegistry(t)
	isolateMigrationSource(t)
	db := newMemoryDB(t)

	noop := func(Database) error { return nil }
	moduleB := []Migration{
		{Version: 300, Name: "module_b_first", Up: noop, Down: noop},
		{Version: 400, Name: "module_b_second", Up: noop, Down: noop},
	}

	SetMigrationSource(func() []Migration { return moduleB[:1] })
	if err := (&MigrateCommand{table: "module_b_migrations"}).Execute(&CommandContext{DB: db}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	SetMigrationSource(func() []Migration { return moduleB })
	out := captureStdout(t, func() {
		if err := (&MigrateListCommand{table: "module_b_migrations"}).Execute(&CommandContext{DB: db}); err != nil {
			t.Errorf("migrate:list: %v", err)
		}
	})

	if !strings.Contains(out, "Total: 2 | Applied: 1 | Pending: 1") {
		t.Errorf("ringkasan salah.\ngot:\n%s", out)
	}
	if strings.Contains(out, "Orphan") {
		t.Errorf("tidak boleh ada orphan.\ngot:\n%s", out)
	}
}
