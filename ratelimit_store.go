package dim

import (
	"context"
	"sync"
	"time"

	"github.com/atfromhome/goreus/pkg/cache"
)

// RateLimitStore mendefinisikan interface untuk backend penyimpanan rate limit.
// Memungkinkan pergantian antara InMemory (single instance) dan Postgres (multi-instance).
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

// --- PostgreSQL Implementation ---

// PostgresRateLimitStore mengimplementasikan RateLimitStore menggunakan PostgreSQL.
// Cocok untuk deployment multi-instance/cluster.
// Menggunakan tabel UNLOGGED untuk performa tinggi (data hilang saat crash dapat diterima untuk rate limits).
type PostgresRateLimitStore struct {
	db Database
}

// NewPostgresRateLimitStore membuat store rate limit PostgreSQL baru.
//
// Parameters:
//   - db: koneksi database yang mengimplementasikan interface Database
func NewPostgresRateLimitStore(db Database) *PostgresRateLimitStore {
	return &PostgresRateLimitStore{db: db}
}

// InitSchema membuat tabel yang diperlukan untuk rate limiting.
// Sebaiknya dipanggil saat startup aplikasi atau migrasi.
func (s *PostgresRateLimitStore) InitSchema(ctx context.Context) error {
	query := `
		CREATE UNLOGGED TABLE IF NOT EXISTS rate_limits (
			key VARCHAR(255) PRIMARY KEY,
			count INT NOT NULL DEFAULT 0,
			expires_at TIMESTAMP NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_rate_limits_expires_at ON rate_limits(expires_at);
	`
	return s.db.Exec(ctx, query)
}

// Allow mengecek dan menaikkan limit menggunakan Atomic UPSERT di PostgreSQL.
func (s *PostgresRateLimitStore) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	now := time.Now()
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
			WHEN rate_limits.expires_at < $3 THEN $2
			ELSE rate_limits.expires_at
		END
		RETURNING count
	`

	var count int
	// $1=key, $2=expiresAt, $3=now
	err := s.db.QueryRow(ctx, query, key, expiresAt, now).Scan(&count)
	if err != nil {
		return false, err
	}

	return count <= limit, nil
}

// Close menutup koneksi (no-op untuk implementasi ini karena DB dikelola di luar).
func (s *PostgresRateLimitStore) Close() error {
	return nil
}