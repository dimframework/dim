package dim

import (
	"fmt"
	"net/http"
)

// Recovery membuat middleware yang recover dari panics dan log mereka.
// Middleware ini:
// 1. Catch panic yang terjadi di handler atau downstream middleware
// 2. Log panic error dengan request details (path, method) untuk debugging
// 3. Return 500 Internal Server Error response ke client
// 4. Prevent application crash dan memastikan graceful error handling
// Berguna untuk production safety dan error monitoring.
//
// Parameters:
//   - logger: *Logger untuk menulis panic error logs
//
// Returns:
//   - MiddlewareFunc: middleware function yang recover dari panics
//
// Example:
//
//	logger := NewLogger(slog.LevelError)
//	router.Use(Recovery(logger))
//	// Jika ada panic di handler, akan logged dan 500 response dikirim ke client
func Recovery(logger *Logger) MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.Error("panic recovered",
						"error", fmt.Sprintf("%v", err),
						"path", r.RequestURI,
						"method", r.Method,
					)

					// Set status code and return error response
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)

					JsonError(w, http.StatusInternalServerError, "Kesalahan server internal", nil)
				}
			}()

			next(w, r)
		}
	}
}
