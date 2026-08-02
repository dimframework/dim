package dim

import (
	"fmt"
	"net/http"
)

// RateLimit membuat middleware yang menerapkan pembatasan kecepatan (rate limiting).
// Middleware ini mencegah penyalahgunaan API dengan membatasi jumlah request per IP atau per User.
//
// Parameters:
//   - config: Struct RateLimitConfig yang berisi aturan limit.
//   - store: (Opsional) Backend storage custom via variadic parameter.
//     Jika kosong, menggunakan InMemoryRateLimitStore.
//     Gunakan NewPostgresRateLimitStore(db) untuk persistensi database.
//
// Returns:
//   - MiddlewareFunc: Middleware function untuk router.
//
// Example:
//
//	// Default In-Memory
//	router.Use(dim.RateLimit(config))
//
//	// Dengan Postgres Store
//	store := dim.NewPostgresRateLimitStore(db)
//	router.Use(dim.RateLimit(config, store))
func RateLimit(config RateLimitConfig, store ...RateLimitStore) MiddlewareFunc {
	if !config.Enabled {
		return func(next HandlerFunc) HandlerFunc {
			return next
		}
	}

	var s RateLimitStore
	if len(store) > 0 {
		s = store[0]
	}

	limiter := NewRateLimiter(config, s)

	return func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			clientIP := GetClientIP(r)

			// Check IP rate limit
			allowed, err := limiter.CheckIPLimit(ctx, clientIP)
			if err != nil {
				// Fail open: Jika store error, biarkan request lewat tapi log error (jika ada logger)
				// Strategi ini mencegah downtime API gara-gara cache/DB down.
			} else if !allowed {
				TooManyRequests(w, int(config.ResetPeriod.Seconds()))
				return
			}

			// Check user rate limit if authenticated
			user, ok := GetUser(r)
			if ok {
				userKey := fmt.Sprintf("user:%s", user.GetID())
				allowed, err := limiter.CheckUserLimit(ctx, userKey)
				if err == nil && !allowed {
					TooManyRequests(w, int(config.ResetPeriod.Seconds()))
					return
				}
			}

			next(w, r)
		}
	}
}
