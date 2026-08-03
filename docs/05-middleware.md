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
- [Client IP Middleware](#client-ip-middleware)
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

// 2. CLIENT IP - SEBELUM apa pun yang membaca IP klien
//    (opsional; hanya jika berjalan di belakang proxy tepercaya)
router.Use(dim.ClientIP(cfg.ClientIP))

// 3. LOGGER
router.Use(dim.LoggerMiddleware(logger))

// 4. CORS & CSRF - SEBELUM AUTH
router.Use(dim.CORS(corsConfig))
router.Use(dim.CSRFMiddleware(csrfConfig))

// 5. AUTH - Per grup/rute
// 6. HANDLER
```

---

## Middleware Bawaan

| # | Nama | Tujuan | Required |
|---|------|--------|----------|
| 1 | `Recovery` | Tangkap panic | ✅ Sangat disarankan |
| 2 | `ClientIP` | Resolusi IP klien di belakang proxy | ⚠️ Jika di belakang proxy |
| 3 | `LoggerMiddleware` | Log request/response | ✅ Sangat disarankan |
| 4 | `CORS` | Handle cross-origin | ✅ Jika ada frontend web |
| 5 | `CSRF` | Proteksi CSRF | ✅ Untuk web tradisional |
| 6 | `RequireAuth` | JWT verification | ✅ Untuk rute terlindungi |
| 7 | `RateLimit` | DDoS protection | ⚠️ Opsional |

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

### Streaming, SSE, dan WebSocket

Middleware ini membungkus `http.ResponseWriter` untuk menangkap status code.
Pembungkusnya mempertahankan antarmuka opsional milik writer aslinya —
`http.Flusher`, `http.Hijacker`, dan `io.ReaderFrom` — sehingga SSE, respons
chunked, upgrade WebSocket, dan jalur cepat penyajian file statis tetap
berfungsi di baliknya.

Untuk handler streaming, gunakan `http.ResponseController` (cara resmi sejak
Go 1.20 untuk writer terbungkus):

```go
func streamHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/event-stream")
    rc := http.NewResponseController(w)

    for _, msg := range messages {
        fmt.Fprintf(w, "data: %s\n\n", msg)
        if err := rc.Flush(); err != nil {
            return
        }
    }
}
```

Type assertion langsung seperti `w.(http.Flusher)` juga tetap bekerja, sehingga
pustaka WebSocket yang mengandalkan `w.(http.Hijacker)` — `gorilla/websocket`
dan `coder/websocket` — dapat dipakai tanpa penyesuaian.

> Pembungkus hanya meneruskan antarmuka yang **benar-benar dimiliki** writer
> aslinya. Di HTTP/2 yang tidak punya `Hijacker`, `w.(http.Hijacker)` tetap
> gagal sebagaimana mestinya, bukan berhasil lalu error saat dipanggil.

Dua catatan:

- `http.Pusher` tidak diteruskan. HTTP/2 server push sudah tidak didukung
  peramban arus utama. Bila tetap dibutuhkan, jangkau lewat
  `http.NewResponseController(w)` atau `Unwrap()`.
- Setelah `Hijack`, koneksi keluar dari kendali `net/http` dan status yang
  tercatat tidak lagi mewakili apa pun yang dikirim ke klien. Log untuk request
  semacam itu diberi tanda `hijacked=true`.

---

## CORS Middleware

Menangani Cross-Origin Resource Sharing. Wajib jika API diakses dari browser dengan domain berbeda.

### Fitur
- Whitelist origin (exact match atau wildcard `*`).
- Support credentials (cookie, auth header).
- Support `ExposedHeaders` agar client bisa membaca custom header.
- Validasi integer `MaxAge` yang ketat.
- Mengembalikan `204 No Content` untuk Preflight (OPTIONS).
- Support `Vary: Origin` untuk caching yang benar.

```go
router.Use(dim.CORS(corsConfig))
```

---

## CSRF Middleware

Melindungi dari serangan Cross-Site Request Forgery (CSRF). Penting untuk aplikasi yang menggunakan cookie session.

### Fitur
- Validasi token via Header (`X-CSRF-Token`) atau Form (`_csrf`).
- **Cookie MaxAge**: Token expires otomatis sesuai konfigurasi (default 12 jam).
- Exempt paths: Skip validasi untuk path tertentu (e.g. webhook, public API).
- Double Submit Cookie pattern.
- Mengembalikan **419 Authentication Timeout** jika token tidak valid atau expired (standar industri modern).

```go
router.Use(dim.CSRFMiddleware(csrfConfig))
```

---

## Auth Middleware

Melindungi route dengan memverifikasi JWT.

### `RequireAuth` (Aman)

Wajib login. Jika token tidak valid, return 401.

Mendukung pengambilan token dari Header (Bearer) maupun Cookie.

```go
// Default: Mengambil dari header "Authorization: Bearer <token>"
api := router.Group("/api", dim.RequireAuth(jwtManager, blocklistStore))

