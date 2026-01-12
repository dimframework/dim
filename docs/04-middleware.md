# Middleware di Framework dim

⚠️ Urutan middleware salah dapat merusak fungsionalitas dan keamanan!

## Daftar Isi

- [Konsep Dasar](#konsep-dasar)
- [Urutan Middleware KRITIS](#urutan-middleware-kritis)
- [Middleware Bawaan](#middleware-bawaan)
- [Recovery Middleware](#recovery-middleware)
- [Logger Middleware](#logger-middleware)
- [CORS Middleware](#cors-middleware)
- [CSRF Middleware](#csrf-middleware)
- [Auth Middleware](#auth-middleware)
- [Rate Limiting Middleware](#rate-limiting-middleware)
- [Custom Middleware](#custom-middleware)
- [Middleware Chaining](#middleware-chaining)
- [Praktik Terbaik](#best-practices)

---

## Konsep Dasar

### Apa itu Middleware?

Middleware adalah fungsi yang memproses request sebelum sampai ke handler, dan memproses response sebelum dikirim ke client.

**Struktur**:
```go
type MiddlewareFunc func(next HandlerFunc) HandlerFunc
type HandlerFunc func(w http.ResponseWriter, r *http.Request)
```

### Alur Middleware

```
Request
  ↓
Middleware 1 ┐
  ↓          ├─ Chain processing (in-order)
Middleware 2 │
  ↓          ├─ Can modify request/context
Middleware 3 │
  ↓          ├─ Can stop chain (return early)
  ↓          ┘
Handler
  ↓
Response
  ↓
Middleware 3 ← Return ke middleware (LIFO)
  ↓
Middleware 2
  ↓
Middleware 1
  ↓
Client
```

### Middleware Example

```go
func loggingMiddleware(next HandlerFunc) HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Sebelum handler
        log.Printf("Request: %s %s", r.Method, r.URL.Path)
        
        // Call next handler/middleware
        next(w, r)
        
        // Setelah handler (optional)
        log.Printf("Request selesai")
    }
}
```

---

## ⚠️ URUTAN MIDDLEWARE KRITIS

**URUTAN INI TIDAK BOLEH DIUBAH!** Urutan salah dapat menyebabkan security issues dan bugs.

### Urutan yang BENAR (WAJIB):

```go
// Asumsikan variabel-variabel ini sudah diinisialisasi sebelumnya
var logger *slog.Logger
var corsConfig dim.CORSConfig
var csrfConfig dim.CSRFConfig
var jwtManager *dim.JWTManager

router := dim.NewRouter()

// 1. RECOVERY - HARUS PERTAMA
router.Use(dim.Recovery(logger))

// 2. LOGGER - HARUS KEDUA
router.Use(dim.LoggerMiddleware(logger))

// 3. CORS - HARUS KETIGA
router.Use(dim.CORS(corsConfig))

// 4. CSRF - HARUS KEEMPAT
router.Use(dim.CSRF(csrfConfig))

// 5. AUTH - (Gunakan per grup atau per rute sesuai kebutuhan)
// Contoh:
// protected := router.Group("/api", dim.RequireAuth(jwtManager))
// protected.Get("/data", dataHandler)

// Routes daftar SETELAH middleware
router.Get("/public/data", publicDataHandler)
```

### Diagram Urutan (dengan penjelasan)

```
┌───────────────────────────────────────────┐
│ 1️⃣  Recovery Middleware                   │
└───────────────────────────────────────────┘
                    ↓
┌───────────────────────────────────────────┐
│ 2️⃣  Logger Middleware                     │
└───────────────────────────────────────────┘
                    ↓
┌───────────────────────────────────────────┐
│ 3️⃣  CORS Middleware                       │
└───────────────────────────────────────────┘
                    ↓
┌───────────────────────────────────────────┐
│ 4️⃣  CSRF Middleware                       │
└───────────────────────────────────────────┘
                    ↓
┌───────────────────────────────────────────┐
│ 5️⃣  Auth Middleware (jika digunakan)      │
└───────────────────────────────────────────┘
                    ↓
┌───────────────────────────────────────────┐
│ 6️⃣  Handler + Response                    │
└───────────────────────────────────────────┘
```

### Mengapa Urutan Ini Penting?

**1. Recovery HARUS PERTAMA**
- Jika middleware lain panic, Recovery harus menangkapnya.

**2. Logger HARUS KEDUA**
- Agar semua request tercatat, bahkan yang mengalami panic.

**3. CORS HARUS SEBELUM CSRF/Auth**
- CORS preflight (OPTIONS) tidak boleh dicek oleh CSRF atau Auth.

**4. CSRF HARUS SEBELUM Auth**
- CSRF melindungi pengguna yang sudah login, jadi validasi CSRF harus terjadi sebelum logika auth yang lebih dalam.

**5. Auth HARUS SEBELUM Handler**
- Handler yang aman memerlukan informasi pengguna dari konteks, yang diatur oleh Auth middleware.

---

## Middleware Bawaan

### Daftar Middleware Framework

| # | Nama       | Tujuan               | Required                   |
|---|------------|----------------------|----------------------------|
| 1 | Recovery   | Tangkap panic        | ✅ Sangat disarankan       |
| 2 | Logger     | Log request/response | ✅ Sangat disarankan       |
| 3 | CORS       | Handle cross-origin  | ✅ Jika ada frontend web   |
| 4 | CSRF       | Validasi CSRF token  | ✅ Untuk app web tradisional |
| 5 | Auth       | JWT verification     | ✅ Untuk rute terlindungi  |
| 6 | RateLimit  | DDoS protection      | ⚠️ Opsional                |

---

## Recovery Middleware

Menangkap panic dan mengembalikan error response 500.

```go
// HARUS PERTAMA
router.Use(dim.Recovery(logger))
```

---

## Logger Middleware

Mencatat semua request dan response.

```go
// HARUS KEDUA (atau setelah Recovery)
router.Use(dim.LoggerMiddleware(logger))
```

---

## CORS Middleware

Menangani Cross-Origin Resource Sharing requests.

```go
// HARUS SEBELUM CSRF dan Auth
router.Use(dim.CORS(cfg.CORS))
```

---

## CSRF Middleware

Proteksi Cross-Site Request Forgery attacks.

```go
// HARUS SEBELUM Auth
router.Use(dim.CSRF(cfg.CSRF))
```

---

## Auth Middleware

Verifikasi token autentikasi JWT untuk melindungi *route*.

### Peringatan Kritis: `RequireAuth` vs `ExpectBearerToken`

Sangat penting untuk menggunakan *middleware* yang tepat:

1.  **`dim.RequireAuth(jwtManager *dim.JWTManager)`**:
    *   **KEAMANAN**: ✅ **AMAN**. Ini adalah **cara yang benar dan direkomendasikan** untuk melindungi *route*.
    *   **Fungsi**: Memverifikasi token (tanda tangan, masa berlaku) DAN menempatkan pengguna di *request context*. Gagal jika token tidak valid.

2.  **`dim.ExpectBearerToken()`**:
    *   **KEAMANAN**: ❌ **TIDAK AMAN JIKA DIGUNAKAN SENDIRI**.
    *   **Fungsi**: HANYA memeriksa keberadaan header `Authorization: Bearer <token>`. **TIDAK** memverifikasi token. Gunakan hanya untuk kasus lanjutan di mana verifikasi dilakukan manual.

### `RequireAuth` (Aman)

Middleware yang mewajibkan pengguna untuk terautentikasi.

```go
// Terapkan ke seluruh grup
api := router.Group("/api", dim.RequireAuth(jwtManager))
api.Get("/users", listUsersHandler)

// Request tanpa token atau dengan token tidak valid:
// Response: 401 {"message": "Token tidak valid atau telah kadaluarsa"}
```

### `OptionalAuth` (Aman)

*Middleware* yang memperbolehkan pengguna terautentikasi ataupun anonim. Jika token valid, data pengguna akan tersedia di *handler*.

```go
router.Get("/posts", 
    func(w http.ResponseWriter, r *http.Request) {
        user, authenticated := dim.GetUser(r)
        if authenticated {
            // Logika untuk pengguna yang login
        } else {
            // Logika untuk pengguna anonim
        }
    },
    dim.OptionalAuth(jwtManager), // Terapkan di sini
)
```

### Mengakses Pengguna di Handler

Setelah `RequireAuth` atau `OptionalAuth` dijalankan, gunakan `dim.GetUser(r)` untuk mengakses data pengguna.

```go
func profileHandler(w http.ResponseWriter, r *http.Request) {
    user, ok := dim.GetUser(r)
    if !ok {
        dim.Unauthorized(w, "Unauthorized")
        return
    }
    dim.OK(w, user)
}
```

---

## Rate Limiting Middleware

Proteksi DDoS dengan membatasi request per IP/user.

```go
// Rate limit untuk endpoint sensitif, setelah autentikasi
sensitive := router.Group("/api/admin", 
    dim.RequireAuth(jwtManager),
    dim.RateLimit(cfg.RateLimit),
)
sensitive.Get("/users", listAllUsersHandler)
```

---

## Custom Middleware

Buat middleware custom untuk kebutuhan spesifik.

```go
// Middleware untuk admin-only
func RequireAdminMiddleware(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        user, ok := dim.GetUser(r) // Bergantung pada `RequireAuth` yang sudah jalan sebelumnya
        if !ok || user.Role != "admin" {
            dim.Forbidden(w, "Hanya untuk admin")
            return
        }
        next(w, r)
    }
}
```

---

## Middleware Chaining

Middleware dapat digabung dengan berbagai cara.

### Global Middleware Chain

```go
router := dim.NewRouter()
router.Use(dim.Recovery(logger))
router.Use(dim.LoggerMiddleware(logger))
// Middleware ini berlaku untuk semua rute
```

### Group Middleware Chain

```go
// Rute di grup admin akan menjalankan 3 middleware: Recovery, Logger, dan RequireAuth.
admin := router.Group("/admin", dim.RequireAuth(jwtManager))
admin.Get("/dashboard", dashboardHandler)
```

### Per-Route Middleware Chain

```go
router.Post("/api/sensitive",
    sensitiveHandler,
    dim.RequireAuth(jwtManager), // 1. Pastikan login
    RequireAdminMiddleware,      // 2. Pastikan admin
    dim.RateLimit(config),       // 3. Batasi request
)
```

---

## Praktik Terbaik

### ✅ DO: Follow Urutan yang Benar

```go
router := dim.NewRouter()

// ✅ BENAR
router.Use(dim.Recovery(logger))
router.Use(dim.LoggerMiddleware(logger))
router.Use(dim.CORS(corsConfig))

// Terapkan Auth per grup atau per rute
api := router.Group("/api", dim.RequireAuth(jwtManager))
```

### ✅ DO: Gunakan Route Grouping

```go
// ✅ BAIK - Terorganisir
api := router.Group("/api", dim.RequireAuth(jwtManager))
api.Get("/users", listUsersHandler)
api.Post("/users", createUserHandler)

admin := router.Group("/admin", 
    dim.RequireAuth(jwtManager),
    RequireAdminMiddleware,
)
admin.Delete("/users/:id", deleteUserHandler)
```

---

## Summary

⚠️ **Urutan Middleware KRITIS:**
```
1. Recovery  → 2. Logger  → 3. CORS / CSRF  → 4. Auth  → Handler
```

Middleware di dim memberikan:
- **Security** - CSRF, Auth, Rate limiting
- **Observability** - Logging, request tracking
- **Reliability** - Panic recovery, error handling
- **Flexibility** - Global, group, per-route middleware

Sekarang pelajari [Autentikasi](05-authentication.md) untuk detail auth flow.

---

**Lihat Juga**:
- [Autentikasi](05-authentication.md) - JWT dan password management
- [Routing](03-routing.md) - Route grouping dengan middleware
- [Konfigurasi](07-configuration.md) - Environment-based config
- [Error Handling](08-error-handling.md) - Error responses