package dim

import (
	"net/http"
	"time"
)

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

			// Wrap response writer. wrapResponseWriter mempertahankan antarmuka
			// opsional milik w (Flusher, Hijacker, ReaderFrom) supaya handler
			// streaming dan upgrade WebSocket tetap berfungsi di balik middleware.
			wrapped, rw := wrapResponseWriter(w)

			next(wrapped, r)

			duration := time.Since(start)

			attrs := []any{
				"request_id", requestID,
				"method", r.Method,
				"path", r.RequestURI,
				"status", rw.statusCode,
				"duration_ms", duration.Milliseconds(),
			}

			// Setelah Hijack, koneksi keluar dari kendali net/http dan status yang
			// tercatat tidak lagi mewakili apa pun yang dikirim ke klien.
			if rw.hijacked {
				attrs = append(attrs, "hijacked", true)
			}

			logger.Info("request completed", attrs...)
		}
	}
}
