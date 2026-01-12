# Referensi API - Framework dim

Dokumentasi API lengkap untuk framework dim.

## Daftar Isi

- [Router API](#router-api)
- [Middleware API](#middleware-api)
- [Context API](#context-api)
- [Response API](#response-api)
- [Error API](#error-api)
- [Validation API](#validation-api)
- [Database API](#database-api)
- [Password API](#password-api)
- [File & Upload API](#file-upload-api)
- [Config & Env API](#config-env-api)
- [Logger API](#logger-api)

---

## Router API

### NewRouter
`func NewRouter() *Router`
Membuat instance Router baru.

### Metode HTTP
- `Get(path string, handler HandlerFunc, middleware ...MiddlewareFunc)`
- `Post(path string, handler HandlerFunc, middleware ...MiddlewareFunc)`
- `Put(path string, handler HandlerFunc, middleware ...MiddlewareFunc)`
- `Delete(path string, handler HandlerFunc, middleware ...MiddlewareFunc)`
- `Patch(path string, handler HandlerFunc, middleware ...MiddlewareFunc)`
- `Options(path string, handler HandlerFunc, middleware ...MiddlewareFunc)`
- `Head(path string, handler HandlerFunc, middleware ...MiddlewareFunc)`

### Group
`func (r *Router) Group(prefix string, middleware ...MiddlewareFunc) *RouterGroup`
Membuat kelompok rute dengan prefiks dan middleware.

### Use
`func (r *Router) Use(middleware ...MiddlewareFunc)`
Menambahkan middleware global.

### SetNotFound
`func (r *Router) SetNotFound(handler HandlerFunc)`
Mengatur handler kustom untuk 404.

---

## Middleware API

### Recovery
`func Recovery(logger *slog.Logger) MiddlewareFunc`
Middleware untuk menangkap dari `panic`.

### LoggerMiddleware
`func LoggerMiddleware(logger *slog.Logger) MiddlewareFunc`
Middleware untuk logging permintaan/respons.

### CORS
`func CORS(config CORSConfig) MiddlewareFunc`
Middleware untuk Cross-Origin Resource Sharing.

### CSRF
`func CSRFMiddleware(config CSRFConfig) MiddlewareFunc`
Middleware untuk perlindungan Cross-Site Request Forgery.

### RequireAuth
`func RequireAuth(jwtManager *JWTManager) MiddlewareFunc`
**Aman & Direkomendasikan.** Mewajibkan dan memverifikasi token JWT.

### OptionalAuth
`func OptionalAuth(jwtManager *JWTManager) MiddlewareFunc`
**Aman & Direkomendasikan.** Secara opsional memverifikasi token JWT.

### ExpectBearerToken
`func ExpectBearerToken() MiddlewareFunc`
**Tidak Aman.** Hanya memeriksa keberadaan header `Authorization: Bearer`.

### AllowBearerToken
`func AllowBearerToken() MiddlewareFunc`
**Tidak Aman.** Middleware pasif yang tidak melakukan apa-apa.

### RateLimit
`func RateLimit(config RateLimitConfig) MiddlewareFunc`
Middleware untuk pembatasan kecepatan (rate limiting).

---

## Context API

### GetUser
`func GetUser(r *http.Request) (*User, bool)`
Mengambil pengguna dari konteks.

### SetUser
`func SetUser(r *http.Request, user *User) *http.Request`
Menyimpan pengguna ke konteks.

### GetParam
`func GetParam(r *http.Request, key string) string`
Mengambil satu parameter jalur.

### GetParams
`func GetParams(r *http.Request) map[string]string`
Mengambil semua parameter jalur.

### GetQueryParam
`func GetQueryParam(r *http.Request, key string) string`
Mengambil satu parameter kueri.

### GetQueryParams
`func GetQueryParams(r *http.Request, keys ...string) map[string]string`
Mengambil beberapa parameter kueri.

### GetHeaderValue
`func GetHeaderValue(r *http.Request, key string) string`
Mengambil nilai header.

### GetAuthToken
`func GetAuthToken(r *http.Request) (string, bool)`
Mengekstrak token Bearer dari header `Authorization`.

### GetRequestID
`func GetRequestID(r *http.Request) string`
Mengambil ID permintaan unik dari konteks.

### SetRequestID
`func SetRequestID(r *http.Request, requestID string) *http.Request`
Menyimpan ID permintaan unik ke konteks.

---

## Response API

### Fungsi Dasar
- `Json(w, status, data)`: Mengirim respons JSON.
- `JsonPagination(w, status, data, meta)`: Mengirim respons JSON dengan paginasi.
- `JsonError(w, status, message, errors)`: Mengirim respons kesalahan JSON.
- `JsonAppError(w, appErr)`: Mengirim `*AppError` sebagai respons JSON.

### Pembantu Sukses
- `OK(w, data)`: Mengirim 200 OK.
- `Created(w, data)`: Mengirim 201 Created.
- `NoContent(w)`: Mengirim 204 No Content.

### Pembantu Error
- `BadRequest(w, message, errors)`: Mengirim 400 Bad Request.
- `Unauthorized(w, message)`: Mengirim 401 Unauthorized.
- `Forbidden(w, message)`: Mengirim 403 Forbidden.
- `NotFound(w, message)`: Mengirim 404 Not Found.
- `Conflict(w, message, errors)`: Mengirim 409 Conflict.
- `InternalServerError(w, message)`: Mengirim 500 Internal Server Error.

### Utilitas Header & Cookie
- `SetStatus(w, status)`: Mengatur kode status HTTP.
- `SetHeader(w, key, value)`: Mengatur satu header.
- `SetHeaders(w, headers)`: Mengatur beberapa header.
- `SetCookie(w, cookie)`: Mengatur cookie.

---

## Error API

- `NewAppError(message string, statusCode int) *AppError`: Membuat `AppError` baru.
- `(e *AppError) WithFieldError(field, message string) *AppError`: Menambahkan kesalahan per-field.
- `IsAppError(err error) bool`: Memeriksa apakah `error` adalah `*AppError`.
- `AsAppError(err error) (*AppError, bool)`: Melakukan type assertion ke `*AppError`.

---

## Validation API

### NewValidator
`func NewValidator() *Validator`
Membuat validator baru.

### Aturan Validasi
- `Required(field, value)`
- `Email(field, value)`
- `MinLength(field, value, min)`
- `MaxLength(field, value, max)`
- `Length(field, value, length)`
- `Pattern(field, value, pattern)`
- `In(field, value, allowed...)`
- `NumRange(field, value, min, max)`
- `Matches(field, value, otherField, otherValue)`
- `Custom(field, fn, value, message)`

### Validasi Opsional (`JsonNull`)
- `OptionalEmail(field, value)`
- `OptionalMinLength(field, value, min)`
- `OptionalMaxLength(field, value, max)`
- `OptionalLength(field, value, length)`
- `OptionalIn(field, value, allowed...)`
- `OptionalMatches(field, value, pattern)`

### Metode Hasil
- `IsValid() bool`
- `Errors() []string`
- `ErrorMap() map[string]string`

---

## Database API
- `NewPostgresDatabase(config DatabaseConfig) (*PostgresDatabase, error)`
- `(db *PostgresDatabase) Query(ctx, query, args...) (Rows, error)`
- `(db *PostgresDatabase) QueryRow(ctx, query, args...) Row`
- `(db *PostgresDatabase) Exec(ctx, query, args...) error`
- `(db *PostgresDatabase) Begin(ctx) (pgx.Tx, error)`
- `(db *PostgresDatabase) WithTx(ctx, fn) error`
- `(db *PostgresDatabase) Close()`

---

## Password API
- `HashPassword(password string) (string, error)`
- `VerifyPassword(hashedPassword, password string) error`
- `NewPasswordValidator() *PasswordValidator`
- `ValidatePasswordStrength(password string) error`

---

## File & Upload API
- `DetectContentType(filename string) string`
- `RegisterMIMEType(ext, mimeType string)`
- `ServeFile(w, filename, filePath, statusCode)`
- `ServeFileInline(w, filename, filePath, statusCode)`
- `UploadFiles(ctx, disk, files, opts...)`

---

## Config & Env API
- `LoadConfig() (*Config, error)`
- `GetEnv(key string) string`
- `GetEnvOrDefault(key, defaultValue string) string`
- `ParseEnvDuration(value string) time.Duration`
- `ParseEnvBool(value string) bool`
- `ParseEnvInt(value string) int`

---

## Logger API
- `NewLogger(level slog.Level) *Logger`
- `NewLoggerWithWriter(w io.Writer, level slog.Level) *Logger`
- `NewTextLogger(level slog.Level) *Logger`
- `NewTextLoggerWithWriter(w io.Writer, level slog.Level) *Logger`
