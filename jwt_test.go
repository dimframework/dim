package dim

import (
	"testing"
	"time"
)

func TestGenerateAccessToken(t *testing.T) {
	config := &JWTConfig{
		HMACSecret:         "test-secret",
		SigningMethod:      "HS256",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	}
	manager, err := NewJWTManager(config)
	if err != nil {
		t.Fatalf("NewJWTManager error: %v", err)
	}

	token, err := manager.GenerateAccessToken("1", "test@example.com", "sid-123", nil)
	if err != nil {
		t.Errorf("GenerateAccessToken() error = %v", err)
	}

	if token == "" {
		t.Errorf("token is empty")
	}
}

func TestVerifyAccessToken(t *testing.T) {
	config := &JWTConfig{
		HMACSecret:         "test-secret",
		SigningMethod:      "HS256",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	}
	manager, err := NewJWTManager(config)
	if err != nil {
		t.Fatalf("NewJWTManager error: %v", err)
	}

	token, _ := manager.GenerateAccessToken("1", "test@example.com", "sid-123", nil)

	claims, err := manager.VerifyToken(token)
	if err != nil {
		t.Errorf("VerifyToken() error = %v", err)
	}

	if sub, ok := claims["sub"].(string); !ok || sub != "1" {
		t.Errorf("sub = %v, want 1", claims["sub"])
	}

	if email, ok := claims["email"].(string); !ok || email != "test@example.com" {
		t.Errorf("email = %v, want test@example.com", claims["email"])
	}
}

func TestVerifyAccessTokenInvalid(t *testing.T) {
	config := &JWTConfig{
		HMACSecret:         "test-secret",
		SigningMethod:      "HS256",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	}
	manager, err := NewJWTManager(config)
	if err != nil {
		t.Fatalf("NewJWTManager error: %v", err)
	}

	_, err = manager.VerifyToken("invalid-token")
	if err == nil {
		t.Errorf("VerifyToken() should fail for invalid token")
	}
}

func TestGenerateRefreshToken(t *testing.T) {
	config := &JWTConfig{
		HMACSecret:         "test-secret",
		SigningMethod:      "HS256",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	}
	manager, err := NewJWTManager(config)
	if err != nil {
		t.Fatalf("NewJWTManager error: %v", err)
	}

	token, err := manager.GenerateRefreshToken("1", "sid-123")
	if err != nil {
		t.Errorf("GenerateRefreshToken() error = %v", err)
	}

	if token == "" {
		t.Errorf("token is empty")
	}
}

func TestVerifyRefreshToken(t *testing.T) {
	config := &JWTConfig{
		HMACSecret:         "test-secret",
		SigningMethod:      "HS256",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	}
	manager, err := NewJWTManager(config)
	if err != nil {
		t.Fatalf("NewJWTManager error: %v", err)
	}

	token, _ := manager.GenerateRefreshToken("1", "sid-123")

	userID, _, err := manager.VerifyRefreshToken(token)
	if err != nil {
		t.Errorf("VerifyRefreshToken() error = %v", err)
	}

	if userID != "1" {
		t.Errorf("userID = %s, want 1", userID)
	}
}

func TestGetTokenExpiry(t *testing.T) {
	config := &JWTConfig{
		HMACSecret:         "test-secret",
		SigningMethod:      "HS256",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	}
	manager, err := NewJWTManager(config)
	if err != nil {
		t.Fatalf("NewJWTManager error: %v", err)
	}

	token, _ := manager.GenerateAccessToken("1", "test@example.com", "sid-123", nil)

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
		HMACSecret:         "test-secret",
		SigningMethod:      "HS256",
		AccessTokenExpiry:  1 * time.Millisecond, // Very short expiry
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	}
	manager, err := NewJWTManager(config)
	if err != nil {
		t.Fatalf("NewJWTManager error: %v", err)
	}

	token, _ := manager.GenerateAccessToken("1", "test@example.com", "sid-123", nil)

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

