package dim

import (
	"net/http"
	"strconv"
	"strings"
)

// CORS membuat middleware yang handle Cross-Origin Resource Sharing (CORS).
// Middleware ini set CORS headers untuk allow cross-origin requests dari specified origins.
// Support preflight requests (OPTIONS method) dan credential requests.
// Origin checking dilakukan dengan exact match atau wildcard (*).
//
// Parameters:
//   - config: CORSConfig yang berisi allowed origins, methods, headers, credentials setting
//
// Returns:
//   - MiddlewareFunc: middleware function yang handle CORS
//
// Example:
//
//	corsConfig := CORSConfig{
//	  AllowedOrigins: []string{"https://example.com", "https://app.example.com"},
//	  AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
//	  AllowedHeaders: []string{"Content-Type", "Authorization"},
//	  AllowCredentials: true,
//	  MaxAge: 3600,
//	}
//	router.Use(CORS(corsConfig))
func CORS(config CORSConfig) MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if origin is allowed
			isAllowed := isOriginAllowed(origin, config.AllowedOrigins)

			if isAllowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Add("Vary", "Origin")

				if config.AllowCredentials {
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				}

				w.Header().Set("Access-Control-Allow-Methods", strings.Join(config.AllowedMethods, ", "))
				w.Header().Set("Access-Control-Allow-Headers", strings.Join(config.AllowedHeaders, ", "))

				if len(config.ExposedHeaders) > 0 {
					w.Header().Set("Access-Control-Expose-Headers", strings.Join(config.ExposedHeaders, ", "))
				}

				if config.MaxAge > 0 {
					w.Header().Set("Access-Control-Max-Age", strconv.Itoa(config.MaxAge))
				}
			}

			// Handle preflight requests
			// Hanya intercept jika method OPTIONS DAN memiliki header Origin (indikasi CORS preflight)
			if r.Method == http.MethodOptions && origin != "" {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next(w, r)
		}
	}
}

// isOriginAllowed mengecek apakah origin yang diberikan ada dalam whitelist allowed origins.
// Mendukung exact match atau wildcard (*) untuk allow semua origins.
// Returns true jika origin diizinkan, false sebaliknya.
func isOriginAllowed(origin string, allowedOrigins []string) bool {
	if origin == "" {
		return false
	}

	for _, allowed := range allowedOrigins {
		if allowed == "*" {
			return true
		}
		if allowed == origin {
			return true
		}
	}

	return false
}
