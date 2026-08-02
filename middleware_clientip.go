package dim

import "net/http"

// ClientIP meresolusi IP address klien satu kali di awal request dan menyimpannya
// di request context, sehingga seluruh komponen hilir — rate limiting, logging,
// audit trail, GetClientIP, dan Ctx.ClientIP() — membaca nilai yang sama.
//
// Tanpa middleware ini, GetClientIP mengembalikan RemoteAddr. Itu adalah default
// yang aman: aplikasi yang terjangkau langsung tetap benar tanpa konfigurasi apa pun.
//
// Pasang middleware ini bila aplikasi berjalan di belakang proxy tepercaya dan
// membutuhkan IP asli klien. Pasang sedini mungkin dalam chain, sebelum middleware
// lain yang membaca IP (misalnya RateLimit dan LoggerMiddleware).
//
// KEAMANAN: TrustedProxyCount > 0 hanya aman bila aplikasi tidak dapat dihubungi
// langsung, yakni seluruh trafik wajib melewati proxy tepercaya. Bila origin masih
// terjangkau langsung, klien dapat menyusun sendiri isi X-Forwarded-For dan
// memalsukan IP-nya.
//
// Parameters:
//   - config: ClientIPConfig yang menentukan jumlah hop proxy tepercaya
//
// Returns:
//   - MiddlewareFunc: middleware yang menyimpan client IP ke request context
//
// Example:
//
//	cfg, _ := dim.LoadConfig()
//
//	router := dim.NewRouter()
//	router.Use(dim.ClientIP(cfg.ClientIP))          // pasang paling awal
//	router.Use(dim.RateLimit(cfg.RateLimit))        // ikut memakai IP hasil resolusi
func ClientIP(config ClientIPConfig) MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			clientIP := GetClientIPWithTrustedProxies(r, config.TrustedProxyCount)
			next(w, SetClientIP(r, clientIP))
		}
	}
}