func TestVerifyToken_RejectsRefreshToken(t *testing.T) {
	config := &JWTConfig{
		HMACSecret:         "test-secret",
		SigningMethod:      "HS256",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	}
	manager, err := NewJWTManager(config)
	if err != nil {
		t.Fatalf("NewJWTManager error: %v", err)
	}

	// Generate refresh token (typ: rt+jwt)
	refreshToken, _ := manager.GenerateRefreshToken("1", "sid-123")

	// Try to verify refresh token using VerifyToken (which expects access token typ: at+jwt)
	_, err = manager.VerifyToken(refreshToken)
	if err == nil {
		t.Errorf("VerifyToken() should reject refresh token")
	}

	// Verify error message indicates token type mismatch
	if err != nil && err.Error() != "invalid token type: expected access token" {
		t.Errorf("Expected 'invalid token type: expected access token', got: %v", err)
	}
}

func TestVerifyRefreshToken_RejectsAccessToken(t *testing.T) {
	config := &JWTConfig{
		HMACSecret:         "test-secret",
		SigningMethod:      "HS256",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	}
	manager, err := NewJWTManager(config)
	if err != nil {
		t.Fatalf("NewJWTManager error: %v", err)
	}

	// Generate access token (typ: at+jwt)
	accessToken, _ := manager.GenerateAccessToken("1", "test@example.com", "sid-123", nil)

	// Try to verify access token using VerifyRefreshToken (which expects refresh token typ: rt+jwt)
	_, _, err = manager.VerifyRefreshToken(accessToken)
	if err == nil {
		t.Errorf("VerifyRefreshToken() should reject access token")
	}

	// Verify error message indicates token type mismatch
	if err != nil && err.Error() != "invalid token type: expected refresh token" {
		t.Errorf("Expected 'invalid token type: expected refresh token', got: %v", err)
	}
}

func TestTokenTypeHeader(t *testing.T) {
	config := &JWTConfig{
		HMACSecret:         "test-secret",
		SigningMethod:      "HS256",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	}
	manager, err := NewJWTManager(config)
	if err != nil {
		t.Fatalf("NewJWTManager error: %v", err)
	}

	t.Run("AccessToken_HasCorrectTypHeader", func(t *testing.T) {
		accessToken, _ := manager.GenerateAccessToken("1", "test@example.com", "sid-123", nil)

		// Verify access token can be verified with VerifyToken
		claims, err := manager.VerifyToken(accessToken)
		if err != nil {
			t.Errorf("VerifyToken() should succeed for access token: %v", err)
		}
		if claims == nil {
			t.Error("Claims should not be nil")
		}
	})

	t.Run("RefreshToken_HasCorrectTypHeader", func(t *testing.T) {
		refreshToken, _ := manager.GenerateRefreshToken("1", "sid-123")

		// Verify refresh token can be verified with VerifyRefreshToken
		userID, sessionID, err := manager.VerifyRefreshToken(refreshToken)
		if err != nil {
			t.Errorf("VerifyRefreshToken() should succeed for refresh token: %v", err)
		}
		if userID != "1" {
			t.Errorf("userID = %s, want 1", userID)
		}
		if sessionID != "sid-123" {
			t.Errorf("sessionID = %s, want sid-123", sessionID)
		}
	})
}

func TestDifferentSecrets(t *testing.T) {
	config1 := &JWTConfig{
		HMACSecret:         "secret1",
		SigningMethod:      "HS256",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	}
	manager1, _ := NewJWTManager(config1)

	config2 := &JWTConfig{
		HMACSecret:         "secret2",
		SigningMethod:      "HS256",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	}
	manager2, _ := NewJWTManager(config2)

	token, _ := manager1.GenerateAccessToken("1", "test@example.com", "sid-123", nil)

	_, err := manager2.VerifyToken(token)
	if err == nil {
		t.Errorf("VerifyToken() with different secret should fail")
	}
}
