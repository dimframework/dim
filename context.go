package dim

import (
	"context"
	"net/http"
	"strings"
)

// Context keys
type contextKey string

const (
	userKey      contextKey = "user"
	requestIDKey contextKey = "request_id"
	routeKey     contextKey = "route"
	clientIPKey  contextKey = "client_ip"
)

// SetUser menyimpan user object ke dalam request context.
// Berguna untuk menyimpan authenticated user info yang dapat diakses di handlers.
// Returns request baru dengan updated context.
//
// Parameters:
//   - r: *http.Request request yang akan diupdate contextnya
//   - user: Authenticatable object yang akan disimpan
//
// Returns:
//   - *http.Request: request baru dengan user disimpan di context
//
// Example:
//
//	req = SetUser(req, user)
//	user, ok := GetUser(req)
func SetUser(r *http.Request, user Authenticatable) *http.Request {
	ctx := context.WithValue(r.Context(), userKey, user)
	return r.WithContext(ctx)
}

// GetUser mengambil user object dari request context.
// Returns user dan boolean indicating apakah user ada di context.
// Returns nil user dan false jika user tidak ditemukan.
//
// Parameters:
//   - r: *http.Request request yang di-check contextnya
//
// Returns:
//   - Authenticatable: user object dari context, nil jika tidak ada
//   - bool: true jika user ada, false jika tidak ada
//
// Example:
//
//	user, ok := GetUser(req)
//	if !ok {
//	  return JsonError(w, 401, "Tidak authorized", nil)
//	}
func GetUser(r *http.Request) (Authenticatable, bool) {
	user, ok := r.Context().Value(userKey).(Authenticatable)
	return user, ok
}

// GetClaims mengambil custom claims dari request context jika user terotentikasi.
// Memeriksa apakah user mengimplementasikan method GetClaims() map[string]interface{}.
//
// Parameters:
//   - r: *http.Request request yang akan diperiksa
//
// Returns:
//   - map[string]interface{}: map claims, atau nil jika tidak ada atau user tidak punya claims
func GetClaims(r *http.Request) map[string]interface{} {
	user, ok := GetUser(r)
	if !ok {
		return nil
	}

	// Check if user has GetClaims method
	if claimsUser, ok := user.(interface{ GetClaims() map[string]interface{} }); ok {
		return claimsUser.GetClaims()
	}

	return nil
}

// SetRequestID menyimpan unique request ID ke dalam context.
// Request ID berguna untuk logging dan tracing requests across systems.
// Biasanya di-set oleh logger middleware di awal request processing.
//
// Parameters:
//   - r: *http.Request request yang akan diupdate contextnya
//   - requestID: string unique identifier untuk request ini
//
// Returns:
//   - *http.Request: request baru dengan requestID disimpan di context
//
// Example:
//
//	requestID := GenerateSecureToken(16)
//	req = SetRequestID(req, requestID)
func SetRequestID(r *http.Request, requestID string) *http.Request {
	ctx := context.WithValue(r.Context(), requestIDKey, requestID)
	return r.WithContext(ctx)
}

// GetRequestID mengambil request ID dari context.
// Returns empty string jika request ID tidak ditemukan.
// Gunakan value ini untuk logging dan request tracing.
//
// Parameters:
//   - r: *http.Request request yang di-check contextnya
//
// Returns:
//   - string: request ID, empty string jika tidak ada
//
// Example:
//
//	requestID := GetRequestID(req)
//	logger.Info("Processing request", "request_id", requestID)
func GetRequestID(r *http.Request) string {
	if requestID, ok := r.Context().Value(requestIDKey).(string); ok {
		return requestID
	}
	return ""
}

// SetClientIP menyimpan client IP address yang sudah diresolusi ke dalam context.
// Biasanya di-set oleh middleware ClientIP di awal request processing, sehingga
// seluruh komponen hilir (rate limit, logger, handler) membaca nilai yang sama.
//
// Parameters:
//   - r: *http.Request request yang akan diupdate contextnya
//   - clientIP: string IP address klien yang sudah diresolusi
//
// Returns:
//   - *http.Request: request baru dengan clientIP disimpan di context
//
// Example:
//
//	req = SetClientIP(req, GetClientIPWithTrustedProxies(req, 1))
func SetClientIP(r *http.Request, clientIP string) *http.Request {
	ctx := context.WithValue(r.Context(), clientIPKey, clientIP)
	return r.WithContext(ctx)
}

// clientIPFromContext mengambil client IP yang sudah diresolusi dari context.
// Returns empty string dan false jika middleware ClientIP belum dipasang.
func clientIPFromContext(r *http.Request) (string, bool) {
	clientIP, ok := r.Context().Value(clientIPKey).(string)
	return clientIP, ok && clientIP != ""
}

// routeParams holds URL parameter key-value pairs captured during tree traversal.
// Parallel slices (not a map) to avoid allocation overhead; last-added value wins
// for duplicate keys (supports nested routers).
type routeParams struct {
	keys []string
	vals []string
}

// routeMatch records what the router matched for one request.
//
// The router installs it before the global middleware chain runs and fills it
// in once a route is found, so middleware registered with Router.Use can read
// the matched pattern after next returns — the point where metrics and access
// logs are written. A value placed in the context during dispatch would only
// be visible downstream of the router, never to the middleware wrapping it.
//
// The buffers back params for up to four parameters, which covers most routes
// without a separate allocation.
type routeMatch struct {
	pattern string
	params  routeParams
	keysBuf [4]string
	valsBuf [4]string
	parent  *routeMatch // match of an enclosing Router when routers are nested
}

