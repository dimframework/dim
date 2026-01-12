# dim - Framework HTTP Sederhana

**dim** (redup - undefined) adalah framework HTTP sederhana berbasis Go yang dirancang untuk keperluan internal dengan fokus pada kesederhanaan, fleksibilitas dan kebutuhan pribadi, jadi belum tentu cocok digunakan.

## Fitur Utama

- **Custom Router**: Router berbasis tree dengan path parameters dan route grouping
- **Authentication**: JWT-based authentication dengan refresh token support
- **Password Management**: Secure password hashing menggunakan bcrypt
- **Middleware System**: Middleware chain yang powerful dan fleksibel
- **CORS Support**: Full CORS configuration dengan wildcard support
- **CSRF Protection**: Cookie-based CSRF protection dengan configurable exemption
- **Rate Limiting**: Per-IP dan per-user rate limiting dengan Goreus Cache (TTL & LRU eviction)
- **Partial Updates**: Support untuk PATCH requests dengan jsonull (distinguish field states)
- **Database Abstraction**: Generic database interface dengan PostgreSQL implementation dan read/write split
- **Validation**: Custom validation system dengan optional field validators
- **Logging**: Structured logging menggunakan slog (Go 1.21+)
- **Configuration**: Environment-based configuration management

## Tech Stack

- **HTTP Server**: `net/http` (stdlib)
- **Router**: Custom implementation
- **JWT**: `github.com/golang-jwt/jwt/v5`
- **Password Hashing**: `golang.org/x/crypto/bcrypt`
- **Database**: `github.com/jackc/pgx/v5`
- **Rate Limiting & Cache**: `github.com/atfromhome/goreus`
- **Partial Updates**: jsonull (via goreus)
- **Logging**: `log/slog` (stdlib)

## 📖 Dokumentasi Lengkap

Dokumentasi komprehensif tersedia di folder `docs/`:
- **[Getting Started](docs/01-getting-started.md)** - Mulai dengan cepat
- **[Architecture](docs/02-architecture.md)** - Desain dan overview
- **[API Reference](docs/19-api-reference.md)** - Referensi API lengkap
- **[Troubleshooting](docs/18-troubleshooting.md)** - Masalah umum dan solusi
- **[Deployment Guide](docs/17-deployment.md)** - Deploy ke production
- **[Handler Patterns](docs/16-handlers.md)** - Pola handler

**[→ Lihat semua dokumentasi](docs/README.md)**

## Instalasi

```bash
go get github.com/nuradiyana/dim
```

## Quick Start

### 1. Setup Environment

```bash
cp examples/.env.example .env
# Edit .env dengan konfigurasi Anda
```

### 2. Inisialisasi Database

```go
import "github.com/nuradiyana/dim"

// Muat config
cfg, _ := dim.LoadConfig()

// Hubungkan ke database
db, _ := dim.NewPostgresDatabase(cfg.Database)
defer db.Close()

// Jalankan migrasi
dim.RunMigrations(db, getMigrations())
```

### 3. Setup Router dan Middleware

```go
router := dim.NewRouter()

// Global middleware
router.Use(dim.Recovery(logger))
router.Use(dim.Logger(logger))
router.Use(dim.CORS(cfg.CORS))

// Daftar routes
router.Post("/auth/login", loginHandler)
router.Get("/health", healthHandler)

// Protected routes
api := router.Group("/api", dim.RequireAuth(jwtManager))
api.Get("/profile", profileHandler)

// Mulai server
http.ListenAndServe(":8080", router)
```

## Dokumentasi API

### Format Response

#### Respons Sukses (Objek Tunggal atau Array)
```json
{"id": 1, "name": "John"}
```

#### Respons dengan Paginasi
```json
{
  "data": [{"id": 1, "name": "John"}],
  "meta": {
    "page": 1,
    "per_page": 10,
    "total": 100,
    "total_pages": 10
  }
}
```

#### Respons Kesalahan
```json
{
  "message": "Validasi gagal",
  "errors": {
    "email": "Format email tidak valid",
    "password": "Password terlalu lemah"
  }
}
```

