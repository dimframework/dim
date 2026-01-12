# Routing di Framework dim

Pelajari cara mendaftar dan mengelola routes HTTP di dim.

## Daftar Isi

- [Konsep Dasar](#konsep-dasar)
- [Route Registration](#route-registration)
- [HTTP Methods](#http-methods)
- [Path Parameters](#path-parameters)
- [Query Parameters](#query-parameters)
- [Route Grouping](#route-grouping)
- [Middleware Per-Route](#middleware-per-route)
- [NotFound Handler](#notfound-handler)
- [Advanced Routing](#advanced-routing)
- [Praktik Terbaik](#best-practices)

---

## Konsep Dasar

Route di dim adalah mapping antara **HTTP method + path** dengan **handler function**.

**Struktur Route**:
```
Method + Path → Handler
GET /users    → func getUsers(w, r)
POST /users   → func createUser(w, r)
```

**Router Tree**:
Framework menggunakan tree-based router untuk performa O(log n):

```
GET requests
├─ /              → homeHandler
├─ /users
│  ├─ (base)      → listUsersHandler
│  └─ /:id
│     ├─ (base)   → getUserHandler
│     └─ /posts
│        └─ /:postId → getUserPostHandler
└─ /health       → healthHandler

POST requests
├─ /users        → createUserHandler
└─ /auth/login   → loginHandler
```

**Static vs Dynamic Routes**:

```go
// Static route (exact match)
router.Get("/users", getUsersHandler)

// Dynamic route (parameter match)
router.Get("/users/:id", getUserHandler)

// Multi-parameter
router.Get("/users/:id/posts/:postId", getUserPostHandler)
```

---

## Route Registration

### HTTP Methods

Daftar route menggunakan method matcher:

```go
router := dim.NewRouter()

// GET request
router.Get("/users", listUsersHandler)

// POST request
router.Post("/users", createUserHandler)

// PUT request
router.Put("/users/:id", updateUserHandler)

// DELETE request
router.Delete("/users/:id", deleteUserHandler)

// PATCH request
router.Patch("/users/:id", patchUserHandler)

// OPTIONS request
router.Options("/users", optionsHandler)
```

### Handler Function Signature

Semua handler harus memiliki signature:

```go
func handlerName(w http.ResponseWriter, r *http.Request)
```

**Contoh Handler**:

```go
func listUsersHandler(w http.ResponseWriter, r *http.Request) {
    // Business logic
    users := []map[string]interface{}{
        {"id": 1, "name": "John"},
        {"id": 2, "name": "Jane"},
    }
    
    // Return response
    dim.Json(w, http.StatusOK, users)
}

func createUserHandler(w http.ResponseWriter, r *http.Request) {
    // Parse request body
    var req struct {
        Name  string `json:"name"`
        Email string `json:"email"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        dim.JsonError(w, http.StatusBadRequest, "Invalid JSON", nil)
        return
    }
    
    // Validate
    v := dim.NewValidator()
    v.Required("name", req.Name)
    v.Email("email", req.Email)
    
    if !v.IsValid() {
        dim.JsonError(w, http.StatusBadRequest, "Validation failed", v.Errors())
        return
    }
    
    // Create user logic...
    
    // Return response
    dim.Json(w, http.StatusCreated, map[string]interface{}{
        "id":    1,
        "name":  req.Name,
        "email": req.Email,
    })
}
```

---

## HTTP Methods

### GET - Retrieve Resource

```go
router.Get("/users", listUsersHandler)
router.Get("/users/:id", getUserHandler)
```

**Karakteristik**:
- Tidak mengubah server state
- Tidak memiliki request body
- Cacheable
- Idempotent

### POST - Create Resource

```go
router.Post("/users", createUserHandler)
router.Post("/auth/login", loginHandler)
```

**Karakteristik**:
- Create resource baru
- Memiliki request body
- Non-idempotent

### PUT - Replace Resource

```go
router.Put("/users/:id", updateUserHandler)
```

**Karakteristik**:
- Replace entire resource
- Memiliki request body
- Idempotent

### DELETE - Remove Resource

```go
router.Delete("/users/:id", deleteUserHandler)
```

**Karakteristik**:
- Remove resource
- Tidak memiliki body
- Idempotent

### PATCH - Partial Update

```go
router.Patch("/users/:id", patchUserHandler)
```

**Karakteristik**:
- Update sebagian resource
- Memiliki request body
- May not be idempotent

---

## Path Parameters

Parameter di-embed di dalam path URL (misalnya, `/users/123`). Framework `dim` menangkap nilai-nilai ini dan menyediakannya melalui *request context*.

Gunakan fungsi `dim.GetParam` untuk mengambil nilai parameter berdasarkan namanya.

### Single Parameter

```go
router.Get("/users/:id", func(w http.ResponseWriter, r *http.Request) {
    // Router menangkap '123' dari path dan menyimpannya di context.
    // dim.GetParam membaca nilai 'id' dari context tersebut.
    id := dim.GetParam(r, "id")
    
    dim.Json(w, http.StatusOK, map[string]string{
        "id": id,
    })
})

// Request: GET /users/123
// Response: {"id": "123"}
```

### Multiple Parameters

```go
router.Get("/users/:userId/posts/:postId", func(w http.ResponseWriter, r *http.Request) {
    userId := dim.GetParam(r, "userId")
    postId := dim.GetParam(r, "postId")
    
    dim.Json(w, http.StatusOK, map[string]string{
        "userId": userId,
        "postId": postId,
    })
})

// Request: GET /users/123/posts/456
// Response: {"userId": "123", "postId": "456"}
```

### Mengambil Semua Parameter

Gunakan `dim.GetParams(r)` untuk mendapatkan sebuah `map[string]string` dari semua parameter yang ditangkap.

```go
router.Get("/users/:id/posts/:postId", func(w http.ResponseWriter, r *http.Request) {
    // Get all parameters as a map
    params := dim.GetParams(r)
    
    // params akan menjadi map[string]string{"id": "123", "postId": "456"}
    dim.Json(w, http.StatusOK, params)
})
```

### Konversi Tipe Data

Parameter path selalu diekstrak sebagai `string`. Anda perlu mengonversinya secara manual ke tipe data lain seperti `int` atau `uuid.UUID`.

```go
router.Get("/users/:id", func(w http.ResponseWriter, r *http.Request) {
    idStr := dim.GetParam(r, "id")
    
    // Konversi ke int64
    id, err := strconv.ParseInt(idStr, 10, 64)
    if err != nil {
        dim.JsonError(w, http.StatusBadRequest, "ID tidak valid", nil)
        return
    }
    
    // Gunakan id sebagai int64
    dim.Json(w, http.StatusOK, map[string]int64{
        "id": id,
    })
})
```

---

## Query Parameters

Query parameters di URL: `/users?page=1&limit=10`

```go
router.Get("/users", func(w http.ResponseWriter, r *http.Request) {
    // Parse query parameters
    query := r.URL.Query()
    
    // Get single value
    page := query.Get("page")          // "1" atau ""
    
    // Get all values for key (jika multiple)
    tags := query["tag"]               // []string
    
    // Check if exists
    if _, exists := query["sort"]; exists {
        // sort parameter exists
    }
    
    dim.Json(w, http.StatusOK, map[string]interface{}{
        "page": page,
        "tags": tags,
    })
})

// Request: GET /users?page=1&limit=10&tag=admin&tag=user
// Response: {
//   "page": "1",
//   "tags": ["admin", "user"]
// }
```

**Helper untuk Query Parameters**:

Anda dapat membuat fungsi pembantu untuk mengambil dan mengonversi parameter *query* dengan nilai *default*.

```go
func GetQueryInt(r *http.Request, key string, defaultVal int) int {
    str := r.URL.Query().Get(key)
    if str == "" {
        return defaultVal
    }
    
    val, err := strconv.Atoi(str)
    if err != nil {
        return defaultVal
    }
    
    return val
}

// Usage
router.Get("/users", func(w http.ResponseWriter, r *http.Request) {
    page := GetQueryInt(r, "page", 1)
    limit := GetQueryInt(r, "limit", 10)
    
    dim.Json(w, http.StatusOK, map[string]interface{}{
        "page":  page,
        "limit": limit,
    })
})
```

---

## Route Grouping

Kelompok routes dengan prefix dan shared middleware:

### Basic Grouping

```go
router := dim.NewRouter()

// Buat group dengan prefix
api := router.Group("/api")

// Register routes di group
api.Get("/users", listUsersHandler)
api.Post("/users", createUserHandler)

// Hasilnya:
// GET /api/users
// POST /api/users
```

### Nested Grouping

```go
router := dim.NewRouter()

// Group level 1
api := router.Group("/api")

// Group level 2 (nested)
v1 := api.Group("/v1")
users := v1.Group("/users")

// Register routes
users.Get("", listUsersHandler)      // GET /api/v1/users
users.Post("", createUserHandler)    // POST /api/v1/users

// Sub-resource routes
users.Get("/:id", getUserHandler)    // GET /api/v1/users/:id
users.Get("/:id/posts", getPostsHandler) // GET /api/v1/users/:id/posts
```

### Group dengan Middleware

*Middleware* yang diterapkan pada grup akan dijalankan untuk semua *route* di dalam grup tersebut.

```go
// Diinisialisasi di tempat lain
var jwtManager *dim.JWTManager 

router := dim.NewRouter()

// Public routes (tanpa auth)
router.Get("/health", healthHandler)
router.Post("/auth/login", loginHandler)

// Protected routes (dengan auth yang aman)
api := router.Group("/api", dim.RequireAuth(jwtManager))

api.Get("/profile", profileHandler)
api.Post("/logout", logoutHandler)

// Admin routes (auth + admin check)
admin := router.Group("/admin", 
    dim.RequireAuth(jwtManager), // Pastikan login
    requireAdminMiddleware,                  // Pastikan adalah admin
)

admin.Get("/users", listAllUsersHandler)
admin.Delete("/users/:id", deleteUserHandler)
```

---

## Middleware Per-Route

Selain *middleware* global (`router.Use(...)`), Anda dapat menerapkan *middleware* pada *route* individual atau pada grup.

### Urutan Eksekusi

Urutan eksekusi *middleware* adalah sebagai berikut:
1.  *Middleware* global (`router.Use`).
2.  *Middleware* level grup (`router.Group`).
3.  *Middleware* level *route*.
4.  *Handler* utama.

### Global Middleware

Diterapkan ke semua *route* yang terdaftar setelah pemanggilan `Use`.

```go
router := dim.NewRouter()

// Middleware ini akan dijalankan untuk SEMUA request
router.Use(dim.Recovery(logger))
router.Use(dim.LoggerMiddleware(logger))

// Daftarkan route setelah middleware
router.Get("/users", listUsersHandler)
router.Get("/health", healthHandler)
```

### Per-Route Middleware (Urutan Argumen Penting!)

Untuk menerapkan *middleware* hanya pada satu *route* spesifik, daftarkan sebagai argumen setelah *handler*.

**Tanda tangan fungsi**: `router.Get(path string, handler HandlerFunc, middleware ...MiddlewareFunc)`

```go
router := dim.NewRouter()

// PENTING: Middleware diletakkan SETELAH handler.
router.Get("/users",
    listUsersHandler, // 1. Path, 2. Handler
    dim.RateLimit(rateLimitConfig), // 3. Middleware (bisa lebih dari satu)
)

router.Post("/users",
    createUserHandler, // 1. Path, 2. Handler
    dim.RateLimit(rateLimitConfig), // 3. Middleware 1
    customValidationMiddleware,     // 4. Middleware 2
)

// Route ini tidak memiliki middleware spesifik
router.Get("/health", healthHandler)
```

### Per-Group Middleware

Ini adalah cara paling umum untuk menerapkan *middleware* seperti autentikasi ke sekelompok *route*. *Middleware* diterapkan pada semua *route* yang didefinisikan di dalam grup.

```go
// Diinisialisasi di tempat lain
var jwtManager *dim.JWTManager 

router := dim.NewRouter()

// Middleware untuk grup /api
// Contoh: Memastikan semua request ke /api harus terautentikasi
api := router.Group("/api", dim.RequireAuth(jwtManager))

// Semua route di dalam grup ini akan menjalankan RequireAuth terlebih dahulu
api.Get("/users", listUsersHandler)
api.Post("/logout", logoutHandler)

// Anda tetap bisa menambahkan middleware tambahan per-route di dalam grup
api.Post("/users",
    createUserHandler,
    dim.RateLimit(rateLimitConfig), // Middleware tambahan
)
```

---

## NotFound Handler

Handler untuk routes yang tidak cocok:

### Default NotFound

```go
router := dim.NewRouter()

// Jika route tidak found, return 404
router.Get("/users", listUsersHandler)

// Request: GET /invalid
// Response: 404 (default handler)
```

### Custom NotFound Handler

```go
router := dim.NewRouter()

router.SetNotFound(func(w http.ResponseWriter, r *http.Request) {
    dim.JsonError(w, http.StatusNotFound, 
        "Endpoint tidak ditemukan", 
        nil,
    )
})

router.Get("/users", listUsersHandler)

// Request: GET /invalid
// Response: {
//   "message": "Endpoint tidak ditemukan",
//   "errors": null
// }
```

---

## Advanced Routing

### Conflicting Routes

Router dapat handle conflicting routes dengan prioritas:

```go
router := dim.NewRouter()

// Static route prioritas lebih tinggi
router.Get("/users/me", getMyProfileHandler)

// Dynamic route
router.Get("/users/:id", getUserHandler)

// Request: GET /users/me
// Matches: /users/me (static, lebih prioritas)

// Request: GET /users/123
// Matches: /users/:id (dynamic)
```

### Method-Based Routing

Handler berbeda per method:

```go
router := dim.NewRouter()

// GET → read
router.Get("/users/:id", func(w http.ResponseWriter, r *http.Request) {
    // GET logic
})

// PUT → update
router.Put("/users/:id", func(w http.ResponseWriter, r *http.Request) {
    // PUT logic
})

// DELETE → remove
router.Delete("/users/:id", func(w http.ResponseWriter, r *http.Request) {
    // DELETE logic
})

// Same path, berbeda method, berbeda handler
```

### Pattern Matching

Router dapat match patterns dinamis:

```go
router := dim.NewRouter()

// UUID pattern
router.Get("/posts/:id", func(w http.ResponseWriter, r *http.Request) {
    id := dim.GetParam(r, "id")
    // id bisa berisi: 123e4567-e89b-12d3-a456-426614174000
})

// Username pattern
router.Get("/users/:username", func(w http.ResponseWriter, r *http.Request) {
    username := dim.GetParam(r, "username")
    // username bisa berisi: john_doe
})
```

---

## Praktik Terbaik

### 1. RESTful URL Structure

Follow REST conventions:

```go
router := dim.NewRouter()

// Resources
router.Get("/users", listUsersHandler)              // List
router.Post("/users", createUserHandler)            // Create
router.Get("/users/:id", getUserHandler)            // Read
router.Put("/users/:id", updateUserHandler)         // Update
router.Delete("/users/:id", deleteUserHandler)      // Delete

// Nested resources
router.Get("/users/:id/posts", getUserPostsHandler)
router.Post("/users/:id/posts", createUserPostHandler)
router.Get("/users/:id/posts/:postId", getUserPostHandler)
```

### 2. Use Route Grouping

Organize routes logically:

```go
// ❌ Tidak terorganisir
router.Get("/api/users", listUsersHandler)
router.Post("/api/users", createUserHandler)
router.Get("/api/posts", listPostsHandler)

// ✅ Terorganisir
api := router.Group("/api")

users := api.Group("/users")
users.Get("", listUsersHandler)
users.Post("", createUserHandler)

posts := api.Group("/posts")
posts.Get("", listPostsHandler)
```

### 3. Consistent Naming

Gunakan konvensi naming konsisten:

```go
// ✅ Baik: Deskriptif dan konsisten
router.Get("/users/:id", getUserByIDHandler)
router.Post("/users", createUserHandler)
router.Put("/users/:id", updateUserHandler)

// ❌ Buruk: Tidak konsisten
router.Get("/user/:id", getUser)
router.Post("/users", addUser)
router.Put("/users/:id", updateUserData)
```

### 4. Use Status Codes Correctly

```go
func createUserHandler(w http.ResponseWriter, r *http.Request) {
    // ... validation ...
    
    // Created (201)
    dim.Json(w, http.StatusCreated, user)
}

func listUsersHandler(w http.ResponseWriter, r *http.Request) {
    // ... fetch ...
    
    // OK (200)
    dim.Json(w, http.StatusOK, users)
}

func getUserHandler(w http.ResponseWriter, r *http.Request) {
    user, err := store.Find(id)
    
    if err != nil {
        // Not Found (404)
        dim.JsonError(w, http.StatusNotFound, "User not found", nil)
        return
    }
    
    // OK (200)
    dim.Json(w, http.StatusOK, user)
}
```

### 5. Route Organization Pattern

```go
// main.go - Simple application
func main() {
    router := setupRoutes(/* ... */)
    http.ListenAndServe(":8080", router)
}

func setupRoutes(/* deps */) *dim.Router {
    router := dim.NewRouter()
    
    // Public routes
    router.Get("/health", healthHandler)
    router.Post("/auth/login", loginHandler)
    router.Post("/auth/register", registerHandler)
    
    // API routes (protected)
    api := router.Group("/api", dim.RequireAuth())
    api.Get("/profile", profileHandler)
    api.Post("/logout", logoutHandler)
    
    // Admin routes (protected + admin)
    admin := router.Group("/admin", 
        dim.RequireAuth(),
        requireAdminMiddleware,
    )
    admin.Get("/users", listAllUsersHandler)
    
    router.SetNotFound(notFoundHandler)
    
    return router
}
```

---

## Summary

Routing di dim:
- **Simple & powerful** - Tree-based router yang cepat
- **RESTful** - Support semua HTTP methods
- **Flexible** - Dynamic parameters, grouping, middleware
- **Safe** - Static routes prioritas lebih tinggi

Sekarang pelajari [Middleware](04-middleware.md) untuk mendalami sistem middleware yang KRITIS untuk keamanan.

---

**Lihat Juga**:
- [Middleware](04-middleware.md) - Middleware system dan urutan
- [Handlers](16-handlers.md) - Handler patterns dan struktur
- [Response Helpers](11-response-helpers.md) - Format response
- [Autentikasi](05-authentication.md) - Protected routes
