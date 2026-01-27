package dim

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"
)

// MockUser implements Authenticatable
type MockUser struct {
	ID       string
	Email    string
	Password string
}

func (u *MockUser) GetID() string        { return u.ID }
func (u *MockUser) GetEmail() string     { return u.Email }
func (u *MockUser) GetPassword() string  { return u.Password }
func (u *MockUser) SetPassword(p string) { u.Password = p }

// MockUserStore implements AuthUserStore
type MockUserStore struct {
	users map[string]*MockUser
}

func NewMockUserStore() *MockUserStore {
	return &MockUserStore{
		users: make(map[string]*MockUser),
	}
}

func (s *MockUserStore) FindByEmail(ctx context.Context, email string) (Authenticatable, error) {
	for _, u := range s.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, errors.New("user not found")
}

func (s *MockUserStore) FindByID(ctx context.Context, id string) (Authenticatable, error) {
	if u, ok := s.users[id]; ok {
		return u, nil
	}
	return nil, errors.New("user not found")
}

func (s *MockUserStore) Update(ctx context.Context, user Authenticatable) error {
	u, ok := user.(*MockUser)
	if !ok {
		return errors.New("invalid user type")
	}
	s.users[u.ID] = u
	return nil
}

func (s *MockUserStore) AddUser(user *MockUser) {
	s.users[user.ID] = user
}

func TestLoginSuccess(t *testing.T) {
	userStore := NewMockUserStore()
	tokenStore := NewMockTokenStore()
	config := &JWTConfig{
		HMACSecret:         "test-secret",
		SigningMethod:      "HS256",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	}

	hashedPassword, _ := HashPassword("ValidPass123!")
	userStore.AddUser(&MockUser{
		ID:       "1",
		Email:    "test@example.com",
		Password: hashedPassword,
	})

	service, err := NewAuthService(userStore, tokenStore, nil, config)
	if err != nil {
		t.Fatalf("NewAuthService error: %v", err)
	}
	ctx := context.Background()

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
		HMACSecret:         "test-secret",
		SigningMethod:      "HS256",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	}

	hashedPassword, _ := HashPassword("ValidPass123!")
	userStore.AddUser(&MockUser{
		ID:       "1",
		Email:    "test@example.com",
		Password: hashedPassword,
	})

	service, err := NewAuthService(userStore, tokenStore, nil, config)
	if err != nil {
		t.Fatalf("NewAuthService error: %v", err)
	}
	ctx := context.Background()

	_, _, err = service.Login(ctx, "test@example.com", "WrongPass")
	if err == nil {
		t.Errorf("Login() should fail for invalid password")
	}
}

func TestRefreshTokenSuccess(t *testing.T) {
	userStore := NewMockUserStore()
	tokenStore := NewMockTokenStore()
	config := &JWTConfig{
		HMACSecret:         "test-secret",
		SigningMethod:      "HS256",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	}

	hashedPassword, _ := HashPassword("ValidPass123!")
	userStore.AddUser(&MockUser{
		ID:       "1",
		Email:    "test@example.com",
		Password: hashedPassword,
	})

	service, err := NewAuthService(userStore, tokenStore, nil, config)
	if err != nil {
		t.Fatalf("NewAuthService error: %v", err)
	}
	ctx := context.Background()

	// Register and login
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
		HMACSecret:         "test-secret",
		SigningMethod:      "HS256",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	}

	hashedPassword, _ := HashPassword("ValidPass123!")
	userStore.AddUser(&MockUser{
		ID:       "1",
		Email:    "test@example.com",
		Password: hashedPassword,
	})

	service, err := NewAuthService(userStore, tokenStore, nil, config)
	if err != nil {
		t.Fatalf("NewAuthService error: %v", err)
	}
	ctx := context.Background()

	// Register and login
	_, refreshToken, _ := service.Login(ctx, "test@example.com", "ValidPass123!")

	// Logout
	err = service.Logout(ctx, refreshToken)
	if err != nil {
		t.Errorf("Logout() error = %v", err)
	}
}