### Endpoint Autentikasi

#### Daftar Pengguna
```
POST /auth/register
Content-Type: application/json

{
  "email": "user@example.com",
  "username": "username",
  "password": "StrongPass123!"
}

Response (201 Created):
{
  "id": 1,
  "email": "user@example.com",
  "username": "username",
  "created_at": "2024-01-10T10:00:00Z",
  "updated_at": "2024-01-10T10:00:00Z"
}
```

#### Login
```
POST /auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "StrongPass123!"
}

Response (200 OK):
{
  "access_token": "eyJ0eXAiOiJKV1QiLCJhbGc...",
  "refresh_token": "eyJ0eXAiOiJKV1QiLCJhbGc...",
  "token_type": "Bearer"
}
```

#### Penyegaran Token
```
POST /auth/refresh
Content-Type: application/json

{
  "refresh_token": "eyJ0eXAiOiJKV1QiLCJhbGc..."
}

Response (200 OK):
{
  "access_token": "eyJ0eXAiOiJKV1QiLCJhbGc...",
  "refresh_token": "eyJ0eXAiOiJKV1QiLCJhbGc...",
  "token_type": "Bearer"
}
```

#### Logout
```
POST /api/logout
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "refresh_token": "refresh_token"
}

Response (200 OK):
{
  "message": "Keluar berhasil"
}
```

### Endpoint Profil

#### Update Profil (Partial Update)
```
PATCH /api/profile
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "email": "newemail@example.com",
  "name": "New Name",
  "password": "NewPassword123"
}

Response (200 OK):
{
  "id": 1,
  "email": "newemail@example.com",
  "name": "New Name",
  "created_at": "2024-01-10T10:00:00Z",
  "updated_at": "2024-01-11T15:30:00Z"
}
```

**Notes:**
- Semua field dalam request bersifat optional
- Jika field tidak dikirim, tidak ada update untuk field tersebut
- Jika field dikirim dengan nilai `null`, field tersebut akan diabaikan (tidak di-update).
- Password akan di-hash sebelum disimpan
- Email dan name memiliki validasi (email format, name length)
- Password minimum 8 characters

#### Get Profil
```
GET /api/profile
Authorization: Bearer <access_token>

Response (200 OK):
{
  "id": 1,
  "email": "user@example.com",
  "name": "User Name",
  "created_at": "2024-01-10T10:00:00Z",
  "updated_at": "2024-01-10T10:00:00Z"
}
```

## Panduan Konfigurasi

### Konfigurasi Server

```
SERVER_PORT=8080              # Port server
SERVER_READ_TIMEOUT=30s       # Durasi read timeout
SERVER_WRITE_TIMEOUT=30s      # Durasi write timeout
```

### Konfigurasi Database

```
DB_WRITE_HOST=localhost       # Host database untuk write
DB_READ_HOSTS=localhost       # Host database untuk read (comma-separated)
DB_PORT=5432                  # Port database
DB_NAME=dim_db                # Nama database
DB_USER=postgres              # User database
DB_PASSWORD=postgres          # Password database
DB_MAX_CONNS=25               # Koneksi maksimum
DB_SSL_MODE=disable           # SSL mode: disable, require, prefer, allow, verify-ca, verify-full
```

**Konfigurasi SSL Mode:**

Variabel environment `DB_SSL_MODE` mengontrol perilaku koneksi SSL/TLS. Mode yang tersedia:
- `disable` (default) - Koneksi tanpa SSL
- `require` - Koneksi SSL diperlukan, tanpa verifikasi sertifikat
- `prefer` - Koneksi SSL disukai, fallback ke non-SSL jika tidak tersedia
- `allow` - Koneksi non-SSL secara default, upgrade ke SSL jika tersedia
- `verify-ca` - Koneksi SSL diperlukan dengan verifikasi CA sertifikat server
- `verify-full` - Koneksi SSL diperlukan dengan verifikasi server dan hostname (paling aman)

