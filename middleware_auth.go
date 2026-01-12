package dim

import (
	"net/http"
)

// ExpectBearerToken adalah middleware yang hanya memeriksa keberadaan header `Authorization: Bearer <token>`.
// **TIDAK AMAN**: Middleware ini TIDAK memverifikasi validitas token itu sendiri.
// Gunakan ini hanya untuk kasus penggunaan lanjutan di mana verifikasi dilakukan secara manual di tempat lain.
// Untuk keamanan, selalu prioritaskan penggunaan `RequireAuth`.
func ExpectBearerToken() MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			_, ok := GetAuthToken(r)
			if !ok {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				JsonError(w, http.StatusUnauthorized, "Header otorisasi hilang atau tidak valid", nil)
				return
			}

			// Note: Token verification should be done at application level
			// This middleware just extracts and validates the presence of the token
			// The actual JWT verification happens in the handler or service layer

			next(w, r)
		}
	}
}

// AllowBearerToken adalah middleware pasif yang tidak melakukan apa-apa.
// Tujuannya adalah untuk secara eksplisit menandai sebuah rute yang memperbolehkan
// header Authorization, meskipun tidak divalidasi di tingkat middleware.
// Penggunaannya sangat jarang dan untuk kasus yang sangat spesifik.
func AllowBearerToken() MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Token is optional, so we don't fail if it's missing
			// Just extract it if present
			_, _ = GetAuthToken(r)

			next(w, r)
		}
	}
}

// RequireAuth adalah middleware yang aman dan direkomendasikan untuk mewajibkan dan memverifikasi token JWT yang valid.
// Middleware ini menggunakan JWTManager untuk memvalidasi token dan menempatkan info pengguna ke dalam konteks.
// Mengembalikan 401 Unauthorized jika token tidak ada, tidak valid, atau kedaluwarsa.
//
// Parameters:
//   - jwtManager: *JWTManager untuk verifikasi token.
//
// Returns:
//   - MiddlewareFunc: Middleware yang memberlakukan autentikasi aman.
//
// Example:
//
//	router.Get("/protected", handler, RequireAuth(jwtManager))
//	// Di dalam handler, gunakan GetUser(req) untuk mendapatkan pengguna yang terautentikasi.
func RequireAuth(jwtManager *JWTManager) MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			token, ok := GetAuthToken(r)
			if !ok {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				JsonError(w, http.StatusUnauthorized, "Header otorisasi hilang atau tidak valid", nil)
				return
			}

			// Verify token
			claims, err := jwtManager.VerifyToken(token)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				JsonError(w, http.StatusUnauthorized, "Token tidak valid atau telah kadaluarsa", nil)
				return
			}

			// Set user in context
			user := &User{
				ID:    claims.UserID,
				Email: claims.Email,
			}
			r = SetUser(r, user)

			next(w, r)
		}
	}
}

// OptionalAuth (sebelumnya OptionalAuthWithManager) adalah middleware yang aman dan direkomendasikan
// untuk secara opsional memverifikasi token JWT jika ada.
// Middleware ini tidak akan gagal jika token tidak ada atau tidak valid.
// Jika token valid, info pengguna akan ditempatkan di konteks.
// Berguna untuk endpoint yang mendukung konteks pengguna opsional.
//
// Parameters:
//   - jwtManager: *JWTManager untuk verifikasi token.
//
// Returns:
//   - MiddlewareFunc: Middleware yang memungkinkan autentikasi opsional dengan verifikasi.
//
// Example:
//
//	router.Get("/semi-protected", handler, OptionalAuth(jwtManager))
//	// Di dalam handler: user, ok := GetUser(req); if ok { /* authenticated */ }
func OptionalAuth(jwtManager *JWTManager) MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			token, ok := GetAuthToken(r)
			if ok {
				// Try to verify token
				if claims, err := jwtManager.VerifyToken(token); err == nil {
					// Token is valid, set user in context
					user := &User{
						ID:    claims.UserID,
						Email: claims.Email,
					}
					r = SetUser(r, user)
				}
			}

			next(w, r)
		}
	}
}
