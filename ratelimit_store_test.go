package dim

import (
	"context"
	"testing"
	"time"
)

// Regresi: query Allow pernah mengirim 5 argumen untuk 4 placeholder, sehingga pgx
// selalu menolaknya dengan "expected 4 arguments, got 5". Karena RateLimit fail open,
// kegagalan itu membuat seluruh rate limit berbasis database tidak berefek.
func TestDatabaseRateLimitStore_Postgres(t *testing.T) {
	db := newTestPostgresDB(t)
	store := NewDatabaseRateLimitStore(db)
	ctx := context.Background()

	if err := store.InitSchema(ctx); err != nil {
		t.Fatalf("InitSchema: %v", err)
	}

	key := "test-ip-postgres"
	if err := db.Exec(ctx, "DELETE FROM rate_limits WHERE key = $1", key); err != nil {
		t.Fatalf("bersihkan sisa data: %v", err)
	}

	window := time.Hour
	for i := 1; i <= 2; i++ {
		allowed, err := store.Allow(ctx, key, 2, window)
		if err != nil {
			t.Fatalf("Allow %d: %v", i, err)
		}
		if !allowed {
			t.Errorf("request %d seharusnya diizinkan", i)
		}
	}

	allowed, err := store.Allow(ctx, key, 2, window)
	if err != nil {
		t.Fatalf("Allow 3: %v", err)
	}
	if allowed {
		t.Error("request ke-3 seharusnya diblokir (limit 2)")
	}
}

// Regresi: SQLiteDatabase.Rebind mengganti setiap "$N" dengan "?" secara posisional,
// sehingga placeholder yang dipakai ulang menggeser pemetaan argumen tanpa error.
// Akibatnya expires_at ditimpa dengan `now` (sudah lewat) dan counter ter-reset di
// setiap request berikutnya.
//
// Test SQLite yang lama tidak menangkap ini karena seluruh panggilannya terjadi dalam
// satu detik yang sama, sedangkan `now` di-truncate ke detik. Test ini sengaja
// melewati batas detik.
func TestDatabaseRateLimitStore_SQLiteAcrossSeconds(t *testing.T) {
	db, err := NewSQLiteDatabase(DatabaseConfig{Database: ":memory:"})
	if err != nil {
		t.Fatalf("NewSQLiteDatabase: %v", err)
	}
	defer db.Close()

	store := NewDatabaseRateLimitStore(db)
	ctx := context.Background()
	if err := store.InitSchema(ctx); err != nil {
		t.Fatalf("InitSchema: %v", err)
	}

	// Window jauh lebih panjang daripada durasi test, jadi tidak boleh ada reset.
	window := time.Hour
	const limit = 2

	for i := 1; i <= 3; i++ {
		allowed, err := store.Allow(ctx, "test-ip", limit, window)
		if err != nil {
			t.Fatalf("Allow %d: %v", i, err)
		}

		wantAllowed := i <= limit
		if allowed != wantAllowed {
			t.Errorf("request %d: allowed = %v, want %v", i, allowed, wantAllowed)
		}

		// expires_at harus tetap di masa depan; bug lama menimpanya dengan `now`.
		var expiresAt time.Time
		q := db.Rebind("SELECT expires_at FROM rate_limits WHERE key = $1")
		if err := db.QueryRow(ctx, q, "test-ip").Scan(&expiresAt); err != nil {
			t.Fatalf("baca expires_at: %v", err)
		}
		if !expiresAt.After(time.Now().UTC()) {
			t.Errorf("request %d: expires_at = %v sudah lewat — window ter-reset", i, expiresAt)
		}

		if i < 3 {
			time.Sleep(1100 * time.Millisecond) // lewati batas detik
		}
	}
}