**Konfigurasi SSL Mode (Berbasis Kode):**

Konfigurasi SSL mode dalam kode:
```go
cfg, _ := dim.LoadConfig()

// Atur SSL mode
cfg.Database.SSLMode = "require"  // atau: disable, prefer, allow, verify-ca, verify-full

// Buat koneksi database
db, _ := dim.NewPostgresDatabase(cfg.Database)
```

**Parameter Runtime Kustom:**

Konfigurasi parameter runtime PostgreSQL kustom dalam kode dengan mengatur `config.RuntimeParams`:
```go
cfg, _ := dim.LoadConfig()

// Konfigurasi parameter runtime kustom
cfg.Database.RuntimeParams = map[string]string{
    "search_path": "myschema",
    "standard_conforming_strings": "on",
    "application_name": "my_app",
}

// Buat koneksi database dengan config kustom
db, _ := dim.NewPostgresDatabase(cfg.Database)
```

**Mode Eksekusi Query:**

Konfigurasi mode eksekusi query untuk pgbouncer SimpleProtocol dalam kode:
```go
cfg.Database.QueryExecMode = "simple"  // Gunakan SimpleProtocol dengan pgbouncer
```

**Kompatibilitas pgbouncer:**

Framework dim menyediakan dukungan lengkap pgbouncer:
- `RuntimeParams` yang dapat dikonfigurasi untuk pengaturan PostgreSQL kustom
- Opsional: Atur `QueryExecMode = "simple"` untuk SimpleProtocol
- Kompatibel dengan `pool_mode = transaction` (direkomendasikan)

Untuk menggunakan dengan pgbouncer, konfigurasi pgbouncer.ini Anda:
```ini
[databases]
dim_db = host=primary_host port=5432 dbname=dim_db

[pgbouncer]
pool_mode = transaction
statement_timeout = 0
```

**Catatan tentang pool_mode:**
- `transaction` (direkomendasikan) - Koneksi dikembalikan setelah transaksi selesai. Terbaik untuk sebagian besar aplikasi
- `session` - Koneksi dipertahankan untuk seluruh sesi
- `statement` - Koneksi dikembalikan setelah setiap statement (jarang digunakan, perlu handling hati-hati)

Kemudian atur variabel environment untuk menunjuk ke pgbouncer:
```
DB_WRITE_HOST=pgbouncer-write-host
DB_READ_HOSTS=pgbouncer-read-host  # Bisa sama atau pgbouncer instance berbeda
DB_PORT=6432                        # Port default pgbouncer
```

### Konfigurasi JWT

```
JWT_SECRET=secret-key         # JWT signing secret (DIPERLUKAN)
JWT_ACCESS_TOKEN_EXPIRY=15m   # Expiry access token
JWT_REFRESH_TOKEN_EXPIRY=168h # Expiry refresh token
```

### Konfigurasi Rate Limiting

```
RATE_LIMIT_ENABLED=true       # Aktifkan rate limiting
RATE_LIMIT_PER_IP=100         # Request per IP per reset period
RATE_LIMIT_PER_USER=200       # Request per user per reset period
RATE_LIMIT_RESET_PERIOD=1h    # Reset period rate limit
```

### Konfigurasi CORS

```
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:8080
CORS_ALLOWED_METHODS=GET,POST,PUT,DELETE,PATCH,OPTIONS
CORS_ALLOWED_HEADERS=Content-Type,Authorization,X-CSRF-Token
CORS_ALLOW_CREDENTIALS=true
CORS_MAX_AGE=3600
```

### Konfigurasi CSRF

```
CSRF_ENABLED=true             # Aktifkan CSRF protection
CSRF_EXEMPT_PATHS=/webhooks   # Paths yang tidak perlu CSRF (comma-separated)
CSRF_TOKEN_LENGTH=32          # Panjang CSRF token
CSRF_COOKIE_NAME=csrf_token   # Nama cookie untuk CSRF token
CSRF_HEADER_NAME=X-CSRF-Token # Nama header untuk CSRF token
```

