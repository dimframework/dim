# Autentikasi & JWT di Framework dim

Pelajari cara mengimplementasikan autentikasi JWT yang aman.

## Daftar Isi

- [Konsep JWT](#konsep-jwt)
- [Alur Autentikasi Lengkap](#alur-autentikasi-lengkap)
- [User Registration](#user-registration)
- [User Login](#user-login)
- [Token Refresh](#token-refresh)
- [Password Reset](#password-reset)
- [Logout](#logout)
- [Mengakses Pengguna Terautentikasi](#mengakses-pengguna-terautentikasi)
- [Melindungi Route (Protected Routes)](#melindungi-route-protected-routes)
- [JWT Configuration](#jwt-configuration)
- [Praktik Terbaik Keamanan](#praktik-terbaik-keamanan)

---

## Konsep JWT

JWT (JSON Web Token) adalah token stateless untuk autentikasi. Strukturnya terdiri dari `[Header].[Payload].[Signature]`. Payload berisi "claims" seperti ID pengguna (`sub`) dan waktu kedaluwarsa (`exp`), sedangkan signature menjamin integritas data.

---

## Alur Autentikasi Lengkap

1.  **Registrasi**: Klien mengirim `email` dan `password`. Server melakukan hash pada password (menggunakan bcrypt) dan menyimpan pengguna baru.
2.  **Login**: Klien mengirim `email` dan `password`. Server memverifikasi kredensial. Jika berhasil, server membuat *Access Token* (berumur pendek, misal 15 menit) dan *Refresh Token* (berumur panjang, misal 7 hari).
3.  **Request Terautentikasi**: Klien menyertakan *Access Token* pada header `Authorization: Bearer <token>` di setiap permintaan ke *endpoint* yang dilindungi.
4.  **Middleware Autentikasi**: Di sisi server, middleware `RequireAuth` memverifikasi token ini. Jika valid, informasi pengguna diekstrak dan disimpan dalam *request context*.
5.  **Refresh Token**: Ketika *Access Token* kedaluwarsa, klien mengirim *Refresh Token* ke endpoint `/auth/refresh` untuk mendapatkan pasangan token baru.
6.  **Logout**: Klien mengirim *Refresh Token* ke endpoint `/api/logout` agar server dapat membatalkannya (misalnya, dengan menambahkannya ke *blacklist*).

---

## User Registration

Handler registrasi bertanggung jawab untuk mem-parsing, memvalidasi, dan memanggil service untuk membuat pengguna baru.

```go
func registerHandler(authService *dim.AuthService) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // 1. Parse request body
        var req struct {
            Email    string `json:"email"`
            Username string `json:"username"`
            Password string `json:"password"`
        }
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            dim.BadRequest(w, "JSON tidak valid", nil)
            return
        }
        
        // 2. Validate input
        v := dim.NewValidator()
        v.Required("email", req.Email).Email("email", req.Email)
        v.Required("username", req.Username).MinLength("username", req.Username, 3)
        
        // Gunakan password validator bawaan
        if err := dim.ValidatePasswordStrength(req.Password); err != nil {
            appErr, _ := dim.AsAppError(err)
            dim.BadRequest(w, appErr.Message, appErr.Errors)
            return
        }
        
        if !v.IsValid() {
            dim.BadRequest(w, "Validasi gagal", v.ErrorMap())
            return
        }
        
        // 3. Panggil service untuk registrasi
        user, err := authService.Register(r.Context(), req.Email, req.Username, req.Password)
        if err != nil {
            // Tangani error dari service (misal, email sudah ada)
            if appErr, ok := dim.AsAppError(err); ok {
                 dim.JsonAppError(w, appErr)
            } else {
                 dim.InternalServerError(w, "Gagal melakukan registrasi")
            }
            return
        }
        
        // 4. Kirim response sukses
        dim.Created(w, user)
    }
}
```

---

## User Login

Handler login memverifikasi kredensial dan mengembalikan token.

```go
func loginHandler(authService *dim.AuthService) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var req struct {
            Email    string `json:"email"`
            Password string `json:"password"`
        }
        // ... parsing dan validasi ...

        // Panggil service
        accessToken, refreshToken, err := authService.Login(r.Context(), req.Email, req.Password)
        if err != nil {
            dim.Unauthorized(w, "Email atau password salah")
            return
        }
        
        // Kirim token
        dim.OK(w, map[string]interface{}{
            "access_token":  accessToken,
            "refresh_token": refreshToken,
            "expires_in":    900,  // 15 menit
            "token_type":    "Bearer",
        })
    }
}
```
---

(Bagian Token Refresh, Password Reset, dan Logout disederhanakan untuk keringkasan)

---

## Mengakses Pengguna Terautentikasi

Setelah token divalidasi oleh middleware `RequireAuth`, Anda dapat mengakses data pengguna di dalam *handler* menggunakan `dim.GetUser(r)`.

```go
// Pastikan route ini dilindungi oleh RequireAuth(jwtManager)
func profileHandler(w http.ResponseWriter, r *http.Request) {
    // Get user dari context
    user, ok := dim.GetUser(r)
    if !ok {
        // Ini seharusnya tidak terjadi jika middleware diterapkan dengan benar
        dim.Unauthorized(w, "Unauthorized")
        return
    }
    
    // Gunakan informasi pengguna
    dim.OK(w, user)
})
```

---

## Melindungi Route (Protected Routes)

### Peringatan Kritis: `RequireAuth` vs `ExpectBearerToken`

Sangat penting untuk menggunakan *middleware* yang tepat:

1.  **`dim.RequireAuth(jwtManager *JWTManager)`**:
    *   **KEAMANAN**: ✅ **AMAN**. Ini adalah **cara yang benar dan direkomendasikan** untuk melindungi *route*.
    *   **Fungsi**: Memverifikasi token (tanda tangan, masa berlaku) DAN menempatkan pengguna di *request context*. Gagal jika token tidak valid.

2.  **`dim.ExpectBearerToken()`**:
    *   **KEAMANAN**: ❌ **TIDAK AMAN JIKA DIGUNAKAN SENDIRI**.
    *   **Fungsi**: HANYA memeriksa keberadaan header `Authorization: Bearer <token>`. **TIDAK** memverifikasi token. Gunakan hanya untuk kasus lanjutan di mana verifikasi dilakukan manual.

### Melindungi Grup Route (Cara yang Direkomendasikan)

Cara paling umum adalah membuat grup *route* yang memerlukan autentikasi.

```go
// main.go

// 1. Buat JWT Manager dengan konfigurasi Anda
cfg, _ := dim.LoadConfig()
jwtManager := dim.NewJWTManager(&cfg.JWT)

router := dim.NewRouter()

// Rute publik (tidak perlu login)
router.Post("/auth/login", loginHandler(authService))

// Rute API yang dilindungi
// Gunakan RequireAuth untuk keamanan!
api := router.Group("/api", dim.RequireAuth(jwtManager))

// Semua rute di dalam grup ini sekarang terlindungi
api.Get("/profile", profileHandler)
api.Get("/users", listUsersHandler)
```

### Otentikasi Opsional

Gunakan `dim.OptionalAuth(jwtManager)` untuk *route* yang dapat diakses publik tetapi memberikan fungsionalitas tambahan jika pengguna login.

```go
// Endpoint ini dapat diakses oleh semua orang
// Tapi akan menampilkan data personal jika user login
router.Get("/posts/:id", 
    getPostHandler, // Handler dieksekusi setelah middleware
    dim.OptionalAuth(jwtManager),
)

func getPostHandler(w http.ResponseWriter, r *http.Request) {
    post := getPublicPost(r.Context())
    user, authenticated := dim.GetUser(r)
    
    if authenticated {
        // Kirim response dengan data tambahan untuk pengguna yang login
        post.CanEdit = (post.AuthorID == user.ID)
    }
    
    dim.OK(w, post)
}
```

### Role-Based Access (Kontrol Akses Berbasis Peran)

Anda dapat dengan mudah membangun *middleware* kustom di atas `RequireAuth` untuk memeriksa peran (*role*) pengguna.

```go
func requireAdminMiddleware(next dim.HandlerFunc) dim.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        user, ok := dim.GetUser(r)
        if !ok {
            dim.Unauthorized(w, "Unauthorized")
            return
        }
        
        // Asumsikan Anda mengambil detail user dari database
        appUser, _ := myUserStore.FindByID(r.Context(), user.ID)
        if appUser.Role != "admin" {
            dim.Forbidden(w, "Akses ditolak: hanya untuk admin")
            return
        }
        
        next(w, r)
    }
}

// Penggunaan:
admin := router.Group("/admin", 
    dim.RequireAuth(jwtManager), // 1. Pastikan user login & valid
    requireAdminMiddleware,      // 2. Pastikan user adalah admin
)
admin.Delete("/users/:id", deleteUserHandler)
```

---

## JWT Configuration

Konfigurasi JWT diatur melalui variabel lingkungan.

`.env`:
```bash
JWT_SECRET=your-super-secret-key-change-in-production
JWT_ACCESS_TOKEN_EXPIRY=15m
JWT_REFRESH_TOKEN_EXPIRY=7d
```

---

## Praktik Terbaik Keamanan

- **Jangan pernah hardcode secrets**. Selalu gunakan variabel lingkungan.
- **Gunakan HTTPS** di production untuk mengenkripsi token saat transit.
- **Simpan token dengan aman** di sisi klien, lebih disukai dalam *HttpOnly cookies*.
- **Implementasikan rotasi refresh token** untuk meningkatkan keamanan.
- **Validasi signature token** di setiap request.
- **Implementasikan *blacklist* token** untuk proses logout yang sesungguhnya.

---

**Lihat Juga**:
- [Middleware](04-middleware.md) - Urutan middleware dan keamanan
- [Validasi](09-validation.md) - Validasi input
- [Keamanan](14-security.md) - Praktik keamanan lainnya