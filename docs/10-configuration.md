# Konfigurasi Framework dim

Pelajari cara mengkonfigurasi framework dim menggunakan environment variables.

## Daftar Isi

- [Konsep Konfigurasi](#konsep-konfigurasi)
- [Environment Variables](#environment-variables)
- [Server Configuration](#server-configuration)
- [Database Configuration](#database-configuration)
- [JWT Configuration](#jwt-configuration)
- [CORS Configuration](#cors-configuration)
- [CSRF Configuration](#csrf-configuration)
- [Rate Limiting Configuration](#rate-limiting-configuration)
- [Email Configuration](#email-configuration)
- [Load Configuration](#load-configuration)
- [Praktik Terbaik](#best-practices)

---

## Konsep Konfigurasi

### Filosofi Konfigurasi

Framework dim mengikuti **12-factor app** principles:
- **Never hardcode secrets** - Gunakan environment variables
- **Environment-specific** - Config berubah per environment (dev, staging, prod)
- **Simple & explicit** - Jelas, tidak magic
- **Type-safe** - Parse dan validate saat load

### Config Structure

```go
type Config struct {
    Server     ServerConfig
    Database   DatabaseConfig
    JWT        JWTConfig
    CORS       CORSConfig
    CSRF       CSRFConfig
    RateLimit  RateLimitConfig
    Email      EmailConfig
}
```

---

## Environment Variables

### File `.env`

Buat file `.env` di root project:

```bash
# Copy dari template
cp .env.example .env

# Edit dengan nilai Anda
vim .env
```

### Load .env File

Framework otomatis load `.env` saat `dim.LoadConfig()`:

```go
cfg, err := dim.LoadConfig()
if err != nil {
    log.Fatal(err)
}
```

### Environment Variable Format

Format: `SECTION_KEY=value`

```bash
SERVER_PORT=8080                    # SERVER section
DB_WRITE_HOST=localhost             # DATABASE section
JWT_SECRET=super-secret-key         # JWT section
CORS_ALLOWED_ORIGINS=http://...     # CORS section
```

---

## Server Configuration

### Environment Variables

```bash
# Server port (default: 8080)
SERVER_PORT=8080

# Read timeout untuk requests (default: 30s)
SERVER_READ_TIMEOUT=30s

# Write timeout untuk responses (default: 30s)
SERVER_WRITE_TIMEOUT=30s
```

### Server Config Struct

```go
type ServerConfig struct {
    Port         string        // "8080", "3000", etc
    ReadTimeout  time.Duration // 30s, 1m, etc
    WriteTimeout time.Duration // 30s, 1m, etc
}
```

### Access Server Config

```go
cfg, _ := dim.LoadConfig()

port := cfg.Server.Port           // "8080"
readTimeout := cfg.Server.ReadTimeout    // 30 * time.Second

// Use in server setup
server := &http.Server{
    Addr:         ":" + cfg.Server.Port,
    Handler:      router,
    ReadTimeout:  cfg.Server.ReadTimeout,
    WriteTimeout: cfg.Server.WriteTimeout,
}

server.ListenAndServe()
```

### Timeout Guide

| Timeout | Purpose | Default | Range |
|---------|---------|---------|-------|
| ReadTimeout | Baca request | 30s | 10s-60s |
| WriteTimeout | Kirim response | 30s | 10s-60s |

---

## Database Configuration

### Environment Variables

```bash
# Write host (Primary/Master)
DB_WRITE_HOST=localhost

# Read hosts (Replicas) - comma-separated untuk multiple
DB_READ_HOSTS=localhost,localhost

# Database port (default: 5432)
DB_PORT=5432

# Database name
DB_NAME=myapp_db

# Database user
DB_USER=postgres

# Database password
DB_PASSWORD=secretpassword

# SSL mode: disable|require|prefer|allow|verify-ca|verify-full (default: disable)
DB_SSL_MODE=disable

# Max connections per pool (default: 25)
DB_MAX_CONNS=25
```

### Database Config Struct

```go
type DatabaseConfig struct {
    WriteHost     string
    ReadHosts     []string
    Port          int
    Database      string
    Username      string
    Password      string
    SSLMode       string
    MaxConns      int
    RuntimeParams map[string]string // Custom runtime parameters (e.g., search_path)
    QueryExecMode string            // e.g., "simple_protocol"
}
```

### Connection String Format

Framework build connection string internally:

```
postgresql://user:password@host:port/database?sslmode=disable
```

### SSL Modes Explained

| Mode | Meaning | Security | Use Case |
|------|---------|----------|----------|
| `disable` | No SSL | ❌ Low | Development, local |
| `require` | SSL required | ✅ High | Production |
| `prefer` | SSL if available | ⚠️ Medium | Staging |
| `allow` | SSL tolerated | ⚠️ Low | Testing |
| `verify-ca` | SSL + CA verify | ✅ High | Strict prod |
| `verify-full` | SSL + full verify | ✅ Very High | Very strict prod |

### Load Database Config

```go
cfg, _ := dim.LoadConfig()

db, err := dim.NewPostgresDatabase(cfg.Database)
if err != nil {
    log.Fatal(err)
}
```

---

## JWT Configuration

### Environment Variables

```bash
# JWT signing secret (CHANGE IN PRODUCTION!)
JWT_SECRET=your-super-secret-key-change-in-production

# Access token expiry (default: 15m)
JWT_ACCESS_TOKEN_EXPIRY=15m

# Refresh token expiry (default: 7d)
JWT_REFRESH_TOKEN_EXPIRY=7d
```

### JWT Config Struct

```go
type JWTConfig struct {
    Secret              string
    AccessTokenExpiry   time.Duration
    RefreshTokenExpiry  time.Duration
}
```

### Token Expiry Guide

```bash
# Short-lived access token
JWT_ACCESS_TOKEN_EXPIRY=15m      # 15 minutes (recommended)

# Long-lived refresh token
JWT_REFRESH_TOKEN_EXPIRY=7d      # 7 days (recommended)

# Alternative values
JWT_ACCESS_TOKEN_EXPIRY=30m      # 30 minutes
JWT_ACCESS_TOKEN_EXPIRY=1h       # 1 hour
JWT_REFRESH_TOKEN_EXPIRY=30d     # 30 days
JWT_REFRESH_TOKEN_EXPIRY=365d    # 1 year
```

### Secret Management

⚠️ **CRITICAL**: Jangan expose JWT secret!

```bash
# ✅ BAIK - Use strong, random secret
JWT_SECRET=generated-by-crypto-rand-256bits-hex

# ❌ BURUK - Weak secret
JWT_SECRET=secret

# ❌ BURUK - Hardcoded di code
const JWTSecret = "my-secret"
```

### Generate Strong Secret

```bash
# macOS/Linux
openssl rand -hex 32

# Output: abc123def456...

# Set di .env
JWT_SECRET=abc123def456...
```

---

## CORS Configuration

### Environment Variables

```bash
# Allowed origins (comma-separated)
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:8080,https://example.com

# Allowed HTTP methods (comma-separated)
CORS_ALLOWED_METHODS=GET,POST,PUT,DELETE,PATCH,OPTIONS

# Allowed headers (comma-separated)
CORS_ALLOWED_HEADERS=Content-Type,Authorization,X-CSRF-Token

# Allow credentials (cookies, auth) - true|false
CORS_ALLOW_CREDENTIALS=true

# Preflight cache time (seconds)
CORS_MAX_AGE=3600
```

### CORS Config Struct

```go
type CORSConfig struct {
    AllowedOrigins   []string
    AllowedMethods   []string
    AllowedHeaders   []string
    AllowCredentials bool
    MaxAge           int
}
```

### Environment-Specific Examples

```bash
# Development
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:8080
CORS_ALLOWED_METHODS=GET,POST,PUT,DELETE,PATCH,OPTIONS
CORS_ALLOWED_HEADERS=Content-Type,Authorization,X-CSRF-Token
CORS_ALLOW_CREDENTIALS=true
CORS_MAX_AGE=3600

# Production
CORS_ALLOWED_ORIGINS=https://example.com,https://app.example.com
CORS_ALLOWED_METHODS=GET,POST,PUT,DELETE,PATCH,OPTIONS
CORS_ALLOWED_HEADERS=Content-Type,Authorization
CORS_ALLOW_CREDENTIALS=true
CORS_MAX_AGE=7200
```

### Load CORS Config

```go
cfg, _ := dim.LoadConfig()
router.Use(dim.CORS(cfg.CORS))
```

---

## CSRF Configuration

### Environment Variables

```bash
# Enable CSRF protection (default: true)
CSRF_ENABLED=true

# Paths yang skip CSRF validation (comma-separated)
CSRF_EXEMPT_PATHS=/webhooks,/health,/api/public

# Token length dalam bytes (default: 32)
CSRF_TOKEN_LENGTH=32

# Cookie name untuk token
CSRF_COOKIE_NAME             → "csrf_token"

# Header name untuk token
CSRF_HEADER_NAME=X-CSRF-Token
```

### CSRF Config Struct

```go
type CSRFConfig struct {
    Enabled     bool
    ExemptPaths []string
    TokenLength int
    CookieName  string
    HeaderName  string
}
```

### Exempt Paths

```bash
# Webhook endpoints (tidak perlu CSRF)
CSRF_EXEMPT_PATHS=/webhooks/stripe,/webhooks/github,/health

# Public endpoints
CSRF_EXEMPT_PATHS=/api/public,/public
```

---

## Rate Limiting Configuration

### Environment Variables

```bash
# Enable rate limiting (default: true)
RATE_LIMIT_ENABLED=true

# Requests per IP per reset period (default: 100)
RATE_LIMIT_PER_IP=100

# Requests per authenticated user per reset period (default: 200)
RATE_LIMIT_PER_USER=200

# Reset period (default: 1h)
RATE_LIMIT_RESET_PERIOD=1h
```

### Rate Limit Config Struct

```go
type RateLimitConfig struct {
    Enabled     bool
    PerIP       int
    PerUser     int
    ResetPeriod time.Duration
}
```

### Rate Limit Strategy Examples

```bash
# Strict (untuk sensitive endpoints)
RATE_LIMIT_PER_IP=10
RATE_LIMIT_PER_USER=50
RATE_LIMIT_RESET_PERIOD=5m

# Moderate (default)
RATE_LIMIT_PER_IP=100
RATE_LIMIT_PER_USER=200
RATE_LIMIT_RESET_PERIOD=1h

# Lenient (untuk public APIs)
RATE_LIMIT_PER_IP=1000
RATE_LIMIT_PER_USER=2000
RATE_LIMIT_RESET_PERIOD=1h
```

---

## Email Configuration

### Environment Variables

```bash
# From email address
EMAIL_FROM=noreply@example.com
```

### Email Config Struct

```go
type EmailConfig struct {
    FromEmail  string
}
```

---

## Load Configuration

### LoadConfig Function

```go
func LoadConfig() (*Config, error) {
    // Load dari .env file
    // Parse environment variables
    // Validate values
    // Return Config struct
}
```

### Usage

```go
package main

import (
    "log"
    "github.com/nuradiyana/dim"
)

func main() {
    cfg, err := dim.LoadConfig()
    if err != nil {
        log.Fatal("Config load failed:", err)
    }
    
    // Gunakan config di aplikasi
    db, _ := dim.NewPostgresDatabase(cfg.Database)
    router := setupRouter(cfg)
    
    logger.Info("Config loaded successfully")
}
```

### Default Values

Jika env var tidak set, framework gunakan default:

```
SERVER_PORT                  → "8080"
SERVER_READ_TIMEOUT          → "30s"
SERVER_WRITE_TIMEOUT         → "30s"
DB_PORT                      → 5432
DB_SSL_MODE                  → "disable"
DB_MAX_CONNS                 → 25
JWT_ACCESS_TOKEN_EXPIRY      → "15m"
JWT_REFRESH_TOKEN_EXPIRY     → "7d"
CORS_MAX_AGE                 → 3600
CSRF_ENABLED                 → true
CSRF_TOKEN_LENGTH            → 32
RATE_LIMIT_ENABLED           → true
RATE_LIMIT_PER_IP            → 100
RATE_LIMIT_PER_USER          → 200
RATE_LIMIT_RESET_PERIOD      → "1h"
```

---

## Environment-Specific Configs

### Development

`.env.dev`:
```bash
SERVER_PORT=8080
DB_WRITE_HOST=localhost
DB_READ_HOSTS=localhost
DB_NAME=myapp_dev
DB_USER=postgres
DB_PASSWORD=postgres
DB_SSL_MODE=disable
JWT_SECRET=dev-secret-key
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:8080
CSRF_ENABLED=false
RATE_LIMIT_ENABLED=false
```

Load dengan:
```bash
cp .env.dev .env
go run main.go
```

### Staging

`.env.staging`:
```bash
SERVER_PORT=8080
DB_WRITE_HOST=db-staging.example.com
DB_READ_HOSTS=db-replica.example.com
DB_NAME=myapp_staging
DB_USER=staging_user
DB_PASSWORD=$SECURE_PASSWORD
DB_SSL_MODE=require
JWT_SECRET=$SECURE_SECRET
CORS_ALLOWED_ORIGINS=https://staging-app.example.com
CSRF_ENABLED=true
RATE_LIMIT_ENABLED=true
```

### Production

`.env.prod`:
```bash
SERVER_PORT=8080
DB_WRITE_HOST=db-prod.example.com
DB_READ_HOSTS=db-replica1.example.com,db-replica2.example.com,db-replica3.example.com
DB_NAME=myapp_prod
DB_USER=prod_user
DB_PASSWORD=$SECURE_PASSWORD
DB_SSL_MODE=verify-full
DB_MAX_CONNS=50
JWT_SECRET=$SECURE_SECRET_32_BYTES
JWT_ACCESS_TOKEN_EXPIRY=15m
JWT_REFRESH_TOKEN_EXPIRY=7d
CORS_ALLOWED_ORIGINS=https://example.com,https://app.example.com,https://api.example.com
CORS_ALLOW_CREDENTIALS=true
CSRF_ENABLED=true
RATE_LIMIT_ENABLED=true
RATE_LIMIT_PER_IP=100
RATE_LIMIT_PER_USER=500
```

---

## Praktik Terbaik

### ✅ DO: Use Environment Variables

```bash
# ✅ BAIK - Configuration via env
JWT_SECRET=secure-random-key
DB_PASSWORD=secure-password

# ❌ BURUK - Hardcoded
const JWTSecret = "secret"
```

### ✅ DO: Create `.env.example`

```bash
# .env.example - Template untuk developer
SERVER_PORT=8080
DB_WRITE_HOST=localhost
DB_NAME=myapp_db
# ... etc
```

Add to `.gitignore`:
```
.env
.env.local
.env.*.local
```

### ✅ DO: Use Strong Secrets in Production

```bash
# Generate random secret
openssl rand -hex 32

# Set di .env
JWT_SECRET=abc123...
```

### ✅ DO: Validate Configuration at Startup

```go
cfg, err := dim.LoadConfig()
if err != nil {
    log.Fatal("Invalid configuration:", err)
}

// Validate critical values
if cfg.JWT.Secret == "" {
    log.Fatal("JWT_SECRET is required")
}

if cfg.Database.WriteHost == "" {
    log.Fatal("DB_WRITE_HOST is required")
}

logger.Info("Configuration validated successfully")
```

### ❌ DON'T: Hardcode Configuration

```go
// ❌ BURUK
const (
    DBHost = "localhost"
    DBPassword = "secret123"
    JWTSecret = "hardcoded-secret"
)

// ✅ BAIK
cfg, _ := dim.LoadConfig()
// cfg.Database.WriteHost dari env
// cfg.JWT.Secret dari env
```

### ✅ DO: Document Configuration

Buat dokumentasi untuk setiap variable:

```bash
# Server configuration
SERVER_PORT=8080                 # Server port number
SERVER_READ_TIMEOUT=30s          # Request read timeout
SERVER_WRITE_TIMEOUT=30s         # Response write timeout

# Database configuration
DB_WRITE_HOST=localhost          # Primary database host
DB_READ_HOSTS=localhost          # Read replica hosts (comma-separated)
DB_PORT=5432                     # PostgreSQL port
DB_NAME=myapp_db                 # Database name
DB_USER=postgres                 # Database user
DB_PASSWORD=password             # Database password
DB_SSL_MODE=disable              # SSL mode (dev) or require (prod)
DB_MAX_CONNS=25                  # Maximum connections per pool
```

### ✅ DO: Use Type-Specific Environment Parsing

```go
// ✅ BAIK - Type conversion saat load
port := strconv.Atoi(GetEnv("DB_PORT"))  // Convert string ke int
timeout := parseDuration(GetEnv("SERVER_READ_TIMEOUT"))  // Parse duration

// ❌ BURUK - Type conversion di handler
port := strconv.Atoi(r.FormValue("port"))  // Parsing di request handler
```

---

## Summary

Konfigurasi di dim:
- **Environment-based** - Berbeda per environment
- **No hardcoding** - Semua dari env vars
- **Type-safe** - Parsing dan validasi saat load
- **Well-documented** - Clear defaults dan examples

Lihat [Server Setup](01-getting-started.md) untuk quick start dengan configuration.

---

**Lihat Juga**:
- [Database](06-database.md) - Database configuration detail
- [Autentikasi](05-authentication.md) - JWT configuration
- [Middleware](04-middleware.md) - CORS/CSRF configuration
