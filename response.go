package dim

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// PaginationMeta contains pagination information
type PaginationMeta struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// PaginationResponse is the response structure for paginated data
type PaginationResponse struct {
	Data interface{}    `json:"data"`
	Meta PaginationMeta `json:"meta"`
}

// ErrorResponse is the response structure for error responses
type ErrorResponse struct {
	Message string            `json:"message"`
	Errors  map[string]string `json:"errors,omitempty"`
}

// Json menulis JSON response dengan status code dan data yang diberikan.
// Content-Type header otomatis di-set ke "application/json".
// Untuk single objects, write langsung tanpa wrapper: {"id": 1, "name": "John"}
// Untuk arrays, write langsung tanpa wrapper: [{"id": 1, "name": "John"}]
//
// Parameters:
//   - w: http.ResponseWriter untuk menulis response
//   - status: HTTP status code (contoh: 200, 400, 500)
//   - data: data yang akan di-encode sebagai JSON
//
// Returns:
//   - error: error jika encoding JSON gagal
//
// Example:
//
//	user := User{ID: 1, Name: "John", Email: "john@example.com"}
//	Json(w, 200, user)
func Json(w http.ResponseWriter, status int, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	return json.NewEncoder(w).Encode(data)
}

// JsonPagination menulis paginated JSON response dengan data dan pagination metadata.
// Response format: {"data": [...], "meta": {"page": 1, "per_page": 10, "total": 100, "total_pages": 10}}
// Content-Type header otomatis di-set ke "application/json".
//
// Parameters:
//   - w: http.ResponseWriter untuk menulis response
//   - status: HTTP status code
//   - data: data array/slice yang akan dipaginate
//   - meta: PaginationMeta berisi pagination information
//
// Returns:
//   - error: error jika encoding JSON gagal
//
// Example:
//
//	users := []User{{ID: 1, Name: "John"}, {ID: 2, Name: "Jane"}}
//	meta := PaginationMeta{Page: 1, PerPage: 10, Total: 100, TotalPages: 10}
//	JsonPagination(w, 200, users, meta)
func JsonPagination(w http.ResponseWriter, status int, data interface{}, meta PaginationMeta) error {
	response := PaginationResponse{
		Data: data,
		Meta: meta,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	return json.NewEncoder(w).Encode(response)
}

// JsonError menulis error JSON response dengan message dan optional field errors.
// Response format: {"message": "error message", "errors": {"field": "error message"}}
// Content-Type header otomatis di-set ke "application/json".
// Gunakan untuk standard error responses dengan field-level error details.
//
// Parameters:
//   - w: http.ResponseWriter untuk menulis response
//   - status: HTTP status code (contoh: 400, 401, 404, 500)
//   - message: error message string
//   - errors: optional map dari field names ke error messages
//
// Returns:
//   - error: error jika encoding JSON gagal
//
// Example:
//
//	JsonError(w, 400, "Validasi gagal", map[string]string{
//	  "email": "Email harus valid",
//	  "password": "Password minimal 8 karakter",
//	})
func JsonError(w http.ResponseWriter, status int, message string, errors map[string]string) error {
	response := ErrorResponse{
		Message: message,
		Errors:  errors,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	return json.NewEncoder(w).Encode(response)
}

// JsonAppError menulis AppError sebagai JSON response.
// Mengekstrak status code, message, dan field errors dari AppError dan mengirimnya.
// Convenience function yang wrap JsonError dengan AppError data.
//
// Parameters:
//   - w: http.ResponseWriter untuk menulis response
//   - appErr: *AppError yang berisi response data
//
// Returns:
//   - error: error jika encoding JSON gagal
//
// Example:
//
//	appErr := NewAppError("Validasi gagal", 400)
//	appErr.WithFieldError("email", "Email sudah terdaftar")
//	JsonAppError(w, appErr)
func JsonAppError(w http.ResponseWriter, appErr *AppError) error {
	return JsonError(w, appErr.StatusCode, appErr.Message, appErr.Errors)
}

// SetContentType menetapkan Content-Type header untuk response.
// Berguna untuk mengset content type custom selain application/json.
//
// Parameters:
//   - w: http.ResponseWriter untuk menulis header
//   - contentType: content type string (contoh: text/plain, text/html, application/xml)
//
// Example:
//
//	SetContentType(w, "text/html")
//	w.Write([]byte("<html><body>Hello</body></html>"))
func SetContentType(w http.ResponseWriter, contentType string) {
	w.Header().Set("Content-Type", contentType)
}

// SetHeader menetapkan single response header.
// Header dapat di-set sebelum WriteHeader/SetStatus dipanggil.
//
// Parameters:
//   - w: http.ResponseWriter untuk menulis header
//   - key: nama header (contoh: X-Custom-Header, Authorization)
//   - value: header value
//
// Example:
//
//	SetHeader(w, "X-Request-ID", "12345")
//	SetHeader(w, "Cache-Control", "no-cache")
func SetHeader(w http.ResponseWriter, key, value string) {
	w.Header().Set(key, value)
}

// SetHeaders menetapkan multiple response headers sekaligus.
// Convenience function untuk mengset banyak headers dalam satu call.
// Headers dapat di-set sebelum WriteHeader/SetStatus dipanggil.
//
// Parameters:
//   - w: http.ResponseWriter untuk menulis headers
//   - headers: map[string]string dari header names ke values
//
// Example:
//
//	SetHeaders(w, map[string]string{
//	  "X-Request-ID": "12345",
//	  "X-Version": "1.0",
//	  "Cache-Control": "no-cache",
//	})
func SetHeaders(w http.ResponseWriter, headers map[string]string) {
	for key, value := range headers {
		w.Header().Set(key, value)
	}
}

// SetCookie menetapkan response cookie.
// Cookie akan dikirim ke client dan disimpan untuk subsequent requests.
//
// Parameters:
//   - w: http.ResponseWriter untuk menulis cookie
//   - cookie: *http.Cookie yang akan dikirim
//
// Example:
//
//	cookie := &http.Cookie{
//	  Name: "session_id",
//	  Value: "abc123",
//	  HttpOnly: true,
//	  Secure: true,
//	  MaxAge: 3600,
//	}
//	SetCookie(w, cookie)
func SetCookie(w http.ResponseWriter, cookie *http.Cookie) {
	http.SetCookie(w, cookie)
}

// SetStatus menulis HTTP response status code.
// Harus dipanggil sebelum menulis response body.
// Setelah status ditulis, tidak bisa diubah lagi.
//
// Parameters:
//   - w: http.ResponseWriter untuk menulis status
//   - status: HTTP status code (contoh: 200, 400, 500)
//
// Example:
//
//	SetStatus(w, 201)  // Created
//	w.Write([]byte(`{"id": 1}`))
func SetStatus(w http.ResponseWriter, status int) {
	w.WriteHeader(status)
}

// NoContent menulis 204 No Content response.
// Berguna untuk successful requests yang tidak mengembalikan data (contoh: DELETE, PUT tanpa response body).
// Tidak ada response body yang dikirim.
//
// Parameters:
//   - w: http.ResponseWriter untuk menulis response
//
// Returns:
//   - error: selalu nil
//
// Example:
//
//	NoContent(w)  // 204 No Content
func NoContent(w http.ResponseWriter) error {
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// Created menulis 201 Created response dengan data.
// Berguna untuk successful POST requests yang membuat resource baru.
//
// Parameters:
//   - w: http.ResponseWriter untuk menulis response
//   - data: resource baru yang dibuat
//
// Returns:
//   - error: error jika encoding JSON gagal
//
// Example:
//
//	user := User{ID: 1, Name: "John"}
//	Created(w, user)  // 201 Created
func Created(w http.ResponseWriter, data interface{}) error {
	return Json(w, http.StatusCreated, data)
}

// OK menulis 200 OK response dengan data.
// Berguna untuk successful GET atau update requests yang mengembalikan data.
//
// Parameters:
//   - w: http.ResponseWriter untuk menulis response
//   - data: response data
//
// Returns:
//   - error: error jika encoding JSON gagal
//
// Example:
//
//	users := []User{{ID: 1}, {ID: 2}}
//	OK(w, users)  // 200 OK
func OK(w http.ResponseWriter, data interface{}) error {
	return Json(w, http.StatusOK, data)
}

// BadRequest menulis 400 Bad Request error response.
// Berguna untuk validation errors atau malformed requests.
//
// Parameters:
//   - w: http.ResponseWriter untuk menulis response
//   - message: error message
//   - errors: optional map dari field names ke error messages untuk validation errors
//
// Returns:
//   - error: error jika encoding JSON gagal
//
// Example:
//
//	BadRequest(w, "Validasi gagal", map[string]string{
//	  "email": "Email tidak valid",
//	})
func BadRequest(w http.ResponseWriter, message string, errors map[string]string) error {
	return JsonError(w, http.StatusBadRequest, message, errors)
}

// Unauthorized menulis 401 Unauthorized error response.
// Berguna untuk requests tanpa authentication atau dengan invalid credentials.
//
// Parameters:
//   - w: http.ResponseWriter untuk menulis response
//   - message: error message
//
// Returns:
//   - error: error jika encoding JSON gagal
//
// Example:
//
//	Unauthorized(w, "Token tidak valid atau telah expired")
func Unauthorized(w http.ResponseWriter, message string) error {
	return JsonError(w, http.StatusUnauthorized, message, nil)
}

// Forbidden menulis 403 Forbidden error response.
// Berguna untuk authenticated requests yang tidak punya permission untuk access resource.
//
// Parameters:
//   - w: http.ResponseWriter untuk menulis response
//   - message: error message
//
// Returns:
//   - error: error jika encoding JSON gagal
//
// Example:
//
//	Forbidden(w, "Anda tidak memiliki permission untuk access resource ini")
func Forbidden(w http.ResponseWriter, message string) error {
	return JsonError(w, http.StatusForbidden, message, nil)
}

// NotFound menulis 404 Not Found error response.
// Berguna untuk requests ke resource yang tidak ada.
//
// Parameters:
//   - w: http.ResponseWriter untuk menulis response
//   - message: error message
//
// Returns:
//   - error: error jika encoding JSON gagal
//
// Example:
//
//	NotFound(w, "User dengan ID 123 tidak ditemukan")
func NotFound(w http.ResponseWriter, message string) error {
	return JsonError(w, http.StatusNotFound, message, nil)
}

// Conflict menulis 409 Conflict error response.
// Berguna untuk requests yang conflict dengan state saat ini (contoh: duplicate email).
//
// Parameters:
//   - w: http.ResponseWriter untuk menulis response
//   - message: error message
//   - errors: optional map dari field names ke error messages
//
// Returns:
//   - error: error jika encoding JSON gagal
//
// Example:
//
//	Conflict(w, "Email sudah terdaftar", map[string]string{
//	  "email": "Email ini sudah digunakan oleh pengguna lain",
//	})
func Conflict(w http.ResponseWriter, message string, errors map[string]string) error {
	return JsonError(w, http.StatusConflict, message, errors)
}

// InternalServerError menulis 500 Internal Server Error response.
// Berguna untuk unexpected server errors.
// Jangan expose detailed error information ke client untuk security.
//
// Parameters:
//   - w: http.ResponseWriter untuk menulis response
//   - message: error message yang aman untuk dikirim ke client
//
// Returns:
//   - error: error jika encoding JSON gagal
//
// Example:
//
//	InternalServerError(w, "Terjadi kesalahan pada server")
func InternalServerError(w http.ResponseWriter, message string) error {
	return JsonError(w, http.StatusInternalServerError, message, nil)
}

// TooManyRequests menulis 429 Too Many Requests response.
// Mengatur header Retry-After dan mengirim pesan error standar.
// Berguna untuk rate limiting middleware.
//
// Parameters:
//   - w: http.ResponseWriter untuk menulis response
//   - retryAfterSeconds: jumlah detik yang harus ditunggu client sebelum retry
//
// Returns:
//   - error: error jika encoding JSON gagal
//
// Example:
//
//	TooManyRequests(w, 60)
func TooManyRequests(w http.ResponseWriter, retryAfterSeconds int) error {
	w.Header().Set("Retry-After", fmt.Sprintf("%d", retryAfterSeconds))
	return JsonError(w, http.StatusTooManyRequests, "Batas tingkat permintaan terlampaui", nil)
}