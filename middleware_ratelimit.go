package dim

import (
	"log/slog"
	"math"
	"net/http"
	"time"
)

// RateLimit membuat middleware yang menerapkan pembatasan kecepatan (rate limiting).
// Middleware ini mencegah penyalahgunaan API dengan membatasi jumlah request per IP,
// per User, atau per kunci yang ditentukan pemanggil lewat KeyFunc.
//
// Setiap limiter memakai ember bernama config.Name. Dua pemanggilan RateLimit yang
// memakai store yang sama tetapi Name berbeda tidak saling mengisi ember; tanpa Name,
// semuanya berbagi satu ember per alamat — perilaku lama, dipertahankan untuk aplikasi
// yang memasang satu RateLimit global.
//
// Bila config.KeyFunc diisi, hanya ember kunci itu yang diperiksa (dengan PerKey);
// PerIP dan PerUser diabaikan. KeyFunc yang mengembalikan string kosong dijawab 500,
// karena itu berarti middleware dipasang sebelum apa yang menyediakan kuncinya.
//
// Request yang ditolak dijawab 429 dengan Retry-After dari sisa waktu ember bila store
// mengimplementasikan RetryAfterRateLimitStore, selain itu dari ResetPeriod. Setiap
// penolakan dicatat lewat slog pada level Warn, supaya ambang dapat disetel dari log.
//
// Parameters:
//   - config: Struct RateLimitConfig yang berisi aturan limit.
//   - store: (Opsional) Backend storage custom via variadic parameter.
//     Jika kosong, menggunakan InMemoryRateLimitStore.
//     Gunakan NewDatabaseRateLimitStore(db) untuk persistensi database.
//
// Returns:
//   - MiddlewareFunc: Middleware function untuk router.
//
// Example:
//
//	// Satu limiter global
//	router.Use(dim.RateLimit(config))
//
//	// Dua limiter per rute dengan ember terpisah
//	store := dim.NewDatabaseRateLimitStore(db)
//	router.Post("/login", login, dim.RateLimit(dim.RateLimitConfig{
//		Enabled: true, Name: "login", PerIP: 5, ResetPeriod: 15 * time.Minute,
//	}, store))
//	router.Post("/donate", donate, dim.RateLimit(dim.RateLimitConfig{
//		Enabled: true, Name: "donate", PerIP: 100, ResetPeriod: 15 * time.Minute,
//	}, store))
//
//	// Ember per akun, dipasang sesudah middleware yang mengisi akunnya
//	router.Post("/profile", profile, requireAccount, dim.RateLimit(dim.RateLimitConfig{
//		Enabled: true, PerKey: 300, ResetPeriod: 15 * time.Minute,
//		KeyFunc: func(r *http.Request) string { return "account:" + accountIDFrom(r) },
//	}, store))
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
			if config.KeyFunc != nil {
				key := config.KeyFunc(r)
				if key == "" {
					slog.Default().ErrorContext(r.Context(), "kunci rate limit kosong, middleware berjalan sebelum penyedia kuncinya",
						"name", config.Name,
						"path", r.URL.Path,
					)
					InternalServerError(w, "Terjadi kesalahan pada server")
					return
				}
				if !rateLimitPasses(w, r, limiter, config, "key", key, config.PerKey) {
					return
				}
				next(w, r)
				return
			}

			if !rateLimitPasses(w, r, limiter, config, "ip", limiter.IPKey(GetClientIP(r)), config.PerIP) {
				return
			}

			if user, ok := GetUser(r); ok {
				if !rateLimitPasses(w, r, limiter, config, "user", limiter.UserKey(user.GetID()), config.PerUser) {
					return
				}
			}

			next(w, r)
		}
	}
}

// rateLimitPasses memeriksa satu ember dan menulis 429 bila ditolak.
// Mengembalikan true bila request boleh lanjut.
func rateLimitPasses(w http.ResponseWriter, r *http.Request, limiter *RateLimiter, config RateLimitConfig, scope, key string, limit int) bool {
	allowed, retryAfter, err := limiter.CheckKeyLimit(r.Context(), key, limit)
	if err != nil {
		// Fail open: jika store error, request tetap diteruskan agar API tidak
		// ikut tumbang saat cache/DB bermasalah. Tetapi kegagalan ini WAJIB
		// terlihat — rate limit yang mati diam-diam berarti proteksi brute
		// force hilang tanpa ada yang tahu.
		slog.Default().ErrorContext(r.Context(), "rate limit store gagal, request diteruskan tanpa pembatasan",
			"name", config.Name,
			"scope", scope,
			"path", r.URL.Path,
			"error", err,
		)
		return true
	}
	if allowed {
		return true
	}

	seconds := retryAfterSeconds(retryAfter)
	slog.Default().WarnContext(r.Context(), "rate limit terlampaui",
		"name", config.Name,
		"scope", scope,
		"path", r.URL.Path,
		"retry_after_seconds", seconds,
	)
	TooManyRequests(w, seconds)
	return false
}

// retryAfterSeconds membulatkan durasi ke atas ke detik penuh, minimal 1.
// Retry-After bernilai 0 berarti "coba lagi sekarang", yang untuk request yang
// baru saja ditolak selalu salah.
func retryAfterSeconds(d time.Duration) int {
	seconds := int(math.Ceil(d.Seconds()))
	if seconds < 1 {
		return 1
	}
	return seconds
}