// Opsi: Mengambil dari Cookie "auth_token" (untuk web app)
web := router.Group("/web", dim.RequireAuth(
    jwtManager, 
    blocklistStore, 
    dim.WithCookieToken("auth_token"),
))
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

## Client IP Middleware

Meresolusi IP address klien satu kali di awal request dan menyimpannya di request
context. Seluruh komponen hilir — `RateLimit`, `LoggerMiddleware`, `GetClientIP`,
dan `Ctx.ClientIP()` — kemudian membaca nilai yang sama.

### Kapan Dibutuhkan

Hanya jika aplikasi berjalan **di belakang proxy tepercaya** (Cloud Run, load
balancer, nginx) dan Anda membutuhkan IP asli klien, bukan IP proxy.

Tanpa middleware ini `GetClientIP` mengembalikan `RemoteAddr`. Itu default yang
aman: header proxy dapat diisi sembarang oleh klien, jadi tidak pernah dipercaya
kecuali aplikasi menyatakan berapa hop yang layak dipercaya.

### Konfigurasi

```go
config := dim.ClientIPConfig{
    TrustedProxyCount: 1,   // di belakang satu proxy; 0 = pakai RemoteAddr
}
```

### Penggunaan

```go
cfg, _ := dim.LoadConfig()

router.Use(dim.ClientIP(cfg.ClientIP))   // sedini mungkin dalam chain
```

Lalu di handler:

```go
func handler(w http.ResponseWriter, r *http.Request) {
    c := dim.Of(w, r)
    ip := c.ClientIP()   // IP asli klien, bukan IP proxy
}
```

### ⚠️ Catatan Keamanan

`TrustedProxyCount > 0` hanya aman bila aplikasi **tidak dapat dihubungi
langsung** — seluruh trafik wajib melewati proxy tepercaya. Bila origin masih
terjangkau langsung, klien dapat menyusun sendiri isi `X-Forwarded-For` dan
memalsukan IP-nya.

Detail lengkap: [Client IP Configuration](10-configuration.md#client-ip-configuration).

---

## Rate Limiting Middleware

Melindungi API dari penyalahgunaan dan serangan DDoS dengan membatasi jumlah permintaan per IP dan per pengguna.

### Cara Kerja

Middleware ini melacak jumlah permintaan dalam periode waktu tertentu (reset period). Jika batas terlampaui, server akan mengembalikan respons `429 Too Many Requests` beserta header `Retry-After`.

Batas per IP memakai IP hasil resolusi middleware `ClientIP`. Bila middleware itu
tidak dipasang, `RemoteAddr` yang dipakai — benar untuk aplikasi tanpa proxy, tetapi
di belakang load balancer seluruh pengguna akan berbagi satu ember rate limit.

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

	// Opsi 2: Database Storage (PostgreSQL/SQLite)
	// rateStore := dim.NewDatabaseRateLimitStore(db)

	r.Use(dim.RateLimit(config, rateStore))
}
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
