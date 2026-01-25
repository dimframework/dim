# Common Issues & Solutions untuk Framework dim

Panduan troubleshooting untuk masalah umum yang mungkin Anda hadapi saat menggunakan framework dim.

## Daftar Isi

- [Application Startup Errors](#application-startup-errors)
- [Database Connection Issues](#database-connection-issues)
- [Authentication & Authorization Problems](#authentication--authorization-problems)
- [API Response Issues](#api-response-issues)
- [Performance Problems](#performance-problems)
- [Middleware & Routing Issues](#middleware--routing-issues)
- [Deployment Issues](#deployment-issues)
- [Security Concerns](#security-concerns)
- [Praktik Terbaik](#praktik-terbaik)

---

## Application Startup Errors

### Error: "failed to load config"

**Penyebab:**
- Environment variables tidak di-set
- File `.env` tidak ditemukan
- Format environment variable salah

**Solusi:**

1. Periksa apakah `.env` file ada di working directory:
```bash
ls -la .env
```

2. Verify format environment variable (tidak ada space di sekitar `=`):
```bash
# ❌ Salah
JWT_SECRET = my-secret

# ✅ Benar
JWT_SECRET=my-secret
```

3. Untuk development, pastikan file `.env` di-load dengan benar:
```go
import "github.com/joho/godotenv"

func main() {
    // Load .env file
    godotenv.Load()
    
    cfg, err := dim.LoadConfig()
    if err != nil {
        log.Fatal("Config error:", err)
    }
}
```

4. Untuk production, set environment variables langsung:
```bash
# Via systemd service
[Service]
Environment="JWT_SECRET=your-secret"
Environment="DB_WRITE_HOST=db.example.com"
```

### Error: "listen tcp :8080: bind: address already in use"

**Penyebab:**
- Port 8080 sudah digunakan oleh aplikasi lain
- Aplikasi sebelumnya tidak di-terminate dengan benar

**Solusi:**

1. Cari process yang menggunakan port 8080:
```bash
lsof -i :8080
# atau di macOS
netstat -an | grep 8080
```

2. Terminate process tersebut:
```bash
kill -9 <PID>
```

3. Gunakan port yang berbeda:
```bash
SERVER_PORT=8081 go run main.go
```

4. Pastikan graceful shutdown di implemented:
```go
server := &http.Server{
    Addr:    ":" + cfg.Server.Port,
    Handler: router,
}

// Handle graceful shutdown
go func() {
    sigint := make(chan os.Signal, 1)
    signal.Notify(sigint, os.Interrupt)
    <-sigint
    
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    if err := server.Shutdown(ctx); err != nil {
        log.Fatal("Shutdown error:", err)
    }
}()

if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
    log.Fatal("Server error:", err)
}
```

### Error: "panic: runtime error: index out of range"

**Penyebab:**
- Middleware chain tidak benar
- Handler tidak mengecek parameter
- Context value tidak di-set sebelum di-access

**Solusi:**

1. Selalu check apakah value ada di context:
```go
// ❌ Salah - bisa panic jika user tidak ada
user := dim.GetUser(r)
fmt.Println(user.Email)

// ✅ Benar
user, ok := dim.GetUser(r)
if !ok {
    dim.JsonError(w, http.StatusUnauthorized, "Unauthorized", nil)
    return
}
fmt.Println(user.Email)
```

2. Gunakan middleware yang benar sesuai urutan:
```go
// Recovery middleware HARUS di-awal untuk catch panic
r.Use(dim.Recovery(logger))
r.Use(dim.Logger(logger))
r.Use(dim.CORS(corsConfig))
```

3. Debug dengan menambahkan logging:
```go
logger.Debug("handler called", "params", dim.GetParams(r))
```

---

## Database Connection Issues

### Error: "failed to connect to database"

**Penyebab:**
- Database server tidak berjalan
- Connection credentials salah
- Network connectivity issue
- Database tidak exist

**Solusi:**

1. Verify database server berjalan:
```bash
# PostgreSQL
psql -h localhost -U postgres -d postgres -c "SELECT version();"
```

2. Check connection credentials di `.env`:
```bash
DB_WRITE_HOST=localhost
DB_PORT=5432
DB_NAME=dim_development
DB_USER=postgres
DB_PASSWORD=your_password
```

3. Test connection manual:
```bash
psql -h localhost -p 5432 -U postgres -d dim_development
```

4. Verify database exist:
```bash
psql -U postgres -l | grep dim_development
```

5. Jika database belum ada, create:
```bash
createdb -U postgres dim_development
```

6. Jalankan migrations:
```bash
# Di dalam aplikasi atau setup script
if err := dim.RunMigrations(db, GetMigrations()); err != nil {
    log.Fatal("Migration failed:", err)
}
```

### Error: "connection pool exhausted"

**Penyebab:**
- Terlalu banyak concurrent connections
- Connections tidak di-close dengan benar
- Connection leak di code

**Solusi:**

1. Increase connection pool size:
```bash
DB_MAX_CONNS=50
```

2. Audit database operations untuk connection leak:
```go
// ❌ Salah - connection tidak di-close
func (s *UserStore) GetUser(ctx context.Context, id int64) (*User, error) {
    row := s.db.QueryRow(ctx, "SELECT id, email FROM users WHERE id = $1", id)
    // row tidak di-close
    var user User
    return &user, nil
}

// ✅ Benar
func (s *UserStore) GetUser(ctx context.Context, id int64) (*User, error) {
    rows, err := s.db.Query(ctx, "SELECT id, email FROM users WHERE id = $1", id)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var user User
    if rows.Next() {
        if err := rows.Scan(&user.ID, &user.Email); err != nil {
            return nil, err
        }
    }
    return &user, nil
}
```

3. Monitor connection pool:
```go
// Setup monitoring di application
ticker := time.NewTicker(10 * time.Second)
go func() {
    for range ticker.C {
        logger.Debug("DB connections active", "count", dbStats.NumConns)
    }
}()
```

### Error: "read replicas returning stale data"

**Penyebab:**
- Read/write split timing issue
- Replication lag
- Read query sebelum write complete

**Solusi:**

1. Untuk data consistency-critical operations, gunakan write connection:
```go
// Force write connection untuk read setelah write
func (s *UserStore) UpdateUserAndRead(ctx context.Context, userID int64, email string) (*User, error) {
    // Write operation
    if err := s.db.Exec(ctx, "UPDATE users SET email = $1 WHERE id = $2", email, userID); err != nil {
        return nil, err
    }
    
    // Add small delay untuk replication
    time.Sleep(100 * time.Millisecond)
    
    // Read dari primary (jika replicas memiliki lag)
    return s.getUserFromPrimary(ctx, userID)
}
```

2. Monitor replication lag:
```bash
# Di PostgreSQL replica
SELECT EXTRACT(EPOCH FROM (NOW() - pg_last_xact_replay_timestamp())) as replication_lag_seconds;
```

3. Configure read/write consistency di application:
```go
type Database interface {
    QueryFromPrimary(ctx context.Context, query string, args ...interface{}) (Rows, error)
    QueryFromReplica(ctx context.Context, query string, args ...interface{}) (Rows, error)
}
```

---

## Authentication & Authorization Problems

### Error: "invalid token" atau "token expired"

**Penyebab:**
- Token sudah expired
- JWT_SECRET berubah
- Token malformed
- Token signature tidak valid

**Solusi:**

1. Verify JWT_SECRET consistent di semua instances:
```bash
# Semua server harus memiliki JWT_SECRET yang sama
JWT_SECRET=your-consistent-secret-across-all-instances
```

2. Check token expiry time di config:
```bash
# Default 15 menit untuk access token
JWT_ACCESS_TOKEN_EXPIRY=15m

# Jika ingin lebih lama
JWT_ACCESS_TOKEN_EXPIRY=1h
```

3. Implement token refresh di client:
```go
// Client side - detect 401 dan refresh
func (c *Client) Do(req *http.Request) (*http.Response, error) {
    resp, err := c.http.Do(req)
    
    if resp.StatusCode == http.StatusUnauthorized {
        // Try refresh token
        newAccessToken, err := c.RefreshToken()
        if err != nil {
            return nil, fmt.Errorf("session expired")
        }
        
        // Retry request dengan token baru
        req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", newAccessToken))
        return c.http.Do(req)
    }
    
    return resp, err
}
```

4. Debug token dengan decoding:
```go
import "github.com/golang-jwt/jwt/v5"

func debugToken(tokenString string) {
    token, _, err := new(jwt.Parser).ParseUnverified(tokenString, &jwt.MapClaims{})
    if err != nil {
        log.Fatal("Parse error:", err)
    }
    
    claims := token.Claims.(jwt.MapClaims)
    for key, val := range claims {
        log.Printf("%s: %v", key, val)
    }
}
```

### Error: "401 Unauthorized" di protected route

**Penyebab:**
- Authorization header tidak di-send
- Authorization header format salah
- User tidak authenticated

**Solusi:**

1. Verify Authorization header di-send dengan format yang benar:
```
Authorization: Bearer <token>
```

2. Check header format di middleware:
```go
// ✅ Correct format
authHeader := r.Header.Get("Authorization")
// "Bearer <token>"

// Parse token
parts := strings.Split(authHeader, " ")
if len(parts) != 2 || parts[0] != "Bearer" {
    dim.JsonError(w, http.StatusUnauthorized, "Invalid authorization header", nil)
    return
}

token := parts[1]
```

3. Verify middleware di-apply ke route:
```go
// ❌ Salah - middleware tidak di-apply
r.Get("/api/profile", profileHandler)

// ✅ Benar - middleware yang aman di-apply per-grup
api := r.Group("/api", dim.RequireAuth(jwtManager))
api.Get("/profile", profileHandler)
```

3. Pastikan `GetUser` digunakan setelah middleware verifikasi:
```go
func RequireAdmin() dim.MiddlewareFunc {
    return func(next http.HandlerFunc) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
            // GetUser HANYA akan bekerja jika RequireAuth sudah dijalankan sebelumnya
            user, ok := dim.GetUser(r)
            if !ok {
                dim.JsonError(w, http.StatusUnauthorized, "Unauthorized", nil)
                return
            }
            
            if user.Role != "admin" {
                dim.JsonError(w, http.StatusForbidden, "Admin access required", nil)
                return
            }
            
            next(w, r)
        }
    }
}
```

### Error: "403 Forbidden"

**Penyebab:**
- User tidak memiliki permission
- CSRF token invalid
- Rate limit exceeded

**Solusi:**

1. Implement authorization middleware:
```go
func RequireAdmin() dim.MiddlewareFunc {
    return func(next http.HandlerFunc) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
            user, ok := dim.GetUser(r)
            if !ok {
                dim.JsonError(w, http.StatusUnauthorized, "Unauthorized", nil)
                return
            }
            
            if user.Role != "admin" {
                dim.JsonError(w, http.StatusForbidden, "Admin access required", nil)
                return
            }
            
            next(w, r)
        }
    }
}

// Usage
admin := r.Group("/admin", RequireAdmin())
admin.Post("/users", deleteUserHandler)
```

2. Verify CSRF token di-send:
```
X-CSRF-Token: <token>
```

3. Check rate limit status:
```go
// Server side - return rate limit info di response headers
w.Header().Set("X-RateLimit-Limit", "100")
w.Header().Set("X-RateLimit-Remaining", "42")
w.Header().Set("X-RateLimit-Reset", "1234567890")
```

---

## API Response Issues

### Error: "unexpected response format"

**Penyebab:**
- Response format berubah
- Error response format tidak konsisten
- Field missing di response

**Solusi:**

1. Pastikan response format konsisten:
```go
// ✅ Consistent format untuk single object
dim.Json(w, http.StatusOK, user)
// {"id": 1, "email": "user@example.com", ...}

// ✅ Consistent format untuk collection
dim.Json(w, http.StatusOK, users)
// [{"id": 1, "email": "..."}, ...]

// ✅ Consistent format untuk error
dim.JsonError(w, http.StatusBadRequest, "Validation failed", map[string]string{
    "email": "Invalid email",
})
// {"message": "Validation failed", "errors": {"email": "Invalid email"}}
```

2. Use response helpers everywhere:
```go
// ❌ Salah - inconsistent format
w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "user": user})

// ✅ Benar - using helper
dim.Json(w, http.StatusOK, user)
```

3. Document API responses di documentation:
```markdown
## GET /api/users/:id

### Response 200 OK
```json
{
  "id": 1,
  "email": "user@example.com",
  "username": "johndoe",
  "created_at": "2024-01-01T00:00:00Z"
}
```

### Response 404 Not Found
```json
{
  "message": "User not found",
  "errors": null
}
```
```

### Error: "content-type: application/json; charset=utf-8" mismatch

**Penyebab:**
- Client expect charset berbeda
- Response header tidak di-set correctly

**Solusi:**

1. Verify content-type di response helpers:
```go
// Di dim.Json() helper
func Json(w http.ResponseWriter, status int, data interface{}) {
    w.Header().Set("Content-Type", "application/json; charset=utf-8")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(data)
}
```

2. Jangan set Content-Type multiple times:
```go
// ❌ Salah
w.Header().Set("Content-Type", "application/json")
w.Header().Set("Content-Type", "text/plain")

// ✅ Benar - hanya set sekali via helper
dim.Json(w, http.StatusOK, data)
```

### Error: "CORS error" atau "No 'Access-Control-Allow-Origin' header"

**Penyebab:**
- CORS middleware tidak configured
- CORS_ALLOWED_ORIGINS tidak match request origin
- Preflight request tidak handled

**Solusi:**

1. Enable CORS middleware:
```go
r.Use(dim.CORS(cfg.CORS))
```

2. Configure CORS_ALLOWED_ORIGINS:
```bash
# .env
CORS_ALLOWED_ORIGINS=http://localhost:3000,https://app.example.com
CORS_ALLOWED_METHODS=GET,POST,PUT,DELETE,PATCH,OPTIONS
CORS_ALLOWED_HEADERS=Content-Type,Authorization,X-CSRF-Token
CORS_ALLOW_CREDENTIALS=true
CORS_MAX_AGE=3600
```

3. Verify CORS preflight handled:
```bash
# Test preflight request
curl -X OPTIONS http://localhost:8080/api/users \
  -H "Origin: http://localhost:3000" \
  -H "Access-Control-Request-Method: POST" \
  -v
```

---

## Performance Problems

### Issue: "Slow API responses"

**Diagnosis:**

1. Measure response time:
```bash
# Benchmark endpoint
ab -n 100 -c 10 http://localhost:8080/api/users
```

2. Check logging middleware untuk response time:
```bash
# Di logs
"method=GET path=/api/users duration_ms=250"
```

3. Profile dengan pprof:
```go
import _ "net/http/pprof"

func main() {
    // Profile akan tersedia di http://localhost:6060/debug/pprof
    go func() {
        http.ListenAndServe("localhost:6060", nil)
    }()
    
    // Start server
}
```

**Solusi:**

1. Optimize database queries:
```go
// ❌ N+1 query problem
users := getAllUsers(ctx)
for _, user := range users {
    posts := getUserPosts(ctx, user.ID) // query per user!
}

// ✅ Batch query
users := getAllUsers(ctx)
posts := getUsersPostsBatch(ctx, users) // single query
```

2. Add database indexes:
```sql
-- Frequently queried columns
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_posts_user_id ON posts(user_id);
```

3. Implement caching:
```go
type CachedUserStore struct {
    store UserStore
    cache map[int64]*User
    mu    sync.RWMutex
}

func (c *CachedUserStore) GetUser(ctx context.Context, id int64) (*User, error) {
    c.mu.RLock()
    if user, ok := c.cache[id]; ok {
        c.mu.RUnlock()
        return user, nil
    }
    c.mu.RUnlock()
    
    user, err := c.store.GetUser(ctx, id)
    if err == nil {
        c.mu.Lock()
        c.cache[id] = user
        c.mu.Unlock()
    }
    return user, err
}
```

4. Enable keep-alive connections:
```go
server := &http.Server{
    Addr:    ":" + cfg.Server.Port,
    Handler: router,
    // Connection pooling
}
```

### Issue: "High memory usage"

**Penyebab:**
- Connection leak
- Memory leak di goroutines
- Unbounded cache

**Solusi:**

1. Monitor goroutines:
```go
import "runtime"

go func() {
    ticker := time.NewTicker(10 * time.Second)
    for range ticker.C {
        logger.Debug("goroutines", "count", runtime.NumGoroutine())
    }
}()
```

2. Implement bounded cache dengan max size:
```go
type BoundedCache struct {
    items map[string]interface{}
    max   int
    mu    sync.RWMutex
}

func (c *BoundedCache) Set(key string, value interface{}) {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    if len(c.items) >= c.max && c.items[key] == nil {
        // Evict oldest item
        for k := range c.items {
            delete(c.items, k)
            break
        }
    }
    
    c.items[key] = value
}
```

3. Use connection pooling limits:
```bash
DB_MAX_CONNS=25
```

---

## Middleware & Routing Issues

### Error: "handler not found" atau 404 Not Found

**Penyebab:**
- Route tidak registered
- Route path berbeda dari request
- HTTP method berbeda

**Solusi:**

1. Verify route registered:
```bash
# Add debug middleware untuk log semua routes
r.Use(func(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        logger.Debug("request", "method", r.Method, "path", r.RequestURI)
        next(w, r)
    }
})
```

2. Check exact route path:
```go
// ❌ Salah
r.Get("/users/:id", getUserHandler)
// Request: GET /users/123 ✅ Match
// Request: GET /api/users/123 ❌ No match

// ✅ Benar
r.Get("/api/users/:id", getUserHandler)
// Request: GET /api/users/123 ✅ Match
```

3. Verify HTTP method:
```go
// ❌ Salah
r.Post("/api/users", getListHandler) // GET request -> 404

// ✅ Benar
r.Get("/api/users", getListHandler)
r.Post("/api/users", createHandler)
```

### Error: "middleware executed in wrong order"

**Penyebab:**
- Middleware order tidak sesuai KRITIS sequence
- Middleware di-apply dengan order yang salah

**Solusi:**

1. Selalu ikuti KRITIS middleware order:
```go
// KRITIS ORDER - JANGAN UBAH!
r.Use(dim.Recovery(logger))      // 1. Recovery first - catch panic
r.Use(dim.Logger(logger))         // 2. Logger - log requests
r.Use(dim.CORS(corsConfig))       // 3. CORS - handle preflight
r.Use(dim.CSRF(csrfConfig))       // 4. CSRF - protect state-changing
r.Use(dim.RateLimit(rateConfig))  // 5. Rate limit - protect resources
// Auth middleware applied per-route
```

2. Apply auth middleware hanya ke protected routes:
```go
// Public routes
r.Post("/auth/login", loginHandler)
r.Post("/auth/register", registerHandler)

// Protected routes
r.Get("/api/profile", profileHandler, dim.RequireAuth())

// Atau dengan group
api := r.Group("/api", dim.RequireAuth())
api.Get("/profile", profileHandler)
api.Post("/posts", createPostHandler)
```

---

## Deployment Issues

### Error: "application won't start in production"

**Penyebab:**
- Environment variables tidak set
- Binary tidak compatible dengan OS
- Dependencies missing

**Solusi:**

1. Verify binary compatibility:
```bash
# Build untuk specific OS
GOOS=linux GOARCH=amd64 go build -o app .

# Check binary info
file app
```

2. Test application start dalam isolated environment:
```bash
# Jalankan di container untuk test
docker run --rm -e JWT_SECRET=test golang:latest ./app
```

3. Verify semua dependencies tersedia di production:
```bash
# Check vendor directory
ls -la vendor/

# Or use module-based
go mod download
```

### Error: "database migration failed"

**Penyebab:**
- Migrations tidak idempotent
- Schema conflicts
- Permissions issue

**Solusi:**

1. Implement idempotent migrations:
```go
// ✅ Idempotent - safe to run multiple times
Up: func(pool *pgxpool.Pool) error {
    _, err := pool.Exec(context.Background(), `
        CREATE TABLE IF NOT EXISTS users (
            id BIGSERIAL PRIMARY KEY,
            email VARCHAR(255) UNIQUE NOT NULL
        )
    `)
    return err
},

// ✅ Idempotent - check before drop
Down: func(pool *pgxpool.Pool) error {
    _, err := pool.Exec(context.Background(), `
        DROP TABLE IF EXISTS users
    `)
    return err
},
```

2. Test migrations di staging terlebih dahulu:
```bash
# Backup database
pg_dump dim_staging > backup.sql

# Run migrations
./app migrate up

# Verify
psql dim_staging -c "\dt"
```

---

## Security Concerns

### Issue: "Possible SQL injection"

**Penyebab:**
- Query building dengan string concatenation
- User input tidak di-sanitize

**Solusi:**

1. Selalu gunakan parameterized queries:
```go
// ❌ Salah - SQL injection risk
query := fmt.Sprintf("SELECT * FROM users WHERE email = '%s'", email)
row := db.QueryRow(query)

// ✅ Benar - parameterized
query := "SELECT * FROM users WHERE email = $1"
row := db.QueryRow(query, email)
```

2. Use ORM atau query builder jika prefer:
```go
// Dengan parameterized query via dim.Database
var user User
err := db.QueryRow(ctx, 
    "SELECT id, email FROM users WHERE email = $1", 
    email,
).Scan(&user.ID, &user.Email)
```

### Issue: "Sensitive data di logs"

**Penyebab:**
- Passwords di-log
- Tokens di-log
- Personal data di-log

**Solusi:**

1. Sanitize sensitive data sebelum logging:
```go
// ❌ Salah
logger.Info("user registered", "password", req.Password)

// ✅ Benar
logger.Info("user registered", "email", req.Email)

// ✅ Atau gunakan custom marshal
type SafeUser struct {
    Email string `json:"email"`
    // Password tidak di-include
}
```

2. Implement log filtering:
```go
func SanitizeLog(msg string) string {
    // Remove password, token, apikey patterns
    re := regexp.MustCompile(`(password|token|apikey)=\S+`)
    return re.ReplaceAllString(msg, "$1=***")
}
```

### Issue: "CSRF token tidak valid"

**Penyebab:**
- Token tidak di-generate
- Token di-mismatch antara cookie dan header
- Token expired

**Solusi:**

1. Verify CSRF middleware enabled:
```go
r.Use(dim.CSRF(dim.CSRFConfig{
    Enabled:     true,
    TokenLength: 32,
    ExemptPaths: []string{"/health", "/webhooks"},
}))
```

2. Generate token di form page:
```html
<!-- Server-side generate token -->
<form method="POST" action="/api/action">
    <input type="hidden" name="_csrf" value="{{.CSRFToken}}">
    <!-- form fields -->
</form>

<!-- JavaScript untuk API requests -->
<script>
const token = document.querySelector('[name="_csrf"]').value;
fetch('/api/action', {
    method: 'POST',
    headers: {
        'X-CSRF-Token': token
    }
});
</script>
```

3. Verify token di request:
```bash
# POST request harus include CSRF token di header atau form field
curl -X POST http://localhost:8080/api/action \
  -H "X-CSRF-Token: <token>" \
  -d "data=value"
```

---

## Praktik Terbaik

### 1. Logging untuk Troubleshooting

- Log di critical points (startup, errors, auth attempts)
- Include context (user ID, request ID, operation)
- Use structured logging dengan slog
- Jangan log sensitive data

```go
logger.Info("user login successful", 
    "user_id", user.ID,
    "method", "email_password",
)
```

### 2. Error Handling Strategy

- Return meaningful error messages
- Include error details di logs
- Expose safe messages ke client
- Handle panics dengan recovery middleware

```go
if err != nil {
    logger.Error("database error", "error", err, "query", query)
    dim.JsonError(w, http.StatusInternalServerError, 
        "Internal server error", nil)
    return
}
```

### 3. Monitoring & Alerting

- Monitor application health (uptime, errors, response time)
- Set up alerts untuk critical issues
- Regular backup checks
- Performance baseline tracking

### 4. Testing Before Production

- Run all tests (`go test ./...`)
- Run race condition detector (`go test -race ./...`)
- Test dengan production-like configuration
- Load test critical paths

### 5. Version Control & Deployment Tracking

- Tag releases di git
- Keep changelog
- Document breaking changes
- Track deployment history

```bash
# Version management
git tag v1.0.0
git push --tags

# Show deployment info
git show v1.0.0
```

### 6. Debugging Tools

**pprof - CPU & Memory profiling:**
```bash
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30
```

**dlv - Debugger:**
```bash
dlv debug ./cmd/main.go
(dlv) break main.main
(dlv) continue
```

**Race detector:**
```bash
go test -race ./...
```

---

## Referensi

- [01-getting-started.md](01-getting-started.md) - Quick start guide
- [07-configuration.md](07-configuration.md) - Configuration reference
- [04-middleware.md](04-middleware.md) - Middleware details
- [05-authentication.md](05-authentication.md) - Auth flow
- [14-security.md](14-security.md) - Security practices
- [12-structured-logging.md](12-structured-logging.md) - Logging guide
- [17-deployment.md](17-deployment.md) - Deployment guide
