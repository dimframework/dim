package dim

import (
	"context"
	"fmt"
	"time"
)

// RateLimiter menangani logika rate limiting dengan backend storage yang dapat dikonfigurasi.
// Memisahkan logika bisnis dari implementasi middleware.
type RateLimiter struct {
	store       RateLimitStore
	perIP       int
	perUser     int
	resetPeriod time.Duration
}

// NewRateLimiter membuat instance RateLimiter baru.
//
// Parameters:
//   - config: Konfigurasi limit (PerIP, PerUser, ResetPeriod).
//   - store: Backend storage (opsional). Jika nil, akan menggunakan InMemoryRateLimitStore default.
//
// Returns:
//   - *RateLimiter: Instance rate limiter yang siap digunakan.
//
// Example:
//
//	// In-memory
//	limiter := NewRateLimiter(config, nil)
//
//	// Postgres
//	store := NewPostgresRateLimitStore(db)
//	limiter := NewRateLimiter(config, store)
func NewRateLimiter(config RateLimitConfig, store RateLimitStore) *RateLimiter {
	if store == nil {
		store = NewInMemoryRateLimitStore(config.ResetPeriod)
	}
	return &RateLimiter{
		store:       store,
		perIP:       config.PerIP,
		perUser:     config.PerUser,
		resetPeriod: config.ResetPeriod,
	}
}

// CheckIPLimit mengecek apakah IP dalam batas rate limit.
//
// Parameters:
//   - ctx: context untuk operasi
//   - ip: alamat IP client
//
// Returns:
//   - bool: true jika diizinkan, false jika limit terlampaui
//   - error: error dari storage backend
func (rl *RateLimiter) CheckIPLimit(ctx context.Context, ip string) (bool, error) {
	key := fmt.Sprintf("ip:%s", ip)
	return rl.store.Allow(ctx, key, rl.perIP, rl.resetPeriod)
}

// CheckUserLimit mengecek apakah user dalam batas rate limit.
//
// Parameters:
//   - ctx: context untuk operasi
//   - userKey: unique identifier user (misal: "user:123")
//
// Returns:
//   - bool: true jika diizinkan, false jika limit terlampaui
//   - error: error dari storage backend
func (rl *RateLimiter) CheckUserLimit(ctx context.Context, userKey string) (bool, error) {
	return rl.store.Allow(ctx, userKey, rl.perUser, rl.resetPeriod)
}
