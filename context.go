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
	paramsKey    contextKey = "params"
)

// SetUser menyimpan user object ke dalam request context.
// Berguna untuk menyimpan authenticated user info yang dapat diakses di handlers.
// Returns request baru dengan updated context.
//
// Parameters:
//   - r: *http.Request request yang akan diupdate contextnya
//   - user: *User object yang akan disimpan
//
// Returns:
//   - *http.Request: request baru dengan user disimpan di context
//
// Example:
//
//	req = SetUser(req, &User{ID: 123, Email: "user@example.com"})
//	user, ok := GetUser(req)
func SetUser(r *http.Request, user *User) *http.Request {
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
//   - *User: user object dari context, nil jika tidak ada
//   - bool: true jika user ada, false jika tidak ada
//
// Example:
//
//	user, ok := GetUser(req)
//	if !ok {
//	  return JsonError(w, 401, "Tidak authorized", nil)
//	}
func GetUser(r *http.Request) (*User, bool) {
	user, ok := r.Context().Value(userKey).(*User)
	return user, ok
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

// GetParam mengambil single path parameter dari request.
// Menggunakan stdlib r.PathValue() untuk pattern {id}.
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
