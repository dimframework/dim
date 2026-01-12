# Dokumentasi Framework dim

Dokumentasi lengkap untuk framework HTTP **dim** - sebuah framework ringan yang fokus pada keamanan untuk membangun server HTTP di Go.

## 📚 Navigasi Cepat

### Dasar-Dasar
- **[01-Memulai](01-getting-started.md)** - Instalasi dan langkah pertama
- **[02-Arsitektur](02-architecture.md)** - Desain sistem dan overview

### Fitur Inti
- **[03-Routing](03-routing.md)** - HTTP routing dan path matching
- **[04-Middleware](04-middleware.md)** - Sistem middleware dan urutan
- **[05-Autentikasi](05-authentication.md)** - JWT dan alur autentikasi

### Data & Konfigurasi
- **[06-Database](06-database.md)** - Koneksi database dan connection pool
- **[07-Konfigurasi](07-configuration.md)** - Konfigurasi environment
- **[08-Error Handling](08-error-handling.md)** - Tipe error dan penanganan

### Utilitas & Support
- **[09-Validasi](09-validation.md)** - Sistem validasi input
- **[20-Query Filtering](20-query-filtering.md)** - Filter query parameter dengan type-safe parsing
- **[10-Request Context](10-request-context.md)** - Manajemen context
- **[11-Response Helpers](11-response-helpers.md)** - Formatting response
- **[12-Logging](12-structured-logging.md)** - Sistem logging
- **[21-File Handling](21-file-handling.md)** - Upload file, MIME type detection, file serving

### Topik Lanjutan
- **[13-Migrations](13-migrations.md)** - Migrasi database
- **[14-Keamanan](14-security.md)** - Best practices keamanan
- **[15-Testing](15-testing.md)** - Pattern testing
- **[16-Handlers](16-handlers.md)** - Pattern handler dan arsitektur
- **[17-Deployment](17-deployment.md)** - Deployment ke production
- **[18-Troubleshooting](18-troubleshooting.md)** - Masalah umum dan solusi

### Referensi
- **[19-API Reference](19-api-reference.md)** - Dokumentasi API lengkap

---

## 🎯 Path Belajar

### Untuk Pemula
Mulai dari sini jika Anda baru dengan dim:
1. [01-Memulai](01-getting-started.md)
2. [02-Arsitektur](02-architecture.md)
3. [03-Routing](03-routing.md)
4. [16-Handlers](16-handlers.md)
5. [04-Middleware](04-middleware.md)
6. [11-Response Helpers](11-response-helpers.md)
7. [19-API Reference](19-api-reference.md)

### Untuk Pengguna Menengah
Bangun di atas dasar-dasarnya:
1. [05-Autentikasi](05-authentication.md)
2. [08-Error Handling](08-error-handling.md)
3. [09-Validasi](09-validation.md)
4. [06-Database](06-database.md)
5. [12-Logging](12-structured-logging.md)
6. [07-Konfigurasi](07-configuration.md)

### Untuk Pengguna Advanced
Kuasai framework sepenuhnya:
1. [14-Keamanan](14-security.md)
2. [15-Testing](15-testing.md)
3. [13-Migrations](13-migrations.md)
4. [17-Deployment](17-deployment.md)
5. [18-Troubleshooting](18-troubleshooting.md)
6. [19-API Reference](19-api-reference.md)

---

## ⚠️ Poin Penting

### Urutan Middleware KRITIS
Urutan middleware sangat penting! Lihat [04-Middleware](04-middleware.md):
```
Recovery → Logger → CORS → CSRF → Auth → Handler
```
Urutan salah bisa merusak fungsionalitas dan keamanan!

### Single Error Per Field
dim menggunakan model validasi single-error-per-field. Pelajari di [09-Validasi](09-validation.md).

### Konfigurasi JWT
Jangan pernah hardcode secret key. Gunakan environment variables. Lihat [05-Autentikasi](05-authentication.md).

---

## 🔍 Mencari Apa yang Anda Butuhkan

