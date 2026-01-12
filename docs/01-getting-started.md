# Memulai dengan dim

Pelajari cara menginstal dim dan membuat server HTTP pertama Anda.

## Daftar Isi

- [Instalasi](#instalasi)
- [Contoh Minimal](#contoh-minimal)
- [Struktur Proyek](#struktur-proyek)
- [Menjalankan Server](#menjalankan-server)
- [Langkah Selanjutnya](#langkah-selanjutnya)
- [Masalah Umum](#masalah-umum)

---

## Instalasi

### Prasyarat

- **Go 1.22+** - [Download Go](https://golang.org/dl/)
- **PostgreSQL 12+** - [Download PostgreSQL](https://www.postgresql.org/download/)

### Dapatkan Framework

Tambahkan dim ke proyek Anda:

```bash
go get github.com/nuradiyana/dim
```

Atau gunakan di `go.mod`:

```go
require github.com/nuradiyana/dim v0.1.0
```

Kemudian jalankan:

```bash
go mod tidy
```

### Verifikasi Instalasi

Buat file test `test.go`:

```go
package main

import (
    "log"
    "net/http"
    "github.com/nuradiyana/dim"
)

func main() {
    router := dim.NewRouter()
    router.Get("/", func(w http.ResponseWriter, r *http.Request) {
        dim.Json(w, http.StatusOK, map[string]string{"pesan": "Halo dim!"})
    })
    
    log.Println("Server berjalan di :8080")
    http.ListenAndServe(":8080", router)
}
```

Jalankan:

```bash
go run test.go
```

Test:

```bash
curl http://localhost:8080
# {"pesan":"Halo dim!"}
```

✅ Instalasi berhasil!

---

## Contoh Minimal

Berikut aplikasi dim yang paling sederhana:

```go
package main

import (
    "log"
    "net/http"
    "github.com/nuradiyana/dim"
)

func main() {
    // Buat router
    router := dim.NewRouter()

    // Daftar route sederhana
    router.Get("/hello", helloHandler)

    // Mulai server
    log.Println("Mulai server di :8080")
    if err := http.ListenAndServe(":8080", router); err != nil {
        log.Fatal(err)
    }
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
    dim.Json(w, http.StatusOK, map[string]string{
        "pesan": "Halo dari dim!",
    })
}
```

**Yang terjadi:**
1. `dim.NewRouter()` - Membuat HTTP router
2. `router.Get()` - Mendaftar route GET
3. `dim.Json()` - Mengirim response JSON
4. `http.ListenAndServe()` - Mulai server

---

## Struktur Proyek

Aplikasi dim yang khas terlihat seperti ini:

```
myapp/
├── main.go              # Entry point aplikasi
├── handler.go           # HTTP handlers
├── config.go            # Konfigurasi
├── .env                 # Environment variables
├── .env.example         # Template environment
├── go.mod               # Go module
├── go.sum               # Dependencies
└── migrations/          # (opsional)
    └── migrations.go
```

### Struktur Rekomendasi untuk Aplikasi Besar

```
myapp/
├── main.go                # Entry point
├── config.go              # Konfigurasi
├── handler/
│   ├── handler.go         # Registrasi handler
│   ├── auth_handler.go    # Endpoint auth
│   └── user_handler.go    # Endpoint user
├── service/
│   └── auth_service.go    # Business logic
├── store/
│   ├── user_store.go      # User repository
│   └── token_store.go     # Token repository
├── middleware/
│   └── middleware.go      # Custom middleware
├── .env                   # Config environment
├── .env.example           # Template config
├── go.mod                 # Go module
└── migrations/
    └── migrations.go      # Migrasi database
```

---

## Menjalankan Server

### 1. Setup Environment

Buat file `.env`:

```bash
cp .env.example .env
```

Edit `.env`:

```bash
SERVER_PORT=8080
DB_WRITE_HOST=localhost
DB_READ_HOSTS=localhost
DB_PORT=5432
DB_NAME=myapp_db
DB_USER=postgres
DB_PASSWORD=postgres
JWT_SECRET=secret-key-ganti-di-production
```

### 2. Buat Database

```bash
createdb myapp_db
```

### 3. Load Konfigurasi

```go
package main

import (
    "log"
    "github.com/nuradiyana/dim"
)

func main() {
    // Load config dari .env
    cfg, err := dim.LoadConfig()
    if err != nil {
        log.Fatal("Gagal load config:", err)
    }

    // cfg sekarang memiliki semua setting dari environment
    log.Printf("Mulai server di port %s", cfg.Server.Port)
}
```

### 4. Jalankan Aplikasi

```bash
go run main.go
```

Expected output:

```
Mulai server di port 8080
```

Test:

```bash
curl http://localhost:8080/health
```

---

## Aplikasi Starter Lengkap

Aplikasi starter yang lengkap dan berfungsi:

```go
package main

import (
    "context"
    "log"
    "log/slog"
    "net/http"
    "os"

    "github.com/nuradiyana/dim"
)

func main() {
    // Load konfigurasi dari .env file
    cfg, err := dim.LoadConfig()
    if err != nil {
        log.Fatalf("Gagal load config: %v", err)
    }

    // Setup logger terstruktur
    logger := dim.NewLogger(slog.LevelInfo)

    // Buat router baru
    router := dim.NewRouter()

    // Tambah middleware global
    router.Use(dim.Recovery(logger))
    router.Use(dim.LoggerMiddleware(logger))
    router.Use(dim.CORS(cfg.CORS))

    // Daftar routes ke handler
    router.Get("/health", healthHandler)
    router.Get("/", indexHandler)

    // Atur custom 404 handler
    router.SetNotFound(func(w http.ResponseWriter, r *http.Request) {
        dim.JsonError(w, http.StatusNotFound, "Endpoint tidak ditemukan", nil)
    })

    // Mulai server HTTP
    port := cfg.Server.Port
    logger.Info("Mulai server", "port", port)

    if err := http.ListenAndServe(":"+port, router); err != nil {
        logger.Error("Server error", "error", err)
        os.Exit(1)
    }
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    dim.Json(w, http.StatusOK, map[string]string{
        "status": "sehat",
    })
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
    dim.Json(w, http.StatusOK, map[string]string{
        "pesan": "Selamat datang di dim framework!",
        "versi": "0.1.0",
    })
}
```

### Penjelasan Kode

-   `dim.LoadConfig()`: Memuat konfigurasi dari file `.env` ke dalam sebuah struct. Lihat [07-Configuration](07-configuration.md) untuk detailnya.
-   `dim.NewLogger()`: Menginisialisasi *logger* terstruktur baru (menggunakan `slog`). Pelajari lebih lanjut di [12-Structured-Logging](12-structured-logging.md).
-   `dim.NewRouter()`: Membuat instance dari *router* utama.
-   `router.Use(...)`: Menerapkan *middleware* global yang akan dieksekusi pada setiap *request*.
    -   `dim.Recovery`: Menangani *panic* dan mengubahnya menjadi *response* 500 Internal Server Error.
    -   `dim.LoggerMiddleware`: Mencatat setiap *request* yang masuk.
    -   `dim.CORS`: Menangani header Cross-Origin Resource Sharing.
    -   Pelajari lebih lanjut tentang *middleware* di [04-Middleware](04-middleware.md).
-   `router.Get(...)`: Memetakan path URL `GET` ke fungsi *handler* tertentu. Lihat [03-Routing](03-routing.md).
-   `router.SetNotFound(...)`: Mengatur *handler* kustom untuk *request* ke path yang tidak ditemukan.
-   `dim.Json` dan `dim.JsonError`: Fungsi pembantu untuk mengirim *response* JSON dengan mudah. Lihat [11-Response-Helpers](11-response-helpers.md).

Simpan sebagai `main.go`, kemudian:

```bash
go run main.go
```

Test endpoints:

```bash
# Health check
curl http://localhost:8080/health

# Index
curl http://localhost:8080/

# 404
curl http://localhost:8080/notfound
```

---

## Langkah Selanjutnya

Setelah server berjalan:

1. **Pelajari Routing** - Lihat [03-Routing](03-routing.md) untuk menambah routes
2. **Tambah Handlers** - Lihat [16-Handlers](16-handlers.md) untuk handler patterns
3. **Setup Database** - Lihat [06-Database](06-database.md) untuk setup database
4. **Tambah Autentikasi** - Lihat [05-Autentikasi](05-authentication.md)
5. **Middleware** - Lihat [04-Middleware](04-middleware.md) untuk middleware system
6. **Error Handling** - Lihat [08-Error Handling](08-error-handling.md)

Atau langsung ke **[Aplikasi Contoh](../examples/simple/README.md)** untuk contoh lengkap dengan autentikasi dan database.

---

## Masalah Umum

### "Address already in use"

Port sudah digunakan. Pilih salah satu:

1. Kill process sebelumnya:
   ```bash
   lsof -i :8080
   kill -9 <PID>
   ```

2. Gunakan port berbeda di `.env`:
   ```
   SERVER_PORT=8081
   ```

### "Connection refused" (database)

PostgreSQL tidak berjalan:

1. Mulai PostgreSQL:
   ```bash
   # macOS dengan Homebrew
   brew services start postgresql
   
   # Linux dengan systemd
   sudo systemctl start postgresql
   
   # Atau menggunakan Docker
   docker run -d -p 5432:5432 postgres:15
   ```

2. Verifikasi koneksi:
   ```bash
   psql -U postgres
   ```

### "No such file or directory: .env"

Buat file .env:

```bash
cp .env.example .env
```

Atau buat manual:

```bash
cat > .env << EOF
SERVER_PORT=8080
DB_WRITE_HOST=localhost
DB_PORT=5432
DB_NAME=myapp
DB_USER=postgres
DB_PASSWORD=postgres
JWT_SECRET=secret
CORS_ALLOWED_ORIGINS=http://localhost:3000
EOF
```

### Import errors

Pastikan versi terbaru:

```bash
go get -u github.com/nuradiyana/dim
go mod tidy
```

---

## Praktik Terbaik

1. **Selalu gunakan .env files** - Jangan hardcode konfigurasi
2. **Set JWT_SECRET unik** - Ganti default di production
3. **Mulai dengan health endpoint** - Membantu debugging
4. **Gunakan middleware** - Tambah logging, recovery, CORS sejak awal
5. **Test endpoints** - Gunakan curl atau Postman untuk verifikasi
6. **Cek logs** - Lihat startup messages untuk masalah

---

## Apa Selanjutnya?

✅ Server dim pertama Anda sudah berjalan!

Sekarang jelajahi:
- [Arsitektur Overview](02-architecture.md) - Bagaimana dim bekerja
- [Routing](03-routing.md) - Tambah endpoint lebih banyak
- [Aplikasi Contoh](../examples/simple/README.md) - Aplikasi lengkap

Selamat membangun! 🚀
