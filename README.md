# Dim Framework

Dim adalah web framework Go yang **sederhana**, dirancang untuk membantu membangun RESTful API dengan cepat.

Saya membangun Dim di atas `http.ServeMux` (Go 1.22+) agar tetap ringan dan kompatibel dengan standar library Go, namun menambahkan berbagai "bumbu" yang biasanya saya butuhkan di aplikasi nyata: konsistensi struktur, database management, dan keamanan.

## Kenapa Dim?

Daripada menyusun ulang *library* yang sama setiap kali memulai project baru (seperti Router, Middleware, Config, Database), Dim menyediakannya dalam satu paket yang kohesif dan siap pakai.

### Fitur yang "Just Works"

- **Routing:** Menggunakan standar `http.ServeMux` Go 1.22+ dengan dukungan parameter (`/users/{id}`) dan method matching.
- **Productivity CLI:** Command line tools bawaan untuk migrasi database, generate file, dan manajemen server.
- **Database Ready:** Wrapper `pgx` untuk PostgreSQL dengan fitur **Auto-Masking Logs** (data sensitif di log database otomatis disensor).
- **Security First:** Bawaan Rate Limiting, CORS, CSRF, dan JWT helpers.
- **Developer Experience:** Helper untuk JSON response, pagination, sorting, dan error handling yang konsisten.

## Instalasi

```bash
go get github.com/dimframework/dim
```

## Dokumentasi Lengkap

Panduan lengkap, referensi API, dan tutorial mendalam tersedia di folder [docs](docs/):

- [Getting Started](docs/01-getting-started.md)
- [Architecture](docs/02-architecture.md)
- [Routing](docs/03-routing.md)
- [Handlers](docs/04-handlers.md)
- [Middleware](docs/05-middleware.md)
- [Request Context](docs/06-request-context.md)
- [Response Helpers](docs/07-response-helpers.md)
- [Database](docs/08-database.md)
- [Migrations](docs/09-migrations.md)
- [Configuration](docs/10-configuration.md)
- [CLI Commands](docs/11-cli-commands.md)
- [Authentication](docs/12-authentication.md)
- [Validation](docs/13-validation.md)
- [Error Handling](docs/14-error-handling.md)
- [Dan lainnya](docs/README.md)

## Cara Pakai

Berikut adalah setup standar aplikasi menggunakan **Console** agar fitur CLI aktif:

```go
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/dimframework/dim"
)

func main() {
	// 1. Load Config & DB
	config, _ := dim.LoadConfig()
	db, _ := dim.NewPostgresDatabase(config.Database)
	defer db.Close()

	// 2. Setup Router
	router := dim.NewRouter()
	router.Use(dim.RecoveryMiddleware)
	router.Use(dim.LoggerMiddleware)

	// 3. Define Routes
	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		dim.Response(w).Ok(dim.Map{"message": "Halo, Dunia!"})
	})

	// 4. Run Console
	router.Build() // Siapkan router untuk introspeksi
	console := dim.NewConsole(db, router, config)
	console.RegisterBuiltInCommands()

	// Menjalankan aplikasi
	if err := console.Run(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}
```

## CLI Tools

Dim menyertakan tool CLI untuk membantu workflow development Anda sehari-hari.

```bash
# Menjalankan Server
go run main.go serve

# Membuat File Migrasi Baru (Timestamped)
go run main.go make:migration create_users_table

# Menjalankan Migrasi Database
go run main.go migrate

# Melihat Status Migrasi
go run main.go migrate:list

# Melihat Daftar Route
go run main.go route:list

# Bantuan Lengkap
go run main.go help
```

## Fitur Populer

### 1. Database Migrations
Tidak perlu tool eksternal atau file `.sql` manual. Migrasi ditulis dalam Go, menggunakan `init()` function untuk registrasi otomatis, dan mendukung timestamp versioning.

### 2. Smart Logging
Framework ini peduli pada keamanan data log Anda. Query database yang mengandung field sensitif seperti `password`, `token`, atau `api_key` akan otomatis disensor (`*****`) sebelum dicetak ke log.

### 3. SPA & Static Files
Dim memudahkan integrasi dengan frontend modern (React, Vue, Svelte). Method `router.SPA()` menangani fallback routing di sisi klien sehingga user yang me-refresh halaman `/dashboard` tidak akan terkena 404 error.

## Kontribusi

Project ini dikembangkan secara terbuka. Jika Anda menemukan bug atau memiliki ide untuk perbaikan, silakan buka Issue atau kirimkan Pull Request.

## Lisensi

[MIT](LICENSE)
