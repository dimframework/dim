package dim

import (
	"context"
	"testing"
	"time"
)

func TestPostgresDatabase_Logic(t *testing.T) {
	// Kita buat instance kosong hanya untuk test logic Rebind dan DriverName
	db := &PostgresDatabase{}

	if db.DriverName() != "postgres" {
		t.Errorf("Expected driver name 'postgres', got %s", db.DriverName())
	}

	query := "SELECT * FROM users WHERE id = $1 AND email = $2"
	rebound := db.Rebind(query)
	if rebound != query {
		t.Errorf("Postgres Rebind should be no-op. Got %s", rebound)
	}
}

func TestSQLiteDatabase_Logic(t *testing.T) {
	db := &SQLiteDatabase{}

	if db.DriverName() != "sqlite" {
		t.Errorf("Expected driver name 'sqlite', got %s", db.DriverName())
	}

	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "SELECT * FROM users WHERE id = $1",
			expected: "SELECT * FROM users WHERE id = ?",
		},
		{
			input:    "INSERT INTO x (a, b) VALUES ($1, $2) RETURNING id",
			expected: "INSERT INTO x (a, b) VALUES (?, ?) RETURNING id",
		},
		{
			input:    "UPDATE x SET a = $1, b = $2 WHERE c = $3",
			expected: "UPDATE x SET a = ?, b = ? WHERE c = ?",
		},
	}

	for _, tt := range tests {
		rebound := db.Rebind(tt.input)
		if rebound != tt.expected {
			t.Errorf("SQLite Rebind failed.\nInput: %s\nGot:   %s\nWant:  %s", tt.input, rebound, tt.expected)
		}
	}
}

func TestSQLiteDatabase_InMemory(t *testing.T) {
	// Setup in-memory SQLite
	config := DatabaseConfig{
		Driver:   "sqlite",
		Database: ":memory:",
	}

	db, err := NewSQLiteDatabase(config)
	if err != nil {
		t.Fatalf("Failed to create in-memory sqlite: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Test basic execution
	err = db.Exec(ctx, "CREATE TABLE test (id INTEGER PRIMARY KEY, name TEXT)")
	if err != nil {
		t.Fatalf("Exec failed: %v", err)
	}

	// Test Insert & QueryRow
	err = db.Exec(ctx, "INSERT INTO test (name) VALUES ($1)", "dim")
	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	var name string
	err = db.QueryRow(ctx, db.Rebind("SELECT name FROM test WHERE id = $1"), 1).Scan(&name)
	if err != nil {
		t.Fatalf("QueryRow failed: %v", err)
	}
	if name != "dim" {
		t.Errorf("Expected name 'dim', got %s", name)
	}

	// Test Transaction
	err = db.WithTx(ctx, func(ctx context.Context, tx Tx) error {
		return tx.Exec(ctx, "INSERT INTO test (name) VALUES ($1)", "tx")
	})
	if err != nil {
		t.Fatalf("WithTx failed: %v", err)
	}

	err = db.QueryRow(ctx, "SELECT COUNT(*) FROM test").Scan(&name) // reuse name var for count
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if name != "2" {
		t.Errorf("Expected count 2, got %s", name)
	}
}

func TestDatabaseTokenStore_SQLite(t *testing.T) {
	db, _ := NewSQLiteDatabase(DatabaseConfig{Database: ":memory:"})
	defer db.Close()

	// Run migrations needed for TokenStore
	migrations := append(GetUserMigrations(), GetTokenMigrations()...)
	err := RunMigrations(db, migrations)
	if err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	store := NewDatabaseTokenStore(db)
	ctx := context.Background()


token := &RefreshToken{
		UserID:    "550e8400-e29b-41d4-a716-446655440000",
		TokenHash: "hash123",
		UserAgent: "test-agent",
		IPAddress: "127.0.0.1",
		ExpiresAt: time.Now().Add(time.Hour),
	}

	// Test Save (Verifies RETURNING clause and Rebind)
	err = store.SaveRefreshToken(ctx, token)
	if err != nil {
		t.Fatalf("SaveRefreshToken failed: %v", err)
	}
	if token.ID == 0 {
		t.Error("Expected token ID to be set by RETURNING clause")
	}

	// Test Find
	found, err := store.FindRefreshToken(ctx, "hash123")
	if err != nil {
		t.Fatalf("FindRefreshToken failed: %v", err)
	}
	if found.TokenHash != "hash123" {
		t.Error("Token hash mismatch")
	}

	// Test Revoke
	err = store.RevokeRefreshToken(ctx, "hash123")
	if err != nil {
		t.Fatalf("RevokeRefreshToken failed: %v", err)
	}
	found, _ = store.FindRefreshToken(ctx, "hash123")
	if found.RevokedAt == nil {
		t.Error("Expected RevokedAt to be set")
	}
}

func TestDatabaseRateLimitStore_SQLite(t *testing.T) {
	db, _ := NewSQLiteDatabase(DatabaseConfig{Database: ":memory:"})
	defer db.Close()

	store := NewDatabaseRateLimitStore(db)
	ctx := context.Background()

	err := store.InitSchema(ctx)
	if err != nil {
		t.Fatalf("InitSchema failed: %v", err)
	}

	// Test Allow (Verifies ON CONFLICT and Rebind)
	window := time.Hour
	allowed, err := store.Allow(ctx, "test-ip", 2, window)
	if err != nil {
		t.Fatalf("Allow 1 failed: %v", err)
	}
	if !allowed {
		t.Error("Expected first request to be allowed")
	}

	allowed, err = store.Allow(ctx, "test-ip", 2, window)
	if err != nil {
		t.Fatalf("Allow 2 failed: %v", err)
	}
	if !allowed {
		t.Error("Expected second request to be allowed")
	}

	allowed, err = store.Allow(ctx, "test-ip", 2, window)
	if err != nil {
		t.Fatalf("Allow 3 failed: %v", err)
	}
	if allowed {
		t.Error("Expected third request to be blocked")
	}
}
