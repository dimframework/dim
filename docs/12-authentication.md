# Autentikasi & JWT di Framework dim

Pelajari cara mengimplementasikan autentikasi JWT yang aman dan standar.

## Daftar Isi

- [Konsep JWT](#konsep-jwt)
- [Konfigurasi](#konfigurasi)
  - [Algoritma yang Didukung](#algoritma-yang-didukung)
- [User Registration](#user-registration)
- [User Login](#user-login)
- [Melindungi Route](#melindungi-route)
- [Mengakses Data User](#mengakses-data-user)
- [Token Refresh](#token-refresh)
- [Praktik Terbaik](#praktik-terbaik)

---

## Konsep JWT

JWT (JSON Web Token) digunakan sebagai token *stateless* untuk autentikasi API. Framework `dim` menyediakan `JWTManager` untuk menangani pembuatan (signing) dan verifikasi token dengan dukungan untuk berbagai algoritma (HS256, RS256, ES256).

---

## Konfigurasi

### Persiapan Database

Fitur autentikasi memerlukan tabel `users` dan `refresh_tokens`. Anda dapat menggunakan sistem migrasi bawaan untuk menyiapkannya:

```go
// Menjalankan migrasi untuk user dan token
dim.RunMigrations(db, append(dim.GetUserMigrations(), dim.GetTokenMigrations()...))
```

### Algoritma yang Didukung

| Family | Algoritma | Jenis | Keterangan |
|--------|-----------|-------|-----------|
| HMAC | `HS256`, `HS384`, `HS512` | Symmetric | Satu secret untuk sign & verify. Cocok untuk single-service. |
| RSA | `RS256`, `RS384`, `RS512` | Asymmetric | Private key untuk sign, public key untuk verify. Cocok untuk multi-service. |
| ECDSA | `ES256`, `ES384`, `ES512` | Asymmetric | Seperti RSA tapi key lebih kecil dengan keamanan setara. |

### Environment Variables

**HMAC (default, paling sederhana):**

```bash
JWT_SIGNING_METHOD=HS256
JWT_SECRET=rahasia-sangat-panjang-dan-aman-minimal-32-karakter
JWT_ACCESS_TOKEN_EXPIRY=15m
JWT_REFRESH_TOKEN_EXPIRY=168h
```

**RSA / ECDSA (asymmetric):**

```bash
JWT_SIGNING_METHOD=RS256   # atau ES256
JWT_ACCESS_TOKEN_EXPIRY=15m
JWT_REFRESH_TOKEN_EXPIRY=168h

# Pilih salah satu format untuk JWT_PRIVATE_KEY:
JWT_PRIVATE_KEY=/etc/secrets/private.pem          # file path
JWT_PRIVATE_KEY=LS0tLS1CRUdJTiBSU0EgUFJJVkFURQ== # base64-encoded PEM (direkomendasikan)

# Untuk key rotation — JSON map kid -> public key (file path / base64 PEM)
# JWT_PUBLIC_KEYS={"old-key-id": "LS0tLS1CRUdJTiBQVUJMSUM="}
```

> **Mengapa base64?** Raw PEM mengandung newline yang bisa bermasalah di Docker env, Kubernetes Secret, dan CI/CD. Encode dulu dengan `base64 -w 0 private.pem` (Linux) atau `base64 -i private.pem` (macOS).

**Generate keypair RSA/ECDSA:**

```bash
# RSA 2048-bit
openssl genrsa -out private.pem 2048
openssl rsa -in private.pem -pubout -out public.pem

# ECDSA P-256 (lebih kecil, setara keamanan RSA-3072)
openssl ecparam -name prime256v1 -genkey -noout -out private.pem
openssl ec -in private.pem -pubout -out public.pem

# Encode ke base64 untuk .env
base64 -w 0 private.pem
```

### Inisialisasi JWT Manager

```go
// Load config
cfg, _ := dim.LoadConfig()

// Init Manager
jwtManager, err := dim.NewJWTManager(&cfg.JWT)
if err != nil {
    log.Fatal("Gagal init JWT:", err)
}
```

---

## User Registration

Saat registrasi, Anda biasanya hanya membuat user di database. Token baru dibuat saat login.

```go
func registerHandler(w http.ResponseWriter, r *http.Request) {
    // 1. Parse & Validate input
    // ...

    // 2. Hash Password (menggunakan helper dim)
    hashedPassword, err := dim.HashPassword(req.Password)
    
    // 3. Simpan ke database
    // ...
    
    dim.Created(w, user)
}
```

---

## User Login

Handler login memverifikasi password dan menghasilkan token.

```go
func loginHandler(w http.ResponseWriter, r *http.Request) {
    // 1. Verifikasi kredensial user dari DB
    // ...
    
    // 2. Cek Password
    if !dim.CheckPasswordHash(req.Password, user.PasswordHash) {
        dim.Unauthorized(w, "Kredensial salah")
        return
    }

    // 3. Generate Access Token & Refresh Token
    // Framework dim sekarang menggunakan Session ID (sid) binding.
    // Kami merekomendasikan menggunakan `AuthService.Login()` yang menangani ini secara otomatis.
    
    // Contoh penggunaan low-level (jika tidak menggunakan AuthService):
    sessionID := dim.NewUUID()
    
    accessToken, err := jwtManager.GenerateAccessToken(
        fmt.Sprintf("%d", user.ID), 
        user.Email,
        sessionID, // Parameter Session ID
        nil,       // Extra claims
    )
    
    refreshToken, err := jwtManager.GenerateRefreshToken(
        fmt.Sprintf("%d", user.ID),
        sessionID,
    )
    
    // ... simpan refresh token dsb ...
}
```

> **Rekomendasi:** Gunakan `dim.AuthService` yang sudah membungkus logika ini (Login, Generate Token, Simpan Refresh Token, Logout) dengan aman.

---

## Melindungi Route

Gunakan middleware `RequireAuth`. Middleware ini fleksibel dan dapat dikonfigurasi untuk mengambil token dari Header (default) atau Cookie.

```go
// Inisialisasi Middleware (Basic - Bearer Token)
authMiddleware := dim.RequireAuth(jwtManager, blocklistStore)

// Opsi Lanjutan: Mengambil token dari Cookie (misal nama cookie "session_id")
// Cocok untuk aplikasi web tradisional / SPA yang menggunakan cookie HttpOnly
cookieAuthMiddleware := dim.RequireAuth(
    jwtManager, 
    blocklistStore,
    dim.WithCookieToken("session_id"),
)

// Terapkan ke Route
router.Get("/profile", profileHandler, authMiddleware)
router.Get("/dashboard", dashboardHandler, cookieAuthMiddleware)
```
        user.Email, 
        nil,
    )
    if err != nil {
        dim.InternalServerError(w, "Gagal generate token")
        return
    }

    // 4. Generate Refresh Token (Opsional, untuk long-lived session)
    refreshToken, _ := jwtManager.GenerateRefreshToken(fmt.Sprintf("%d", user.ID))

    // 5. Return Response
    dim.OK(w, map[string]string{
        "access_token":  accessToken,
        "refresh_token": refreshToken,
        "type":          "Bearer",
    })
}
```

---

## Melindungi Route

Gunakan middleware `dim.RequireAuth` untuk memproteksi endpoint.

### Basic Protection

```go
// Endpoint ini hanya bisa diakses jika header Authorization valid
router.Get("/profile", profileHandler, dim.RequireAuth(jwtManager))
```

### Group Protection (Recommended)

```go
api := router.Group("/api", dim.RequireAuth(jwtManager))

// Semua route di bawah /api sekarang terlindungi
api.Get("/users", listUsers)
api.Post("/posts", createPost)
```

### Optional Authentication

Gunakan `dim.OptionalAuth` jika endpoint bisa diakses publik tapi butuh konteks user jika ada.

```go
router.Get("/articles/:id", articleHandler, dim.OptionalAuth(jwtManager))
```

---

## Mengakses Data User

Di dalam handler yang terlindungi, ambil data user dari context.

```go
func profileHandler(w http.ResponseWriter, r *http.Request) {
    // Ambil user dari context
    user, ok := dim.GetUser(r)
    if !ok {
        dim.Unauthorized(w, "Tidak terautentikasi")
        return
    }

    // Akses field user
    fmt.Printf("User ID: %s, Email: %s\n", user.ID, user.Email)
    
    // Akses claims tambahan
    if role, ok := user.Claims["role"]; ok {
        fmt.Println("Role:", role)
    }

    dim.OK(w, user)
}
```

---

## Token Refresh

Endpoint untuk memperbarui access token menggunakan refresh token.

```go
func refreshHandler(w http.ResponseWriter, r *http.Request) {
    // Parse refresh token dari body
    var req struct {
        RefreshToken string `json:"refresh_token"`
    }
    // ... decode json ...

    // Verifikasi Refresh Token
    userID, err := jwtManager.VerifyRefreshToken(req.RefreshToken)
    if err != nil {
        dim.Unauthorized(w, "Refresh token tidak valid")
        return
    }

    // Cari user di DB untuk mendapatkan data terbaru (email, dll)
    user := userStore.FindByID(userID)

    // Generate Access Token BARU
    newAccessToken, _ := jwtManager.GenerateAccessToken(userID, user.Email, nil)

    dim.OK(w, map[string]string{
        "access_token": newAccessToken,
    })
}
```

---

## Praktik Terbaik

1.  **HTTPS Wajib**: Jangan kirim token via HTTP biasa.
2.  **Short-Lived Access Token**: Set expiry pendek (misal 15-30 menit).
3.  **Secure Storage**: Di sisi client, simpan token seaman mungkin (HttpOnly Cookie disarankan untuk web).
4.  **Jangan Simpan Data Sensitif di Claims**: Token bisa didecode oleh siapa saja (hanya di-sign, tidak di-encrypt). Jangan taruh password atau data pribadi di claims.

```