## Arsitektur

```
Alur Request
├── Recovery Middleware
│   └── Tangkap panics dan return 500 error
├── Logger Middleware
│   └── Log request details dan performance metrics
├── CORS Middleware
│   └── Tangani cross-origin requests
├── CSRF Middleware
│   └── Validasi CSRF token untuk unsafe methods
├── Auth Middleware (RequireAuth)
│   └── Verifikasi JWT token dan set user context
├── Rate Limit Middleware (opsional)
│   └── Cek per-IP dan per-user limits
└── Handler
    ├── Service Layer
    │   └── Business logic (auth, validation, dll)
    ├── Store Layer
    │   └── Data access (user store, token store)
    └── Database
        ├── Write Pool (single host)
        └── Read Pools (multiple hosts dengan round-robin)
```

## Komponen Inti

### Router

```go
router := dim.NewRouter()

// Daftar routes
router.Get("/users/:id", getUserHandler)
router.Post("/users", createUserHandler)

// Route grouping
api := router.Group("/api", dim.RequireAuth(jwtManager))
api.Get("/profile", profileHandler)

// Global middleware
router.Use(dim.Recovery(logger))

// Mulai server
http.ListenAndServe(":8080", router)
```

### Middleware

```go
// Wajib autentikasi
router.Use(dim.RequireAuth(jwtManager))

// CORS
router.Use(dim.CORS(corsConfig))

// Rate limiting
router.Use(dim.RateLimit(rateLimitConfig))

// Custom middleware
customMiddleware := func(next dim.HandlerFunc) dim.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Sebelum handler
        next(w, r)
        // Setelah handler
    }
}
router.Use(customMiddleware)
```

#### Rate Limiting dengan Goreus Cache

Framework menggunakan Goreus Cache untuk rate limiting, yang menyediakan:
- **TTL-based expiration**: Automatic cleanup of expired entries
- **LRU eviction**: Memory-efficient dengan maximum size limit
- **Thread-safe**: Aman untuk concurrent access
- **Per-IP & Per-User limiting**: Flexible rate limit strategies

```go
// Konfigurasi rate limiting
rateLimitConfig := dim.RateLimitConfig{
    PerIP:       100,           // 100 requests per IP
    PerUser:     200,           // 200 requests per authenticated user
    ResetPeriod: 1 * time.Hour, // Reset every hour
}

// Aktifkan rate limiting
router.Use(dim.RateLimit(rateLimitConfig))

// Atau untuk specific routes
sensitive := router.Group("/api/sensitive", dim.RateLimit(rateLimitConfig))
sensitive.Post("/action", actionHandler)
```

**Fitur Goreus Cache:**
- Automatic TTL-based expiration menghilangkan kebutuhan manual cleanup
- LRU eviction memastikan memory usage terbatas
- Thread-safe operations dengan internal mutex
- Cocok untuk rate limiting, caching, dan session management

### Autentikasi

```go
// Daftar
user, err := authService.Register(ctx, "user@example.com", "name", "password")

// Login
accessToken, refreshToken, err := authService.Login(ctx, "user@example.com", "password")

// Penyegaran
newAccessToken, newRefreshToken, err := authService.RefreshToken(ctx, refreshToken)

// Logout
err := authService.Logout(ctx, refreshToken)
```

### Validasi

```go
validator := dim.NewValidator()
validator.Required("email", email).
          Email("email", email).
          MinLength("password", password, 8)

if !validator.IsValid() {
    errors := validator.Errors()
    // Tangani errors
}
```

### Partial Updates (PATCH)

Framework mendukung partial updates untuk PATCH requests menggunakan `jsonull` yang dapat membedakan antara:
- **Field tidak dikirim** (tidak update field)
- **Field dikirim sebagai null** (tidak update field jika value is null)
- **Field dikirim dengan value** (update field dengan value baru)

#### Contoh Request

```json
PATCH /api/profile
Authorization: Bearer <access_token>

{
  "email": "newemail@example.com",
  "name": null,
  "password": "newpassword123"
}
```

