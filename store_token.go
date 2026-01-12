package dim

import (
	"context"
	"fmt"
	"time"
)

// RefreshToken represents a refresh token entity
type RefreshToken struct {
	ID        int64      `json:"id"`
	UserID    int64      `json:"user_id"`
	TokenHash string     `json:"-"`
	UserAgent string     `json:"user_agent"`
	IPAddress string     `json:"ip_address"`
	ExpiresAt time.Time  `json:"expires_at"`
	CreatedAt time.Time  `json:"created_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
}

// PasswordResetToken represents a password reset token entity
type PasswordResetToken struct {
	ID        int64      `json:"id"`
	UserID    int64      `json:"user_id"`
	TokenHash string     `json:"-"`
	ExpiresAt time.Time  `json:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// TokenStore defines the interface for token storage operations
type TokenStore interface {
	SaveRefreshToken(ctx context.Context, token *RefreshToken) error
	FindRefreshToken(ctx context.Context, tokenHash string) (*RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, tokenHash string) error
	RevokeAllUserTokens(ctx context.Context, userID int64) error

	SavePasswordResetToken(ctx context.Context, token *PasswordResetToken) error
	FindPasswordResetToken(ctx context.Context, tokenHash string) (*PasswordResetToken, error)
	MarkPasswordResetUsed(ctx context.Context, tokenHash string) error
}

// PostgresTokenStore is the PostgreSQL implementation of TokenStore
type PostgresTokenStore struct {
	db Database
}

// NewPostgresTokenStore membuat PostgreSQL token store baru.
// Store ini menangani operasi CRUD untuk refresh tokens dan password reset tokens.
//
// Parameters:
//   - db: Database instance untuk execute queries
//
// Returns:
//   - *PostgresTokenStore: token store instance
//
// Example:
//
//	tokenStore := NewPostgresTokenStore(db)
func NewPostgresTokenStore(db Database) *PostgresTokenStore {
	return &PostgresTokenStore{db: db}
}

// SaveRefreshToken menyimpan refresh token ke database.
// Token disimpan dengan hash, user_agent, ip_address, dan expiry time.
//
// Parameters:
//   - ctx: context untuk membatalkan operasi
//   - token: RefreshToken struct dengan data yang akan disimpan
//
// Returns:
//   - error: error jika INSERT query gagal
//
// Example:
//
//	err := tokenStore.SaveRefreshToken(ctx, &refreshToken)
func (s *PostgresTokenStore) SaveRefreshToken(ctx context.Context, token *RefreshToken) error {
	err := s.db.QueryRow(ctx,
		`INSERT INTO refresh_tokens (user_id, token_hash, user_agent, ip_address, expires_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, created_at`,
		token.UserID,
		token.TokenHash,
		token.UserAgent,
		token.IPAddress,
		token.ExpiresAt,
		time.Now(),
	).Scan(&token.ID, &token.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to save refresh token: %w", err)
	}

	return nil
}

// FindRefreshToken mencari refresh token berdasarkan hash.
//
// Parameters:
//   - ctx: context untuk membatalkan operasi
//   - tokenHash: hash dari token yang akan dicari
//
// Returns:
//   - *RefreshToken: RefreshToken struct jika ditemukan
//   - error: error jika token tidak ditemukan atau query gagal
//
// Example:
//
//	token, err := tokenStore.FindRefreshToken(ctx, tokenHash)
func (s *PostgresTokenStore) FindRefreshToken(ctx context.Context, tokenHash string) (*RefreshToken, error) {
	token := &RefreshToken{}
	err := s.db.QueryRow(ctx,
		`SELECT id, user_id, token_hash, user_agent, ip_address, expires_at, created_at, revoked_at
		 FROM refresh_tokens WHERE token_hash = $1`,
		tokenHash,
	).Scan(&token.ID, &token.UserID, &token.TokenHash, &token.UserAgent, &token.IPAddress, &token.ExpiresAt, &token.CreatedAt, &token.RevokedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to find refresh token: %w", err)
	}

	return token, nil
}

// RevokeRefreshToken membatalkan/revoke refresh token dengan set revoked_at timestamp.
//
// Parameters:
//   - ctx: context untuk membatalkan operasi
//   - tokenHash: hash dari token yang akan di-revoke
//
// Returns:
//   - error: error jika UPDATE query gagal
//
// Example:
//
//	err := tokenStore.RevokeRefreshToken(ctx, tokenHash)
func (s *PostgresTokenStore) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	err := s.db.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = $1 WHERE token_hash = $2`,
		time.Now(),
		tokenHash,
	)

	if err != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}

	return nil
}

// RevokeAllUserTokens membatalkan semua refresh tokens milik pengguna tertentu.
// Berguna untuk security setelah password reset atau logout dari semua device.
//
// Parameters:
//   - ctx: context untuk membatalkan operasi
//   - userID: ID dari user yang token-nya akan di-revoke
//
// Returns:
//   - error: error jika UPDATE query gagal
//
// Example:
//
//	err := tokenStore.RevokeAllUserTokens(ctx, userID)
func (s *PostgresTokenStore) RevokeAllUserTokens(ctx context.Context, userID int64) error {
	err := s.db.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = $1 WHERE user_id = $2 AND revoked_at IS NULL`,
		time.Now(),
		userID,
	)

	if err != nil {
		return fmt.Errorf("failed to revoke all user tokens: %w", err)
	}

	return nil
}

