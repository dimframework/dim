package dim

import (
	"context"
	"fmt"
	"time"
)

// RefreshToken represents a refresh token entity
type RefreshToken struct {
	ID        int64      `json:"id"`
	UserID    string     `json:"user_id"`
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
	UserID    string     `json:"user_id"`
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
	RevokeAllUserTokens(ctx context.Context, userID string) error

	SavePasswordResetToken(ctx context.Context, token *PasswordResetToken) error
	FindPasswordResetToken(ctx context.Context, tokenHash string) (*PasswordResetToken, error)
	MarkPasswordResetUsed(ctx context.Context, tokenHash string) error
}

// DatabaseTokenStore is the SQL implementation of TokenStore (PostgreSQL & SQLite)
type DatabaseTokenStore struct {
	db Database
}

// NewDatabaseTokenStore creates a new SQL token store.
// Handles CRUD operations for refresh tokens and password reset tokens.
func NewDatabaseTokenStore(db Database) *DatabaseTokenStore {
	return &DatabaseTokenStore{db: db}
}

// Deprecated: Use NewDatabaseTokenStore instead
func NewPostgresTokenStore(db Database) *DatabaseTokenStore {
	return NewDatabaseTokenStore(db)
}

// SaveRefreshToken saves a refresh token to the database.
func (s *DatabaseTokenStore) SaveRefreshToken(ctx context.Context, token *RefreshToken) error {
	now := time.Now().UTC().Truncate(time.Second)
	query := `INSERT INTO refresh_tokens (user_id, token_hash, user_agent, ip_address, expires_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, created_at`
	
	err := s.db.QueryRow(ctx, s.db.Rebind(query),
		token.UserID,
		token.TokenHash,
		token.UserAgent,
		token.IPAddress,
		token.ExpiresAt.UTC().Truncate(time.Second),
		now,
	).Scan(&token.ID, &token.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to save refresh token: %w", err)
	}

	return nil
}

// FindRefreshToken finds a refresh token by hash.
func (s *DatabaseTokenStore) FindRefreshToken(ctx context.Context, tokenHash string) (*RefreshToken, error) {
	token := &RefreshToken{}
	query := `SELECT id, user_id, token_hash, user_agent, ip_address, expires_at, created_at, revoked_at
		 FROM refresh_tokens WHERE token_hash = $1`
	
	err := s.db.QueryRow(ctx, s.db.Rebind(query), tokenHash).Scan(
		&token.ID, &token.UserID, &token.TokenHash, &token.UserAgent, &token.IPAddress, 
		&token.ExpiresAt, &token.CreatedAt, &token.RevokedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to find refresh token: %w", err)
	}

	return token, nil
}

// RevokeRefreshToken revokes a refresh token by setting revoked_at timestamp.
func (s *DatabaseTokenStore) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	query := `UPDATE refresh_tokens SET revoked_at = $1 WHERE token_hash = $2`
	
	err := s.db.Exec(ctx, s.db.Rebind(query), time.Now().UTC().Truncate(time.Second), tokenHash)

	if err != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}

	return nil
}

// RevokeAllUserTokens revokes all refresh tokens for a specific user.
func (s *DatabaseTokenStore) RevokeAllUserTokens(ctx context.Context, userID string) error {
	query := `UPDATE refresh_tokens SET revoked_at = $1 WHERE user_id = $2 AND revoked_at IS NULL`
	
	err := s.db.Exec(ctx, s.db.Rebind(query), time.Now().UTC().Truncate(time.Second), userID)

	if err != nil {
		return fmt.Errorf("failed to revoke all user tokens: %w", err)
	}

	return nil
}

// SavePasswordResetToken saves a password reset token to the database.
func (s *DatabaseTokenStore) SavePasswordResetToken(ctx context.Context, token *PasswordResetToken) error {
	now := time.Now().UTC().Truncate(time.Second)
	query := `INSERT INTO password_reset_tokens (user_id, token_hash, expires_at, created_at)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, created_at`
	
	err := s.db.QueryRow(ctx, s.db.Rebind(query),
		token.UserID,
		token.TokenHash,
		token.ExpiresAt.UTC().Truncate(time.Second),
		now,
	).Scan(&token.ID, &token.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to save password reset token: %w", err)
	}

	return nil
}

