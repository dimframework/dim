package dim

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCSRFMiddlewareGETRequest(t *testing.T) {
	config := CSRFConfig{
		Enabled:     true,
		TokenLength: 32,
		CookieName:  "csrf_token",
		HeaderName:  "X-CSRF-Token",
	}

	csrfMiddleware := CSRFMiddleware(config)
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	wrappedHandler := csrfMiddleware(handler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	wrappedHandler(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("GET request should pass CSRF check")
	}
}

func TestCSRFMiddlewarePOSTValid(t *testing.T) {
	config := CSRFConfig{
		Enabled:     true,
		TokenLength: 32,
		CookieName:  "csrf_token",
		HeaderName:  "X-CSRF-Token",
	}

	token, _ := GenerateCSRFToken(32)

	csrfMiddleware := CSRFMiddleware(config)
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	wrappedHandler := csrfMiddleware(handler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/", nil)
	r.Header.Set("X-CSRF-Token", token)

	// Set cookie
	cookie := &http.Cookie{
		Name:  "csrf_token",
		Value: token,
	}
	r.AddCookie(cookie)

	wrappedHandler(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("POST with valid CSRF token should pass")
	}
}

func TestCSRFMiddlewarePOSTInvalid(t *testing.T) {
	config := CSRFConfig{
		Enabled:     true,
		TokenLength: 32,
		CookieName:  "csrf_token",
		HeaderName:  "X-CSRF-Token",
	}

	csrfMiddleware := CSRFMiddleware(config)
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	wrappedHandler := csrfMiddleware(handler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/", nil)
	r.Header.Set("X-CSRF-Token", "invalid_token")

	// Set different cookie
	cookie := &http.Cookie{
		Name:  "csrf_token",
		Value: "different_token",
	}
	r.AddCookie(cookie)

	wrappedHandler(w, r)

	if w.Code != 419 {
		t.Errorf("POST with invalid CSRF token should return 419, got %d", w.Code)
	}
}

func TestCSRFMiddlewareExemptPath(t *testing.T) {
	config := CSRFConfig{
		Enabled:     true,
		ExemptPaths: []string{"/webhooks", "/health"},
		CookieName:  "csrf_token",
		HeaderName:  "X-CSRF-Token",
	}

	csrfMiddleware := CSRFMiddleware(config)
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	wrappedHandler := csrfMiddleware(handler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/webhooks", nil)
	// No CSRF token

	wrappedHandler(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("exempt path should bypass CSRF check")
	}
}

func TestCSRFMiddlewareDisabled(t *testing.T) {
	config := CSRFConfig{
		Enabled: false,
	}

	csrfMiddleware := CSRFMiddleware(config)
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	wrappedHandler := csrfMiddleware(handler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/", nil)
	// No token

	wrappedHandler(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("disabled CSRF should allow request")
	}
}

func TestGenerateCSRFToken(t *testing.T) {
	token1, err := GenerateCSRFToken(32)
	if err != nil {
		t.Errorf("GenerateCSRFToken() error = %v", err)
	}

	token2, _ := GenerateCSRFToken(32)

	if token1 == "" || token2 == "" {
		t.Errorf("tokens should not be empty")
	}

	if token1 == token2 {
		t.Errorf("tokens should be different")
	}
}

func TestSetCSRFToken(t *testing.T) {
	config := CSRFConfig{
		CookieName:   "csrf_token",
		CookieMaxAge: 3600,
	}

	token := "test_token_123"

	w := httptest.NewRecorder()
	SetCSRFToken(w, token, config)

	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Errorf("expected 1 cookie, got %d", len(cookies))
	}

	if cookies[0].Value != token {
		t.Errorf("cookie value mismatch")
	}

	if cookies[0].MaxAge != 3600 {
		t.Errorf("expected cookie MaxAge 3600, got %d", cookies[0].MaxAge)
	}
}
