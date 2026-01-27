package dim

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Tests for the new `ExpectBearerToken` function (previously `RequireAuth`)
func TestExpectBearerTokenMissing(t *testing.T) {
	authMiddleware := ExpectBearerToken()
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
	wrappedHandler := authMiddleware(handler)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	wrappedHandler(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("missing token should return 401")
	}
}

func TestExpectBearerTokenInvalid(t *testing.T) {
	authMiddleware := ExpectBearerToken()
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
	wrappedHandler := authMiddleware(handler)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "InvalidFormat")
	wrappedHandler(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("invalid auth header should return 401")
	}
}

func TestExpectBearerTokenValid(t *testing.T) {
	authMiddleware := ExpectBearerToken()
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
	wrappedHandler := authMiddleware(handler)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "Bearer valid_token")
	wrappedHandler(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("valid bearer token should pass")
	}
}

// Test for the new `AllowBearerToken` function (previously `OptionalAuth`)
func TestAllowBearerToken(t *testing.T) {
	authMiddleware := AllowBearerToken()
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
	wrappedHandler := authMiddleware(handler)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	wrappedHandler(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("optional auth should pass without token")
	}
}

// Tests for the new `RequireAuth` function (previously `RequireAuthWithManager`)
func TestRequireAuthValid(t *testing.T) {
	config := &JWTConfig{
		HMACSecret:        "test-secret",
		SigningMethod:     "HS256",
		AccessTokenExpiry: 15 * time.Minute,
	}
	jwtManager, _ := NewJWTManager(config)
	token, _ := jwtManager.GenerateAccessToken("1", "test@example.com", "sid-123", nil)

	authMiddleware := RequireAuth(jwtManager, nil)
	handler := func(w http.ResponseWriter, r *http.Request) {
		user, ok := GetUser(r)
		if !ok || user.GetID() != "1" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
	wrappedHandler := authMiddleware(handler)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "Bearer "+token)
	wrappedHandler(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("valid token should pass and set user context")
	}
}

func TestRequireAuthInvalid(t *testing.T) {
	config := &JWTConfig{
		HMACSecret:        "test-secret",
		SigningMethod:     "HS256",
		AccessTokenExpiry: 15 * time.Minute,
	}
	jwtManager, _ := NewJWTManager(config)
	authMiddleware := RequireAuth(jwtManager, nil)
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
	wrappedHandler := authMiddleware(handler)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "Bearer invalid_token")
	wrappedHandler(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("invalid token should return 401")
	}
}

// Tests for the new `OptionalAuth` function (previously `OptionalAuthWithManager`)
func TestOptionalAuthNoToken(t *testing.T) {
	config := &JWTConfig{
		HMACSecret:        "test-secret",
		SigningMethod:     "HS256",
		AccessTokenExpiry: 15 * time.Minute,
	}
	jwtManager, _ := NewJWTManager(config)
	authMiddleware := OptionalAuth(jwtManager)
	handler := func(w http.ResponseWriter, r *http.Request) {
		_, ok := GetUser(r)
		if ok {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
	wrappedHandler := authMiddleware(handler)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	wrappedHandler(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("optional auth should pass without token")
	}
}

func TestOptionalAuthValidToken(t *testing.T) {
	config := &JWTConfig{
		HMACSecret:        "test-secret",
		SigningMethod:     "HS256",
		AccessTokenExpiry: 15 * time.Minute,
	}
	jwtManager, _ := NewJWTManager(config)
	token, _ := jwtManager.GenerateAccessToken("1", "test@example.com", "sid-123", nil)
	authMiddleware := OptionalAuth(jwtManager)
	handler := func(w http.ResponseWriter, r *http.Request) {
		user, ok := GetUser(r)
		if !ok || user.GetID() != "1" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
	wrappedHandler := authMiddleware(handler)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "Bearer "+token)
	wrappedHandler(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("optional auth should set user context with valid token")
	}
}

func TestRequireAuthWithLogger_LogsTokenTypeError(t *testing.T) {
	config := &JWTConfig{
		HMACSecret:         "test-secret",
		SigningMethod:      "HS256",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	}
	jwtManager, _ := NewJWTManager(config)

	// Generate refresh token (typ: rt+jwt) - wrong type for RequireAuth
	refreshToken, _ := jwtManager.GenerateRefreshToken("1", "sid-123")

	// Create logger with buffer to capture output
	var buf bytes.Buffer
	logger := NewLoggerWithWriter(&buf, slog.LevelDebug)

	// Create middleware with logger
	authMiddleware := RequireAuth(jwtManager, nil, WithAuthLogger(logger))
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
	wrappedHandler := authMiddleware(handler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test-path", nil)
	r.Header.Set("Authorization", "Bearer "+refreshToken)
	wrappedHandler(w, r)

	// Should return 401
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 Unauthorized, got %d", w.Code)
	}

	// Check log output contains the error
	logOutput := buf.String()
	if !strings.Contains(logOutput, "Token verification failed") {
		t.Errorf("Expected log to contain 'Token verification failed', got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "invalid token type: expected access token") {
		t.Errorf("Expected log to contain 'invalid token type: expected access token', got: %s", logOutput)
	}
}

func TestRequireAuthWithoutLogger_NoLog(t *testing.T) {
	config := &JWTConfig{
		HMACSecret:         "test-secret",
		SigningMethod:      "HS256",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	}
	jwtManager, _ := NewJWTManager(config)

	// Generate refresh token (wrong type)
	refreshToken, _ := jwtManager.GenerateRefreshToken("1", "sid-123")

	// Create middleware WITHOUT logger - should not panic
	authMiddleware := RequireAuth(jwtManager, nil)
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
	wrappedHandler := authMiddleware(handler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test-path", nil)
	r.Header.Set("Authorization", "Bearer "+refreshToken)

	// Should not panic and return 401
	wrappedHandler(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 Unauthorized, got %d", w.Code)
	}
}
