# Deployment

Panduan untuk mendeploy aplikasi berbasis framework dim ke production.

## Daftar Isi

- [Single Binary Deployment](#single-binary-deployment)
- [Docker Deployment](#docker-deployment)
- [Environment Variables](#environment-variables)
- [Graceful Shutdown](#graceful-shutdown)

---

## Single Binary Deployment

Salah satu kekuatan utama Go adalah kemampuan untuk menghasilkan satu file binary statis yang berisi semua kebutuhan aplikasi, termasuk aset frontend (HTML/CSS/JS).

### Menggunakan `embed`

Framework dim mendukung penuh `embed.FS` untuk routing file statis dan SPA.

**Struktur Proyek**:
```
/
├── main.go
├── go.mod
└── dist/          # Folder hasil build frontend (React/Vue)
    ├── index.html
    └── assets/
        └── style.css
```

**Kode `main.go`**:

```go
package main

import (
    "embed"
    "io/fs"
    "github.com/dimframework/dim"
)

//go:embed dist/*
var distFS embed.FS

func main() {
    router := dim.NewRouter()

    // 1. API Routes
    router.Group("/api", apiHandler)

    // 2. Setup File System
    // Sub-root ke folder "dist" agar path dimulai dari root folder tersebut
    publicFS, _ := fs.Sub(distFS, "dist")

    // 3. Static Files
    // Melayani file statis seperti /assets/style.css
    router.Static("/assets/", publicFS)

    // 4. SPA Fallback
    // Menangani routing frontend (React/Vue)
    router.SPA(publicFS, "index.html")

    // Start Server
    dim.StartServer(context.Background(), config, router)
}
```

### Build Perintah

```bash
# Build binary untuk Linux (misal server Ubuntu)
GOOS=linux GOARCH=amd64 go build -o app-server main.go

# Upload hanya file 'app-server' ke server Anda
scp app-server user@your-server:/opt/app/
```

Di server, Anda hanya perlu satu file ini dan file `.env` (opsional). Tidak perlu install Node.js, Nginx, atau dependensi lain.

---

## Docker Deployment

Jika Anda lebih suka menggunakan container:

**Dockerfile**:

```dockerfile
# Stage 1: Build Frontend (opsional jika repo menyatu)
# FROM node:18 AS frontend-builder
# ... npm run build ...

# Stage 2: Build Backend
FROM golang:1.22 AS builder
WORKDIR /app
COPY . .
# COPY --from=frontend-builder /app/dist ./dist
RUN CGO_ENABLED=0 GOOS=linux go build -o server .

# Stage 3: Final Image
FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/server .
COPY .env . 

EXPOSE 8080
CMD ["./server"]
```

---

## Environment Variables

Pastikan variabel berikut diset di server production:

| Variable | Deskripsi | Contoh |
|----------|-----------|--------|
| `PORT` | Port aplikasi | `8080` |
| `DATABASE_URL` | Koneksi DB | `postgres://user:pass@host:5432/db` |
| `JWT_SECRET` | Secret key (Wajib) | `random-string-panjang` |
| `ENV` | Environment | `production` |

---

## Graceful Shutdown

Framework dim secara otomatis menangani `SIGINT` dan `SIGTERM`.

Saat Anda me-restart service (misal via `systemd` atau `docker restart`):
1.  Server berhenti menerima koneksi baru.
2.  Server menunggu request yang sedang berjalan selesai (hingga batas `ShutdownTimeout`).
3.  Koneksi database ditutup dengan aman.
4.  Proses berhenti.

Ini memastikan tidak ada request pengguna yang terputus di tengah jalan saat deployment.
