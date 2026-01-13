# Arsitektur Framework dim

Pahami desain dan struktur internal framework dim.

## Daftar Isi

- [Overview Arsitektur](#overview-arsitektur)
- [Komponen Inti](#komponen-inti)
- [Alur Request](#alur-request)
- [Alur Data](#alur-data)
- [Design Principles](#design-principles)
- [Database Layer](#database-layer)

---

## Overview Arsitektur

Framework dim menggunakan arsitektur berlapis dengan fokus pada **simplicity**, **security**, dan **observability**. Struktur framework:

```
┌──────────────────────────────────────────────────────────┐
│                 HTTP Request (net/http)                  │
└──────────────────────────────────────────────────────────┘
                          ↓
┌──────────────────────────────────────────────────────────┐
│               Router (http.ServeMux Go 1.22+)            │
└──────────────────────────────────────────────────────────┘
                         ↓
┌──────────────────────────────────────────────────────────┐
│    Middleware Chain (ordered execution)                  │
├──────────────────────────────────────────────────────────┤
│ 1. Recovery   - Catch panics                             │
│ 2. Logger     - Log requests/responses                   │
│ 3. CORS       - Cross-origin handling                    │
│ 4. CSRF       - Token validation                         │
│ 5. Auth       - JWT verification                         │
└──────────────────────────────────────────────────────────┘
                      ↓
┌─────────────────────────────────────────────────────────┐
│           Handler (HTTP business logic)                 │
└─────────────────────────────────────────────────────────┘
                      ↓
┌───────────────────┬──────────────────┬──────────────────┐
│   Service         │   Store          │   Validation     │
└───────────────────┴──────────────────┴──────────────────┘
                         ↓
┌─────────────────────────────────────────────────────────┐
│        Database Layer (Read/Write Split + Trace)        │
└─────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────┐
│             PostgreSQL Server                           │
└─────────────────────────────────────────────────────────┘
```

---

## Komponen Inti

### 1. Router Modern

**Tujuan**: Mencocokkan HTTP request ke handler yang tepat.

**Karakteristik**:
- Dibangun di atas `http.ServeMux` (standar Go 1.22+)
- Support HTTP methods: GET, POST, PUT, DELETE, dll.
- Path parameters: `/users/{id}`
- Wildcards: `/files/{path...}`
- **Static & SPA Support**: Dukungan native untuk `fs.FS` (lokal & embed)

### 2. Middleware System

**Tujuan**: Memproses request/response dengan cara terstruktur.

**Karakteristik**:
- Chainable middleware execution
- **Security First**: Default middleware stack mencakup Recovery dan Logger
- **Flexible**: Mendukung middleware global, per-grup, dan per-route

**Urutan Execution (KRITIS)**:
1.  **Recovery**: Menjaga server tetap hidup saat panic.
2.  **Logger**: Mencatat semua akses (termasuk yang gagal).
3.  **CORS/CSRF**: Keamanan browser.
4.  **Auth**: Validasi identitas pengguna.
5.  **Handler**: Logika bisnis.

### 3. Request Context

**Tujuan**: Membawa data request melalui middleware dan handler.

**Data yang Disimpan**:
- User info (dari JWT) via `dim.GetUser(r)`
- Path parameters via `r.PathValue("id")`
- Request ID

### 4. Database Layer & Observability

**Tujuan**: Abstraksi database dengan fitur enterprise.

**Fitur Unggulan**:
*   **Automatic Tracer**: Mencatat setiap query SQL dan durasinya.
*   **Data Masking**: Secara otomatis menyembunyikan argumen sensitif (password, token, email) di log query.
*   **Read/Write Splitting**: Memisahkan beban kerja ke primary dan replica.

```
Log Output:
INFO msg="query executed" sql="INSERT INTO users..." args=["*****", "*****"] duration=2ms
```

### 5. Authentication & JWT

**Tujuan**: Autentikasi aman menggunakan JWT.

**Alur**:
1.  User login → Server verifikasi password (bcrypt).
2.  Generate JWT (Access + Refresh Token).
3.  Client mengirim token di header `Authorization`.
4.  Middleware `RequireAuth` memverifikasi token & inject user ke context.

---

## Alur Request

### Lifecycle Request Normal

```
1. HTTP Request masuk
   └─ GET /api/users/123

2. Router menerima
   └─ Match method (GET) dan path (/api/users/{id})
   └─ Extract params: id="123"

3. Middleware chain mulai
   └─ Recovery middleware
   └─ Logger middleware
   └─ Auth middleware
      └─ Verify JWT token
      └─ Set user context
   └─ Handler

4. Handler execute
   └─ Get user dari context
   └─ Query database via store (Auto-Trace & Masking)
   └─ Format response

5. Response return
   └─ Set headers
   └─ Write body JSON
   └─ Logger log response (200 OK)
```

---

## Database Layer

### Arsitektur Koneksi

```
┌──────────────────────────┐
│   Application Code       │
└────────────┬─────────────┘
             │
      ┌──────▼────────┐
      │  Tracer &     │  <-- Masking terjadi di sini
      │  Masking      │
      └──────┬────────┘
             │
    ┌────────┴────────┐
    │ Routing Logic   │
    ▼                 ▼
┌─────────┐      ┌──────────┐
│ Read    │      │ Write    │
│ Pool(s) │      │ Pool     │
│ (LB)    │      │ (Single) │
└────┬────┘      └────┬─────┘
     │                │
  ┌──▼────────────────▼──┐
  │  PostgreSQL Servers  │
  └──────────────────────┘
```

**Routing Logic**:
- `Query()`/`QueryRow()` (SELECT) → Read Pool (Load Balanced)
- `Exec()` (INSERT/UPDATE) → Write Pool
- Transaksi (`Begin`) → Selalu Write Pool

---

## Design Principles

### 1. Simplicity & Standards
Menggunakan standar library Go (`net/http`, `slog`) sebanyak mungkin untuk mengurangi dependensi eksternal dan memudahkan maintenance.

### 2. Security by Default
- Middleware urutan yang ketat.
- Masking otomatis data sensitif di log.
- Header keamanan otomatis untuk Static/SPA.

### 3. Production Ready
- **Graceful Shutdown**: Server menangani sinyal OS dengan benar.
- **Single Binary**: Dukungan `embed` untuk deployment mudah.
- **Structured Logging**: Log dalam format JSON/Text yang mudah di-parse.