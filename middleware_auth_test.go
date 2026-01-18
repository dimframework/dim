package dim

import (
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
