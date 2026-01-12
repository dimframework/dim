# Production Deployment Guide untuk Framework dim

Pelajari cara mendeploy aplikasi framework dim ke production dengan aman dan efisien.

## Daftar Isi

- [Pre-deployment Checklist](#pre-deployment-checklist)
- [Environment Configuration](#environment-configuration)
- [Database Preparation](#database-preparation)
- [Build & Compilation](#build--compilation)
- [Server Setup](#server-setup)
- [Reverse Proxy Configuration](#reverse-proxy-configuration)
- [SSL/TLS Setup](#ssltls-setup)
- [Monitoring & Logging](#monitoring--logging)
- [Backup & Recovery](#backup--recovery)
- [Deployment Strategies](#deployment-strategies)
- [Performance Tuning](#performance-tuning)
- [Praktik Terbaik](#praktik-terbaik)

---

## Pre-deployment Checklist

Sebelum mendeploy ke production, pastikan:

### Kode & Build

- ✅ Semua tests passing (`go test ./...`)
- ✅ Tidak ada race conditions (`go test -race ./...`)
- ✅ Code coverage acceptable (minimal 70%)
- ✅ Dependencies updated dan secure (`go mod tidy`)
- ✅ Build berhasil di production environment
- ✅ No debug logging di production code
- ✅ Error handling lengkap di semua endpoints

### Konfigurasi

- ✅ Environment variables di-setup di server
- ✅ Database credentials aman (jangan hardcode)
- ✅ JWT_SECRET yang kuat dan random
- ✅ CORS origins yang tepat (jangan wildcard)
- ✅ Rate limiting configured
- ✅ Timeouts configured appropriately

### Database

- ✅ Database sudah created di production
- ✅ Migrations sudah di-run
- ✅ Database backups configured
- ✅ Connection pooling configured
- ✅ Indexes created untuk query yang sering

### Keamanan

- ✅ HTTPS/TLS configured
- ✅ Security headers set
- ✅ CSRF protection enabled
- ✅ Input validation lengkap
- ✅ No sensitive data di logs
- ✅ SQL injection protection (parameterized queries)

### Monitoring

- ✅ Logging system configured
- ✅ Error alerting configured
- ✅ Performance monitoring setup
- ✅ Health check endpoint working
- ✅ Uptime monitoring configured

---

## Environment Configuration

### Production Environment Variables

Buat file `.env.production` untuk production environment:

```bash
# Server
SERVER_PORT=8080
SERVER_READ_TIMEOUT=30s
SERVER_WRITE_TIMEOUT=30s

# Database
DB_WRITE_HOST=prod-db-primary.example.com
DB_READ_HOSTS=prod-db-replica1.example.com,prod-db-replica2.example.com
DB_PORT=5432
DB_NAME=dim_production
DB_USER=app_user
DB_PASSWORD=<strong-random-password>
DB_MAX_CONNS=25
DB_SSL_MODE=require

# JWT
JWT_SECRET=<very-strong-random-secret-min-32-chars>
JWT_ACCESS_TOKEN_EXPIRY=15m
JWT_REFRESH_TOKEN_EXPIRY=168h

# CORS
CORS_ALLOWED_ORIGINS=https://app.example.com,https://www.example.com
CORS_ALLOWED_METHODS=GET,POST,PUT,DELETE,PATCH,OPTIONS
CORS_ALLOWED_HEADERS=Content-Type,Authorization,X-CSRF-Token
CORS_ALLOW_CREDENTIALS=true
CORS_MAX_AGE=3600

# CSRF
CSRF_ENABLED=true
CSRF_EXEMPT_PATHS=/webhooks/stripe,/webhooks/github
CSRF_TOKEN_LENGTH=32
CSRF_COOKIE_NAME=csrf_token

# Rate Limiting
RATE_LIMIT_ENABLED=true
RATE_LIMIT_PER_IP=100
RATE_LIMIT_PER_USER=200
RATE_LIMIT_RESET_PERIOD=1h

# Email
EMAIL_FROM=noreply@example.com
```

### Generate Strong Secrets

```bash
# Generate JWT_SECRET (32 bytes random)
openssl rand -base64 32

# Generate database password
openssl rand -base64 24
```

### Load Environment Variables

```bash
# Load from .env.production
export $(cat .env.production | xargs)

# Atau gunakan script
source .env.production
```

---

(Sisa dokumen tidak diubah dan dihilangkan untuk keringkasan)