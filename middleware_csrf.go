package dim

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

// CSRFMiddleware membuat middleware yang handle CSRF (Cross-Site Request Forgery) protection.
// Middleware ini verify CSRF token untuk unsafe HTTP methods (POST, PUT, DELETE, PATCH).
// Safe methods (GET, HEAD, OPTIONS) dan exempt paths di-skip dari CSRF check.
// Token divalidasi dengan membandingkan value dari header/form dengan value dari cookie.
// Mengembalikan 403 Forbidden jika token tidak valid atau tidak match.
//
// Parameters:
//   - config: CSRFConfig yang berisi enabled status, header name, cookie name, exempt paths
//
// Returns:
//   - MiddlewareFunc: middleware function yang handle CSRF protection
//
// Example:
//
//	csrfConfig := CSRFConfig{
//	  Enabled: true,
//	  HeaderName: "X-CSRF-Token",
//	  CookieName: "_csrf",
//	  ExemptPaths: []string{"/api/public/*"},
//	}
//	router.Use(CSRFMiddleware(csrfConfig))
func CSRFMiddleware(config CSRFConfig) MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Skip CSRF check for safe methods and exempt paths
			if !config.Enabled || IsSafeHttpMethod(r.Method) || PathMatches(r.URL.Path, config.ExemptPaths) {
				next(w, r)
				return
			}

			// For unsafe methods, verify CSRF token
			token := GetCSRFToken(r, config.HeaderName)
			cookieToken := GetCookie(r, config.CookieName)

			if token == "" || cookieToken == "" || token != cookieToken {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				JsonError(w, http.StatusForbidden, "Validasi token CSRF gagal", nil)
				return
			}

			next(w, r)
		}
	}
}

// GenerateCSRFToken menghasilkan token CSRF baru dengan secure random bytes.
// Token di-encode sebagai hex string dengan specified length.
// Token ini harus disimpan di cookie dan dikirim di request header atau form untuk verification.
//
// Parameters:
//   - length: jumlah random bytes untuk generate (contoh: 32)
//
// Returns:
//   - string: hex-encoded CSRF token
//   - error: error jika random byte generation gagal
//
// Example:
//
//	token, err := GenerateCSRFToken(32)
//	if err != nil {
//	  return err
//	}
//	SetCSRFToken(w, token, csrfConfig)
func GenerateCSRFToken(length int) (string, error) {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// SetCSRFToken menyimpan CSRF token dalam cookie response.
// Cookie di-set dengan HttpOnly=false sehingga accessible dari JavaScript.
// SameSite=Lax digunakan untuk prevent CSRF attacks sambil tetap allow cross-site form submissions.
//
// Parameters:
//   - w: http.ResponseWriter untuk menulis cookie
//   - token: CSRF token string yang akan disimpan
//   - config: CSRFConfig yang berisi cookie configuration
//
// Example:
//
//	token, _ := GenerateCSRFToken(32)
//	SetCSRFToken(w, token, csrfConfig)
func SetCSRFToken(w http.ResponseWriter, token string, config CSRFConfig) {
	cookie := &http.Cookie{
		Name:     config.CookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: false, // Must be accessible from JavaScript
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)
}

// GetCSRFToken mengekstrak CSRF token dari request dengan mencek multiple sources.
// Cek dilakukan dalam urutan: header (X-CSRF-Token), form data (_csrf field).
// Returns empty string jika token tidak ditemukan di manapun.
// Header check diprioritaskan untuk API requests, form data untuk traditional forms.
//
// Parameters:
//   - r: *http.Request yang akan di-check token-nya
//   - headerName: nama header untuk cek CSRF token (contoh: X-CSRF-Token)
//
// Returns:
//   - string: CSRF token jika ditemukan, empty string jika tidak ada
//
// Example:
//
//	token := GetCSRFToken(req, "X-CSRF-Token")
//	if token == "" {
//	  return JsonError(w, 400, "Token CSRF diperlukan", nil)
//	}
func GetCSRFToken(r *http.Request, headerName string) string {
	// Try to get from header first
	if token := r.Header.Get(headerName); token != "" {
		return token
	}

	// Try to get from form data
	if err := r.ParseForm(); err == nil {
		if token := r.FormValue("_csrf"); token != "" {
			return token
		}
	}

	return ""
}