func TestRequestPasswordResetSuccess(t *testing.T) {
	userStore := NewMockUserStore()
	tokenStore := NewMockTokenStore()
	config := &JWTConfig{
		HMACSecret:         "test-secret",
		SigningMethod:      "HS256",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	}

	hashedPassword, _ := HashPassword("ValidPass123!")
	userStore.AddUser(&MockUser{
		ID:       "1",
		Email:    "test@example.com",
		Password: hashedPassword,
	})

	service, err := NewAuthService(userStore, tokenStore, nil, config)
	if err != nil {
		t.Fatalf("NewAuthService error: %v", err)
	}
	ctx := context.Background()

	// Request password reset
	token, err := service.RequestPasswordReset(ctx, "test@example.com")
	if err != nil {
		t.Errorf("RequestPasswordReset() error = %v", err)
	}
	if token == "" {
		t.Error("RequestPasswordReset() should return token")
	}
}

func TestAuthService_WithLogger(t *testing.T) {
	userStore := NewMockUserStore()
	tokenStore := NewMockTokenStore()
	config := &JWTConfig{
		HMACSecret:         "test-secret",
		SigningMethod:      "HS256",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	}

	var buf bytes.Buffer
	logger := NewLoggerWithWriter(&buf, slog.LevelDebug)

	service, err := NewAuthService(userStore, tokenStore, nil, config)
	if err != nil {
		t.Fatalf("NewAuthService error: %v", err)
	}

	// Test chaining
	result := service.WithLogger(logger)
	if result != service {
		t.Error("WithLogger should return the same service instance for chaining")
	}
}

func TestAuthService_RefreshToken_LogsTokenTypeError(t *testing.T) {
	userStore := NewMockUserStore()
	tokenStore := NewMockTokenStore()
	config := &JWTConfig{
		HMACSecret:         "test-secret",
		SigningMethod:      "HS256",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	}

	hashedPassword, _ := HashPassword("ValidPass123!")
	userStore.AddUser(&MockUser{
		ID:       "1",
		Email:    "test@example.com",
		Password: hashedPassword,
	})

	// Create logger with buffer to capture output
	var buf bytes.Buffer
	logger := NewLoggerWithWriter(&buf, slog.LevelDebug)

	service, err := NewAuthService(userStore, tokenStore, nil, config)
	if err != nil {
		t.Fatalf("NewAuthService error: %v", err)
	}
	service.WithLogger(logger)

	ctx := context.Background()

	// Login to get tokens
	accessToken, _, err := service.Login(ctx, "test@example.com", "ValidPass123!")
	if err != nil {
		t.Fatalf("Login error: %v", err)
	}

	// Try to use ACCESS token as refresh token - should fail and log error
	_, _, err = service.RefreshToken(ctx, accessToken)
	if err == nil {
		t.Error("RefreshToken should fail when using access token")
	}

	// Check log output contains the error
	logOutput := buf.String()
	if !strings.Contains(logOutput, "Refresh token verification failed") {
		t.Errorf("Expected log to contain 'Refresh token verification failed', got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "invalid token type: expected refresh token") {
		t.Errorf("Expected log to contain 'invalid token type: expected refresh token', got: %s", logOutput)
	}
}

func TestAuthService_RefreshToken_NoLogWithoutLogger(t *testing.T) {
	userStore := NewMockUserStore()
	tokenStore := NewMockTokenStore()
	config := &JWTConfig{
		HMACSecret:         "test-secret",
		SigningMethod:      "HS256",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	}

	hashedPassword, _ := HashPassword("ValidPass123!")
	userStore.AddUser(&MockUser{
		ID:       "1",
		Email:    "test@example.com",
		Password: hashedPassword,
	})

	service, err := NewAuthService(userStore, tokenStore, nil, config)
	if err != nil {
		t.Fatalf("NewAuthService error: %v", err)
	}
	// Note: NOT calling WithLogger - logger is nil

	ctx := context.Background()

	// Login to get tokens
	accessToken, _, err := service.Login(ctx, "test@example.com", "ValidPass123!")
	if err != nil {
		t.Fatalf("Login error: %v", err)
	}

	// Try to use ACCESS token as refresh token - should fail but NOT panic
	_, _, err = service.RefreshToken(ctx, accessToken)
	if err == nil {
		t.Error("RefreshToken should fail when using access token")
	}

	// Test passes if no panic occurred
}
