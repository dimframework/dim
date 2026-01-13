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

- **Go 1.22+** - Framework ini menggunakan `http.ServeMux` modern.
- **PostgreSQL 12+** - (Opsional) Jika menggunakan database.

### Dapatkan Framework

Tambahkan dim ke proyek Anda:

```bash
go get github.com/nuradiyana/dim
```

---

## Contoh Minimal

Berikut aplikasi dim yang paling sederhana:

```go
package main

import (
    "context"
    "log/slog"
    "net/http"
    
    "github.com/nuradiyana/dim"
)

func main() {
    // 1. Buat router
    router := dim.NewRouter()

    // 2. Daftar route sederhana
    router.Get("/hello", func(w http.ResponseWriter, r *http.Request) {
        dim.Response(w).Ok(dim.Map{
            "pesan": "Halo dari dim!",
        })
    })

    // 3. Konfigurasi Server
    config := dim.ServerConfig{
        Port: "8080",
    }

    // 4. Mulai server dengan Graceful Shutdown
    ctx := context.Background()
    slog.Info("Mulai server di :8080")
    
    if err := dim.StartServer(ctx, config, router); err != nil {
        slog.Error("Gagal menjalankan server", "error", err)
    }
}
```

**Yang terjadi:**
1. `dim.NewRouter()` - Membuat HTTP router modern.
2. `router.Get()` - Mendaftar route GET.
3. `dim.Response(w).Ok()` - Helper fluent untuk mengirim response JSON.
4. `dim.StartServer()` - Menjalankan server dengan *graceful shutdown* built-in.

---

## Struktur Proyek

Aplikasi dim yang khas terlihat seperti ini:

```
myapp/
├── main.go              # Entry point aplikasi
├── handler.go           # HTTP handlers
├── config.go            # Konfigurasi
├── .env                 # Environment variables
├── go.mod               # Go module
└── go.sum               # Dependencies
```

### Struktur Rekomendasi untuk Aplikasi Besar

```
myapp/
├── cmd/
│   └── api/
│       └── main.go        # Entry point
├── internal/
│   ├── config/            # Load config
│   ├── handler/           # HTTP handlers
│   ├── service/           # Business logic
│   └── repository/        # Data access (PostgreSQL)
├── migrations/            # SQL migrations
├── public/                # Static assets (jika ada)
├── .env                   # Config local
└── go.mod
```

---

## Menjalankan Server

### 1. Setup Environment

Buat file `.env`:

```bash
SERVER_PORT=8080
DB_WRITE_HOST=localhost
DB_PORT=5432
DB_NAME=myapp_db
DB_USER=postgres
DB_PASSWORD=postgres
JWT_SECRET=rahasia-sangat-panjang-dan-aman
```

### 2. Jalankan Aplikasi

```bash
go run main.go
```

Test endpoint:

```bash
curl http://localhost:8080/hello
```

---

## Aplikasi Starter Lengkap

Berikut contoh aplikasi yang menggunakan Middleware, Database, dan Routing:

```go
package main

import (
    "context"
    "log/slog"
    "net/http"
    "os"

    "github.com/nuradiyana/dim"
)

func main() {
    // 1. Setup Logger
    logger := dim.NewLogger(slog.LevelInfo)

    // 2. Setup Router & Middleware
    router := dim.NewRouter()
    
    // Global Middleware (Urutan PENTING!)
    router.Use(dim.Recovery(logger))
    router.Use(dim.LoggerMiddleware(logger))

    // 3. Public Routes
    router.Get("/", func(w http.ResponseWriter, r *http.Request) {
        dim.Response(w).Ok(dim.Map{
            "app": "Dim API",
            "version": "1.0.0",
        })
    })

    // 4. Start Server
    config := dim.ServerConfig{Port: "8080"}
    
    logger.Info("Server berjalan", "port", config.Port)
    if err := dim.StartServer(context.Background(), config, router); err != nil {
        logger.Error("Server berhenti", "error", err)
        os.Exit(1)
    }
}
```

---

## Langkah Selanjutnya

Setelah server berjalan:

1. **Pelajari Routing** - Lihat [03-Routing](03-routing.md) untuk sintaks parameter `{id}`.
2. **Setup Database** - Lihat [06-Database](06-database.md) untuk koneksi PostgreSQL dan Tracer.
3. **Tambah Autentikasi** - Lihat [05-Autentikasi](05-authentication.md) untuk JWT.
4. **Deploy SPA** - Lihat [17-Deployment](17-deployment.md) untuk serving React/Vue.

---

## Masalah Umum

### "Address already in use"
Port 8080 sedang dipakai. Matikan proses lama atau ganti port di `.env`.

### "Connection refused"
Pastikan PostgreSQL berjalan jika Anda mengonfigurasi koneksi database.

### "undefined: dim.StartServer"
Pastikan Anda menggunakan versi `dim` terbaru. Jalankan `go get -u github.com/nuradiyana/dim`.