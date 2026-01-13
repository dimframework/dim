# Dim Framework

Dim adalah web framework Go yang modern, beropini, dan kaya fitur, dirancang untuk membangun RESTful API yang *scalable*. Framework ini memanfaatkan kekuatan standar library `http.ServeMux` (Go 1.22+) sambil menyediakan berbagai *tools* pendukung untuk aplikasi skala *enterprise*.

## Fitur Utama

- **🚀 Modern Routing:** Dibangun di atas `http.ServeMux` Go 1.22+ dengan dukungan native untuk path parameters (`/users/{id}`), pencocokan metode HTTP, dan wildcards.
- **🖥️ CLI & Console:** Sistem CLI built-in yang powerful untuk menjalankan server, migrasi database, dan introspeksi route.
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

Berikut adalah cara standar menginisialisasi aplikasi menggunakan sistem **Console**:

```go
package main

import (
	"log"
	"os"

	"github.com/nuradiyana/dim"
)

func main() {
	// 1. Load Config
	config, err := dim.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	// 2. Setup Database
	db, err := dim.NewPostgresDatabase(config.Database)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// 3. Init Router
	router := dim.NewRouter()
	router.Use(dim.RecoveryMiddleware)
	router.Use(dim.LoggerMiddleware)

	// Register Routes
	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		dim.Response(w).Ok(dim.Map{"message": "Hello from Dim!"})
	})

	// Build Router (Optimasi untuk introspeksi)
	router.Build()

	// 4. Init Console & Run
	console := dim.NewConsole(db, router, config)
	console.RegisterBuiltInCommands()

	// Menjalankan aplikasi via CLI
	// Default: menjalankan server (serve)
	if err := console.Run(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}
```

## CLI & Console

Framework dim dilengkapi dengan CLI bawaan untuk memudahkan operasional. Setelah setup di atas, Anda dapat menggunakan perintah berikut:

```bash
# Menjalankan Server (Default)
go run main.go
go run main.go serve -port 3000

# Manajemen Database
go run main.go migrate              # Jalankan pending migrations
go run main.go migrate:list         # Cek status migrasi
go run main.go migrate:rollback     # Batalkan migrasi terakhir

# Introspeksi Route
go run main.go route:list           # Lihat semua route yang terdaftar

# Bantuan
go run main.go help
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