package dim

import (
	"context"
	"testing"
	"time"
)

func TestMockTokenStoreSaveRefreshToken(t *testing.T) {
	store := NewMockTokenStore()
	ctx := context.Background()

	token := &RefreshToken{
		UserID:    1,
		TokenHash: "hash123",
		UserAgent: "Mozilla/5.0",
		IPAddress: "192.168.1.1",
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	err := store.SaveRefreshToken(ctx, token)
	if err != nil {
		t.Errorf("SaveRefreshToken() error = %v", err)
	}

	if token.ID != 1 {
		t.Errorf("token ID = %d, want 1", token.ID)
	}
}

func TestMockTokenStoreFindRefreshToken(t *testing.T) {
	store := NewMockTokenStore()
	ctx := context.Background()

	token := &RefreshToken{
		UserID:    1,
		TokenHash: "hash123",
		UserAgent: "Mozilla/5.0",
		IPAddress: "192.168.1.1",
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	store.SaveRefreshToken(ctx, token)

	found, err := store.FindRefreshToken(ctx, "hash123")
	if err != nil {
		t.Errorf("FindRefreshToken() error = %v", err)
	}

	if found.UserID != token.UserID {
		t.Errorf("user ID mismatch")
	}
}

func TestMockTokenStoreRevokeRefreshToken(t *testing.T) {
	store := NewMockTokenStore()
	ctx := context.Background()

	token := &RefreshToken{
		UserID:    1,
		TokenHash: "hash123",
		UserAgent: "Mozilla/5.0",
		IPAddress: "192.168.1.1",
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	store.SaveRefreshToken(ctx, token)

	err := store.RevokeRefreshToken(ctx, "hash123")
	if err != nil {
		t.Errorf("RevokeRefreshToken() error = %v", err)
	}

	found, _ := store.FindRefreshToken(ctx, "hash123")
	if found.RevokedAt == nil {
		t.Errorf("token should be revoked")
	}
}

func TestMockTokenStoreRevokeAllUserTokens(t *testing.T) {
	store := NewMockTokenStore()
	ctx := context.Background()

	token1 := &RefreshToken{
		UserID:    1,
		TokenHash: "hash1",
		UserAgent: "Mozilla/5.0",
		IPAddress: "192.168.1.1",
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	token2 := &RefreshToken{
		UserID:    1,
		TokenHash: "hash2",
		UserAgent: "Chrome/90.0",
		IPAddress: "192.168.1.2",
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	store.SaveRefreshToken(ctx, token1)
	store.SaveRefreshToken(ctx, token2)

	err := store.RevokeAllUserTokens(ctx, 1)
	if err != nil {
		t.Errorf("RevokeAllUserTokens() error = %v", err)
	}

	found1, _ := store.FindRefreshToken(ctx, "hash1")
	found2, _ := store.FindRefreshToken(ctx, "hash2")

	if found1.RevokedAt == nil || found2.RevokedAt == nil {
		t.Errorf("all tokens should be revoked")
	}
}

func TestMockTokenStoreSavePasswordResetToken(t *testing.T) {
	store := NewMockTokenStore()
	ctx := context.Background()

	token := &PasswordResetToken{
		UserID:    1,
		TokenHash: "reset_hash123",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	err := store.SavePasswordResetToken(ctx, token)
	if err != nil {
		t.Errorf("SavePasswordResetToken() error = %v", err)
	}

	if token.ID != 1 {
		t.Errorf("token ID = %d, want 1", token.ID)
	}
}

func TestMockTokenStoreFindPasswordResetToken(t *testing.T) {
	store := NewMockTokenStore()
	ctx := context.Background()

	token := &PasswordResetToken{
		UserID:    1,
		TokenHash: "reset_hash123",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	store.SavePasswordResetToken(ctx, token)

	found, err := store.FindPasswordResetToken(ctx, "reset_hash123")
	if err != nil {
		t.Errorf("FindPasswordResetToken() error = %v", err)
	}

	if found.UserID != token.UserID {
		t.Errorf("user ID mismatch")
	}
}

func TestMockTokenStoreMarkPasswordResetUsed(t *testing.T) {
	store := NewMockTokenStore()
	ctx := context.Background()

	token := &PasswordResetToken{
		UserID:    1,
		TokenHash: "reset_hash123",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	store.SavePasswordResetToken(ctx, token)

	err := store.MarkPasswordResetUsed(ctx, "reset_hash123")
	if err != nil {
		t.Errorf("MarkPasswordResetUsed() error = %v", err)
	}

	found, _ := store.FindPasswordResetToken(ctx, "reset_hash123")
	if found.UsedAt == nil {
		t.Errorf("token should be marked as used")
	}
}
