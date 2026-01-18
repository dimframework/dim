package dim

import (
	"context"
	"fmt"
	"time"

	"github.com/atfromhome/goreus/pkg/cache"
)

// TokenBlocklist mendefinisikan interface untuk mekanisme pencabutan token (denylist).
// Digunakan untuk menyimpan identifier (JTI atau Session ID) dari token yang di-invalidate sebelum waktunya.
type TokenBlocklist interface {
	// Invalidate memasukkan token/session ke daftar hitam.
	// identifier: Unique identifier (bisa JTI atau SID).
	// expiresIn: Sisa durasi hidup token tersebut (sampai exp).
	Invalidate(ctx context.Context, identifier string, expiresIn time.Duration) error

	// IsRevoked mengecek apakah identifier ada di daftar hitam dan belum expired.
	IsRevoked(ctx context.Context, identifier string) (bool, error)
}

// --- InMemory Implementation ---

// InMemoryBlocklist implementasi TokenBlocklist menggunakan goreus/cache.
// Cocok untuk single-instance deployment atau testing.
// PERHATIAN: Data hilang saat restart.
type InMemoryBlocklist struct {
	tokens *cache.InMemoryCache[string, time.Time]
}

// NewInMemoryBlocklist membuat instance baru InMemoryBlocklist.
// Menggunakan kapasistas 100,000 dan default eviction 7 hari (aman untuk refresh token).
func NewInMemoryBlocklist() *InMemoryBlocklist {
	// 7 hari = 168 jam
	return &InMemoryBlocklist{
		tokens: cache.NewInMemoryCache[string, time.Time](100000, 168*time.Hour),
	}
}

func (m *InMemoryBlocklist) Invalidate(ctx context.Context, identifier string, expiresIn time.Duration) error {
	// Kita simpan waktu kedaluwarsa token di value cache untuk validasi manual
	// Meskipun goreus cache memiliki TTL sendiri (7 hari), token mungkin expire lebih cepat (misal 15 menit).
	m.tokens.Set(ctx, identifier, time.Now().Add(expiresIn))
	return nil
}

func (m *InMemoryBlocklist) IsRevoked(ctx context.Context, identifier string) (bool, error) {
	deadline, exists := m.tokens.Get(ctx, identifier)
	if !exists {
		return false, nil
	}

	// Jika token sebenarnya sudah expired (melewati expires_at di value),
	// maka secara teknis ia tidak perlu "di-blacklist" lagi karena sudah invalid by nature.
	if time.Now().After(deadline) {
		return false, nil
	}

	return true, nil
}

// --- PostgreSQL Implementation ---

// PostgresBlocklist implementasi TokenBlocklist menggunakan PostgreSQL UNLOGGED Table.
// Cocok untuk production & multi-instance deployment.
type PostgresBlocklist struct {
	db Database
}

// NewPostgresBlocklist membuat instance baru PostgresBlocklist.
func NewPostgresBlocklist(db Database) *PostgresBlocklist {
	return &PostgresBlocklist{db: db}
}

// InitSchema membuat tabel UNLOGGED 'token_blocklist' jika belum ada.
// UNLOGGED table tidak ditulis ke WAL (Write Ahead Log), performa lebih cepat (seperti Redis)
// dengan tradeoff data hilang jika DB crash (yang acceptable untuk cache/blocklist).
func (p *PostgresBlocklist) InitSchema(ctx context.Context) error {
	query := `
		CREATE UNLOGGED TABLE IF NOT EXISTS token_blocklist (
			identifier VARCHAR(255) PRIMARY KEY,
			expires_at TIMESTAMP NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_token_blocklist_expires_at ON token_blocklist(expires_at);
	`
	return p.db.Exec(ctx, query)
}

func (p *PostgresBlocklist) Invalidate(ctx context.Context, identifier string, expiresIn time.Duration) error {
	query := `INSERT INTO token_blocklist (identifier, expires_at) VALUES ($1, $2)`
	// Gunakan time.Now().UTC() untuk konsistensi
	return p.db.Exec(ctx, query, identifier, time.Now().UTC().Add(expiresIn))
}

func (p *PostgresBlocklist) IsRevoked(ctx context.Context, identifier string) (bool, error) {
	var exists bool
	// Cek existensi token yang masil valid (expires_at > NOW)
	query := `SELECT EXISTS(SELECT 1 FROM token_blocklist WHERE identifier = $1 AND expires_at > NOW())`
	err := p.db.QueryRow(ctx, query, identifier).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check blocklist: %w", err)
	}
	return exists, nil
}

// Cleanup menghapus token yang sudah expired dari database.
// Sebaiknya dijalankan sebagai cron job atau background task.
func (p *PostgresBlocklist) Cleanup(ctx context.Context) error {
	query := `DELETE FROM token_blocklist WHERE expires_at <= NOW()`
	return p.db.Exec(ctx, query)
}
