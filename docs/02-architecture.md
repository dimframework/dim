# Arsitektur Framework dim

Pahami desain dan struktur internal framework dim.

## Daftar Isi

- [Overview Arsitektur](#overview-arsitektur)
- [Komponen Inti](#komponen-inti)
- [Alur Request](#alur-request)
- [Alur Data](#alur-data)
- [Design Principles](#design-principles)
- [Interaksi Komponen](#interaksi-komponen)
- [Model Layering](#model-layering)
- [Connection Pool Architecture](#connection-pool-architecture)

---

## Overview Arsitektur

Framework dim menggunakan arsitektur berlapis dengan fokus pada **simplicity** dan **security**. Struktur framework:

```
┌──────────────────────────────────────────────────────────┐
│                 HTTP Request (net/http)                  │
└──────────────────────────────────────────────────────────┘
                          ↓
┌──────────────────────────────────────────────────────────┐
│               Router (Custom Tree-based)                 │
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
│        Database Layer (Read/Write Split)                │
└─────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────┐
│             PostgreSQL Server                           │
└─────────────────────────────────────────────────────────┘
```

---

## Komponen Inti

### 1. Router

**Tujuan**: Mencocokkan HTTP request ke handler yang tepat.

**Karakteristik**:
- Implementasi tree-based untuk performa O(log n)
- Support HTTP methods: GET, POST, PUT, DELETE, PATCH, OPTIONS
- Path parameters: `/users/:id/posts/:postId`
- Route grouping dengan prefix: `/api`, `/admin`
- Middleware per-route dan per-group
- Pattern matching dengan validation

**Struktur Data**:
```go
type Router struct {
    trees      map[string]*node      // Satu tree per method
    middleware []MiddlewareFunc      // Global middleware
    notFound   HandlerFunc           // 404 handler
}

type node struct {
    path      string
    handler   HandlerFunc
    children  []*node
    params    map[string]string
    middleware []MiddlewareFunc
}
```

**Alur Matching**:
1. Terima request (method + path)
2. Pilih tree berdasarkan method
3. Traverse tree dengan path segments
4. Match dynamic parameters (`:id`)
5. Kumpulkan middleware (global + route-specific)
6. Jalankan handler dengan middleware chain

### 2. Middleware System

**Tujuan**: Memproses request/response dengan cara terstruktur.

**Karakteristik**:
- Chainable middleware execution
- Urutan execution yang tepat KRITIS untuk keamanan
- Middleware dapat memodifikasi context
- Middleware dapat menghentikan chain early
- Per-route dan global middleware support

**Tipe Middleware**:
```go
type MiddlewareFunc func(next HandlerFunc) HandlerFunc
type HandlerFunc func(w http.ResponseWriter, r *http.Request)
```

**Urutan Execution (KRITIS)**:
```
1. Recovery      - Menangkap panic
2. Logger        - Mencatat request
3. CORS          - Menangani CORS headers
4. CSRF          - Validasi CSRF token
5. Auth          - Verifikasi JWT dengan `RequireAuth`
6. Handler       - Business logic
7. Response      - Balikan response
```

⚠️ **Urutan salah dapat menyebabkan security issues!**

### 3. Request Context
`

**Tujuan**: Membawa data request melalui middleware dan handler.

**Data yang Disimpan**:
- User info (dari JWT)
- Path parameters
- Request ID
- Custom values

**Penggunaan**:
```go
// Di handler
user := dim.GetUser(r)           // Get authenticated user
param := dim.GetParam(r, "id")   // Get path parameter
userID := dim.GetRequestID(r)    // Get request ID
```

### 4. Response Formatter

**Tujuan**: Format HTTP response secara konsisten.

**Tipe Response**:

**Single Object**:
```json
{"id": 1, "name": "John"}
```

**Collection**:
```json
[{"id": 1, "name": "John"}, {"id": 2, "name": "Jane"}]
```

**Pagination**:
```json
{
  "data": [...],
  "meta": {
    "page": 1,
    "per_page": 10,
    "total": 100,
    "total_pages": 10
  }
}
```

**Error**:
```json
{
  "message": "Validation failed",
  "errors": {
    "email": "Invalid email",
    "password": "Too weak"
  }
}
```

### 5. Error Handling

**Tujuan**: Error terstruktur dengan status code dan detail.

**Struktur Error**:
```go
type AppError struct {
    Message    string            // Error message
    StatusCode int               // HTTP status
    Errors     map[string]string // Field-level errors
}
```

**Predefined Errors**:
```
400 BadRequest      - Validation failed
401 Unauthorized    - Auth required
403 Forbidden       - Auth failed
404 NotFound        - Resource not found
409 Conflict        - Duplicate entry
500 InternalError   - Server error
```

### 6. Database Layer

**Tujuan**: Abstraksi database dengan read/write splitting.

**Karakteristik**:
- Generic interface (bisa diganti implementation)
- Read/Write connection pool terpisah
- Multiple read replicas dengan load balancing
- Connection pooling (pgx)
- Transaction support

**Arsitektur**:
```
┌──────────────────────────┐
│   Application Code       │
└────────────┬─────────────┘
             │
      ┌──────▼────────┐
      │ Database      │
      │ Interface     │
      └──────┬────────┘
             │
    ┌────────┴────────┐
    │                 │
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

**Read/Write Splitting**:
- `Query()` dan `QueryRow()` → Read pool
- `Exec()` → Write pool
- Transparent ke application code
- Multiple read replicas dengan round-robin load balancing

### 7. Authentication & JWT

**Tujuan**: Autentikasi aman menggunakan JWT.

**Alur**:
```
1. User login dengan email+password
2. Verifikasi password (bcrypt)
3. Generate JWT tokens (access + refresh)
4. Return tokens ke client
5. Client kirim access token di header Authorization
6. Server verifikasi token di middleware
7. Extract user info dari claims
8. Inject ke request context
```

**Token Types**:
- **Access Token** - Short-lived (15 minutes default), untuk API requests
- **Refresh Token** - Long-lived (7 days default), untuk mendapat access token baru

**Claims Structure**:
```json
{
  "sub": 1,                           // User ID
  "email": "user@example.com",
  "username": "john",
  "iat": 1234567890,                  // Issued at
  "exp": 1234568790                   // Expiry
}
```

---

## Alur Request

### Lifecycle Request Normal

```
1. HTTP Request masuk
   └─ GET /api/users/123

2. Router menerima
   └─ Match method (GET) dan path (/api/users/:id)
   └─ Extract params: {id: "123"}

3. Middleware chain mulai
   └─ Recovery middleware
   └─ Logger middleware
   └─ CORS middleware
   └─ CSRF middleware
   └─ Auth middleware
      └─ Verify JWT token
      └─ Get user info
      └─ Set context
   └─ Handler

4. Handler execute
   └─ Get user dari context
   └─ Query database via store
   └─ Format response

5. Response return
   └─ Set headers (Content-Type, etc)
   └─ Write body JSON
   └─ Logger log response

6. Client terima response
```

### Lifecycle Request dengan Error

```
1. HTTP Request masuk
   └─ POST /auth/login

2. Router match endpoint

3. Middleware chain
   └─ Recovery middleware
   └─ Logger middleware
   └─ Middleware jalankan normal

4. Handler execute
   └─ Parse request body
   └─ Validasi input
   └─ ❌ Validation gagal
   └─ Return error response

5. Response error return
   └─ Status 400
   └─ Body: {"message": "...", "errors": {...}}

6. Client terima error response
```

### Lifecycle Request Panic

```
1. HTTP Request masuk

2. Router match

3. Middleware chain
   └─ Recovery middleware ← READY
   └─ Handler
      └─ ❌ PANIC!
      └─ Recovery middleware CATCH
      └─ Log panic
      └─ Return 500 response

4. Response error return

5. Client terima error
```

---

## Alur Data

### Authentication Flow

```
┌──────────┐
│  Client  │
└────┬─────┘
     │ POST /auth/register
     │ {email, username, password}
     ▼
┌──────────────┐
│   Handler    │
├──────────────┤
│ Validasi     │
│ Parse JSON   │
└────┬─────────┘
     │
     ▼
┌──────────────┐
│   Service    │
├──────────────┤
│ Validasi     │
│ Hash pwd     │
│ Check email  │
└────┬─────────┘
     │
     ▼
┌──────────────┐
│   Store      │
├──────────────┤
│ Insert user  │
└────┬─────────┘
     │
     ▼
┌──────────────┐
│   Database   │
├──────────────┤
│ Save to DB   │
└────┬─────────┘
     │
     ▼ (success)
┌──────────────┐
│   Service    │ ← Generate JWT
├──────────────┤
│ Access token │
│ Refresh tok. │
└────┬─────────┘
     │
     ▼
┌──────────────┐
│   Handler    │
├──────────────┤
│ Format JSON  │
└────┬─────────┘
     │
     ▼
┌──────────────┐
│   Response   │
├──────────────┤
│ Status 201   │
│ Tokens       │
└────┬─────────┘
     │
     ▼
┌──────────┐
│  Client  │
└──────────┘
```

### Authenticated Request Flow

```
┌───────────────────────────────┐
│            Client             │
├───────────────────────────────┤
│ GET /api/profile              │
│ Authorization: Bearer <token> │
└───────────────┬───────────────┘
                │
                ▼
┌───────────────────────────────┐
│            Router             │
├───────────────────────────────┤
│ GET /api/profile              │
│ Authorization: Bearer <token> │
└───────────────┬───────────────┘
                │
                ▼
┌───────────────────────────────┐
│            Middleware         │
├───────────────────────────────┤
│ Auth:                         │
│ - Verify JWT                  │
│ - Get claims                  │
│ - Set user                    │
└─────────────┬─────────────────┘
              │
              ▼
┌───────────────────────────────┐
│            Handler            │
├───────────────────────────────┤
│ dim.GetUser()                 │
│ Get profile                   │
└────┬──────────────────────────┘
     │
     ▼
┌───────────────────────────────┐
│            Response           │
├───────────────────────────────┤
│          User data            │
└────┬──────────────────────────┘
     │
     ▼
┌───────────────────────────────┐
│            Client             │
└───────────────────────────────┘
```

---

## Design Principles

### 1. Simplicity

- **Flat structure** - Semua file di satu folder (tidak nested)
- **Clear interfaces** - Interface kecil dan focused
- **Minimal dependencies** - Gunakan stdlib sebanyak mungkin
- **No magic** - Explicit > implicit

### 2. Security First

- **Urutan middleware kritis** - Lihat dokumentasi untuk order yang tepat
- **Single error per field** - Tidak bocor informasi sensitif
- **JWT validation** - Semua request authenticated diverifikasi
- **CSRF protection** - Token-based CSRF mitigation
- **Rate limiting** - DDoS protection

### 3. Composability

- **Middleware chainable** - Mix and match middleware
- **Handler patterns flexible** - Factory, Struct, Direct patterns
- **Database interface generic** - Bisa swap implementation
- **Store pattern** - Repository pattern untuk data access

### 4. Production Ready

- **Error handling** - Structured errors dengan status codes
- **Logging** - Structured logging dengan slog
- **Migrations** - Database versioning
- **Configuration** - Environment-based config
- **Connection pooling** - Efficient resource usage

---

## Interaksi Komponen

### Handler → Service → Store → Database

```go
// Handler menerima request
func getProfileHandler(w http.ResponseWriter, r *http.Request) {
    user := dim.GetUser(r)  // Dari middleware
    
    // Call service
    profile := userService.GetProfile(user.ID)
    
    dim.Json(w, http.StatusOK, profile)
}

// Service business logic
func (s *UserService) GetProfile(userID int64) (*User, error) {
    // Call store
    user, err := s.store.FindByID(userID)
    if err != nil {
        return nil, err
    }
    return user, nil
}

// Store query database
func (s *UserStore) FindByID(userID int64) (*User, error) {
    // Query ke database
    var user User
    err := s.db.QueryRow("SELECT ... WHERE id = $1", userID).
        Scan(&user.ID, &user.Email, ...)
    return &user, err
}

// Database jalankan query
func (db *Database) QueryRow(query string, args ...interface{}) Row {
    // Pilih read connection
    pool := db.getReadPool()
    return pool.QueryRow(context.Background(), query, args...)
}
```

### Middleware → Context → Handler

```go
// Auth middleware
func requireAuthWithManager(jwtManager *dim.JWTManager) dim.MiddlewareFunc {
    return func(next dim.HandlerFunc) dim.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
            token, ok := dim.GetAuthToken(r)
            if !ok {
                dim.Unauthorized(w, "Token otorisasi tidak ada")
                return
            }
            
            // Verifikasi token
            claims, err := jwtManager.VerifyToken(token)
            if err != nil {
                dim.Unauthorized(w, "Token tidak valid")
                return
            }
            
            // Buat objek pengguna dan atur di konteks
            user := &dim.User{ID: claims.UserID, Email: claims.Email}
            r = dim.SetUser(r, user)
            
            // Panggil handler berikutnya
            next(w, r)
        }
    }
}

// Handler access user
func profileHandler(w http.ResponseWriter, r *http.Request) {
    user, _ := dim.GetUser(r) // Ambil dari konteks
    dim.OK(w, user)
}
```

---

## Model Layering

Framework menggunakan 3-tier layering:

### Layer 1: HTTP (net/http)
```
- Router (custom implementation)
- Middleware chain
- Request/Response handling
```

### Layer 2: Business Logic
```
- Services (auth, user, etc)
- Handlers (HTTP endpoint logic)
- Validation
```

### Layer 3: Data Access
```
- Stores (repositories)
- Database interface
- Migrations
```

**Alur Data Antar Layer**:

```
HTTP Layer
├─ Router mematch request
├─ Middleware process request
└─ Handler dijalankan

Business Logic Layer
├─ Handler parse input
├─ Validasi input
└─ Call service

Service Layer
├─ Business logic
├─ Call store
└─ Return result

Data Access Layer
├─ Store execute query
├─ Database jalankan
└─ Return data

Response
├─ Format JSON
├─ Set headers
└─ Send to client
```

---

## Connection Pool Architecture

### Read/Write Connection Splitting

Framework menggunakan **read/write connection splitting** untuk scalability:

```
┌─────────────────────────────────────┐
│          Application Layer          │
└──────────────────┬──────────────────┘
                   │
                   ▼
┌─────────────────────────────────────┐
│     Database Interface              │
├─────────────────────────────────────┤
│ Router request ke connection pool   │
│ Query/QueryRow → Read pool          │
│ Exec → Write pool                   │
└────────┬────────────────────┬───────┘
         │                    │
         ▼                    ▼
┌──────────────────┐   ┌──────────────┐
│   Read Pool 1    │   │  Write Pool  │
│ (Host A)         │   │  (Host M)    │
└────────┬─────────┘   └──────┬───────┘
         │                    │
┌────────▼─────────┐          │
│   Read Pool 2    │          │
│ (Host B)         │          │  
└────────┬─────────┘          │
         │                    │
┌────────▼────────────────────▼──────┐
│   PostgreSQL Servers               │
│ Read replicas (A, B)               │
│ Write primary (M)                  │
└────────────────────────────────────┘
```

### Load Balancing untuk Read

Multiple read replicas di-balance dengan **round-robin**:

```go
// Di database.go
readPools []*pgxpool.Pool    // Array of read pools
readIndex atomic.Uint32       // Round-robin counter

// Saat query
func (db *PostgresDatabase) getReadPool() *pgxpool.Pool {
    idx := db.readIndex.Add(1) - 1
    return db.readPools[idx % len(db.readPools)]
}

// Hasil: Request 1 → Host A
//        Request 2 → Host B
//        Request 3 → Host A
//        Request 4 → Host B
```

### Connection Configuration

```env
# Single write host
DB_WRITE_HOST=db-primary.example.com

# Multiple read hosts (comma-separated)
DB_READ_HOSTS=db-replica1.example.com,db-replica2.example.com

# Pool size
DB_MAX_CONNS=25

# SSL mode
DB_SSL_MODE=require
```

**Fallback Behavior**:
- Jika `DB_READ_HOSTS` kosong → Gunakan write host untuk read
- Jika connect gagal → Error ditangani di layer handler

---

## Summary

Framework dim didesain dengan:

1. **Clear separation of concerns** - HTTP, business logic, data access
2. **Middleware-based extensibility** - Tambah fitur via middleware
3. **Strong type safety** - Go interfaces dan error handling
4. **Performance** - Connection pooling, read/write split, caching
5. **Security** - JWT, CSRF, rate limiting, input validation
6. **Simplicity** - Minimal dependencies, explicit code, flat structure

Pada bagian berikutnya, pelajari [Routing](03-routing.md) untuk detail route registration.

---

**Lihat Juga**:
- [Middleware](04-middleware.md) - Urutan dan konfigurasi middleware
- [Handlers](16-handlers.md) - Handler patterns dan struktur
- [Database](06-database.md) - Konfigurasi database detail
