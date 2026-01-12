package dim

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLoggerMiddleware(t *testing.T) {
	logger := NewLogger(slog.LevelInfo)

	var capturedRequestID string
	handler := func(w http.ResponseWriter, r *http.Request) {
		// Capture the request ID from within the handler
		capturedRequestID = GetRequestID(r)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}

	loggerMiddleware := LoggerMiddleware(logger)
	wrappedHandler := loggerMiddleware(handler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)

	wrappedHandler(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
	}

	// Check that request ID was set (should be captured from within the handler)
	if capturedRequestID == "" {
		t.Errorf("request ID should be set")
	}
}

func TestLoggerMiddlewareStatusCode(t *testing.T) {
	logger := NewLogger(slog.LevelInfo)

	tests := []struct {
		name       string
		statusCode int
	}{
		{"OK", http.StatusOK},
		{"Created", http.StatusCreated},
		{"BadRequest", http.StatusBadRequest},
		{"NotFound", http.StatusNotFound},
		{"InternalServerError", http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}

			loggerMiddleware := LoggerMiddleware(logger)
			wrappedHandler := loggerMiddleware(handler)

			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/test", nil)

			wrappedHandler(w, r)

			if w.Code != tt.statusCode {
				t.Errorf("status code = %d, want %d", w.Code, tt.statusCode)
			}
		})
	}
}

func TestLoggerMiddlewareDefaultStatus(t *testing.T) {
	logger := NewLogger(slog.LevelInfo)

	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	}

	loggerMiddleware := LoggerMiddleware(logger)
	wrappedHandler := loggerMiddleware(handler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)

	wrappedHandler(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("default status code should be 200 OK")
	}
}
