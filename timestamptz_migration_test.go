package dim

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// Sesi di zona selain UTC adalah tempat kolom tanpa zona salah: nilai jam dinding
// UTC yang ditulis dim dibandingkan dengan NOW() yang ditafsirkan di zona sesi.
const testNonUTCZone = "Asia/Jakarta" // +07, tanpa DST

// newIsolatedPostgresDB membuat schema baru yang kosong dan koneksi yang
// search_path-nya hanya schema itu, dengan zona sesi testNonUTCZone. Schema-nya
// dihapus saat test selesai.
func newIsolatedPostgresDB(t *testing.T) *PostgresDatabase {
	t.Helper()
	return newIsolatedPostgresDBWithMode(t, "")
}

// newIsolatedPostgresDBWithMode sama dengan newIsolatedPostgresDB, dengan
// QueryExecMode tertentu ("simple" untuk pgbouncer).
func newIsolatedPostgresDBWithMode(t *testing.T, execMode string) *PostgresDatabase {
	t.Helper()
	admin := newTestPostgresDB(t)
	schema := fmt.Sprintf("dim_tz_%d", time.Now().UnixNano())
	ctx := context.Background()
	if err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() { admin.Exec(context.Background(), "DROP SCHEMA "+schema+" CASCADE") })

	cfg := testPostgresConfig(t)
	cfg.RuntimeParams = map[string]string{"search_path": schema, "timezone": testNonUTCZone}
	cfg.QueryExecMode = execMode
	db, err := NewPostgresDatabase(cfg)
	if err != nil {
		t.Fatalf("NewPostgresDatabase: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// columnType mengembalikan tipe kolom seperti format_type, dicari lewat
// search_path koneksi — sama seperti SQL tak-berkualifikasi milik dim.
func columnType(t *testing.T, db Database, table, column string) string {
	t.Helper()
	var typ string
	err := db.QueryRow(context.Background(), `
		SELECT format_type(a.atttypid, a.atttypmod)
		FROM pg_attribute a
		WHERE a.attrelid = to_regclass($1) AND a.attname = $2 AND NOT a.attisdropped`,
		table, column).Scan(&typ)
	if err != nil {
		t.Fatalf("tipe %s.%s: %v", table, column, err)
	}
	return typ
}

func TestFrameworkMigrations_Postgres_TimeColumnsAreTimestamptz(t *testing.T) {
	db := newIsolatedPostgresDB(t)
	if err := RunMigrations(db, GetFrameworkMigrations()); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}

	for _, c := range frameworkTimeColumns {
		for _, col := range c.columns {
			if got := columnType(t, db, c.table, col); got != "timestamp with time zone" {
				t.Errorf("%s.%s = %s, harus timestamp with time zone", c.table, col, got)
			}
		}
	}
	if got := columnType(t, db, "migrations", "applied_at"); got != "timestamp with time zone" {
		t.Errorf("migrations.applied_at = %s, harus timestamp with time zone", got)
	}
}

// Database yang dimigrasi sebelum perbaikan ini: tabelnya dibuat dengan kolom
// tanpa zona dan berisi jam dinding UTC. Konversinya harus mempertahankan
// instannya meski sesi berjalan di zona lain.
func TestFrameworkMigrations_Postgres_ConvertsExistingTimestampColumns(t *testing.T) {
	db := newIsolatedPostgresDB(t)
	ctx := context.Background()

	legacy := []string{
		`CREATE TABLE migrations (version BIGINT PRIMARY KEY, name VARCHAR(255) NOT NULL, applied_at TIMESTAMP DEFAULT NOW())`,
		`CREATE TABLE users (id UUID PRIMARY KEY, email VARCHAR(255) UNIQUE NOT NULL, name VARCHAR(100), password VARCHAR(255) NOT NULL, created_at TIMESTAMP DEFAULT NOW(), updated_at TIMESTAMP DEFAULT NOW())`,
		`CREATE TABLE refresh_tokens (id BIGSERIAL PRIMARY KEY, user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE, token_hash VARCHAR(255) UNIQUE NOT NULL, user_agent TEXT NOT NULL, ip_address VARCHAR(45) NOT NULL, expires_at TIMESTAMP NOT NULL, created_at TIMESTAMP DEFAULT NOW(), revoked_at TIMESTAMP)`,
		`CREATE TABLE password_reset_tokens (id BIGSERIAL PRIMARY KEY, user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE, token_hash VARCHAR(255) UNIQUE NOT NULL, expires_at TIMESTAMP NOT NULL, used_at TIMESTAMP, created_at TIMESTAMP DEFAULT NOW())`,
		`CREATE UNLOGGED TABLE token_blocklist (identifier VARCHAR(255) PRIMARY KEY, expires_at TIMESTAMP NOT NULL)`,
		`CREATE UNLOGGED TABLE rate_limits (key VARCHAR(255) PRIMARY KEY, count INT NOT NULL DEFAULT 0, expires_at TIMESTAMP NOT NULL)`,
		`INSERT INTO migrations (version, name, applied_at) VALUES
			(1, 'create_users_table', '2026-01-02 03:04:05'),
			(2, 'create_refresh_tokens_table', '2026-01-02 03:04:05'),
			(3, 'create_password_reset_tokens_table', '2026-01-02 03:04:05'),
			(4, 'create_token_blocklist_table', '2026-01-02 03:04:05'),
			(5, 'create_rate_limits_table', '2026-01-02 03:04:05')`,
		`INSERT INTO users (id, email, password, created_at, updated_at) VALUES
			('00000000-0000-0000-0000-000000000001', 'a@example.com', 'x', '2026-01-02 03:04:05', '2026-01-02 03:04:05')`,
		`INSERT INTO refresh_tokens (user_id, token_hash, user_agent, ip_address, expires_at, created_at, revoked_at) VALUES
			('00000000-0000-0000-0000-000000000001', 'rt', 'ua', '127.0.0.1', '2026-01-02 03:04:05', '2026-01-02 03:04:05', '2026-01-02 03:04:05')`,
		`INSERT INTO password_reset_tokens (user_id, token_hash, expires_at, used_at, created_at) VALUES
			('00000000-0000-0000-0000-000000000001', 'pr', '2026-01-02 03:04:05', '2026-01-02 03:04:05', '2026-01-02 03:04:05')`,
		`INSERT INTO token_blocklist (identifier, expires_at) VALUES ('jti', '2026-01-02 03:04:05')`,
		`INSERT INTO rate_limits (key, count, expires_at) VALUES ('ip:1', 1, '2026-01-02 03:04:05')`,
	}
	for _, q := range legacy {
		if err := db.Exec(ctx, q); err != nil {
			t.Fatalf("siapkan skema lama: %v\n%s", err, q)
		}
	}

	if err := RunMigrations(db, GetFrameworkMigrations()); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}

	want := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	check := func(table, column string) {
		t.Helper()
		if got := columnType(t, db, table, column); got != "timestamp with time zone" {
			t.Errorf("%s.%s = %s, harus timestamp with time zone", table, column, got)
		}
		var got time.Time
		q := fmt.Sprintf("SELECT %s FROM %s LIMIT 1", column, table)
		if err := db.QueryRow(ctx, q).Scan(&got); err != nil {
			t.Fatalf("baca %s.%s: %v", table, column, err)
		}
		if !got.Equal(want) {
			t.Errorf("%s.%s = %v, harus %v — instannya bergeser saat konversi", table, column, got.UTC(), want)
		}
	}
	for _, c := range frameworkTimeColumns {
		for _, col := range c.columns {
			check(c.table, col)
		}
	}
	check("migrations", "applied_at")
}

