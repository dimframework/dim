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
- [Advanced: Middleware Chaining](#advanced-middleware-chaining)
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

---

## ⚠️ URUTAN MIDDLEWARE KRITIS

**URUTAN INI TIDAK BOLEH DIUBAH!**

### Urutan yang BENAR (WAJIB):

```go
router := dim.NewRouter()

// 1. RECOVERY - HARUS PERTAMA
router.Use(dim.Recovery(logger))

// 2. LOGGER - HARUS KEDUA
router.Use(dim.LoggerMiddleware(logger))

// 3. CORS & CSRF - SEBELUM AUTH
router.Use(dim.CORS(corsConfig))
router.Use(dim.CSRF(csrfConfig))

// 4. AUTH - Per grup/rute
// 5. HANDLER
```

---

## Middleware Bawaan

| # | Nama | Tujuan | Required |
|---|------|--------|----------|
| 1 | `Recovery` | Tangkap panic | ✅ Sangat disarankan |
| 2 | `LoggerMiddleware` | Log request/response | ✅ Sangat disarankan |
| 3 | `CORS` | Handle cross-origin | ✅ Jika ada frontend web |
| 4 | `CSRF` | Proteksi CSRF | ✅ Untuk web tradisional |
| 5 | `RequireAuth` | JWT verification | ✅ Untuk rute terlindungi |
| 6 | `RateLimit` | DDoS protection | ⚠️ Opsional |

---

## Recovery Middleware

Menangkap panic dan mengembalikan error response 500 JSON.

```go
router.Use(dim.Recovery(logger))
```

---

## Logger Middleware

Mencatat detail request (method, path, status code, duration) dengan format terstruktur.

```go
router.Use(dim.LoggerMiddleware(logger))
```

---

## Auth Middleware

Melindungi route dengan memverifikasi JWT.

### `RequireAuth` (Aman)

Wajib login. Jika token tidak valid, return 401.

```go
api := router.Group("/api", dim.RequireAuth(jwtManager))
```

### `OptionalAuth`

Boleh login atau tidak. Jika login, user context diisi.

```go
router.Get("/news", listNewsHandler, dim.OptionalAuth(jwtManager))
```

### Mengakses User

```go
func myHandler(w http.ResponseWriter, r *http.Request) {
    user, ok := dim.GetUser(r) // Mengembalikan *TokenUser, bool
    if ok {
        fmt.Println("User ID:", user.ID)
    }
}
```

---

## Rate Limiting Middleware

Melindungi API dari penyalahgunaan dan serangan DDoS dengan membatasi jumlah permintaan per IP dan per pengguna.

### Cara Kerja

Middleware ini melacak jumlah permintaan dalam periode waktu tertentu (reset period). Jika batas terlampaui, server akan mengembalikan respons `429 Too Many Requests` beserta header `Retry-After`.

### Konfigurasi

```go
config := dim.RateLimitConfig{
    Enabled:     true,
    PerIP:       100,           // Maks 100 request per IP
    PerUser:     200,           // Maks 200 request per user (jika login)
    ResetPeriod: time.Hour,     // Periode reset counter
}
```

### Storage Backends (Pluggable)

Dim mendukung penyimpanan counter rate limit di beberapa backend:

1.  **In-Memory (Default)**
    *   **Pros**: Sangat cepat, tidak butuh setup tambahan.
    *   **Cons**: Data hilang saat restart, limit tidak terbagi antar instance (jika horizontal scaling).
    *   **Use Case**: Single server deployment.

2.  **PostgreSQL**
    *   **Pros**: Persistent, mendukung multi-instance/cluster (distributed rate limiting).
    *   **Cons**: Sedikit overhead network database.
    *   **Tech**: Menggunakan `UNLOGGED` table untuk performa maksimal.

### Penggunaan

**Opsi 1: Default (In-Memory)**

```go
// Otomatis menggunakan in-memory store
router.Use(dim.RateLimit(config))
```

**Opsi 2: Distributed (PostgreSQL)**

```go
// 1. Setup koneksi DB
db, _ := dim.NewPostgresDatabase(dbConfig)

// 2. Jalankan migrasi rate limit
// Ini akan membuat tabel 'rate_limits' yang diperlukan
dim.RunMigrations(db, dim.GetRateLimitMigrations())

// 3. Buat store rate limit
rateStore := dim.NewPostgresRateLimitStore(db)

// 4. Gunakan di middleware
router.Use(dim.RateLimit(config, rateStore))
```

---

## Advanced: Middleware Chaining

Dim menyediakan helper canggih untuk mengelola komposisi middleware.

### `Chain`

Menerapkan urutan middleware ke satu handler.

```go
finalHandler := dim.Chain(
    myHandler, 
    dim.RequireAuth(jwt), 
    dim.RateLimit(limit),
)
router.Get("/sensitive", finalHandler)
```

### `ChainMiddleware`

Menggabungkan beberapa middleware menjadi satu unit reusable.

```go
// Buat "Paket Middleware" untuk endpoint publik
publicStack := dim.ChainMiddleware(
    dim.Recovery(logger),
    dim.LoggerMiddleware(logger),
    dim.CORS(corsConfig),
)

// Gunakan di router
router.Use(publicStack)
```

### `Compose`

Mirip `ChainMiddleware`, membuat middleware baru dari komposisi yang ada.

```go
// Gabung Auth + AdminCheck
adminStack := dim.Compose(
    dim.RequireAuth(jwt),
    requireAdminMiddleware,
)

// Terapkan
router.Group("/admin", adminStack)
```

---

## Praktik Terbaik

1.  **Selalu Gunakan Recovery**: Jangan biarkan server crash karena satu panic.
2.  **Auth di Level Grup**: Lebih aman menerapkan auth ke grup `/api` daripada satu per satu route (rawan lupa).
3.  **CORS Global**: CORS biasanya perlu diterapkan secara global.
4.  **Chain Middleware**: Gunakan `ChainMiddleware` untuk menghindari duplikasi kode setup middleware yang panjang.
