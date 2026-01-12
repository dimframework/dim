package dim

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/atfromhome/goreus/pkg/cache"
)

// RateLimiter handles rate limiting using Goreus Cache
type RateLimiter struct {
	cache       *cache.InMemoryCache[string, int]
	perIP       int
	perUser     int
	resetPeriod time.Duration
	ctx         context.Context
	mu          sync.Mutex
}

// NewRateLimiter membuat RateLimiter baru menggunakan in-memory cache.
// Cache digunakan untuk menyimpan request count per IP dan per user.
// ResetPeriod menentukan kapan counter di-reset ke 0 (contoh: 1 menit).
//
// Parameters:
//   - config: RateLimitConfig yang berisi enabled status, per-IP limit, per-user limit, reset period
//
// Returns:
//   - *RateLimiter: rate limiter instance yang siap digunakan
//
// Example:
//
//	config := RateLimitConfig{
//	  Enabled: true,
//	  PerIP: 100,
//	  PerUser: 1000,
//	  ResetPeriod: time.Minute,
//	}
//	limiter := NewRateLimiter(config)
func NewRateLimiter(config RateLimitConfig) *RateLimiter {
	return &RateLimiter{
		cache:       cache.NewInMemoryCache[string, int](10000, config.ResetPeriod),
		perIP:       config.PerIP,
		perUser:     config.PerUser,
		resetPeriod: config.ResetPeriod,
		ctx:         context.Background(),
	}
}

// RateLimit membuat middleware yang apply rate limiting berdasarkan IP dan user.
// Middleware ini:
// 1. Check request count per IP terhadap configured limit (jika enabled)
// 2. Check request count per authenticated user terhadap configured limit (jika user authenticated)
// 3. Return 429 Too Many Requests jika limit terlampaui
// 4. Set Retry-After header untuk indicate client kapan bisa retry
// Berguna untuk protect API dari abuse dan denial-of-service attacks.
//
// Parameters:
//   - config: RateLimitConfig untuk rate limiting configuration
//
// Returns:
//   - MiddlewareFunc: middleware function yang apply rate limiting
//
// Example:
//
//	config := RateLimitConfig{Enabled: true, PerIP: 100, PerUser: 1000, ResetPeriod: time.Minute}
//	router.Use(RateLimit(config))
func RateLimit(config RateLimitConfig) MiddlewareFunc {
	if !config.Enabled {
		// If disabled, just pass through
		return func(next HandlerFunc) HandlerFunc {
			return next
		}
	}

	limiter := NewRateLimiter(config)

	return func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			clientIP := GetClientIP(r)

			// Check IP rate limit
			if !limiter.CheckIPLimit(clientIP) {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", fmt.Sprintf("%d", int(config.ResetPeriod.Seconds())))
				w.WriteHeader(http.StatusTooManyRequests)
				JsonError(w, http.StatusTooManyRequests, "Batas tingkat permintaan terlampaui", nil)
				return
			}

			// Check user rate limit if authenticated
			user, ok := GetUser(r)
			if ok {
				userKey := fmt.Sprintf("user:%d", user.ID)
				if !limiter.CheckUserLimit(userKey) {
					w.Header().Set("Content-Type", "application/json")
					w.Header().Set("Retry-After", fmt.Sprintf("%d", int(config.ResetPeriod.Seconds())))
					w.WriteHeader(http.StatusTooManyRequests)
					JsonError(w, http.StatusTooManyRequests, "Batas tingkat permintaan terlampaui", nil)
					return
				}
			}

			next(w, r)
		}
	}
}

// CheckIPLimit mengecek apakah IP dalam batas rate limit.
// Increment counter untuk IP dan return true jika masih dalam limit, false jika exceeded.
// Format key: "ip:<ip_address>"
//
// Parameters:
//   - ip: client IP address string
//
// Returns:
//   - bool: true jika request allowed, false jika rate limit exceeded
//
// Example:
//
//	if !limiter.CheckIPLimit("192.168.1.1") {
//	  // Rate limit exceeded for this IP
//	}
func (rl *RateLimiter) CheckIPLimit(ip string) bool {
	key := fmt.Sprintf("ip:%s", ip)
	return rl.checkLimit(key, rl.perIP)
}

// CheckUserLimit mengecek apakah user dalam batas rate limit.
// Increment counter untuk user dan return true jika masih dalam limit, false jika exceeded.
// Format key: "user:<user_id>"
//
// Parameters:
//   - userKey: user key string (contoh: "user:123")
//
// Returns:
//   - bool: true jika request allowed, false jika rate limit exceeded
//
// Example:
//
//	if !limiter.CheckUserLimit("user:456") {
//	  // Rate limit exceeded for this user
//	}
func (rl *RateLimiter) CheckUserLimit(userKey string) bool {
	return rl.checkLimit(userKey, rl.perUser)
}

// checkLimit checks if a key is within the rate limit
func (rl *RateLimiter) checkLimit(key string, limit int) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	count, exists := rl.cache.Get(rl.ctx, key)
	if !exists {
		count = 0
	}

	count++
	rl.cache.Set(rl.ctx, key, count)

	return count <= limit
}

// Reset clears the rate limiter cache
func (rl *RateLimiter) Reset() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.cache.Close()
	rl.cache = cache.NewInMemoryCache[string, int](10000, rl.resetPeriod)
}

// GetLimit returns the current limit count for a key
func (rl *RateLimiter) GetLimit(key string) int {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	count, exists := rl.cache.Get(rl.ctx, key)
	if !exists {
		return 0
	}
	return count
}