### Berdasarkan Fitur
- **Routing**: [03-Routing](03-routing.md)
- **Autentikasi & JWT**: [05-Autentikasi](05-authentication.md)
- **Database**: [06-Database](06-database.md)
- **Middleware**: [04-Middleware](04-middleware.md)
- **Error Handling**: [08-Error Handling](08-error-handling.md)
- **Validasi**: [09-Validasi](09-validation.md)
- **Query Filtering**: [20-Query Filtering](20-query-filtering.md)
- **File Handling**: [21-File Handling](21-file-handling.md)
- **HTTP Responses**: [11-Response Helpers](11-response-helpers.md)
- **Konfigurasi**: [07-Konfigurasi](07-configuration.md)
- **Logging**: [12-Logging](12-structured-logging.md)
- **Testing**: [15-Testing](15-testing.md)
- **Keamanan**: [14-Keamanan](14-security.md)

### Berdasarkan Use Case
- **Membangun API baru**: [01-Memulai](01-getting-started.md) → [03-Routing](03-routing.md) → [05-Autentikasi](05-authentication.md)
- **Melindungi routes**: [04-Middleware](04-middleware.md) → [05-Autentikasi](05-authentication.md) → [14-Keamanan](14-security.md)
- **Validasi input**: [09-Validasi](09-validation.md) → [08-Error Handling](08-error-handling.md)
- **Filter & query parameter**: [20-Query Filtering](20-query-filtering.md) → [08-Error Handling](08-error-handling.md)
- **Operasi database**: [06-Database](06-database.md) → [13-Migrations](13-migrations.md)
- **Debug masalah**: [18-Troubleshooting](18-troubleshooting.md) → [12-Logging](12-structured-logging.md)
- **Production deployment**: [17-Deployment](17-deployment.md) → [14-Keamanan](14-security.md)

---

## 📖 Overview Framework

**dim** adalah framework HTTP ringan yang dibangun di atas `net/http` Go dengan:

- **Routing sederhana** - Router berbasis tree dengan path parameters
- **Sistem middleware** - Middleware chainable dengan urutan yang benar
- **Autentikasi JWT** - Alur auth lengkap (register, login, refresh, logout)
- **Abstraksi database** - PostgreSQL dengan read/write splitting
- **Error handling** - Error terstruktur dengan detail level field
- **Validasi** - Fluent validation API
- **Middleware keamanan** - CORS, CSRF, rate limiting
- **Production-ready** - Logging, migrations, konfigurasi

---

## 🚀 Quick Start

```bash
# Clone dan setup
git clone https://github.com/yourusername/dim
cd dim

# Review aplikasi contoh
cd examples/simple
cp .env.example .env

# Edit .env dengan database credentials Anda
vim .env

# Buat database
createdb dim_simple

# Jalankan contoh
go run .
```

Lihat [01-Memulai](01-getting-started.md) untuk instruksi detail.

---

## 📋 Statistik Dokumentasi

- **Total Files**: 19 guides + README
- **Total Kata**: ~30,000+ kata (Bahasa Indonesia)
- **Total Waktu Baca**: ~3-4 jam
- **Contoh Kode**: 150+
- **Diagram**: Diagram arsitektur ASCII

---

## 🆘 Butuh Bantuan?

1. **Tidak tahu harus mulai dari mana?** → [01-Memulai](01-getting-started.md)
2. **Mengalami error?** → [18-Troubleshooting](18-troubleshooting.md)
3. **Butuh API docs khusus?** → [19-API Reference](19-api-reference.md)
4. **Kekhawatiran keamanan?** → [14-Keamanan](14-security.md)
5. **Cara membuat handlers?** → [16-Handlers](16-handlers.md)

---

## 📝 Kontribusi

Perbaikan dokumentasi sangat diterima! Pastikan:
- Contoh kode sudah ditest dan berfungsi
- Link valid dan tidak broken
- Formatting konsisten dengan dokumentasi lain
- Contoh baru menyertakan error handling

---

## 📄 Lisensi

Semua dokumentasi adalah bagian dari proyek framework dim.