// SavePasswordResetToken menyimpan password reset token ke database.
// Token disimpan dengan hash dan expiry time untuk password reset flow.
//
// Parameters:
//   - ctx: context untuk membatalkan operasi
//   - token: PasswordResetToken struct dengan data yang akan disimpan
//
// Returns:
//   - error: error jika INSERT query gagal
//
// Example:
//
//	err := tokenStore.SavePasswordResetToken(ctx, &resetToken)
func (s *PostgresTokenStore) SavePasswordResetToken(ctx context.Context, token *PasswordResetToken) error {
	err := s.db.QueryRow(ctx,
		`INSERT INTO password_reset_tokens (user_id, token_hash, expires_at, created_at)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, created_at`,
		token.UserID,
		token.TokenHash,
		token.ExpiresAt,
		time.Now(),
	).Scan(&token.ID, &token.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to save password reset token: %w", err)
	}

	return nil
}

// FindPasswordResetToken mencari password reset token berdasarkan hash.
//
// Parameters:
//   - ctx: context untuk membatalkan operasi
//   - tokenHash: hash dari reset token yang akan dicari
//
// Returns:
//   - *PasswordResetToken: PasswordResetToken struct jika ditemukan
//   - error: error jika token tidak ditemukan atau query gagal
//
// Example:
//
//	token, err := tokenStore.FindPasswordResetToken(ctx, tokenHash)
func (s *PostgresTokenStore) FindPasswordResetToken(ctx context.Context, tokenHash string) (*PasswordResetToken, error) {
	token := &PasswordResetToken{}
	err := s.db.QueryRow(ctx,
		`SELECT id, user_id, token_hash, expires_at, created_at, used_at
		 FROM password_reset_tokens WHERE token_hash = $1`,
		tokenHash,
	).Scan(&token.ID, &token.UserID, &token.TokenHash, &token.ExpiresAt, &token.CreatedAt, &token.UsedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to find password reset token: %w", err)
	}

	return token, nil
}

// MarkPasswordResetUsed menandai password reset token sudah digunakan dengan set used_at timestamp.
// Mencegah reuse dari token yang sama untuk reset password.
//
// Parameters:
//   - ctx: context untuk membatalkan operasi
//   - tokenHash: hash dari reset token yang akan ditandai sebagai used
//
// Returns:
//   - error: error jika UPDATE query gagal
//
// Example:
//
//	err := tokenStore.MarkPasswordResetUsed(ctx, tokenHash)
func (s *PostgresTokenStore) MarkPasswordResetUsed(ctx context.Context, tokenHash string) error {
	err := s.db.Exec(ctx,
		`UPDATE password_reset_tokens SET used_at = $1 WHERE token_hash = $2`,
		time.Now(),
		tokenHash,
	)

	if err != nil {
		return fmt.Errorf("failed to mark password reset token as used: %w", err)
	}

	return nil
}

// MockTokenStore is a mock implementation for testing
type MockTokenStore struct {
	refreshTokens map[string]*RefreshToken
	resetTokens   map[string]*PasswordResetToken
}

// NewMockTokenStore membuat mock token store untuk testing.
// Mock store menyimpan tokens dalam memory dan cocok untuk unit tests.
//
// Returns:
//   - *MockTokenStore: mock store instance dengan empty token maps
//
// Example:
//
//	mockStore := NewMockTokenStore()
//	// use in tests
func NewMockTokenStore() *MockTokenStore {
	return &MockTokenStore{
		refreshTokens: make(map[string]*RefreshToken),
		resetTokens:   make(map[string]*PasswordResetToken),
	}
}

