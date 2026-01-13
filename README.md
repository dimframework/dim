# Dim Framework

Dim adalah web framework Go yang modern, beropini, dan kaya fitur, dirancang untuk membangun RESTful API yang *scalable*. Framework ini memanfaatkan kekuatan standar library `http.ServeMux` (Go 1.22+) sambil menyediakan berbagai *tools* pendukung untuk aplikasi skala *enterprise*.

## Fitur Utama

- **🚀 Modern Routing:** Dibangun di atas `http.ServeMux` Go 1.22+ dengan dukungan native untuk path parameters (`/users/{id}`), pencocokan metode HTTP, dan wildcards.
- **🛡️ Middleware Lengkap:** Dukungan bawaan untuk CORS, CSRF, Rate Limiting, Recovery, dan Structured Logging.
- **🗄️ Integrasi Database Canggih:** 
  - Wrapper untuk `pgx` guna interaksi PostgreSQL performa tinggi.
  - **Automatic Query Tracing** dengan fitur masking data sensitif (perlindungan PII di log).
- **🔐 Keamanan:** 
  - Helper untuk JWT Authentication.
  - Utilitas hashing password yang aman.
  - Validasi berbasis context.
- **📄 Standar JSON:API:** Alat bantu untuk memudahkan pagination, sorting, dan filtering respons API.
- **⚙️ Manajemen Konfigurasi:** Pemuatan konfigurasi berbasis *environment variables*.
- **📝 Structured Logging:** Integrasi logger bawaan untuk *observability* yang lebih baik.

## Instalasi

```bash
go get github.com/nuradiyana/dim
```

## Mulai Cepat (Quick Start)

Berikut adalah contoh lengkap cara membuat API dengan **Database**, **JWT Authentication**, dan **Protected Routes**:

```go
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/nuradiyana/dim"
)

func main() {
	// 1. Setup Database
	dbConfig := dim.DatabaseConfig{
		WriteHost:     "localhost",
		Port:          5432,
		Database:      "myapp_db",
		Username:      "postgres",
		Password:      "secret",
		SSLMode:       "disable",
		MaxConns:      10,
	}
	
	db, err := dim.NewPostgresDatabase(dbConfig)
	if err != nil {
		slog.Error("Gagal connect ke database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// 2. Setup JWT Manager
	jwtConfig := &dim.JWTConfig{
		SigningMethod:     "HS256",
		HMACSecret:        "super-secret-key-change-me",
		AccessTokenExpiry: 24 * time.Hour,
	}
	
	jwtManager, err := dim.NewJWTManager(jwtConfig)
	if err != nil {
		slog.Error("Gagal init JWT", "error", err)
		os.Exit(1)
	}

	// 3. Init Router
	router := dim.NewRouter()

	// Global Middleware
	router.Use(dim.RecoveryMiddleware)
	router.Use(dim.LoggerMiddleware)

	// --- Public Routes ---
	router.Post("/login", func(w http.ResponseWriter, r *http.Request) {
		// Contoh login sederhana (di real app, verifikasi password user dari DB)
		userID := "123"
		email := "user@example.com"

		// Generate Token
		token, _ := jwtManager.GenerateAccessToken(userID, email, nil)
		
		dim.Response(w).Ok(dim.Map{
			"token": token,
		})
	})

	// --- Protected Routes ---
	// Menggunakan Middleware RequireAuth
	api := router.Group("/api", dim.RequireAuth(jwtManager))
	
	api.Get("/profile", func(w http.ResponseWriter, r *http.Request) {
		// ... (kode sebelumnya)
		dim.Response(w).Ok(dim.Map{
			"id":    user.ID,
			"email": user.Email,
			"name":  name,
		})
	})

	// --- Static Files & SPA ---
	// Melayani file statis (css, js, images)
	router.Static("/public/", os.DirFS("./assets"))

	// Melayani SPA (React/Vue/dll) dengan fallback ke index.html
	// Penting: SPA didaftarkan terakhir agar tidak bentrok dengan API
	router.SPA(os.DirFS("./dist"), "index.html")

	// 4. Start Server
	config := dim.ServerConfig{Port: "8080"}
	ctx := context.Background()
	slog.Info("Server berjalan di port :8080")
	
	if err := dim.StartServer(ctx, config, router); err != nil {
		slog.Error("Server stopped", "error", err)
	}
}
```

## Modul Inti

### Routing & Middleware
Dim menyediakan API yang *fluent* untuk mendefinisikan route dan grup. Middleware dapat diterapkan secara global, per grup, atau spesifik untuk route tertentu.

```go
// Middleware Grup
adminGroup := router.Group("/admin", dim.RequireAuth(jwtManager))

// Middleware Spesifik Route
router.Post("/upload", uploadHandler, RateLimitMiddleware)
```

### Database & Observability
Framework ini menyertakan *database tracer* yang kuat yang secara otomatis mencatat query database sambil melindungi informasi sensitif (seperti password, token, dan email).

```go
// Pada konfigurasi database Anda
connConfig, _ := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
// Tracer akan otomatis terpasang jika Anda menggunakan utilitas database dim
```

**Logika Masking:**
Setiap query yang mengandung kata kunci sensitif (`password`, `email`, `token`, `secret`, `api_key`) akan secara otomatis menyembunyikan argumennya menjadi `*****` di dalam log untuk mencegah kebocoran data.

### Utilitas JSON:API
Memudahkan penanganan endpoint *list* yang kompleks dengan standar pagination, filtering, dan sorting.

```go
// Secara otomatis memparsing query params: ?page[number]=1&page[size]=10&sort=-created_at
pagination := dim.GetPagination(r)
filters := dim.GetFilter(r)
sorts := dim.GetSort(r)
```

### Static Files & SPA Support
Dim memudahkan integrasi dengan frontend modern (React, Vue, Svelte) atau sekadar menyajikan aset statis menggunakan interface `fs.FS` (mendukung folder lokal maupun `embed`).

**Static Assets:**
```go
// Menggunakan folder lokal
router.Static("/public/", os.DirFS("./assets"))

// Menggunakan embed (Single Binary)
//go:embed assets/*
var assetsFS embed.FS
router.Static("/public/", assetsFS)
```

**Single Page Application (SPA):**
Menangani fallback routing sisi klien (misal: user refresh di `/dashboard` tidak akan 404, tapi kembali ke `index.html`).
```go
// Pastikan route API didefinisikan SEBELUM memanggil SPA
router.Group("/api", apiHandler)

// Contoh 1: Menggunakan folder lokal (development)
router.SPA(os.DirFS("./dist"), "index.html")

// Contoh 2: Menggunakan embed (production - Single Binary)
//go:embed dist/*
var distFS embed.FS
// Gunakan fs.Sub agar root FS langsung mengarah ke isi folder dist
rootFS, _ := fs.Sub(distFS, "dist")
router.SPA(rootFS, "index.html")
```

### File Handling
Utilitas bawaan untuk menangani upload file secara aman.

```go
file, header, err := dim.GetFile(r, "avatar")
if err != nil {
    // handle error
}
// Validasi dan simpan file...
```

## Kontribusi

Kontribusi sangat diterima! Pastikan setiap fitur baru yang Anda buat disertai dengan *unit test* yang sesuai.

## Lisensi

[MIT](LICENSE)