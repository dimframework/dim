package dim

import (
	"testing"
	"time"
)

func TestGenerateAccessToken(t *testing.T) {
	config := &JWTConfig{
		Secret:             "test-secret",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	}
	manager := NewJWTManager(config)

	token, err := manager.GenerateAccessToken(1, "test@example.com")
	if err != nil {
		t.Errorf("GenerateAccessToken() error = %v", err)
	}

	if token == "" {
		t.Errorf("token is empty")
	}
}

func TestVerifyAccessToken(t *testing.T) {
	config := &JWTConfig{
		Secret:             "test-secret",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	}
	manager := NewJWTManager(config)

	token, _ := manager.GenerateAccessToken(1, "test@example.com")

	claims, err := manager.VerifyToken(token)
	if err != nil {
		t.Errorf("VerifyToken() error = %v", err)
	}

	if claims.UserID != 1 {
		t.Errorf("UserID = %d, want 1", claims.UserID)
	}

	if claims.Email != "test@example.com" {
		t.Errorf("Email = %s, want test@example.com", claims.Email)
	}
}

func TestVerifyAccessTokenInvalid(t *testing.T) {
	config := &JWTConfig{
		Secret:             "test-secret",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	}
	manager := NewJWTManager(config)

	_, err := manager.VerifyToken("invalid-token")
	if err == nil {
		t.Errorf("VerifyToken() should fail for invalid token")
	}
}

func TestGenerateRefreshToken(t *testing.T) {
	config := &JWTConfig{
		Secret:             "test-secret",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	}
	manager := NewJWTManager(config)

	token, err := manager.GenerateRefreshToken(1)
	if err != nil {
		t.Errorf("GenerateRefreshToken() error = %v", err)
	}

	if token == "" {
		t.Errorf("token is empty")
	}
}

func TestVerifyRefreshToken(t *testing.T) {
	config := &JWTConfig{
		Secret:             "test-secret",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	}
	manager := NewJWTManager(config)

	token, _ := manager.GenerateRefreshToken(1)

	userID, err := manager.VerifyRefreshToken(token)
	if err != nil {
		t.Errorf("VerifyRefreshToken() error = %v", err)
	}

	if userID != 1 {
		t.Errorf("userID = %d, want 1", userID)
	}
}

func TestGetTokenExpiry(t *testing.T) {
	config := &JWTConfig{
		Secret:             "test-secret",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	}
	manager := NewJWTManager(config)

	token, _ := manager.GenerateAccessToken(1, "test@example.com")

	expiry, err := manager.GetTokenExpiry(token)
	if err != nil {
		t.Errorf("GetTokenExpiry() error = %v", err)
	}

	now := time.Now()
	if expiry.Before(now) || expiry.After(now.Add(20*time.Minute)) {
		t.Errorf("token expiry is out of expected range")
	}
}

func TestIsTokenExpired(t *testing.T) {
	config := &JWTConfig{
		Secret:             "test-secret",
		AccessTokenExpiry:  1 * time.Millisecond, // Very short expiry
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	}
	manager := NewJWTManager(config)

	token, _ := manager.GenerateAccessToken(1, "test@example.com")

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	expired, err := manager.IsTokenExpired(token)
	if err != nil {
		t.Errorf("IsTokenExpired() error = %v", err)
	}

	if !expired {
		t.Errorf("token should be expired")
	}
}

func TestDifferentSecrets(t *testing.T) {
	config1 := &JWTConfig{
		Secret:             "secret1",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	}
	manager1 := NewJWTManager(config1)

	config2 := &JWTConfig{
		Secret:             "secret2",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	}
	manager2 := NewJWTManager(config2)

	token, _ := manager1.GenerateAccessToken(1, "test@example.com")

	_, err := manager2.VerifyToken(token)
	if err == nil {
		t.Errorf("VerifyToken() with different secret should fail")
	}
}
