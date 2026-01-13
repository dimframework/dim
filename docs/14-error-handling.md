# Error Handling di Framework dim

Pelajari cara menangani error dengan terstruktur di framework dim.

## Daftar Isi

- [Konsep Error Handling](#konsep-error-handling)
- [Error Types](#error-types)
- [HTTP Status Codes](#http-status-codes)
- [Error Response Format](#error-response-format)
- [Validation Errors](#validation-errors)
- [Database Errors](#database-errors)
- [Custom Error Types](#custom-error-types)
- [Error Logging](#error-logging)
- [Error Recovery](#error-recovery)
- [Praktik Terbaik](#best-practices)

---

## Konsep Error Handling

### Error Flow

```
Request
  ↓
Middleware/Handler
  ↓ Error terjadi?
  ├─ YES → Create error response
  └─ NO → Continue
  ↓
Return Response
  ├─ Success: 200, 201, 204, etc
  └─ Error: 400, 401, 403, 404, 409, 500, etc
  ↓
Client
```

### Error Handling Principles

Framework dim mengikuti prinsip:
1. **Structured errors** - Setiap error memiliki struktur tertentu
2. **HTTP status codes** - Gunakan status code yang tepat
3. **Field-level errors** - Validasi per-field, bukan generic
4. **No sensitive data** - Jangan expose internal details
5. **Logging** - Log semua errors untuk debugging

---

## Error Types

### Struct AppError

Struktur error utama yang digunakan di seluruh framework.

```go
type AppError struct {
    Message    string            // Pesan error yang aman untuk ditampilkan ke pengguna.
    StatusCode int               // Kode status HTTP yang sesuai (misalnya, 400, 404, 500).
    Errors     map[string]string // Opsional: Error per-field untuk validasi.
}
```

### Error Bawaan Framework

Framework menyediakan beberapa variabel error siap pakai untuk kasus umum. Variabel-variabel ini sudah memiliki `StatusCode` yang sesuai, namun `Message` dan `Errors`-nya bisa Anda timpa.

```go
var (
    ErrBadRequest          = NewAppError("Permintaan tidak valid", 400)
    ErrValidation          = NewAppError("Validasi gagal", 400)
    ErrUnauthorized        = NewAppError("Tidak terotorisasi", 401)
    ErrForbidden           = NewAppError("Dilarang", 403)
    ErrNotFound            = NewAppError("Tidak ditemukan", 404)
    ErrConflict            = NewAppError("Konflik", 409)
    ErrInternalServerError = NewAppError("Kesalahan server internal", 500)
)
```

### Contoh Error Level Aplikasi

Anda sangat dianjurkan untuk mendefinisikan error spesifik untuk aplikasi Anda sendiri agar penanganan error menjadi lebih bersih dan dapat di-reuse.

```go
// Definisikan error-error ini di dalam aplikasi Anda
var (
    ErrEmailAlreadyExists    = &dim.AppError{StatusCode: 409, Message: "Email sudah terdaftar"}
    ErrUsernameAlreadyExists = &dim.AppError{StatusCode: 409, Message: "Username sudah digunakan"}
    ErrPasswordTooWeak       = &dim.AppError{StatusCode: 400, Message: "Password terlalu lemah"}
    ErrInvalidCredentials    = &dim.AppError{StatusCode: 401, Message: "Email atau password salah"}
    ErrTokenExpired          = &dim.AppError{StatusCode: 401, Message: "Token sudah kedaluwarsa"}
    ErrTokenInvalid          = &dim.AppError{StatusCode: 401, Message: "Token tidak valid"}
    ErrUserNotFound          = &dim.AppError{StatusCode: 404, Message: "Pengguna tidak ditemukan"}
)
```
**Penggunaan:**
```go
func (s *AuthService) Register(email string) error {
    exists, _ := s.userStore.EmailExists(email)
    if exists {
        return ErrEmailAlreadyExists // Mengembalikan error yang sudah didefinisikan
    }
    // ...
}
```

### Type-Checking AppError

Framework menyediakan dua fungsi pembantu untuk bekerja dengan `error` interface secara aman.

-   **`IsAppError(err error) bool`**: Mengembalikan `true` jika `err` adalah sebuah `*AppError`.
-   **`AsAppError(err error) (*AppError, bool)`**: Melakukan konversi `err` ke `*AppError`.

```go
// Misalkan sebuah fungsi mengembalikan error interface
err := someFunction()

if err != nil {
    // Periksa apakah error ini adalah AppError
    if appErr, ok := dim.AsAppError(err); ok {
        // Ya, ini adalah AppError, kita bisa gunakan field-nya
        log.Printf("AppError terjadi. Status: %d, Pesan: %s", appErr.StatusCode, appErr.Message)
        dim.JsonAppError(w, appErr) // Kirim response yang sesuai
        return
    }

    // Bukan AppError, mungkin error lain (I/O, dll.)
    // Tangani sebagai error server internal
    log.Printf("Terjadi error tak terduga: %v", err)
    dim.InternalServerError(w, "Terjadi kesalahan internal")
}
```

---

## HTTP Status Codes

### Success Codes (2xx)

| Code | Name | Usage |
|------|------|-------|
| 200 | OK | Successful GET, PUT, PATCH |
| 201 | Created | Successful POST |
| 204 | No Content | Successful DELETE (no body) |

### Client Error Codes (4xx)

| Code | Name | Usage | Example |
|------|------|-------|---------|
| 400 | Bad Request | Invalid request format/validation | `{"message": "Validation failed", "errors": {...}}` |
| 401 | Unauthorized | Auth required or invalid | `{"message": "Unauthorized"}` |
| 403 | Forbidden | Authenticated but no permission | `{"message": "Forbidden"}` |
| 404 | Not Found | Resource not found | `{"message": "User not found"}` |
| 409 | Conflict | Duplicate or state conflict | `{"message": "Email already exists"}` |
| 429 | Too Many Requests | Rate limit exceeded | `{"message": "Rate limit exceeded"}` |

### Server Error Codes (5xx)

| Code | Name | Usage |
|------|------|-------|
| 500 | Internal Server Error | Unexpected server error |
| 503 | Service Unavailable | Database unavailable, etc |

---

## Error Response Format

### Simple Error

```json
{
  "message": "Error description"
}
```

**Handler code**:
```go
dim.JsonError(w, http.StatusNotFound, "User tidak ditemukan", nil)
```

### Validation Error

```json
{
  "message": "Validation failed",
  "errors": {
    "email": "Invalid email format",
    "password": "Password too weak"
  }
}
```

**Handler code**:
```go
v := dim.NewValidator()
v.Required("email", email)
v.Email("email", email)
v.Required("password", password)

if !v.IsValid() {
    dim.JsonError(w, http.StatusBadRequest, "Validation failed", v.Errors())
    return
}
```

### Error dengan Additional Info

```json
{
  "message": "Rate limit exceeded",
  "errors": {
    "retry_after": "3600"
  }
}
```

**Handler code**:
```go
dim.JsonError(w, http.StatusTooManyRequests, "Rate limit exceeded", map[string]string{
    "retry_after": "3600",
})
```

---

## Validation Errors

### Single Field Error

```go
func validateEmail(email string) string {
    if email == "" {
        return "Email is required"
    }
    if !isValidEmail(email) {
        return "Invalid email format"
    }
    return ""
}

// In handler
v := dim.NewValidator()
if err := validateEmail(email); err != nil {
    v.Errors()["email"] = err
}

if !v.IsValid() {
    dim.JsonError(w, 400, "Validation failed", v.Errors())
}
```

### Multiple Field Errors

```go
func registerHandler(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Email    string
        Username string
        Password string
    }
    
    json.NewDecoder(r.Body).Decode(&req)
    
    // Validate
    v := dim.NewValidator()
    v.Required("email", req.Email)
    v.Email("email", req.Email)
    v.Required("username", req.Username)
    v.MinLength("username", req.Username, 3)
    v.Required("password", req.Password)
    v.MinLength("password", req.Password, 8)
    
    if !v.IsValid() {
        dim.JsonError(w, http.StatusBadRequest, "Validation failed", v.Errors())
        return
    }
    
    // Continue with registration
}
```

### Custom Validation

```go
v := dim.NewValidator()

// Custom validation function
v.Custom("age", func() bool {
    age, _ := strconv.Atoi(ageStr)
    return age >= 18
}, "Must be 18 or older")

v.Custom("password_confirm", func() bool {
    return password == passwordConfirm
}, "Passwords don't match")

if !v.IsValid() {
    dim.JsonError(w, 400, "Validation failed", v.Errors())
}
```

---

## Database Errors

### No Rows Found

```go
user, err := userStore.FindByID(ctx, userID)

if err == sql.ErrNoRows {
    dim.JsonError(w, http.StatusNotFound, "User tidak ditemukan", nil)
    return
}

if err != nil {
    logger.Error("Database error", "error", err)
    dim.JsonError(w, http.StatusInternalServerError, "Database error", nil)
    return
}

// Use user
```

### Duplicate Key

```go
import "github.com/jackc/pgconn"

func isDuplicateKeyError(err error) bool {
    var pgErr *pgconn.PgError
    if errors.As(err, &pgErr) {
        return pgErr.Code == "23505"  // PostgreSQL duplicate key code
    }
    return false
}

// In handler
err := userStore.Create(ctx, user)

if isDuplicateKeyError(err) {
    dim.JsonError(w, http.StatusConflict, "Email sudah terdaftar", nil)
    return
}

if err != nil {
    logger.Error("Create user error", "error", err)
    dim.JsonError(w, http.StatusInternalServerError, "Gagal membuat user", nil)
    return
}
```

### Foreign Key Violation

```go
func isForeignKeyError(err error) bool {
    var pgErr *pgconn.PgError
    if errors.As(err, &pgErr) {
        return pgErr.Code == "23503"  // PostgreSQL FK error
    }
    return false
}

// In handler
err := postStore.Create(ctx, post)

if isForeignKeyError(err) {
    dim.JsonError(w, http.StatusBadRequest, "User tidak ditemukan", nil)
    return
}
```

### Connection Error

```go
func isConnectionError(err error) bool {
    // Check if connection-related error
    return strings.Contains(err.Error(), "connection") ||
           strings.Contains(err.Error(), "connect")
}

// In handler
users, err := userStore.GetAll(ctx)

if isConnectionError(err) {
    dim.JsonError(w, http.StatusServiceUnavailable, "Database unavailable", nil)
    return
}
```

---

## Custom Error Types

### Define Custom Error

```go
type ValidationError struct {
    Field   string
    Message string
}

type AuthError struct {
    Code    string
    Message string
}

type NotFoundError struct {
    Resource string
    ID       interface{}
}
```

### Use Custom Error

```go
func (store *UserStore) FindByID(ctx context.Context, id int64) (*User, error) {
    // ... query ...
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, &NotFoundError{
                Resource: "User",
                ID:       id,
            }
        }
        return nil, err
    }
    return &user, nil
}

// In handler
user, err := userStore.FindByID(ctx, userID)

var notFoundErr *NotFoundError
if errors.As(err, &notFoundErr) {
    dim.JsonError(w, 404, fmt.Sprintf("%s %v not found", 
        notFoundErr.Resource, notFoundErr.ID), nil)
    return
}

if err != nil {
    logger.Error("Query error", "error", err)
    dim.JsonError(w, 500, "Internal error", nil)
    return
}
```

---

## Error Logging

### Log Error dengan Context

```go
func registerHandler(authService *AuthService) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        user, err := authService.Register(r.Context(), email, username, password)
        
        if err != nil {
            // Log dengan context
            logger.Error("Registration failed",
                "email", email,
                "error", err.Error(),
                "request_id", dim.GetRequestID(r),
            )
            
            // Return generic error ke client
            dim.JsonError(w, 500, "Registration failed", nil)
            return
        }
        
        // Log success
        logger.Info("User registered successfully",
            "user_id", user.ID,
            "email", email,
        )
    }
}
```

### Log Levels

```go
// Debug - Detailed info
logger.Debug("User data", "user", user, "timestamp", time.Now())

// Info - General info
logger.Info("User registered", "user_id", user.ID)

// Warn - Warning condition
logger.Warn("High error rate detected", "error_count", 100)

// Error - Error condition
logger.Error("Database connection failed", "error", err)
```

### Structured Logging

```go
logger.Info("Request processed",
    "method", r.Method,
    "path", r.URL.Path,
    "status", http.StatusOK,
    "duration_ms", duration.Milliseconds(),
    "user_id", user.ID,
)
```

---

## Error Recovery

### Recovery Middleware

```go
func Recovery(logger *slog.Logger) MiddlewareFunc {
    return func(next HandlerFunc) HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
            defer func() {
                if err := recover(); err != nil {
                    // Log panic dengan stack trace
                    logger.Error("Panic recovered",
                        "error", err,
                        "path", r.URL.Path,
                        "method", r.Method,
                    )
                    
                    // Return error response
                    w.Header().Set("Content-Type", "application/json")
                    w.WriteHeader(http.StatusInternalServerError)
                    json.NewEncoder(w).Encode(map[string]string{
                        "message": "Internal server error",
                    })
                }
            }()
            
            next(w, r)
        }
    }
}
```

### Panic vs Error

```go
// ❌ DON'T - Panic untuk recoverable errors
if user == nil {
    panic("User is nil")  // Recovery middleware tangkap, tapi generic
}

// ✅ DO - Return error untuk recoverable errors
if user == nil {
    dim.JsonError(w, 404, "User not found", nil)
    return
}

// ✅ OK - Panic hanya untuk unrecoverable (programming errors)
if handler == nil {
    panic("Handler must not be nil")  // Development bug
}
```

---

## Praktik Terbaik

### ✅ DO: Use Specific HTTP Status Codes

```go
// ✅ BAIK - Specific status codes
switch err {
case sql.ErrNoRows:
    dim.JsonError(w, http.StatusNotFound, "Not found", nil)
case isDuplicateKey:
    dim.JsonError(w, http.StatusConflict, "Already exists", nil)
case isValidationError:
    dim.JsonError(w, http.StatusBadRequest, "Invalid input", v.Errors())
}

// ❌ BURUK - Always 500
dim.JsonError(w, http.StatusInternalServerError, err.Error(), nil)
```

### ✅ DO: Log with Context

```go
// ✅ BAIK - Log dengan context
logger.Error("Query failed",
    "table", "users",
    "user_id", userID,
    "error", err.Error(),
    "request_id", requestID,
)

// ❌ BURUK - Log hanya error message
log.Println("Error:", err)
```

### ✅ DO: Never Expose Internal Details

```go
// ✅ BAIK - Generic error ke client
dim.JsonError(w, 500, "Database error", nil)

// ❌ BURUK - Expose internal details
dim.JsonError(w, 500, fmt.Sprintf("Failed to connect to %s:%d", host, port), nil)

// ❌ BURUK - Expose SQL query
dim.JsonError(w, 400, fmt.Sprintf("Query failed: %s", sqlQuery), nil)
```

### ✅ DO: Validate Input Early

```go
func registerHandler(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Email    string
        Username string
        Password string
    }
    
    // ✅ Parse early
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        dim.JsonError(w, http.StatusBadRequest, "Invalid JSON", nil)
        return
    }
    
    // ✅ Validate early
    v := dim.NewValidator()
    v.Required("email", req.Email)
    v.Email("email", req.Email)
    // ... more validations ...
    
    if !v.IsValid() {
        dim.JsonError(w, http.StatusBadRequest, "Validation failed", v.Errors())
        return
    }
    
    // ✅ Proceed with business logic
}
```

### ❌ DON'T: Ignore Errors

```go
// ❌ BURUK - Ignore error
json.NewDecoder(r.Body).Decode(&req)
userStore.FindByEmail(ctx, email)

// ✅ BAIK - Check error
if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
    dim.JsonError(w, 400, "Invalid JSON", nil)
    return
}

user, err := userStore.FindByEmail(ctx, email)
if err != nil {
    logger.Error("Query error", "error", err)
    dim.JsonError(w, 500, "Database error", nil)
    return
}
```

### ✅ DO: Use Error Type Assertions

```go
// ✅ BAIK - Type assertion untuk specific error
var pgErr *pgconn.PgError
if errors.As(err, &pgErr) {
    if pgErr.Code == "23505" {
        dim.JsonError(w, 409, "Email already exists", nil)
        return
    }
}
```

### ✅ DO: Return Consistent Error Format

```go
// ✅ BAIK - Consistent format
type ErrorResponse struct {
    Message string            `json:"message"`
    Errors  map[string]string `json:"errors,omitempty"`
}

dim.JsonError(w, 400, "Validation failed", map[string]string{
    "email": "Invalid email",
    "password": "Too short",
})

// ❌ BURUK - Inconsistent format
w.WriteHeader(400)
json.NewEncoder(w).Encode("Error")
json.NewEncoder(w).Encode(map[string]interface{}{"err": err})
```

---

## Summary

Error handling di dim:
- **Structured** - Setiap error punya struktur tertentu
- **Status codes** - HTTP status codes yang sesuai
- **Field-level** - Validation errors per-field
- **No exposure** - Jangan expose internal details
- **Well-logged** - Semua errors tercatat untuk debugging

Lihat [Validasi](09-validation.md) untuk validation error detail.

---

**Lihat Juga**:
- [Validasi](09-validation.md) - Input validation
- [Middleware](04-middleware.md) - Error recovery middleware
- [Response Helpers](11-response-helpers.md) - Response formatting