// Konversi hanya boleh menyentuh kolom yang masih tanpa zona. Pada kolom yang
// sudah timestamptz, `USING col AT TIME ZONE 'UTC'` menggeser nilainya sebesar
// offset zona sesi.
func TestTimestamptzMigration_Postgres_IsIdempotent(t *testing.T) {
	db := newIsolatedPostgresDB(t)
	ctx := context.Background()
	if err := RunMigrations(db, GetFrameworkMigrations()); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}

	want := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	if err := db.Exec(ctx, "INSERT INTO token_blocklist (identifier, expires_at) VALUES ('jti', $1)", want); err != nil {
		t.Fatalf("insert: %v", err)
	}

	m := convertTimestampsToTimestamptzMigration()
	if err := m.Up(db); err != nil {
		t.Fatalf("Up kedua: %v", err)
	}
	if err := ensureMigrationsTable(db, DefaultMigrationsTable); err != nil {
		t.Fatalf("ensureMigrationsTable kedua: %v", err)
	}

	var got time.Time
	if err := db.QueryRow(ctx, "SELECT expires_at FROM token_blocklist WHERE identifier = 'jti'").Scan(&got); err != nil {
		t.Fatalf("baca: %v", err)
	}
	if !got.Equal(want) {
		t.Errorf("expires_at = %v, harus %v", got.UTC(), want)
	}
}

