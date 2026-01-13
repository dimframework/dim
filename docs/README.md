# Dokumentasi Framework dim

Dokumentasi lengkap untuk framework HTTP **dim** - framework ringan dan modern untuk membangun RESTful API yang aman di Go.

## 📚 Navigasi Cepat

### Dasar-Dasar
- **[01-Memulai](01-getting-started.md)** - Instalasi dan langkah pertama
- **[02-Arsitektur](02-architecture.md)** - Desain sistem, router modern, dan database layer

### Fitur Inti
- **[03-Routing](03-routing.md)** - Routing modern (Go 1.22+), Path Params, dan Static/SPA
- **[04-Middleware](04-middleware.md)** - Sistem middleware, chaining, dan urutan eksekusi
- **[05-Autentikasi](05-authentication.md)** - JWT flow, login, register, dan proteksi route

### Data & Konfigurasi
- **[06-Database](06-database.md)** - PostgreSQL wrapper, Tracer, Masking, dan Read/Write Split
- **[07-Konfigurasi](07-configuration.md)** - Manajemen environment variables
- **[08-Error Handling](08-error-handling.md)** - Standar error response JSON

### Utilitas & Support
- **[09-Validasi](09-validation.md)** - Validasi input
- **[10-Request Context](10-request-context.md)** - Mengakses user dan request ID
- **[20-JSON:API](20-jsonapi-features.md)** - Filtering, Pagination, dan Sorting standar JSON:API
- **[11-Response Helpers](11-response-helpers.md)** - Helper untuk JSON response sukses/gagal
- **[12-Logging](12-structured-logging.md)** - Structured logging dengan `slog`
- **[21-File Handling](21-file-handling.md)** - Upload dan validasi file

### Topik Lanjutan
- **[13-Migrations](13-migrations.md)** - Manajemen skema database
- **[14-Keamanan](14-security.md)** - Best practices (CORS, CSRF, Headers)
- **[15-Testing](15-testing.md)** - Strategi testing handler dan middleware
- **[17-Deployment](17-deployment.md)** - Single Binary Deployment (Embed) & Docker

---

## 🎯 Mulai Dari Mana?

### 1. Fundamental
Pelajari cara kerja router dan server:
*   [01-Memulai](01-getting-started.md)
*   [03-Routing](03-routing.md)

### 2. Membangun API
Tambahkan logika bisnis dan data:
*   [06-Database](06-database.md)
*   [05-Autentikasi](05-authentication.md)
*   [11-Response Helpers](11-response-helpers.md)

### 3. Production Ready
Siapkan aplikasi untuk dunia nyata:
*   [04-Middleware](04-middleware.md) (Keamanan)
*   [12-Logging](12-structured-logging.md) (Observability)
*   [17-Deployment](17-deployment.md) (Rilis)

---

## ⚠️ Fitur Unggulan

### Database Tracer & Masking
Framework ini secara otomatis mencatat query database ke log, TETAPI menyembunyikan data sensitif (password, token, email) untuk mencegah kebocoran data. Lihat [06-Database](06-database.md).

### Single Binary Distribution
Dukungan penuh untuk `embed.FS` memungkinkan Anda membungkus seluruh aplikasi (Backend + Frontend React/Vue + Migrasi SQL) menjadi satu file binary yang mudah didistribusikan. Lihat [17-Deployment](17-deployment.md).

---

## 🆘 Butuh Bantuan?

Jika Anda mengalami masalah atau error, cek panduan:
- [18-Troubleshooting](18-troubleshooting.md)

---

## 📝 Lisensi

Semua dokumentasi adalah bagian dari proyek framework dim.