#### Implementasi Handler

```go
type UserHandler struct {
    userStore *dim.PostgresUserStore
}

func (h *UserHandler) UpdateProfileHandler(w http.ResponseWriter, r *http.Request) {
    // Get authenticated user
    user, ok := dim.GetUser(r)
    if !ok {
        dim.JsonError(w, http.StatusUnauthorized, "Unauthorized", nil)
        return
    }

    // Parse request body dengan jsonull support
    var req dim.UpdateUserRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        dim.JsonError(w, http.StatusBadRequest, "Invalid request body", nil)
        return
    }

    // Validate optional fields
    v := dim.NewValidator()
    v.OptionalEmail("email", req.Email)
    v.OptionalMinLength("name", req.Name, 3)
    v.OptionalMinLength("password", req.Password, 8)

    if !v.IsValid() {
        dim.JsonError(w, http.StatusBadRequest, "Validation failed", v.ErrorMap())
        return
    }

    // Perform partial update
    if err := h.userStore.UpdatePartial(r.Context(), user.ID, &req); err != nil {
        dim.JsonError(w, http.StatusInternalServerError, "Update failed", nil)
        return
    }

    // Return updated user
    updated, _ := h.userStore.FindByID(r.Context(), user.ID)
    dim.Json(w, http.StatusOK, updated)
}
```

#### UpdateUserRequest Structure

```go
type UpdateUserRequest struct {
    Email    JsonNull[string] `json:"email"`
    Name     JsonNull[string] `json:"name"`
    Password JsonNull[string] `json:"password"`
}
```

#### Optional Field Validators

Framework menyediakan optional validators untuk jsonull fields:

```go
v := dim.NewValidator()

// OptionalEmail - hanya validate jika field present dan valid
v.OptionalEmail("email", req.Email)

// OptionalMinLength - validate minimum length
v.OptionalMinLength("name", req.Name, 3)

// OptionalMaxLength - validate maximum length
v.OptionalMaxLength("name", req.Name, 100)

// OptionalLength - validate exact length
v.OptionalLength("username", req.Username, 20)

// OptionalIn - validate value in list
v.OptionalIn("role", req.Role, []string{"admin", "user", "guest"})

// OptionalMatches - validate dengan regex pattern
v.OptionalMatches("phone", req.Phone, `^\d{10,12}$`)
```

#### JsonNull Helpers

```go
// Create valid value
email := dim.NewJsonNull("user@example.com")

// Create null value
nullEmail := dim.NewJsonNullNull[string]()

// Convert pointer to JsonNull
var name *string = nil
nameNull := dim.JsonNullFromPtr(name)  // Creates null

name = "John"
nameNull = dim.JsonNullFromPtr(name)   // Creates valid value with "John"
```

### Utility Environment

Framework menyediakan utility functions untuk parsing environment variables dan loading .env files:

#### GetEnv
Ambil nilai environment variable:
```go
secret := dim.GetEnv("JWT_SECRET")
```

#### GetEnvOrDefault
Ambil environment variable atau return default value:
```go
port := dim.GetEnvOrDefault("SERVER_PORT", "8080")
```

#### ParseEnvDuration
Parse string duration (e.g., "15m", "1h", "30s"):
```go
timeout := dim.ParseEnvDuration("30s")
expiry := dim.ParseEnvDuration("168h")
```

#### ParseEnvBool
Parse string boolean (supports: "true", "yes", "1", "on"):
```go
debug := dim.ParseEnvBool("true")
enabled := dim.ParseEnvBool(os.Getenv("FEATURE_ENABLED"))
```

#### ParseEnvInt
Parse string integer:
```go
port := dim.ParseEnvInt("8080")
maxConns := dim.ParseEnvInt(os.Getenv("DB_MAX_CONNS"))
```