// Down mengembalikan kolom ke tanpa zona dengan jam dinding UTC, sehingga kode
// versi sebelumnya membaca nilai yang sama.
func TestTimestamptzMigration_Postgres_DownRestoresUTCWallClock(t *testing.T) {
	db := newIsolatedPostgresDB(t)
	ctx := context.Background()
	if err := RunMigrations(db, GetFrameworkMigrations()); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	want := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	if err := db.Exec(ctx, "INSERT INTO token_blocklist (identifier, expires_at) VALUES ('jti', $1)", want); err != nil {
		t.Fatalf("insert: %v", err)
	}

	if err := convertTimestampsToTimestamptzMigration().Down(db); err != nil {
		t.Fatalf("Down: %v", err)
	}

	if got := columnType(t, db, "token_blocklist", "expires_at"); got != "timestamp without time zone" {
		t.Errorf("token_blocklist.expires_at = %s sesudah Down, harus timestamp without time zone", got)
	}
	var wall string
	if err := db.QueryRow(ctx, "SELECT expires_at::text FROM token_blocklist WHERE identifier = 'jti'").Scan(&wall); err != nil {
		t.Fatalf("baca: %v", err)
	}
	if wall != "2026-01-02 03:04:05" {
		t.Errorf("jam dinding = %s, harus 2026-01-02 03:04:05 (UTC)", wall)
	}
}

// Token yang dicabut harus tetap tercabut sampai expires_at, berapa pun zona
// sesinya. Dengan kolom tanpa zona, sesi +07 membuatnya kedaluwarsa 7 jam lebih
// awal — token yang dicabut dengan sisa umur 1 jam langsung diterima lagi.
func TestDatabaseBlocklist_Postgres_NonUTCSession(t *testing.T) {
	db := newIsolatedPostgresDB(t)
	if err := RunMigrations(db, GetFrameworkMigrations()); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	bl := NewDatabaseBlocklist(db)
	ctx := context.Background()

	if err := bl.Invalidate(ctx, "jti-1h", time.Hour); err != nil {
		t.Fatalf("Invalidate: %v", err)
	}
	revoked, err := bl.IsRevoked(ctx, "jti-1h")
	if err != nil {
		t.Fatalf("IsRevoked: %v", err)
	}
	if !revoked {
		t.Error("token yang dicabut dengan sisa umur 1 jam dianggap tidak tercabut")
	}
}