// FindPasswordResetToken finds a password reset token by hash.
func (s *DatabaseTokenStore) FindPasswordResetToken(ctx context.Context, tokenHash string) (*PasswordResetToken, error) {
	token := &PasswordResetToken{}
	query := `SELECT id, user_id, token_hash, expires_at, created_at, used_at
		 FROM password_reset_tokens WHERE token_hash = $1`
	
	err := s.db.QueryRow(ctx, s.db.Rebind(query), tokenHash).Scan(
		&token.ID, &token.UserID, &token.TokenHash, &token.ExpiresAt, &token.CreatedAt, &token.UsedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to find password reset token: %w", err)
	}

	return token, nil
}

// MarkPasswordResetUsed marks a password reset token as used.
func (s *DatabaseTokenStore) MarkPasswordResetUsed(ctx context.Context, tokenHash string) error {
	query := `UPDATE password_reset_tokens SET used_at = $1 WHERE token_hash = $2`
	
	err := s.db.Exec(ctx, s.db.Rebind(query), time.Now().UTC().Truncate(time.Second), tokenHash)

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

// NewMockTokenStore creates a new mock token store.
func NewMockTokenStore() *MockTokenStore {
	return &MockTokenStore{
		refreshTokens: make(map[string]*RefreshToken),
		resetTokens:   make(map[string]*PasswordResetToken),
	}
}

// SaveRefreshToken saves a refresh token in mock store.
func (s *MockTokenStore) SaveRefreshToken(ctx context.Context, token *RefreshToken) error {
	token.ID = int64(len(s.refreshTokens) + 1)
	token.CreatedAt = time.Now()
	s.refreshTokens[token.TokenHash] = token
	return nil
}

// FindRefreshToken finds a refresh token in mock store.
func (s *MockTokenStore) FindRefreshToken(ctx context.Context, tokenHash string) (*RefreshToken, error) {
	token, exists := s.refreshTokens[tokenHash]
	if !exists {
		return nil, fmt.Errorf("refresh token not found")
	}
	return token, nil
}

// RevokeRefreshToken revokes a refresh token in mock store.
func (s *MockTokenStore) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	if token, exists := s.refreshTokens[tokenHash]; exists {
		now := time.Now()
		token.RevokedAt = &now
	}
	return nil
}

// RevokeAllUserTokens revokes all user tokens in mock store.
func (s *MockTokenStore) RevokeAllUserTokens(ctx context.Context, userID string) error {
	now := time.Now()
	for _, token := range s.refreshTokens {
		if token.UserID == userID && token.RevokedAt == nil {
			token.RevokedAt = &now
		}
	}
	return nil
}

// SavePasswordResetToken saves a password reset token in mock store.
func (s *MockTokenStore) SavePasswordResetToken(ctx context.Context, token *PasswordResetToken) error {
	token.ID = int64(len(s.resetTokens) + 1)
	token.CreatedAt = time.Now()
	s.resetTokens[token.TokenHash] = token
	return nil
}

// FindPasswordResetToken finds a password reset token in mock store.
func (s *MockTokenStore) FindPasswordResetToken(ctx context.Context, tokenHash string) (*PasswordResetToken, error) {
	token, exists := s.resetTokens[tokenHash]
	if !exists {
		return nil, fmt.Errorf("password reset token not found")
	}
	return token, nil
}

// MarkPasswordResetUsed marks a password reset token as used in mock store.
func (s *MockTokenStore) MarkPasswordResetUsed(ctx context.Context, tokenHash string) error {
	if token, exists := s.resetTokens[tokenHash]; exists {
		now := time.Now()
		token.UsedAt = &now
	}
	return nil
}
