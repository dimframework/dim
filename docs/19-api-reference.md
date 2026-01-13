# Referensi API - Framework dim

Dokumentasi API lengkap untuk framework dim.

## Daftar Isi

- [Router API](#router-api)
- [Middleware API](#middleware-api)
- [Context API](#context-api)
- [Response API](#response-api)
- [Error API](#error-api)
- [Validation API](#validation-api)
- [Database & Migration API](#database--migration-api)
- [JSON:API (Filter/Sort/Page)](#jsonapi-filter-sort-page)
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

### Static & SPA
- `Static(prefix string, root fs.FS, middleware ...MiddlewareFunc)`: Melayani file statis.
- `SPA(root fs.FS, index string, middleware ...MiddlewareFunc)`: Melayani Single Page Application dengan fallback.

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

### RateLimit
`func RateLimit(config RateLimitConfig, store ...RateLimitStore) MiddlewareFunc`
Middleware untuk pembatasan kecepatan. Mendukung variadic store (default: InMemory).

### Middleware Helpers
- `Chain(handler HandlerFunc, middleware ...MiddlewareFunc) HandlerFunc`
- `ChainMiddleware(middleware ...MiddlewareFunc) MiddlewareFunc`
- `Compose(middleware ...MiddlewareFunc) MiddlewareFunc`

---

## Context API

### GetUser
`func GetUser(r *http.Request) (Authenticatable, bool)`
Mengambil pengguna dari konteks.

### SetUser
`func SetUser(r *http.Request, user Authenticatable) *http.Request`
Menyimpan pengguna ke konteks.

### GetParam
`func GetParam(r *http.Request, key string) string`
Mengambil satu parameter jalur (path).

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
- `TooManyRequests(w, retryAfter int)`: Mengirim 429 Too Many Requests dengan header Retry-After.

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
- `OptionalEmail`, `OptionalMinLength`, `OptionalMaxLength`, `OptionalIn`, dll.

---

## Database & Migration API

### Database
- `NewPostgresDatabase(config DatabaseConfig) (*PostgresDatabase, error)`
- `(db *PostgresDatabase) Query(ctx, query, args...) (Rows, error)`
- `(db *PostgresDatabase) Exec(ctx, query, args...) error`
- `(db *PostgresDatabase) Begin(ctx) (pgx.Tx, error)`
- `(db *PostgresDatabase) WithTx(ctx, fn) error`
- `(db *PostgresDatabase) Close()`

### Rate Limit Storage
- `NewInMemoryRateLimitStore(window time.Duration)`
- `NewPostgresRateLimitStore(db Database)`

### Migrations
- `GetFrameworkMigrations() []Migration`: Mendapatkan semua migrasi inti.
- `GetUserMigrations() []Migration`
- `GetTokenMigrations() []Migration`
- `GetRateLimitMigrations() []Migration`
- `RunMigrations(db, migrations)`: Menjalankan migrasi.
- `RollbackMigration(db, migration)`: Membatalkan migrasi.

---

## JSON:API (Filter, Sort, Page)

### Filtering
- `NewFilterParser(r *http.Request) *FilterParser`
- `(fp) WithMaxValues(max int)`
- `(fp) WithTimezone(tz *time.Location)`
- `(fp) Parse(target interface{})`
- `(fp) HasErrors() bool`

### Pagination
- `NewPaginationParser(defaultLimit, maxLimit int) *PaginationParser`
- `(p) Parse(r *http.Request) (*Pagination, error)`

### Sorting
- `NewSortParser(allowedFields []string) *SortParser`
- `(p) Parse(r *http.Request) ([]SortField, error)`

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