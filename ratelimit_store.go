package dim

import (
	"context"
	"sync"
	"time"

	"github.com/atfromhome/goreus/pkg/cache"
)

// RateLimitStore mendefinisikan interface untuk backend penyimpanan rate limit.
// Memungkinkan pergantian antara InMemory (single instance) dan Database (multi-instance).
type RateLimitStore interface {
	// Allow mengecek apakah request diizinkan dan menaikkan counter.
	// Mengembalikan true jika diizinkan, false jika tidak.
	//
	// Parameters:
	//   - ctx: context untuk operasi
	//   - key: unique key untuk rate limit (misal: "ip:1.2.3.4")
	//   - limit: batas maksimum request
	//   - window: durasi waktu reset
	Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error)

	// Close membersihkan resource yang digunakan.
	Close() error
}

// --- InMemory Implementation ---

// InMemoryRateLimitStore mengimplementasikan RateLimitStore menggunakan goreus/cache.
// Cocok untuk deployment single-instance. Data counter disimpan di memori dan hilang saat restart.
type InMemoryRateLimitStore struct {
	cache *cache.InMemoryCache[string, int]
	mu    sync.Mutex
}

// NewInMemoryRateLimitStore membuat store rate limit in-memory baru.
//
// Parameters:
//   - window: durasi waktu untuk TTL cache (biasanya sama dengan ResetPeriod)
func NewInMemoryRateLimitStore(window time.Duration) *InMemoryRateLimitStore {
	return &InMemoryRateLimitStore{
		cache: cache.NewInMemoryCache[string, int](10000, window),
	}
}

// Allow mengecek dan menaikkan limit di in-memory cache.
func (s *InMemoryRateLimitStore) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	count, exists := s.cache.Get(ctx, key)
	if !exists {
		count = 0
	}

	count++
	s.cache.Set(ctx, key, count)

	return count <= limit, nil
}

// Close menutup cache in-memory.
func (s *InMemoryRateLimitStore) Close() error {
	s.cache.Close()
	return nil
}

// --- Database Implementation (PostgreSQL & SQLite) ---

// DatabaseRateLimitStore mengimplementasikan RateLimitStore menggunakan database SQL.
// Cocok untuk deployment multi-instance/cluster.
// Menggunakan tabel UNLOGGED untuk performa tinggi di PostgreSQL.
type DatabaseRateLimitStore struct {
	db Database
}

// NewDatabaseRateLimitStore membuat store rate limit database baru.
//
// Parameters:
//   - db: koneksi database yang mengimplementasikan interface Database
func NewDatabaseRateLimitStore(db Database) *DatabaseRateLimitStore {
	return &DatabaseRateLimitStore{db: db}
}

// Deprecated: Use NewDatabaseRateLimitStore instead
func NewPostgresRateLimitStore(db Database) *DatabaseRateLimitStore {
	return NewDatabaseRateLimitStore(db)
}

// InitSchema membuat tabel yang diperlukan untuk rate limiting.
// Sebaiknya dipanggil saat startup aplikasi atau migrasi.
func (s *DatabaseRateLimitStore) InitSchema(ctx context.Context) error {
	var query string
	if s.db.DriverName() == "sqlite" {
		query = `
			CREATE TABLE IF NOT EXISTS rate_limits (
				key TEXT PRIMARY KEY,
				count INT NOT NULL DEFAULT 0,
				expires_at TIMESTAMP NOT NULL
			);
			CREATE INDEX IF NOT EXISTS idx_rate_limits_expires_at ON rate_limits(expires_at);
		`
	} else {
		query = `
			CREATE UNLOGGED TABLE IF NOT EXISTS rate_limits (
				key VARCHAR(255) PRIMARY KEY,
				count INT NOT NULL DEFAULT 0,
				expires_at TIMESTAMP NOT NULL
			);
			CREATE INDEX IF NOT EXISTS idx_rate_limits_expires_at ON rate_limits(expires_at);
		`
	}
	return s.db.Exec(ctx, query)
}

// Allow mengecek dan menaikkan limit menggunakan Atomic UPSERT.
func (s *DatabaseRateLimitStore) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	now := time.Now().UTC().Truncate(time.Second)
	expiresAt := now.Add(window)

	// Atomic UPSERT (Insert or Update) dengan logika sliding window.
	// Jika record ada tapi expired (expires_at < now), reset count ke 1 dan update expires_at.
	// Jika record ada dan valid, increment count.
	// Jika record baru, insert dengan count 1.
	query := `
		INSERT INTO rate_limits (key, count, expires_at)
		VALUES ($1, 1, $2)
		ON CONFLICT (key) DO UPDATE
		SET count = CASE
			WHEN rate_limits.expires_at < $3 THEN 1
			ELSE rate_limits.count + 1
		END,
		expires_at = CASE
			WHEN rate_limits.expires_at < $3 THEN $4
			ELSE rate_limits.expires_at
		END
		RETURNING count
	`

	var count int
	// Placeholders: $1=key, $2=expiresAt, $3=now, $4=expiresAt, $5=now
	// Note: We repeat args because database/sql (SQLite) doesn't always support named positional reuse like pgx.
	query = s.db.Rebind(query)
	err := s.db.QueryRow(ctx, query, key, expiresAt, now, expiresAt, now).Scan(&count)
	if err != nil {
		return false, err
	}

	return count <= limit, nil
}

// Close menutup koneksi (no-op untuk implementasi ini karena DB dikelola di luar).
func (s *DatabaseRateLimitStore) Close() error {
	return nil
}
