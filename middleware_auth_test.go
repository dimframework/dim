package dim

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
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

func TestRequireAuthWithLogger(t *testing.T) {
	config := &JWTConfig{
		HMACSecret:        "test-secret",
		SigningMethod:     "HS256",
		AccessTokenExpiry: 15 * time.Minute,
	}
	jwtManager, _ := NewJWTManager(config)

	var logBuf bytes.Buffer
	logger := NewLoggerWithWriter(&logBuf, slog.LevelInfo)

	authMiddleware := RequireAuth(jwtManager, nil, WithAuthLogger(logger))
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
	wrappedHandler := authMiddleware(handler)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "Bearer invalid-token")
	wrappedHandler(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("invalid token should return 401")
	}
}

func TestOptionalAuthValid(t *testing.T) {
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
		t.Errorf("valid token in optional auth should process and set user")
	}
}

func TestOptionalAuthMissing(t *testing.T) {
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
			w.WriteHeader(http.StatusBadRequest) // Should not be authenticated
			return
		}
		w.WriteHeader(http.StatusOK)
	}
	wrappedHandler := authMiddleware(handler)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	wrappedHandler(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("missing token in optional auth should pass")
	}
}

func TestOptionalAuthInvalid(t *testing.T) {
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
			w.WriteHeader(http.StatusBadRequest) // Should not be authenticated
			return
		}
		w.WriteHeader(http.StatusOK)
	}
	wrappedHandler := authMiddleware(handler)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "Bearer invalid-token")
	wrappedHandler(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("invalid token in optional auth should pass ignored")
	}
}

func TestRequireAuthWithCookie(t *testing.T) {
	config := &JWTConfig{
		HMACSecret:        "test-secret",
		SigningMethod:     "HS256",
		AccessTokenExpiry: 15 * time.Minute,
	}
	jwtManager, _ := NewJWTManager(config)
	token, _ := jwtManager.GenerateAccessToken("2", "user@example.com", "sid-456", nil)

	// Use WithCookieToken option
	authMiddleware := RequireAuth(jwtManager, nil, WithCookieToken("auth_token"))
	handler := func(w http.ResponseWriter, r *http.Request) {
		user, ok := GetUser(r)
		if !ok || user.GetID() != "2" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
	wrappedHandler := authMiddleware(handler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r.AddCookie(&http.Cookie{Name: "auth_token", Value: token})

	wrappedHandler(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("valid cookie token should pass")
	}
}

func TestRequireAuthWithMultipleSources(t *testing.T) {
	config := &JWTConfig{
		HMACSecret:        "test-secret",
		SigningMethod:     "HS256",
		AccessTokenExpiry: 15 * time.Minute,
	}
	jwtManager, _ := NewJWTManager(config)

	authMiddleware := RequireAuth(jwtManager, nil,
		WithBearerToken(),
		WithCookieToken("auth_token"),
	)

	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
	wrappedHandler := authMiddleware(handler)

	// Case 1: Bearer token only
	token1, _ := jwtManager.GenerateAccessToken("1", "user@example.com", "sid-1", nil)
	w1 := httptest.NewRecorder()
	r1 := httptest.NewRequest("GET", "/", nil)
	r1.Header.Set("Authorization", "Bearer "+token1)
	wrappedHandler(w1, r1)
	if w1.Code != http.StatusOK {
		t.Errorf("valid bearer token should pass when both configured")
	}

	// Case 2: Cookie token only
	token2, _ := jwtManager.GenerateAccessToken("2", "user@example.com", "sid-2", nil)
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest("GET", "/", nil)
	r2.AddCookie(&http.Cookie{Name: "auth_token", Value: token2})
	wrappedHandler(w2, r2)
	if w2.Code != http.StatusOK {
		t.Errorf("valid cookie token should pass when both configured")
	}

	// Case 3: Missing both
	w3 := httptest.NewRecorder()
	r3 := httptest.NewRequest("GET", "/", nil)
	wrappedHandler(w3, r3)
	if w3.Code != http.StatusUnauthorized {
		t.Errorf("missing token from both sources should fail")
	}
}