// SaveRefreshToken menyimpan refresh token dalam mock store (memory).
//
// Parameters:
//   - ctx: context (tidak digunakan dalam mock)
//   - token: RefreshToken struct yang akan disimpan
//
// Returns:
//   - error: selalu nil untuk mock
//
// Example:
//
//	err := mockStore.SaveRefreshToken(ctx, &token)
func (s *MockTokenStore) SaveRefreshToken(ctx context.Context, token *RefreshToken) error {
	token.ID = int64(len(s.refreshTokens) + 1)
	token.CreatedAt = time.Now()
	s.refreshTokens[token.TokenHash] = token
	return nil
}

// FindRefreshToken mencari refresh token dalam mock store.
//
// Parameters:
//   - ctx: context (tidak digunakan dalam mock)
//   - tokenHash: hash dari token yang akan dicari
//
// Returns:
//   - *RefreshToken: token jika ditemukan, nil jika tidak
//   - error: error message jika token tidak ditemukan
//
// Example:
//
//	token, err := mockStore.FindRefreshToken(ctx, tokenHash)
func (s *MockTokenStore) FindRefreshToken(ctx context.Context, tokenHash string) (*RefreshToken, error) {
	token, exists := s.refreshTokens[tokenHash]
	if !exists {
		return nil, fmt.Errorf("refresh token not found")
	}
	return token, nil
}

// RevokeRefreshToken membatalkan refresh token dalam mock store.
//
// Parameters:
//   - ctx: context (tidak digunakan dalam mock)
//   - tokenHash: hash dari token yang akan di-revoke
//
// Returns:
//   - error: selalu nil untuk mock
//
// Example:
//
//	err := mockStore.RevokeRefreshToken(ctx, tokenHash)
func (s *MockTokenStore) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	if token, exists := s.refreshTokens[tokenHash]; exists {
		now := time.Now()
		token.RevokedAt = &now
	}
	return nil
}

// RevokeAllUserTokens membatalkan semua tokens user dalam mock store.
//
// Parameters:
//   - ctx: context (tidak digunakan dalam mock)
//   - userID: ID dari user yang semua token-nya akan di-revoke
//
// Returns:
//   - error: selalu nil untuk mock
//
// Example:
//
//	err := mockStore.RevokeAllUserTokens(ctx, userID)
func (s *MockTokenStore) RevokeAllUserTokens(ctx context.Context, userID int64) error {
	now := time.Now()
	for _, token := range s.refreshTokens {
		if token.UserID == userID && token.RevokedAt == nil {
			token.RevokedAt = &now
		}
	}
	return nil
}

// SavePasswordResetToken menyimpan password reset token dalam mock store.
//
// Parameters:
//   - ctx: context (tidak digunakan dalam mock)
//   - token: PasswordResetToken struct yang akan disimpan
//
// Returns:
//   - error: selalu nil untuk mock
//
// Example:
//
//	err := mockStore.SavePasswordResetToken(ctx, &token)
func (s *MockTokenStore) SavePasswordResetToken(ctx context.Context, token *PasswordResetToken) error {
	token.ID = int64(len(s.resetTokens) + 1)
	token.CreatedAt = time.Now()
	s.resetTokens[token.TokenHash] = token
	return nil
}

// FindPasswordResetToken mencari password reset token dalam mock store.
//
// Parameters:
//   - ctx: context (tidak digunakan dalam mock)
//   - tokenHash: hash dari reset token yang akan dicari
//
// Returns:
//   - *PasswordResetToken: token jika ditemukan, nil jika tidak
//   - error: error message jika token tidak ditemukan
//
// Example:
//
//	token, err := mockStore.FindPasswordResetToken(ctx, tokenHash)
func (s *MockTokenStore) FindPasswordResetToken(ctx context.Context, tokenHash string) (*PasswordResetToken, error) {
	token, exists := s.resetTokens[tokenHash]
	if !exists {
		return nil, fmt.Errorf("password reset token not found")
	}
	return token, nil
}

// MarkPasswordResetUsed menandai password reset token sebagai used dalam mock store.
//
// Parameters:
//   - ctx: context (tidak digunakan dalam mock)
//   - tokenHash: hash dari reset token yang akan ditandai as used
//
// Returns:
//   - error: selalu nil untuk mock
//
// Example:
//
//	err := mockStore.MarkPasswordResetUsed(ctx, tokenHash)
func (s *MockTokenStore) MarkPasswordResetUsed(ctx context.Context, tokenHash string) error {
	if token, exists := s.resetTokens[tokenHash]; exists {
		now := time.Now()
		token.UsedAt = &now
	}
	return nil
}
