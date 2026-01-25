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
	m.tokens.Set(ctx, identifier, time.Now().UTC().Add(expiresIn))
	return nil
}

func (m *InMemoryBlocklist) IsRevoked(ctx context.Context, identifier string) (bool, error) {
	deadline, exists := m.tokens.Get(ctx, identifier)
	if !exists {
		return false, nil
	}

	// Jika token sebenarnya sudah expired (melewati expires_at di value),
	// maka secara teknis ia tidak perlu "di-blacklist" lagi karena sudah invalid by nature.
	if time.Now().UTC().After(deadline) {
		return false, nil
	}

	return true, nil
}

// --- Database Implementation ---

// DatabaseBlocklist implementasi TokenBlocklist menggunakan database SQL.
// Cocok untuk production & multi-instance deployment.
type DatabaseBlocklist struct {
	db Database
}

// NewDatabaseBlocklist membuat instance baru DatabaseBlocklist.
func NewDatabaseBlocklist(db Database) *DatabaseBlocklist {
	return &DatabaseBlocklist{db: db}
}

// Deprecated: Use NewDatabaseBlocklist instead
func NewPostgresBlocklist(db Database) *DatabaseBlocklist {
	return NewDatabaseBlocklist(db)
}

// InitSchema membuat tabel 'token_blocklist' jika belum ada.
func (p *DatabaseBlocklist) InitSchema(ctx context.Context) error {
	var query string
	if p.db.DriverName() == "sqlite" {
		query = `
			CREATE TABLE IF NOT EXISTS token_blocklist (
				identifier TEXT PRIMARY KEY,
				expires_at TIMESTAMP NOT NULL
			);
			CREATE INDEX IF NOT EXISTS idx_token_blocklist_expires_at ON token_blocklist(expires_at);
		`
	} else {
		query = `
			CREATE UNLOGGED TABLE IF NOT EXISTS token_blocklist (
				identifier VARCHAR(255) PRIMARY KEY,
				expires_at TIMESTAMP NOT NULL
			);
			CREATE INDEX IF NOT EXISTS idx_token_blocklist_expires_at ON token_blocklist(expires_at);
		`
	}
	return p.db.Exec(ctx, query)
}

func (p *DatabaseBlocklist) Invalidate(ctx context.Context, identifier string, expiresIn time.Duration) error {
	query := p.db.Rebind(`INSERT INTO token_blocklist (identifier, expires_at) VALUES ($1, $2)`)
	return p.db.Exec(ctx, query, identifier, time.Now().UTC().Add(expiresIn).Truncate(time.Second))
}

func (p *DatabaseBlocklist) IsRevoked(ctx context.Context, identifier string) (bool, error) {
	var exists bool
	var query string
	if p.db.DriverName() == "sqlite" {
		query = p.db.Rebind(`SELECT EXISTS(SELECT 1 FROM token_blocklist WHERE identifier = $1 AND expires_at > CURRENT_TIMESTAMP)`)
	} else {
		query = p.db.Rebind(`SELECT EXISTS(SELECT 1 FROM token_blocklist WHERE identifier = $1 AND expires_at > NOW())`)
	}
	
	err := p.db.QueryRow(ctx, query, identifier).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check blocklist: %w", err)
	}
	return exists, nil
}

// Cleanup menghapus token yang sudah expired dari database.
func (p *DatabaseBlocklist) Cleanup(ctx context.Context) error {
	var query string
	if p.db.DriverName() == "sqlite" {
		query = `DELETE FROM token_blocklist WHERE expires_at <= CURRENT_TIMESTAMP`
	} else {
		query = `DELETE FROM token_blocklist WHERE expires_at <= NOW()`
	}
	return p.db.Exec(ctx, query)
}