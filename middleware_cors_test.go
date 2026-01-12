package dim

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSMiddlewareAllowedOrigin(t *testing.T) {
	config := CORSConfig{
		AllowedOrigins:   []string{"http://localhost:3000", "http://example.com"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           3600,
	}

	corsMiddleware := CORS(config)
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	wrappedHandler := corsMiddleware(handler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Origin", "http://localhost:3000")

	wrappedHandler(w, r)

	if w.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Errorf("CORS origin header not set correctly")
	}

	if w.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Errorf("CORS credentials header not set")
	}
}

func TestCORSMiddlewareDisallowedOrigin(t *testing.T) {
	config := CORSConfig{
		AllowedOrigins: []string{"http://localhost:3000"},
	}

	corsMiddleware := CORS(config)
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	wrappedHandler := corsMiddleware(handler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Origin", "http://evil.com")

	wrappedHandler(w, r)

	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("CORS should not allow disallowed origin")
	}
}

func TestCORSMiddlewarePreflight(t *testing.T) {
	config := CORSConfig{
		AllowedOrigins: []string{"http://localhost:3000"},
		AllowedMethods: []string{"GET", "POST"},
		AllowedHeaders: []string{"Content-Type"},
	}

	corsMiddleware := CORS(config)
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	wrappedHandler := corsMiddleware(handler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("OPTIONS", "/", nil)
	r.Header.Set("Origin", "http://localhost:3000")

	wrappedHandler(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("preflight status code = %d, want 200", w.Code)
	}

	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Errorf("CORS methods header not set")
	}
}

func TestCORSMiddlewareWildcard(t *testing.T) {
	config := CORSConfig{
		AllowedOrigins: []string{"*"},
	}

	corsMiddleware := CORS(config)
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	wrappedHandler := corsMiddleware(handler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Origin", "http://any-origin.com")

	wrappedHandler(w, r)

	if w.Header().Get("Access-Control-Allow-Origin") != "http://any-origin.com" {
		t.Errorf("wildcard CORS should allow any origin")
	}
}

func TestCORSMiddlewareNoOrigin(t *testing.T) {
	config := CORSConfig{
		AllowedOrigins: []string{"http://localhost:3000"},
	}

	corsMiddleware := CORS(config)
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	wrappedHandler := corsMiddleware(handler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	// No Origin header

	wrappedHandler(w, r)

	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("CORS should not set origin when no origin header")
	}
}
