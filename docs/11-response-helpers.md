# Response Helpers di Framework dim

Pelajari cara mengirim response dengan format yang konsisten dan terstruktur.

## Daftar Isi

- [Konsep Response](#konsep-response)
- [Response Format Standards](#response-format-standards)
- [Json Helper](#json-helper)
- [JsonPagination Helper](#jsonpagination-helper)
- [JsonError Helper](#jsonerror-helper)
- [Custom Headers](#custom-headers)
- [Response Status Codes](#response-status-codes)
- [Streaming Responses](#streaming-responses)
- [Praktik Terbaik](#best-practices)

---

## Konsep Response

### Response Flow

```go
func handler(w http.ResponseWriter, r *http.Request) {
    // 1. Process request
    data := processRequest(r)
    
    // 2. Set headers (optional)
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("X-Custom-Header", "value")
    
    // 3. Write status code
    w.WriteHeader(http.StatusOK)
    
    // 4. Write body
    json.NewEncoder(w).Encode(data)
}
```

### Response Helper Principles

Framework dim menyediakan helpers yang:
- **Consistent** - Format response yang sama untuk semua endpoints
- **Simple** - Satu function call untuk response
- **Type-safe** - No magic, clear parameters
- **Flexible** - Support berbagai response formats

---

## Response Format Standards

### Single Object

```json
{
  "id": 1,
  "name": "John",
  "email": "john@example.com"
}
```

**Handler**:
```go
dim.Json(w, http.StatusOK, user)
```

### Collection/Array

```json
[
  {"id": 1, "name": "John"},
  {"id": 2, "name": "Jane"}
]
```

**Handler**:
```go
dim.Json(w, http.StatusOK, users)
```

### Wrapped Object dengan Meta

```json
{
  "data": {"id": 1, "name": "John"},
  "meta": {"created_at": "2024-01-10T00:00:00Z"}
}
```

**Handler**:
```go
dim.Json(w, http.StatusOK, map[string]interface{}{
    "data": user,
    "meta": map[string]interface{}{
        "created_at": user.CreatedAt,
    },
})
```

### Pagination Response

```json
{
  "data": [
    {"id": 1, "name": "John"},
    {"id": 2, "name": "Jane"}
  ],
  "meta": {
    "page": 1,
    "per_page": 10,
    "total": 100,
    "total_pages": 10
  }
}
```

**Handler**:
```go
meta := dim.PaginationMeta{
    Page:       1,
    PerPage:    10,
    Total:      100,
    TotalPages: 10,
}
dim.JsonPagination(w, http.StatusOK, users, meta)
```

### Error Response

```json
{
  "message": "Validation failed",
  "errors": {
    "email": "Invalid email format",
    "password": "Too weak"
  }
}
```

**Handler**:
```go
dim.JsonError(w, http.StatusBadRequest, "Validation failed", 
    map[string]string{
        "email": "Invalid email format",
        "password": "Too weak",
    })
```

---

## Json Helper

### Simple Response

Mengirim single object atau array:

```go
// Single object
func getUserHandler(w http.ResponseWriter, r *http.Request) {
    user := &User{ID: 1, Name: "John"}
    dim.Json(w, http.StatusOK, user)
    
    // Response:
    // {"id": 1, "name": "John"}
}

// Array of objects
func listUsersHandler(w http.ResponseWriter, r *http.Request) {
    users := []*User{
        {ID: 1, Name: "John"},
        {ID: 2, Name: "Jane"},
    }
    dim.Json(w, http.StatusOK, users)
    
    // Response:
    // [{"id": 1, "name": "John"}, {"id": 2, "name": "Jane"}]
}
```

### Response dengan Map

```go
func loginHandler(w http.ResponseWriter, r *http.Request) {
    accessToken := "eyJ..."
    refreshToken := "eyJ..."
    
    dim.Json(w, http.StatusOK, map[string]interface{}{
        "access_token":  accessToken,
        "refresh_token": refreshToken,
        "expires_in":    900,
        "token_type":    "Bearer",
    })
    
    // Response:
    // {
    //   "access_token": "eyJ...",
    //   "refresh_token": "eyJ...",
    //   "expires_in": 900,
    //   "token_type": "Bearer"
    // }
}
```

### Json Signature

```go
func Json(w http.ResponseWriter, status int, data interface{})
```

**Parameters**:
- `w http.ResponseWriter` - HTTP response writer
- `status int` - HTTP status code (200, 201, 400, etc)
- `data interface{}` - Any data to encode as JSON

---

## JsonPagination Helper

### Pagination Response

```go
func listUsersHandler(w http.ResponseWriter, r *http.Request) {
    // Parse page and limit
    page := getQueryInt(r, "page", 1)
    limit := getQueryInt(r, "limit", 10)
    
    // Fetch data
    users, total, err := userStore.GetPaginated(r.Context(), page, limit)
    if err != nil {
        dim.JsonError(w, 500, "Database error", nil)
        return
    }
    
    // Calculate meta
    totalPages := (total + limit - 1) / limit  // Ceiling division
    meta := dim.PaginationMeta{
        Page:       page,
        PerPage:    limit,
        Total:      total,
        TotalPages: totalPages,
    }
    
    // Send response
    dim.JsonPagination(w, http.StatusOK, users, meta)
    
    // Response:
    // {
    //   "data": [...],
    //   "meta": {
    //     "page": 1,
    //     "per_page": 10,
    //     "total": 100,
    //     "total_pages": 10
    //   }
    // }
}
```

### PaginationMeta Struct

```go
type PaginationMeta struct {
    Page       int `json:"page"`
    PerPage    int `json:"per_page"`
    Total      int `json:"total"`
    TotalPages int `json:"total_pages"`
}
```

### JsonPagination Signature

```go
func JsonPagination(w http.ResponseWriter, status int, data interface{}, meta PaginationMeta)
```

### Helper Function

```go
func calculateTotalPages(total, perPage int) int {
    if total == 0 {
        return 1
    }
    return (total + perPage - 1) / perPage  // Ceiling division
}

// Usage
totalPages := calculateTotalPages(100, 10)  // 10
totalPages := calculateTotalPages(105, 10)  // 11
totalPages := calculateTotalPages(1, 10)    // 1
```

---

## JsonError Helper

### Simple Error

```go
func handler(w http.ResponseWriter, r *http.Request) {
    // Not found
    dim.JsonError(w, http.StatusNotFound, "User not found", nil)
    
    // Response:
    // {"message": "User not found"}
}
```

### Error dengan Field Details

```go
func registerHandler(w http.ResponseWriter, r *http.Request) {
    // Validation error
    dim.JsonError(w, http.StatusBadRequest, "Validation failed",
        map[string]string{
            "email": "Invalid email format",
            "password": "Too short",
        })
    
    // Response:
    // {
    //   "message": "Validation failed",
    //   "errors": {
    //     "email": "Invalid email format",
    //     "password": "Too short"
    //   }
    // }
}
```

### JsonError Signature

```go
func JsonError(w http.ResponseWriter, status int, message string, errors map[string]string)
```

**Parameters**:
- `w http.ResponseWriter` - HTTP response writer
- `status int` - HTTP status code
- `message string` - Error message untuk user
- `errors map[string]string` - Optional field-level errors (nil jika tidak ada)

---

## Pembantu Tambahan

Selain tiga fungsi utama, `dim` menyediakan banyak pembantu untuk membuat kode *handler* Anda lebih bersih dan ekspresif.

### Pembantu Sukses

-   **`OK(w, data)`**: Mengirim response 200 OK. Wrapper untuk `Json(w, 200, data)`.
-   **`Created(w, data)`**: Mengirim response 201 Created. Wrapper untuk `Json(w, 201, data)`.
-   **`NoContent(w)`**: Mengirim response 204 No Content.

```go
// GET /users/1 -> Mengembalikan user
func getUser(w http.ResponseWriter, r *http.Request) {
    user, _ := userStore.FindByID(1)
    dim.OK(w, user) // HTTP 200
}

// POST /users -> Membuat user
func createUser(w http.ResponseWriter, r *http.Request) {
    newUser, _ := userStore.Create(...)
    dim.Created(w, newUser) // HTTP 201
}

// DELETE /users/1
func deleteUser(w http.ResponseWriter, r *http.Request) {
    userStore.Delete(1)
    dim.NoContent(w) // HTTP 204
}
```

### Pembantu Error

-   **`BadRequest(w, message, errors)`**: Mengirim response 400 Bad Request.
-   **`Unauthorized(w, message)`**: Mengirim response 401 Unauthorized.
-   **`Forbidden(w, message)`**: Mengirim response 403 Forbidden.
-   **`NotFound(w, message)`**: Mengirim response 404 Not Found.
-   **`Conflict(w, message, errors)`**: Mengirim response 409 Conflict.
-   **`InternalServerError(w, message)`**: Mengirim response 500 Internal Server Error.
-   **`JsonAppError(w, appErr)`**: Mengurai `*AppError` dan mengirim `JsonError` yang sesuai.

```go
// Validasi gagal
func createUser(w http.ResponseWriter, r *http.Request) {
    v := dim.NewValidator()
    v.Required("email", req.Email)
    if !v.IsValid() {
        dim.BadRequest(w, "Validasi gagal", v.ErrorMap()) // HTTP 400
        return
    }
}

// Resource tidak ditemukan
func getUser(w http.ResponseWriter, r *http.Request) {
    user, err := userStore.FindByID(1)
    if err != nil {
        dim.NotFound(w, "User tidak ditemukan") // HTTP 404
        return
    }
}
```

### Utilitas Lainnya

-   **`SetStatus(w, status)`**: Hanya mengatur status HTTP.
-   **`SetHeader(w, key, value)`**: Mengatur satu header.
-   **`SetHeaders(w, headers)`**: Mengatur beberapa header dari sebuah map.
-   **`SetCookie(w, cookie)`**: Melampirkan cookie ke response.

```go
func customHandler(w http.ResponseWriter, r *http.Request) {
    dim.SetHeader(w, "X-Request-ID", "xyz-123")
    dim.SetCookie(w, &http.Cookie{Name: "my-cookie", Value: "val"})
    dim.SetStatus(w, http.StatusOK)
    w.Write([]byte("Payload kustom"))
}
```

---

## Custom Headers

### Add Response Headers

```go
func handler(w http.ResponseWriter, r *http.Request) {
    // Add headers sebelum WriteHeader
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("X-Request-ID", requestID)
    w.Header().Set("X-RateLimit-Remaining", "99")
    
    // Then send response
    dim.Json(w, http.StatusOK, data)
}
```

### Common Headers

```go
// Content type (automatic dengan dim.Json)
w.Header().Set("Content-Type", "application/json")

// Request ID (untuk tracing)
w.Header().Set("X-Request-ID", requestID)

// Rate limit info
w.Header().Set("X-RateLimit-Limit", "100")
w.Header().Set("X-RateLimit-Remaining", "99")
w.Header().Set("X-RateLimit-Reset", "1704868245")

// Caching
w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

// CORS (handled by middleware)
w.Header().Set("Access-Control-Allow-Origin", "*")

// Security
w.Header().Set("X-Content-Type-Options", "nosniff")
w.Header().Set("X-Frame-Options", "DENY")
```

### Header Helper

```go
func addResponseHeaders(w http.ResponseWriter, headers map[string]string) {
    for key, value := range headers {
        w.Header().Set(key, value)
    }
}

// Usage
addResponseHeaders(w, map[string]string{
    "X-Request-ID": requestID,
    "X-Total-Count": strconv.Itoa(total),
})

dim.Json(w, http.StatusOK, data)
```

---

## Response Status Codes

### Success Codes (2xx)

| Code | Name | Usage |
|------|------|-------|
| 200 | OK | GET, PUT, PATCH successful |
| 201 | Created | POST successful |
| 204 | No Content | DELETE successful (no body) |

**Examples**:
```go
// GET - return data
dim.Json(w, http.StatusOK, user)

// POST - return created resource
dim.Json(w, http.StatusCreated, newUser)

// DELETE - no content
w.WriteHeader(http.StatusNoContent)
```

### Client Error Codes (4xx)

| Code | Name | Usage |
|------|------|-------|
| 400 | Bad Request | Validation failed |
| 401 | Unauthorized | Auth required/invalid |
| 403 | Forbidden | Authenticated but no permission |
| 404 | Not Found | Resource doesn't exist |
| 409 | Conflict | Duplicate/state conflict |
| 429 | Too Many Requests | Rate limit |

**Examples**:
```go
// Validation error
dim.JsonError(w, http.StatusBadRequest, "Invalid input", errors)

// Auth required
dim.JsonError(w, http.StatusUnauthorized, "Login required", nil)

// No permission
dim.JsonError(w, http.StatusForbidden, "Access denied", nil)

// Not found
dim.JsonError(w, http.StatusNotFound, "User not found", nil)

// Duplicate
dim.JsonError(w, http.StatusConflict, "Email already exists", nil)
```

### Server Error Codes (5xx)

| Code | Name | Usage |
|------|------|-------|
| 500 | Internal Server Error | Unexpected error |
| 503 | Service Unavailable | DB down, etc |

**Examples**:
```go
// Unexpected error
dim.JsonError(w, http.StatusInternalServerError, "Server error", nil)

// Database unavailable
dim.JsonError(w, http.StatusServiceUnavailable, "Database unavailable", nil)
```

---

## Streaming Responses

### Streaming Large Data

```go
func downloadUsersHandler(w http.ResponseWriter, r *http.Request) {
    // Set headers
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Content-Disposition", "attachment; filename=users.json")
    
    // Write array opening
    w.Write([]byte("["))
    
    // Stream data
    encoder := json.NewEncoder(w)
    first := true
    
    rows, _ := db.Query(r.Context(), "SELECT * FROM users")
    for rows.Next() {
        var user User
        rows.Scan(&user.ID, &user.Email, &user.Name)
        
        if !first {
            w.Write([]byte(","))
        }
        
        encoder.Encode(user)
        first = false
    }
    
    // Write array closing
    w.Write([]byte("]"))
}
```

### Streaming with Context Cancellation

```go
func streamDataHandler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    
    w.Header().Set("Content-Type", "application/json")
    w.Write([]byte("["))
    
    encoder := json.NewEncoder(w)
    first := true
    
    for i := 0; i < 10000; i++ {
        // Check if client still connected
        select {
        case <-ctx.Done():
            // Client disconnected
            return
        default:
        }
        
        if !first {
            w.Write([]byte(","))
        }
        
        data := map[string]int{"id": i}
        encoder.Encode(data)
        first = false
        
        // Flush to client
        if f, ok := w.(http.Flusher); ok {
            f.Flush()
        }
    }
    
    w.Write([]byte("]"))
}
```

---

## Praktik Terbaik

### ✅ DO: Use Response Helpers

```go
// ✅ BAIK - Consistent format
dim.Json(w, http.StatusOK, user)
dim.JsonPagination(w, 200, users, meta)
dim.JsonError(w, 400, "Invalid", errors)

// ❌ BURUK - Inconsistent format
w.WriteHeader(200)
json.NewEncoder(w).Encode(user)

json.NewEncoder(w).Encode(map[string]interface{}{
    "data": users,
    "page": 1,
})

w.WriteHeader(400)
json.NewEncoder(w).Encode(map[string]string{"error": "Invalid"})
```

### ✅ DO: Use Correct Status Codes

```go
// ✅ BAIK
dim.Json(w, http.StatusCreated, newUser)       // POST
dim.Json(w, http.StatusOK, user)               // GET/PUT
w.WriteHeader(http.StatusNoContent)            // DELETE
dim.JsonError(w, http.StatusNotFound, ...)     // Not found

// ❌ BURUK
dim.Json(w, http.StatusOK, newUser)            // Should be 201
dim.Json(w, http.StatusOK, user)               // For DELETE?
dim.JsonError(w, http.StatusOK, "Error", ...)  // 200 for error?
```

### ✅ DO: Set Content-Type Header

```go
// ✅ BAIK - dim.Json automatically sets it
dim.Json(w, http.StatusOK, data)

// Manual (if not using helper)
w.Header().Set("Content-Type", "application/json; charset=utf-8")
json.NewEncoder(w).Encode(data)

// ❌ BURUK - No content-type
json.NewEncoder(w).Encode(data)
```

### ✅ DO: Handle Empty Results

```go
// ✅ BAIK - Return empty array, not null
users := []*User{}  // Empty slice
dim.Json(w, http.StatusOK, users)  // Returns []

// With pagination
meta := dim.PaginationMeta{
    Page: 1, PerPage: 10, Total: 0, TotalPages: 0,
}
dim.JsonPagination(w, http.StatusOK, users, meta)

// ❌ BURUK - Return nil
var users []*User  // nil slice
dim.Json(w, http.StatusOK, users)  // Returns null
```

### ✅ DO: Use Pagination for Large Results

```go
// ✅ BAIK - Paginate large results
page := getQueryInt(r, "page", 1)
limit := getQueryInt(r, "limit", 10)
users, total, _ := userStore.GetPaginated(ctx, page, limit)

meta := dim.PaginationMeta{
    Page: page, PerPage: limit,
    Total: total,
    TotalPages: (total + limit - 1) / limit,
}
dim.JsonPagination(w, http.StatusOK, users, meta)

// ❌ BURUK - Return all results
users, _ := userStore.GetAll(ctx)  // Could be millions
dim.Json(w, http.StatusOK, users)
```

### ✅ DO: Document Response Format

```go
// Handler documentation
// GetUsers godoc
// @Summary List all users
// @Description Get paginated list of users
// @Produce json
// @Param page query int false "Page number"
// @Param limit query int false "Results per page"
// @Success 200 {object} PaginationResponse{data=[]User}
// @Failure 400 {object} ErrorResponse
// @Router /users [get]
func listUsersHandler(w http.ResponseWriter, r *http.Request) {
    // ...
}
```

---

## Summary

Response helpers di dim:
- **Consistent** - Standard response formats
- **Simple** - One-liner responses
- **Status codes** - Correct HTTP status
- **Pagination** - Built-in pagination support
- **Errors** - Structured error responses

Lihat [Error Handling](08-error-handling.md) untuk error response detail.

---

**Lihat Juga**:
- [Error Handling](08-error-handling.md) - Error response formatting
- [Routing](03-routing.md) - HTTP methods dan status codes
- [Validasi](09-validation.md) - Validation error responses