// withRouteMatch returns req carrying a fresh, empty routeMatch.
func withRouteMatch(req *http.Request) (*http.Request, *routeMatch) {
	m := &routeMatch{parent: routeMatchFrom(req)}
	m.params.keys = m.keysBuf[:0]
	m.params.vals = m.valsBuf[:0]
	return req.WithContext(context.WithValue(req.Context(), routeKey, m)), m
}

// routeMatchFrom returns the routeMatch installed by the router, or nil when
// the request never passed through Router.ServeHTTP.
func routeMatchFrom(r *http.Request) *routeMatch {
	m, _ := r.Context().Value(routeKey).(*routeMatch)
	return m
}

// GetRoutePattern mengembalikan pola rute yang cocok dengan request, persis
// seperti saat didaftarkan — termasuk prefix Group — misalnya "/users/{id}"
// untuk request ke /users/42.
//
// Dapat dibaca dari handler, dari middleware rute, dan dari middleware global
// (Router.Use) setelah next kembali. Gunakan nilai ini, bukan r.URL.Path,
// sebagai label metrik (satu deret per rute, bukan per nilai parameter) dan
// saat mencatat rute yang membawa rahasia di path-nya, seperti /reset/{token}.
//
// Mengembalikan string kosong bila tidak ada rute yang cocok (404, 405,
// redirect trailing slash), bila request dilayani Static/SPA, dan bila request
// tidak melewati Router sama sekali.
//
// Parameters:
//   - r: *http.Request request yang di-check
//
// Returns:
//   - string: pola rute yang cocok, empty string jika tidak ada
//
// Example:
//
//	func Metrics(next dim.HandlerFunc) dim.HandlerFunc {
//		return func(w http.ResponseWriter, r *http.Request) {
//			start := time.Now()
//			next(w, r)
//			route := dim.GetRoutePattern(r) // "/users/{id}"
//			requestDuration.WithLabelValues(r.Method, route).Observe(time.Since(start).Seconds())
//		}
//	}
func GetRoutePattern(r *http.Request) string {
	if m := routeMatchFrom(r); m != nil {
		return m.pattern
	}
	return ""
}

// GetParam mengambil single path parameter dari request.
// Membaca parameter yang ditangkap radix tree dengan fallback ke r.PathValue()
// untuk route yang dilayani langsung oleh http.ServeMux (Static/SPA).
// Returns empty string jika parameter tidak ditemukan.
//
// Parameters:
//   - r: *http.Request request yang di-check parameternya
//   - key: nama parameter yang akan diambil
//
// Returns:
//   - string: parameter value, empty string jika tidak ditemukan
//
// Example:
//
//	// Route: GET /users/{id}
//	userID := GetParam(req, "id")
func GetParam(r *http.Request, key string) string {
	// With nested routers, the innermost router that captured params answers.
	for m := routeMatchFrom(r); m != nil; m = m.parent {
		rp := &m.params
		if len(rp.keys) == 0 {
			continue
		}
		// Reverse iteration: last-added value wins (supports nested groups).
		for i := len(rp.keys) - 1; i >= 0; i-- {
			if rp.keys[i] == key {
				return rp.vals[i]
			}
		}
		break
	}
	// Fallback for mux-served routes (Static, SPA).
	return r.PathValue(key)
}

// GetQueryParam mengambil single query parameter dari request URL.
// Query parameters adalah bagian dari URL setelah "?" (contoh: ?name=value).
// Returns empty string jika parameter tidak ditemukan.
//
// Parameters:
//   - r: *http.Request request yang di-check query parameternya
//   - key: nama query parameter yang akan diambil
//
// Returns:
//   - string: query parameter value, empty string jika tidak ditemukan
//
// Example:
//
//	page := GetQueryParam(req, "page")  // dari URL: /users?page=2
func GetQueryParam(r *http.Request, key string) string {
	return r.URL.Query().Get(key)
}

// GetQueryParams mengambil multiple query parameters dari request URL.
// Berguna untuk mengambil beberapa query parameters sekaligus.
// Returns map dengan empty string values untuk parameters yang tidak ditemukan.
//
// Parameters:
//   - r: *http.Request request yang di-check query parameternya
//   - keys: variadic list dari query parameter names yang akan diambil
//
// Returns:
//   - map[string]string: map dari parameter names ke values
//
// Example:
//
//	params := GetQueryParams(req, "page", "limit", "sort")
//	page := params["page"]
func GetQueryParams(r *http.Request, keys ...string) map[string]string {
	result := make(map[string]string)
	for _, key := range keys {
		result[key] = r.URL.Query().Get(key)
	}
	return result
}

// GetHeaderValue mengambil header value dari HTTP request.
// Header names case-insensitive (Go automatically handles this).
// Returns empty string jika header tidak ditemukan.
//
// Parameters:
//   - r: *http.Request request yang di-check headernya
//   - key: nama header yang akan diambil
//
// Returns:
//   - string: header value, empty string jika tidak ditemukan
//
// Example:
//
//	contentType := GetHeaderValue(req, "Content-Type")
func GetHeaderValue(r *http.Request, key string) string {
	return r.Header.Get(key)
}

// GetAuthToken mengekstrak JWT token dari Authorization header.
// Mengharapkan format: "Bearer <token>"
// Returns token dan boolean indicating apakah token ditemukan dan valid format.
//
// Parameters:
//   - r: *http.Request request yang di-check Authorization headernya
//
// Returns:
//   - string: JWT token string (tanpa "Bearer" prefix)
//   - bool: true jika token valid format, false jika tidak ada atau invalid format
//
// Example:
//
//	token, ok := GetAuthToken(req)
//	if !ok {
//	  return JsonError(w, 401, "Token tidak ditemukan", nil)
//	}
//	userID, err := jwtManager.VerifyAccessToken(token)
func GetAuthToken(r *http.Request) (string, bool) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", false
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", false
	}

	return parts[1], true
}
