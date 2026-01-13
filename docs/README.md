# Dokumentasi Framework dim

Dokumentasi lengkap untuk framework HTTP **dim** - framework ringan dan modern untuk membangun RESTful API yang aman di Go.

## 📚 Navigasi Cepat

### Dasar-Dasar
- **[01-Memulai](01-getting-started.md)** - Instalasi dan langkah pertama
- **[02-Arsitektur](02-architecture.md)** - Desain sistem, router modern, dan database layer

### Core HTTP Components
- **[03-Routing](03-routing.md)** - Routing modern (Go 1.22+), Path Params, dan Introspection
- **[04-Handlers](04-handlers.md)** - Menulis handler, parsing request, dan response
- **[05-Middleware](05-middleware.md)** - Sistem middleware, chaining, dan urutan eksekusi
- **[06-Request Context](06-request-context.md)** - Mengakses user dan request ID
- **[07-Response Helpers](07-response-helpers.md)** - Helper untuk JSON response sukses/gagal

### Data & Configuration
- **[08-Database](08-database.md)** - PostgreSQL wrapper, Tracer, Masking, dan Read/Write Split
- **[09-Migrations](09-migrations.md)** - Manajemen skema database (Migrate/Rollback)
- **[10-Konfigurasi](10-configuration.md)** - Manajemen environment variables
- **[11-CLI & Console](11-cli-commands.md)** - Command Line Interface (Serve, Migrate, Route List)

### Security & Validation
- **[12-Autentikasi](12-authentication.md)** - JWT flow, login, register, dan proteksi route
- **[13-Validasi](13-validation.md)** - Validasi input struct
- **[14-Error Handling](14-error-handling.md)** - Standar error response JSON
- **[15-Logging](15-structured-logging.md)** - Structured logging dengan `slog`
- **[16-Keamanan](16-security.md)** - Best practices (CORS, CSRF, Headers)

### Advanced Features
- **[17-File Handling](17-file-handling.md)** - Upload dan validasi file
- **[18-JSON:API Features](18-jsonapi-features.md)** - Standar JSON:API (Sorting, Pagination)
- **[19-Query Filtering](19-query-filtering.md)** - Advanced query filtering system

### Lifecycle & Operations
- **[20-Testing](20-testing.md)** - Strategi testing handler dan middleware
- **[21-Deployment](21-deployment.md)** - Single Binary Deployment (Embed) & Docker
- **[22-Troubleshooting](22-troubleshooting.md)** - Solusi masalah umum
- **[23-API Reference](23-api-reference.md)** - Referensi lengkap API

---

## 🎯 Mulai Dari Mana?

### 1. Fundamental
Pelajari cara kerja router dan server:
*   [01-Memulai](01-getting-started.md)
*   [03-Routing](03-routing.md)
*   [04-Handlers](04-handlers.md)

### 2. Membangun API
Tambahkan logika bisnis dan data:
*   [08-Database](08-database.md)
*   [12-Autentikasi](12-authentication.md)
*   [07-Response Helpers](07-response-helpers.md)

### 3. Production Ready
Siapkan aplikasi untuk dunia nyata:
*   [05-Middleware](05-middleware.md) (Keamanan)
*   [11-CLI & Console](11-cli-commands.md) (Operasional)
*   [21-Deployment](21-deployment.md) (Rilis)

---

## ⚠️ Fitur Unggulan

### Database Tracer & Masking
Framework ini secara otomatis mencatat query database ke log, TETAPI menyembunyikan data sensitif (password, token, email) untuk mencegah kebocoran data. Lihat [08-Database](08-database.md).

### Single Binary Distribution
Dukungan penuh untuk `embed.FS` memungkinkan Anda membungkus seluruh aplikasi (Backend + Frontend React/Vue + Migrasi SQL) menjadi satu file binary yang mudah didistribusikan. Lihat [21-Deployment](21-deployment.md).

### CLI Tooling
Framework menyertakan built-in CLI untuk memudahkan manajemen database (migrate/rollback) dan debugging routes. Lihat [11-CLI & Console](11-cli-commands.md).

---

## 🆘 Butuh Bantuan?

Jika Anda mengalami masalah atau error, cek panduan:
- [22-Troubleshooting](22-troubleshooting.md)

---

## 📝 Lisensi

Semua dokumentasi adalah bagian dari proyek framework dim.
