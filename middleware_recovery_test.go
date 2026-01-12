package dim

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRecoveryMiddleware(t *testing.T) {
	logger := NewLogger(slog.LevelInfo)

	panicHandler := func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	}

	recoveryMiddleware := Recovery(logger)
	handler := recoveryMiddleware(panicHandler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)

	// This should not panic
	handler(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestRecoveryMiddlewareNoPanic(t *testing.T) {
	logger := NewLogger(slog.LevelInfo)

	normalHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	recoveryMiddleware := Recovery(logger)
	handler := recoveryMiddleware(normalHandler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)

	handler(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
	}
}
