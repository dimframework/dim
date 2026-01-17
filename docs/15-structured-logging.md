# Logging di Framework dim

Pelajari cara menggunakan structured logging dengan slog di framework dim.

## Daftar Isi

- [Konsep Logging](#konsep-logging)
- [Setup Logger](#setup-logger)
- [Log Levels](#log-levels)
- [Structured Logging](#structured-logging)
- [Logger Middleware](#logger-middleware)
- [Praktik Terbaik Logging](#logging-best-practices)
- [Log Aggregation](#log-aggregation)

---

## Konsep Logging

### Filosofi Logging di dim

Framework dim menggunakan **structured logging** dengan `slog` (Go's standard library logger):

```go
// Unstructured (BAD)
log.Printf("User %s registered with email %s", username, email)

// Structured (GOOD)
logger.Info("User registered",
    "username", username,
    "email", email,
)
```

### Logging Hierarchy

```
Application
  ├─ Logger (configured)
  └─ Components use logger
     ├─ Middleware (log requests)
     ├─ Handlers (log business logic)
     ├─ Services (log operations)
     └─ Stores (log queries)
```

### Log Output Example

```
2024-01-10T10:30:45.123Z INFO   msg="User registered"
  username="john" email="john@example.com"

2024-01-10T10:30:46.456Z ERROR  msg="Database query failed"
  query="SELECT * FROM users" error="connection refused"
```

---

## Setup Logger

### Membuat Instance Logger

Cara paling umum adalah membuat *logger* JSON yang menulis ke *standard output*.

```go
import (
    "log/slog"
    "github.com/dimframework/dim"
)

func main() {
    // Membuat logger dengan level log Info
    logger := dim.NewLogger(slog.LevelInfo)
    
    // Menggunakan logger
    logger.Info("Aplikasi dimulai")
}
```

### Konstruktor Logger Tambahan

Framework `dim` menyediakan beberapa konstruktor untuk kasus penggunaan yang berbeda:

-   **`NewLoggerWithWriter(writer, level)`**: Membuat *logger* JSON yang menulis ke `io.Writer` kustom (misalnya, sebuah file).
-   **`NewTextLogger(level)`**: Membuat *logger* dalam format teks (bukan JSON) yang menulis ke *standard output*. Berguna untuk lingkungan pengembangan lokal.
-   **`NewTextLoggerWithWriter(writer, level)`**: Membuat *logger* format teks yang menulis ke `io.Writer` kustom.

```go
// Contoh: Logging ke sebuah file
file, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
if err != nil {
    log.Fatal(err)
}
defer file.Close()

// Logger JSON yang menulis ke file
fileLogger := dim.NewLoggerWithWriter(file, slog.LevelInfo)
fileLogger.Info("Ini akan masuk ke dalam file log.")

// Logger teks untuk development
devLogger := dim.NewTextLogger(slog.LevelDebug)
devLogger.Debug("Ini adalah pesan debug untuk konsol.")
```

### Level-Level Logger

```go
// Level Debug - Informasi sangat detail untuk debugging.
logger := dim.NewLogger(slog.LevelDebug)

// Level Info - Pesan informasional umum.
logger := dim.NewLogger(slog.LevelInfo)

// Level Warn - Pesan peringatan untuk situasi yang tidak kritis.
logger := dim.NewLogger(slog.LevelWarn)

// Level Error - Pesan error yang membutuhkan perhatian.
logger := dim.NewLogger(slog.LevelError)
```

### Pola Logger Global

Mendefinisikan *logger* di level paket untuk akses mudah di seluruh aplikasi.

```go
// logger.go
package main

import (
    "log/slog"
    "github.com/dimframework/dim"
)

var logger *slog.Logger

func init() {
    logger = dim.NewLogger(slog.LevelInfo)
}

// main.go
func main() {
    // logger dapat diakses di seluruh paket
    logger.Info("Server dimulai")
}
```

### Mengirim Logger ke Komponen

Praktik terbaik adalah mengirim instance *logger* sebagai dependensi.

```go
type AuthService struct {
    userStore UserStore
    logger    *slog.Logger
}

func NewAuthService(store UserStore, logger *slog.Logger) *AuthService {
    return &AuthService{
        userStore: store,
        logger:    logger,
    }
}

func (s *AuthService) Register(ctx context.Context, email, username, password string) (*User, error) {
    s.logger.Info("User registration attempt",
        "email", email,
        "username", username,
    )
    
    // ... logika registrasi ...
    
    s.logger.Info("User registered successfully",
        "user_id", user.ID,
        "email", user.Email,
    )
    
    return user, nil
}
```

---

## Log Levels

### Debug

Detailed information untuk debugging:

```go
logger.Debug("Query executed",
    "query", "SELECT * FROM users WHERE id = $1",
    "args", []interface{}{1},
    "rows", 1,
)

logger.Debug("Request body parsed",
    "content_type", "application/json",
    "size_bytes", 256,
)
```

**When to use**:
- Database queries
- Request/response details
- Variable values during processing
- Flow control decisions

**Output**: Only shown when level is Debug

### Info

General informational messages:

```go
logger.Info("Application started",
    "port", 8080,
    "env", "production",
)

logger.Info("User registered",
    "user_id", 123,
    "email", "john@example.com",
)

logger.Info("Request processed",
    "method", "POST",
    "path", "/auth/register",
    "status", 201,
    "duration_ms", 45,
)
```

**When to use**:
- Application lifecycle (start, shutdown)
- Important business events
- User actions
- Request completion

**Output**: Default level

### Warn

Warning messages about potentially problematic situations:

```go
logger.Warn("High error rate detected",
    "error_rate", "5%",
    "threshold", "1%",
)

logger.Warn("Slow query executed",
    "query", "...",
    "duration_ms", 2000,
    "threshold_ms", 1000,
)

logger.Warn("Rate limit approaching",
    "user_id", 123,
    "requests", 95,
    "limit", 100,
)
```

**When to use**:
- Unusual but recoverable conditions
- Performance degradation
- Resource limits approaching
- Deprecated API usage

**Output**: Always shown (unless level is Error)

### Error

Error messages about failures:

```go
logger.Error("Database connection failed",
    "host", "db.example.com",
    "error", err.Error(),
)

logger.Error("User registration failed",
    "email", "john@example.com",
    "reason", "email_already_exists",
)

logger.Error("Request processing error",
    "path", "/api/users",
    "error", err.Error(),
    "user_id", userID,
)
```

**When to use**:
- Errors that need attention
- Failed operations
- Exceptions caught
- System failures (but recoverable)

**Output**: Always shown

---

## Structured Logging

### Basic Structured Log

```go
logger.Info("User logged in",
    "user_id", 123,
    "email", "john@example.com",
    "ip_address", "192.168.1.1",
)

// Output:
// {"time":"...","level":"INFO","msg":"User logged in","user_id":123,"email":"john@example.com","ip_address":"192.168.1.1"}
```

### Menambahkan Konteks ke Logger

Terkadang Anda ingin menambahkan atribut yang sama ke sekelompok pesan *log*. Gunakan `WithGroup` atau `WithAttrs` untuk membuat *logger* baru dengan konteks yang sudah ada.

-   **`WithGroup(name string)`**: Membuat grup atribut bersarang.
-   **`WithAttrs(attrs ...slog.Attr)`**: Menambahkan atribut ke level atas *logger*.

```go
// Logger awal
logger := dim.NewLogger(slog.LevelInfo)

// Menambahkan atribut persisten ke logger baru
requestLogger := logger.With(
    slog.String("request_id", "abc-123"),
    slog.String("user_agent", "Go-Client/1.1"),
)

requestLogger.Info("Request started", "path", "/users")
// Output akan menyertakan request_id dan user_agent
// {"time":"...","level":"INFO","msg":"Request started","request_id":"abc-123","user_agent":"Go-Client/1.1","path":"/users"}

// Menggunakan grup untuk data terkait
dbLogger := logger.WithGroup("database")
dbLogger.Info("Connection established", "host", "localhost")
// Output:
// {"time":"...","level":"INFO","msg":"Connection established","database":{"host":"localhost"}}
```

### Multiple Types

```go
logger.Info("Request completed",
    "method", "POST",                    // string
    "status", 200,                       // int
    "duration_ms", 45,                   // int
    "success", true,                     // bool
    "timestamp", time.Now(),             // time.Time
    "tags", []string{"api", "auth"},     // slice
)
```

### Complex Nested Data

```go
type RequestLog struct {
    Method     string
    Path       string
    StatusCode int
    Duration   time.Duration
    UserID     int64
}

log := RequestLog{
    Method:     "GET",
    Path:       "/users/123",
    StatusCode: 200,
    Duration:   45 * time.Millisecond,
    UserID:     123,
}

logger.Info("Request", 
    "method", log.Method,
    "path", log.Path,
    "status", log.StatusCode,
    "duration_ms", log.Duration.Milliseconds(),
    "user_id", log.UserID,
)
```

### Error Logging dengan Context

```go
err := userStore.FindByID(ctx, userID)
if err != nil {
    logger.Error("Database query failed",
        "operation", "FindByID",
        "user_id", userID,
        "error", err.Error(),
        "error_type", fmt.Sprintf("%T", err),
    )
    return nil, err
}
```

---

## Logger Middleware

### Setup

```go
func main() {
    logger := dim.NewLogger(slog.LevelInfo)
    router := dim.NewRouter()
    
    // Add logger middleware
    router.Use(dim.LoggerMiddleware(logger))
    
    // ... other setup ...
}
```

### What It Logs

```
[Request start]
2024-01-10T10:30:45.123Z INFO  msg="Request started"
  method="POST" path="/auth/login" ip="192.168.1.1"

[Business logic processes]

[Request complete]
2024-01-10T10:30:45.456Z INFO  msg="Request completed"
  method="POST" path="/auth/login" status=200
  duration_ms=333 bytes=512
```

### Log Fields

```go
logger.Info("Request completed",
    "method", r.Method,           // POST
    "path", r.URL.Path,           // /auth/login
    "status", statusCode,         // 200
    "duration_ms", duration,      // 333
    "bytes_written", bytesWritten,// 512
    "ip_address", r.RemoteAddr,   // 192.168.1.1
)
```

---

## Logging Best Practices

### ✅ DO: Use Structured Logging

```go
// ✅ BAIK - Structured
logger.Info("User created",
    "user_id", user.ID,
    "email", user.Email,
    "created_at", user.CreatedAt,
)

// ❌ BURUK - Unstructured
log.Printf("User created: %v", user)
```

### ✅ DO: Include Context

```go
// ✅ BAIK - Include request context
logger.Error("Query failed",
    "query", query,
    "user_id", user.ID,
    "request_id", requestID,
    "error", err.Error(),
)

// ❌ BURUK - Minimal context
log.Println("Error:", err)
```

### ✅ DO: Use Consistent Field Names

```go
// ✅ BAIK - Consistent naming
logger.Info("Event",
    "user_id", 123,
    "email", "user@example.com",
    "created_at", time.Now(),
)

// ❌ BURUK - Inconsistent naming
logger.Info("Event",
    "uid", 123,
    "user_email", "user@example.com",
    "creation_time", time.Now(),
)
```

### ✅ DO: Log at Appropriate Levels

```go
// ✅ BAIK
logger.Debug("Query executed", "query", sql)       // Dev debugging
logger.Info("User registered", "email", email)     // Business event
logger.Warn("Slow query", "duration_ms", 2000)     // Warning condition
logger.Error("DB failed", "error", err)            // Error condition

// ❌ BURUK
logger.Info("Query executed", "query", sql)        // Too verbose for Info
logger.Error("User registered", "email", email)    // Not an error
```

### ✅ DO: Avoid Logging Sensitive Data

```go
// ✅ BAIK - No sensitive data
logger.Info("User login attempt",
    "user_id", userID,
    "method", "email",
)

logger.Info("API call",
    "endpoint", "/users",
    "method", "POST",
)

// ❌ BURUK - Logs sensitive data
logger.Info("User login",
    "email", email,
    "password", password,  // NEVER!
)

logger.Info("API call",
    "auth_token", token,   // NEVER!
)
```

### ✅ DO: Log Request ID for Tracing

```go
// In middleware, set request ID
func requestIDMiddleware(next HandlerFunc) HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        requestID := generateUUID()
        r = SetRequestID(r, requestID)
        next(w, r)
    }
}

// In handler, include request ID
func handler(w http.ResponseWriter, r *http.Request) {
    requestID := GetRequestID(r)
    
    logger.Info("Processing request",
        "request_id", requestID,
        "path", r.URL.Path,
    )
    
    // Later in code
    logger.Error("Something failed",
        "request_id", requestID,
        "error", err.Error(),
    )
}
```

### ✅ DO: Use Error Helper for Exceptions

```go
// ✅ BAIK - Proper error logging
func handler(w http.ResponseWriter, r *http.Request) {
    user, err := userStore.FindByID(ctx, userID)
    
    if err != nil {
        logger.Error("User query failed",
            "user_id", userID,
            "error", err.Error(),
        )
        dim.JsonError(w, 500, "Database error", nil)
        return
    }
}

// ❌ BURUK - Ignore error
func handler(w http.ResponseWriter, r *http.Request) {
    user, _ := userStore.FindByID(ctx, userID)
    // What if err != nil?
}
```

---

## Log Aggregation

### Structured Log Format

Framework dim logs dalam format JSON untuk mudah di-aggregate:

```json
{
  "timestamp": "2024-01-10T10:30:45.123Z",
  "level": "INFO",
  "msg": "User registered",
  "user_id": 123,
  "email": "john@example.com"
}
```

### Export ke ELK Stack

Contoh docker-compose untuk ELK:

```yaml
version: '3'
services:
  elasticsearch:
    image: docker.elastic.co/elasticsearch/elasticsearch:8.0.0
    environment:
      - discovery.type=single-node
    ports:
      - "9200:9200"

  kibana:
    image: docker.elastic.co/kibana/kibana:8.0.0
    ports:
      - "5601:5601"
    depends_on:
      - elasticsearch

  filebeat:
    image: docker.elastic.co/beats/filebeat:8.0.0
    volumes:
      - ./logs:/var/log/app
      - ./filebeat.yml:/usr/share/filebeat/filebeat.yml
    depends_on:
      - elasticsearch
```

### Filebeat Configuration

```yaml
filebeat.inputs:
- type: log
  enabled: true
  paths:
    - /var/log/app/*.log

output.elasticsearch:
  hosts: ["elasticsearch:9200"]

processors:
  - decode_json_fields:
      fields: ["message"]
      target: ""
```

---

## Complete Logging Example

### Application Setup

```go
package main

import (
    "log/slog"
    "os"
    "github.com/dimframework/dim"
)

var logger *slog.Logger

func init() {
    // Create logger
    logLevel := slog.LevelInfo
    if os.Getenv("DEBUG") == "1" {
        logLevel = slog.LevelDebug
    }
    
    logger = dim.NewLogger(logLevel)
}

func main() {
    logger.Info("Application starting")
    
    // Load config
    cfg, err := dim.LoadConfig()
    if err != nil {
        logger.Error("Failed to load config", "error", err.Error())
        os.Exit(1)
    }
    
    // Setup database
    db, err := dim.NewPostgresDatabase(cfg.Database)
    if err != nil {
        logger.Error("Failed to connect to database",
            "host", cfg.Database.WriteHost,
            "error", err.Error(),
        )
        os.Exit(1)
    }
    defer db.Close()
    
    logger.Info("Database connected",
        "host", cfg.Database.WriteHost,
        "database", cfg.Database.Database,
    )
    
    // Setup router
    router := dim.NewRouter()
    
    // Add middleware
    router.Use(dim.Recovery(logger))
    router.Use(dim.LoggerMiddleware(logger))
    router.Use(dim.CORS(cfg.CORS))
    
    logger.Info("Router configured")
    
    // Setup services
    userStore := dim.NewPostgresUserStore(db)
    authService := dim.NewAuthService(userStore, nil, nil, &cfg.JWT)
    
    // Register routes
    router.Post("/auth/register", registerHandler(authService, logger))
    
    // Start server
    logger.Info("Server starting",
        "port", cfg.Server.Port,
        "env", os.Getenv("ENV"),
    )
    
    if err := http.ListenAndServe(":"+cfg.Server.Port, router); err != nil {
        logger.Error("Server error", "error", err.Error())
        os.Exit(1)
    }
}

func registerHandler(authService *AuthService, logger *slog.Logger) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        requestID := dim.GetRequestID(r)
        
        // Parse request
        var req struct {
            Email    string
            Username string
            Password string
        }
        
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            logger.Warn("Invalid JSON in register request",
                "request_id", requestID,
                "error", err.Error(),
            )
            dim.JsonError(w, 400, "Invalid JSON", nil)
            return
        }
        
        logger.Debug("Registration request parsed",
            "request_id", requestID,
            "email", req.Email,
            "username", req.Username,
        )
        
        // Register user
        user, err := authService.Register(r.Context(), 
            req.Email, req.Username, req.Password)
        
        if err != nil {
            logger.Error("User registration failed",
                "request_id", requestID,
                "email", req.Email,
                "error", err.Error(),
            )
            dim.JsonError(w, 500, "Registration failed", nil)
            return
        }
        
        logger.Info("User registered successfully",
            "request_id", requestID,
            "user_id", user.ID,
            "email", user.Email,
        )
        
        dim.Json(w, 201, user)
    }
}
```

---

## Summary

Logging di dim:
- **Structured** - JSON format untuk aggregation
- **Leveled** - Debug, Info, Warn, Error
- **Contextual** - Include relevant data
- **Safe** - No sensitive data
- **Traceable** - Request IDs untuk debugging

Lihat [Error Handling](08-error-handling.md) untuk error logging detail.

---

**Lihat Juga**:
- [Error Handling](08-error-handling.md) - Error logging
- [Middleware](04-middleware.md) - Logger middleware
- [Konfigurasi](07-configuration.md) - Log level configuration
