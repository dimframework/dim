package dim

import (
	"net/http"
	"time"
)

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

// WriteHeader captures the status code
func (rw *responseWriter) WriteHeader(statusCode int) {
	if !rw.written {
		rw.statusCode = statusCode
		rw.written = true
		rw.ResponseWriter.WriteHeader(statusCode)
	}
}

// Write captures writes
func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.statusCode = http.StatusOK
		rw.written = true
	}
	return rw.ResponseWriter.Write(b)
}

// LoggerMiddleware membuat middleware yang log HTTP requests dan responses.
// Middleware ini:
// 1. Generate unique request ID dan set di context untuk request tracing
// 2. Wrap response writer untuk capture response status code
// 3. Measure request duration
// 4. Log request details termasuk method, path, status code, dan duration
// Berguna untuk debugging, monitoring, dan audit trail.
//
// Parameters:
//   - logger: *Logger untuk menulis log entries
//
// Returns:
//   - MiddlewareFunc: middleware function yang log request/response
//
// Example:
//
//	logger := NewLogger(slog.LevelInfo)
//	router.Use(LoggerMiddleware(logger))
//	// Log output: time=... level=INFO msg="request completed" request_id=abc123 method=GET path=/users status=200 duration_ms=45
func LoggerMiddleware(logger *Logger) MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Generate request ID and set it in context
			requestID, _ := GenerateSecureToken(16)
			r = SetRequestID(r, requestID)

			// Wrap response writer
			rw := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			next(rw, r)

			duration := time.Since(start)

			// Log the request
			logger.Info("request completed",
				"request_id", requestID,
				"method", r.Method,
				"path", r.RequestURI,
				"status", rw.statusCode,
				"duration_ms", duration.Milliseconds(),
			)
		}
	}
}
