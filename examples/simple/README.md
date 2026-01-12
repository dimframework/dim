# dim - Simple Example Application

Aplikasi contoh sederhana yang mendemonstrasikan penggunaan framework HTTP **dim**.

## Fitur

- Pendaftaran pengguna dengan hashing password
- Login pengguna dengan token JWT
- Mekanisme refresh token
- Rute yang dilindungi dengan otentikasi JWT
- Migrasi database otomatis
- Dukungan CORS
- Penanganan error yang tepat
- Validasi untuk input, termasuk update parsial (PATCH)

## Prasyarat

- Go 1.22+
- PostgreSQL 12+

## Struktur Proyek

```
examples/simple/
├── main.go                    # Inisialisasi dan setup server
├── handler.go                 # Registrasi rute
├── auth_handler.go            # Handler untuk otentikasi (struct-based)
├── user_handler.go            # Handler untuk operasi pengguna (misal: update profil)
├── common_handler.go          # Handler umum (health, 404)
├── .env.example               # Template konfigurasi environment
├── go.mod                     # Definisi Go module
├── go.sum                     # File lock dependensi
└── README.md                  # File ini
```

## Organisasi File

- **`main.go`**: Inisialisasi server, router, koneksi database, migrasi, dan service.
- **`handler.go`**: Registrasi semua rute dan middleware.
- **`auth_handler.go`**: Menangani logika untuk registrasi, login, refresh token, dan logout.
- **`user_handler.go`**: Menangani logika terkait pengguna, seperti update profil menggunakan metode PATCH dan tipe `JsonNull`.
- **`common_handler.go`**: Menyediakan endpoint umum seperti *health check* dan handler 404.

## Setup

### 1. Salin file environment

```bash
cp .env.example .env
```

### 2. Edit `.env` dengan konfigurasi database Anda

```bash
# Edit .env
DB_WRITE_HOST=localhost
DB_READ_HOSTS=localhost
DB_PORT=5432
DB_NAME=dim_simple
DB_USER=postgres
DB_PASSWORD=postgres
```

### 3. Buat database

```bash
createdb dim_simple
```

### 4. Jalankan aplikasi

```bash
go run .
```

Aplikasi akan berjalan di port 8080 dan siap menerima permintaan.

## API Endpoints

### Health Check (Publik)
`GET /health` - Memeriksa status kesehatan server.

### Otentikasi (Publik)
- `POST /auth/register` - Mendaftarkan pengguna baru.
- `POST /auth/login` - Login dan mendapatkan token.
- `POST /auth/refresh` - Memperbarui access token.

### Profil Pengguna (Dilindungi)
- `GET /api/profile` - Mendapatkan profil pengguna yang sedang login.
- `PATCH /api/profile` - Memperbarui sebagian data profil (nama, email, password).
- `POST /api/logout` - Logout dan membatalkan refresh token.

## Pola Handler

Aplikasi ini mendemonstrasikan dua pola handler utama yang direkomendasikan:

### 1. Handler Struct (auth_handler.go, user_handler.go)
Pola ini mengelompokkan handler terkait sebagai method dari sebuah struct, yang memudahkan pengelolaan dependensi.
```go
type AuthHandler struct {
    authService *dim.AuthService
}

func (h *AuthHandler) RegisterHandler(w http.ResponseWriter, r *http.Request) {
    // ...
}

// Di router:
// router.Post("/auth/register", authHandler.RegisterHandler)
```

### 2. Direct Handler (common_handler.go)
Pola ini menggunakan fungsi biasa sebagai handler, cocok untuk endpoint sederhana tanpa dependensi.
```go
func HealthHandler(w http.ResponseWriter, r *http.Request) {
    dim.OK(w, map[string]string{"status": "healthy"})
}

// Di router:
// router.Get("/health", HealthHandler)
```

## Konfigurasi

Variabel lingkungan di `.env`:

```
# Server
SERVER_PORT=8080
LOG_LEVEL=info

# Database
DB_WRITE_HOST=localhost
DB_NAME=dim_simple
DB_USER=postgres
DB_PASSWORD=postgres
DB_MAX_CONNS=25
DB_SSL_MODE=disable

# JWT
JWT_SECRET=secret-key
JWT_ACCESS_TOKEN_EXPIRY=15m
JWT_REFRESH_TOKEN_EXPIRY=168h

# CORS
CORS_ALLOWED_ORIGINS=http://localhost:3000

# CSRF
CSRF_ENABLED=true
CSRF_COOKIE_NAME=csrf_token
CSRF_HEADER_NAME=X-CSRF-Token

# Rate Limiting
RATE_LIMIT_ENABLED=true
RATE_LIMIT_PER_IP=100
```

## Troubleshooting

### Koneksi Ditolak
Pastikan PostgreSQL berjalan dan kredensial di `.env` sudah benar.

### Token Tidak Valid
Pastikan `JWT_SECRET` sudah diatur dan header `Authorization: Bearer <token>` dikirim dengan benar.

---
Untuk detail lebih lanjut, lihat dokumentasi utama di direktori `../../docs`.