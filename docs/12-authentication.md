# Autentikasi & Token di Framework dim

Pelajari cara mengimplementasikan autentikasi token yang aman menggunakan JWT atau Branca.

## Daftar Isi

- [Konsep Token](#konsep-token)
  - [Memilih Token Provider](#memilih-token-provider)
- [Konfigurasi](#konfigurasi)
  - [Algoritma yang Didukung (JWT)](#algoritma-yang-didukung-jwt)
  - [Konfigurasi JWT](#konfigurasi-jwt)
  - [Konfigurasi Branca](#konfigurasi-branca)
- [Inisialisasi Token Manager](#inisialisasi-token-manager)
- [User Registration](#user-registration)
- [User Login](#user-login)
- [Melindungi Route](#melindungi-route)
- [Mengakses Data User](#mengakses-data-user)
- [Token Refresh](#token-refresh)
- [Praktik Terbaik](#praktik-terbaik)

---

## Konsep Token

Framework `dim` menggunakan interface `TokenManager` sebagai abstraksi untuk semua operasi token — generate, verify, dan cek expiry. Ada dua implementasi bawaan:

- **`JWTManager`** — JSON Web Token yang di-*sign* (payload terbaca client, aman via signature)
- **`BrancaManager`** — Token Branca yang di-*encrypt* (payload tidak bisa dibaca client sama sekali)

Karena keduanya mengimplementasikan interface yang sama, kode aplikasi tidak perlu berubah saat berpindah provider.

### Memilih Token Provider

| | JWT | Branca |
|---|---|---|
| Payload | Terbaca client (base64) | Terenkripsi (XChaCha20-Poly1305) |
| Kunci | Asymmetric (RS/ES) atau symmetric (HS) | Symmetric 32-byte |
| Cocok untuk | API publik, multi-service, JWKS | Payload sensitif, internal service |
| Key rotation | Didukung via `kid` header | Ganti key, token lama otomatis invalid |

---

## Konfigurasi

### Persiapan Database

Fitur autentikasi memerlukan tabel `users` dan `refresh_tokens`. Anda dapat menggunakan sistem migrasi bawaan untuk menyiapkannya:

```go
// Menjalankan migrasi untuk user dan token
dim.RunMigrations(db, append(dim.GetUserMigrations(), dim.GetTokenMigrations()...))
```

### Algoritma yang Didukung (JWT)

| Family | Algoritma | Jenis | Keterangan |
|--------|-----------|-------|-----------|
| HMAC | `HS256`, `HS384`, `HS512` | Symmetric | Satu secret untuk sign & verify. Cocok untuk single-service. |
| RSA | `RS256`, `RS384`, `RS512` | Asymmetric | Private key untuk sign, public key untuk verify. Cocok untuk multi-service. |
| ECDSA | `ES256`, `ES384`, `ES512` | Asymmetric | Seperti RSA tapi key lebih kecil dengan keamanan setara. |

### Konfigurasi JWT

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

### Konfigurasi Branca

Branca membutuhkan satu symmetric key 32-byte. Key dapat diberikan dalam format hex (64 karakter), base64, atau raw string 32 karakter.

```bash
# Generate key (hex, direkomendasikan)
openssl rand -hex 32

# Set di .env
BRANCA_KEY=a3f1c2d4e5b6a7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2
BRANCA_ACCESS_TOKEN_EXPIRY=15m
BRANCA_REFRESH_TOKEN_EXPIRY=168h
```

> **Keamanan:** Berbeda dengan JWT yang hanya men-*sign* payload, Branca mengenkripsi seluruh payload menggunakan XChaCha20-Poly1305. Client tidak bisa membaca isi token sama sekali — cocok untuk menyimpan data yang tidak boleh terekspos.

## Inisialisasi Token Manager

Framework menyediakan `TokenManager` interface sehingga JWT dan Branca bisa dipakai secara bergantian.

**Menggunakan JWT:**

```go
cfg, _ := dim.LoadConfig()

jwtManager, err := dim.NewJWTManager(&cfg.JWT)
if err != nil {
    log.Fatal("Gagal init JWT manager:", err)
}

// AuthService via JWT
authService, err := dim.NewAuthService(userStore, tokenStore, blocklist, &cfg.JWT)
```

**Menggunakan Branca:**

```go
cfg, _ := dim.LoadConfig()

brancaManager, err := dim.NewBrancaManager(&cfg.Branca)
if err != nil {
    log.Fatal("Gagal init Branca manager:", err)
}

// AuthService via Branca — gunakan NewAuthServiceWithManager
authService, err := dim.NewAuthServiceWithManager(userStore, tokenStore, blocklist, brancaManager)
```

`NewAuthService` (lama) tetap berfungsi untuk JWT. `NewAuthServiceWithManager` menerima `TokenManager` apapun.

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

Gunakan middleware `RequireAuth`. Parameternya menerima `TokenManager` — bisa `*JWTManager` atau `*BrancaManager` tanpa perubahan kode lain.

```go
// JWT
authMiddleware := dim.RequireAuth(jwtManager, blocklistStore)

// Branca — signature sama persis
authMiddleware := dim.RequireAuth(brancaManager, blocklistStore)

// Dengan Cookie token
cookieAuthMiddleware := dim.RequireAuth(
    jwtManager,
    blocklistStore,
    dim.WithCookieToken("session_id"),
)

// Terapkan ke route
router.Get("/profile", profileHandler, authMiddleware)
router.Get("/dashboard", dashboardHandler, cookieAuthMiddleware)
```

### Group Protection (Recommended)

```go
api := router.Group("/api", dim.RequireAuth(jwtManager, nil))

// Semua route di bawah /api terlindungi
api.Get("/users", listUsers)
api.Post("/posts", createPost)
```

### Optional Authentication

```go
// User diisi di context jika token valid, tapi request tidak ditolak jika tidak ada token
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

1. **HTTPS Wajib** — Jangan kirim token via HTTP biasa.
2. **Short-Lived Access Token** — Set expiry pendek (15–30 menit).
3. **Secure Storage** — Di sisi client, simpan token seaman mungkin (HttpOnly Cookie disarankan untuk web).
4. **Jangan Simpan Data Sensitif di Claims JWT** — JWT payload hanya di-*sign*, bukan di-*encrypt*. Siapa pun bisa membaca isinya. Gunakan Branca jika payload mengandung data yang tidak boleh terbaca client.
5. **Pilih Branca untuk Internal Service** — Jika token tidak perlu dibaca client dan kerahasiaan payload penting, Branca lebih tepat dari JWT.

```