// Store mengembalikan waktu dalam UTC, sama seperti saat kolomnya tanpa zona;
// pgx memindai timestamptz ke time.Local, yang akan bocor ke JSON sebagai
// offset mesin.
func TestDatabaseTokenStore_Postgres_ReturnsUTC(t *testing.T) {
	db := newIsolatedPostgresDB(t)
	if err := RunMigrations(db, GetFrameworkMigrations()); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	ctx := context.Background()
	userID := "00000000-0000-0000-0000-000000000001"
	if err := db.Exec(ctx, "INSERT INTO users (id, email, password) VALUES ($1, 'a@example.com', 'x')", userID); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	store := NewDatabaseTokenStore(db)
	expires := time.Now().Add(time.Hour)

	rt := &RefreshToken{UserID: userID, TokenHash: "rt", UserAgent: "ua", IPAddress: "127.0.0.1", ExpiresAt: expires}
	if err := store.SaveRefreshToken(ctx, rt); err != nil {
		t.Fatalf("SaveRefreshToken: %v", err)
	}
	if err := store.RevokeRefreshToken(ctx, "rt"); err != nil {
		t.Fatalf("RevokeRefreshToken: %v", err)
	}
	found, err := store.FindRefreshToken(ctx, "rt")
	if err != nil {
		t.Fatalf("FindRefreshToken: %v", err)
	}
	assertUTC(t, "SaveRefreshToken CreatedAt", rt.CreatedAt)
	assertUTC(t, "RefreshToken.ExpiresAt", found.ExpiresAt)
	assertUTC(t, "RefreshToken.CreatedAt", found.CreatedAt)
	if found.RevokedAt == nil {
		t.Fatal("RevokedAt nil sesudah RevokeRefreshToken")
	}
	assertUTC(t, "RefreshToken.RevokedAt", *found.RevokedAt)
	if !found.ExpiresAt.Equal(expires.Truncate(time.Second)) {
		t.Errorf("ExpiresAt = %v, harus %v", found.ExpiresAt, expires.Truncate(time.Second).UTC())
	}

	pr := &PasswordResetToken{UserID: userID, TokenHash: "pr", ExpiresAt: expires}
	if err := store.SavePasswordResetToken(ctx, pr); err != nil {
		t.Fatalf("SavePasswordResetToken: %v", err)
	}
	if err := store.MarkPasswordResetUsed(ctx, "pr"); err != nil {
		t.Fatalf("MarkPasswordResetUsed: %v", err)
	}
	foundPR, err := store.FindPasswordResetToken(ctx, "pr")
	if err != nil {
		t.Fatalf("FindPasswordResetToken: %v", err)
	}
	assertUTC(t, "SavePasswordResetToken CreatedAt", pr.CreatedAt)
	assertUTC(t, "PasswordResetToken.ExpiresAt", foundPR.ExpiresAt)
	assertUTC(t, "PasswordResetToken.CreatedAt", foundPR.CreatedAt)
	if foundPR.UsedAt == nil {
		t.Fatal("UsedAt nil sesudah MarkPasswordResetUsed")
	}
	assertUTC(t, "PasswordResetToken.UsedAt", *foundPR.UsedAt)
}

func assertUTC(t *testing.T, name string, v time.Time) {
	t.Helper()
	if v.Location() != time.UTC {
		t.Errorf("%s berlokasi %v, harus UTC", name, v.Location())
	}
}

func TestGCRARateLimitStore_Postgres_NonUTCSession(t *testing.T) {
	db := newIsolatedPostgresDB(t)
	if err := RunMigrations(db, GetFrameworkMigrations()); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	assertGCRAStoreBehaviour(t, NewGCRARateLimitStore(db))
}

// Query GCRA tidak meng-cast placeholder waktunya, sehingga ia benar atas
// kolom TIMESTAMPTZ maupun TIMESTAMP lama (tabel yang dibuat aplikasi sendiri
// dan tidak dikonversi), di protokol extended maupun simple.
func TestGCRARateLimitStore_Postgres_ColumnTypesAndExecModes(t *testing.T) {
	for _, colType := range []string{"TIMESTAMPTZ", "TIMESTAMP"} {
		for _, mode := range []string{"", "simple"} {
			t.Run(colType+"/"+mode, func(t *testing.T) {
				db := newIsolatedPostgresDBWithMode(t, mode)
				ddl := "CREATE TABLE rate_limits (key VARCHAR(255) PRIMARY KEY, count INT NOT NULL DEFAULT 0, expires_at " + colType + " NOT NULL)"
				if err := db.Exec(context.Background(), ddl); err != nil {
					t.Fatalf("create: %v", err)
				}
				assertGCRAStoreBehaviour(t, NewGCRARateLimitStore(db))
			})
		}
	}
}

func TestDatabaseRateLimitStore_Postgres_NonUTCSession(t *testing.T) {
	db := newIsolatedPostgresDB(t)
	if err := RunMigrations(db, GetFrameworkMigrations()); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	store := NewDatabaseRateLimitStore(db)
	ctx := context.Background()

	for i := 1; i <= 2; i++ {
		if allowed, err := store.Allow(ctx, "k", 2, time.Hour); err != nil || !allowed {
			t.Fatalf("request %d: allowed=%v err=%v", i, allowed, err)
		}
	}
	allowed, retryAfter, err := store.AllowWithRetryAfter(ctx, "k", 2, time.Hour)
	if err != nil {
		t.Fatalf("request 3: %v", err)
	}
	if allowed {
		t.Error("request 3 harus ditolak")
	}
	if retryAfter <= 59*time.Minute || retryAfter > time.Hour {
		t.Errorf("retryAfter = %v, harus mendekati 1 jam", retryAfter)
	}
}
