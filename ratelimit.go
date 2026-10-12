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
	name        string
	perIP       int
	perUser     int
	resetPeriod time.Duration
}

// NewRateLimiter membuat instance RateLimiter baru.
//
// Parameters:
//   - config: Konfigurasi limit (Name, PerIP, PerUser, ResetPeriod).
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
		name:        config.Name,
		perIP:       config.PerIP,
		perUser:     config.PerUser,
		resetPeriod: config.ResetPeriod,
	}
}

// IPKey membentuk kunci ember IP: "ip:<ip>" tanpa Name, "ip:<name>:<ip>" dengan Name.
//
// Dua limiter dengan Name berbeda tidak pernah berbagi ember meski memakai store
// dan alamat yang sama; dua limiter tanpa Name berbagi, persis seperti sebelum Name ada.
func (rl *RateLimiter) IPKey(ip string) string {
	if rl.name == "" {
		return fmt.Sprintf("ip:%s", ip)
	}
	return fmt.Sprintf("ip:%s:%s", rl.name, ip)
}

// UserKey membentuk kunci ember user: "user:<id>" tanpa Name, "user:<name>:<id>" dengan Name.
func (rl *RateLimiter) UserKey(userID string) string {
	if rl.name == "" {
		return fmt.Sprintf("user:%s", userID)
	}
	return fmt.Sprintf("user:%s:%s", rl.name, userID)
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
	allowed, _, err := rl.CheckKeyLimit(ctx, rl.IPKey(ip), rl.perIP)
	return allowed, err
}

// CheckUserLimit mengecek apakah user dalam batas rate limit.
//
// Parameters:
//   - ctx: context untuk operasi
//   - userKey: unique identifier user (misal: "user:123"); dipakai apa adanya sebagai kunci
//
// Returns:
//   - bool: true jika diizinkan, false jika limit terlampaui
//   - error: error dari storage backend
func (rl *RateLimiter) CheckUserLimit(ctx context.Context, userKey string) (bool, error) {
	allowed, _, err := rl.CheckKeyLimit(ctx, userKey, rl.perUser)
	return allowed, err
}

// CheckKeyLimit mengecek kunci apa pun terhadap limit yang diberikan dan mengembalikan
// berapa lama klien harus menunggu bila ditolak.
//
// retryAfter diambil dari store bila store mengimplementasikan RetryAfterRateLimitStore;
// selain itu dipakai ResetPeriod — batas atas yang aman, bukan sisa waktu sebenarnya.
// Saat diizinkan, retryAfter selalu 0.
//
// Parameters:
//   - ctx: context untuk operasi
//   - key: kunci ember, dipakai apa adanya
//   - limit: jumlah request maksimum per window untuk kunci ini
//
// Returns:
//   - allowed: true jika diizinkan
//   - retryAfter: durasi tunggu yang disarankan bila ditolak
//   - err: error dari storage backend
func (rl *RateLimiter) CheckKeyLimit(ctx context.Context, key string, limit int) (allowed bool, retryAfter time.Duration, err error) {
	if withRetry, ok := rl.store.(RetryAfterRateLimitStore); ok {
		allowed, retryAfter, err = withRetry.AllowWithRetryAfter(ctx, key, limit, rl.resetPeriod)
	} else {
		allowed, err = rl.store.Allow(ctx, key, limit, rl.resetPeriod)
		retryAfter = rl.resetPeriod
	}
	if err != nil || allowed {
		return allowed, 0, err
	}
	return false, retryAfter, nil
}