#### LoadEnvFile
Muat environment variables dari .env file:
```go
// Muat dari path spesifik
if err := dim.LoadEnvFile(".env"); err != nil {
    log.Fatal(err)
}

// Muat dari directory (akan cari .env di directory tersebut)
if err := dim.LoadEnvFileFromPath("."); err != nil {
    log.Fatal(err)
}
```

**Fitur LoadEnvFile:**
- Parse format key=value
- Support quoted values (single dan double quotes)
- Support comments (lines yang diawali dengan #)
- Skip empty lines
- Hanya set env var jika belum ada (tidak override existing values)
- Graceful handling jika file tidak exist

Semua parsing functions memberikan default/safe value jika parsing gagal.

### Response Helpers

```go
// Objek tunggal
dim.Json(w, http.StatusOK, data)

// Array
dim.Json(w, http.StatusOK, []data)

// Paginasi
dim.JsonPagination(w, http.StatusOK, data, meta)

// Kesalahan
dim.JsonError(w, http.StatusBadRequest, "message", errors)

// Shorthand helpers
dim.OK(w, data)
dim.Created(w, data)
dim.BadRequest(w, "message", errors)
dim.Unauthorized(w, "message")
dim.NotFound(w, "message")
```

## Best Practices

### Keamanan

1. **Selalu gunakan HTTPS di production**
2. **Rotasi JWT secrets secara berkala**
3. **Gunakan password yang kuat** (min 8 chars, uppercase, lowercase, digit, special)
4. **Aktifkan CSRF protection** untuk operasi yang mengubah state
5. **Implementasikan rate limiting** untuk mencegah brute force attacks
6. **Validasi dan sanitasi** semua input pengguna
7. **Gunakan prepared statements** untuk mencegah SQL injection
8. **Atur CORS origins yang tepat** (hindari wildcards di production)

### Performa

1. **Gunakan read replicas** untuk scaling operasi read
2. **Aktifkan connection pooling** (DB_MAX_CONNS)
3. **Atur timeouts yang sesuai** (SERVER_READ_TIMEOUT, SERVER_WRITE_TIMEOUT)
4. **Gunakan paginasi** untuk large result sets
5. **Tambahkan database indexes** pada kolom yang sering di-query
6. **Monitor dan log** performance metrics

### Error Handling

```go
// Selalu check untuk errors
if err != nil {
    if appErr, ok := dim.AsAppError(err); ok {
        dim.JsonAppError(w, appErr)
    } else {
        dim.JsonError(w, http.StatusInternalServerError, "Server error", nil)
    }
    return
}
```

## Testing

### Unit Tests

```bash
go test ./... -v
```

### Test Coverage

```bash
go test ./... -cover
```

### Mock Implementations

```go
// Mock stores
userStore := dim.NewMockUserStore()
tokenStore := dim.NewMockTokenStore()

// Gunakan dalam tests
service := dim.NewAuthService(userStore, tokenStore, jwtConfig)
```

## Migration System

### Membuat Migrasi

```go
migrations := []dim.Migration{
    {
        Version: 1,
        Name:    "create_users_table",
        Up: func(pool *pgxpool.Pool) error {
            _, err := pool.Exec(context.Background(), `
                CREATE TABLE users (...)
            `)
            return err
        },
        Down: func(pool *pgxpool.Pool) error {
            _, err := pool.Exec(context.Background(), `
                DROP TABLE users
            `)
            return err
        },
    },
}

// Jalankan migrasi
dim.RunMigrations(db, migrations)
```

## Contoh

Lihat direktori `examples/` untuk contoh lengkap yang bekerja dengan:
- User registration dan authentication
- Token refresh
- Password reset
- Protected routes
- Error handling

## Berkontribusi

Kontribusi sangat dipersilakan! Pastikan:
- Kode mengikuti Go best practices
- Tests disertakan untuk fitur baru
- Dokumentasi diperbarui
- Commit messages deskriptif

## Lisensi

Proyek ini dilisensikan di bawah MIT License.

## Dukungan

Untuk issues, pertanyaan, atau saran, silakan buka issue di GitHub.

---

**Selamat coding dengan dim!** 🚀
