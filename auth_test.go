package dim

import (
	"context"
	"testing"
	"time"
)

func TestRegisterSuccess(t *testing.T) {
	userStore := NewMockUserStore()
	tokenStore := NewMockTokenStore()
	config := &JWTConfig{
		Secret:             "test-secret",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	}

	service := NewAuthService(userStore, tokenStore, config)
	ctx := context.Background()

	user, err := service.Register(ctx, "test@example.com", "Test User", "ValidPass123!")
	if err != nil {
		t.Errorf("Register() error = %v", err)
	}

	if user.Email != "test@example.com" {
		t.Errorf("email mismatch")
	}
}

func TestRegisterInvalidEmail(t *testing.T) {
	userStore := NewMockUserStore()
	tokenStore := NewMockTokenStore()
	config := &JWTConfig{
		Secret:             "test-secret",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	}

	service := NewAuthService(userStore, tokenStore, config)
	ctx := context.Background()

	_, err := service.Register(ctx, "invalid-email", "Test User", "ValidPass123!")
	if err == nil {
		t.Errorf("Register() should fail for invalid email")
	}
}

func TestRegisterWeakPassword(t *testing.T) {
	userStore := NewMockUserStore()
	tokenStore := NewMockTokenStore()
	config := &JWTConfig{
		Secret:             "test-secret",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	}

	service := NewAuthService(userStore, tokenStore, config)
	ctx := context.Background()

	_, err := service.Register(ctx, "test@example.com", "Test User", "weak")
	if err == nil {
		t.Errorf("Register() should fail for weak password")
	}
}

func TestLoginSuccess(t *testing.T) {
	userStore := NewMockUserStore()
	tokenStore := NewMockTokenStore()
	config := &JWTConfig{
		Secret:             "test-secret",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	}

	service := NewAuthService(userStore, tokenStore, config)
	ctx := context.Background()

	// Register first
	service.Register(ctx, "test@example.com", "Test User", "ValidPass123!")

	// Then login
	accessToken, refreshToken, err := service.Login(ctx, "test@example.com", "ValidPass123!")
	if err != nil {
		t.Errorf("Login() error = %v", err)
	}

	if accessToken == "" {
		t.Errorf("access token is empty")
	}

	if refreshToken == "" {
		t.Errorf("refresh token is empty")
	}
}

func TestLoginInvalidPassword(t *testing.T) {
	userStore := NewMockUserStore()
	tokenStore := NewMockTokenStore()
	config := &JWTConfig{
		Secret:             "test-secret",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	}

	service := NewAuthService(userStore, tokenStore, config)
	ctx := context.Background()

	// Register first
	service.Register(ctx, "test@example.com", "Test User", "ValidPass123!")

	// Try login with wrong password
	_, _, err := service.Login(ctx, "test@example.com", "WrongPassword123!")
	if err == nil {
		t.Errorf("Login() should fail with wrong password")
	}
}

func TestRefreshTokenSuccess(t *testing.T) {
	userStore := NewMockUserStore()
	tokenStore := NewMockTokenStore()
	config := &JWTConfig{
		Secret:             "test-secret",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	}

	service := NewAuthService(userStore, tokenStore, config)
	ctx := context.Background()

	// Register and login
	service.Register(ctx, "test@example.com", "Test User", "ValidPass123!")
	_, refreshToken, _ := service.Login(ctx, "test@example.com", "ValidPass123!")

	// Refresh token
	newAccessToken, newRefreshToken, err := service.RefreshToken(ctx, refreshToken)
	if err != nil {
		t.Errorf("RefreshToken() error = %v", err)
	}

	if newAccessToken == "" {
		t.Errorf("new access token is empty")
	}

	if newRefreshToken == "" {
		t.Errorf("new refresh token is empty")
	}
}

func TestLogoutSuccess(t *testing.T) {
	userStore := NewMockUserStore()
	tokenStore := NewMockTokenStore()
	config := &JWTConfig{
		Secret:             "test-secret",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	}

	service := NewAuthService(userStore, tokenStore, config)
	ctx := context.Background()

	// Register and login
	service.Register(ctx, "test@example.com", "Test User", "ValidPass123!")
	_, refreshToken, _ := service.Login(ctx, "test@example.com", "ValidPass123!")

	// Logout
	err := service.Logout(ctx, refreshToken)
	if err != nil {
		t.Errorf("Logout() error = %v", err)
	}
}

func TestRequestPasswordResetSuccess(t *testing.T) {
	userStore := NewMockUserStore()
	tokenStore := NewMockTokenStore()
	config := &JWTConfig{
		Secret:             "test-secret",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	}

	service := NewAuthService(userStore, tokenStore, config)
	ctx := context.Background()

	// Register
	service.Register(ctx, "test@example.com", "Test User", "ValidPass123!")

	// Request password reset
	err := service.RequestPasswordReset(ctx, "test@example.com")
	if err != nil {
		t.Errorf("RequestPasswordReset() error = %v", err)
	}
}
