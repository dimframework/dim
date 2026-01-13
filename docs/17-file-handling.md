# File Handling dan Upload

Dokumentasi lengkap tentang file handling, MIME type detection, dan file upload dengan keamanan di framework dim menggunakan abstraksi storage dari goreus.

## Table of Contents

- [Overview](#overview)
- [MIME Type Detection](#mime-type-detection)
- [File Upload](#file-upload)
- [Goreus Storage Integration](#goreus-storage-integration)
- [Security Features](#security-features)
- [Best Practices](#best-practices)
- [Examples](#examples)

## Overview

Framework dim menyediakan utilitas lengkap untuk:
1. **Deteksi MIME Type** - Identifikasi file type dengan 122+ MIME types
2. **File Upload** - Upload dengan validasi, sanitisasi, dan concurrent processing
3. **File Serving** - Melayani file dengan proper headers (download atau inline)
4. **Security** - Sanitisasi, validasi ekstensi, dan content-type verification
5. **Storage Abstraction** - Integrasi dengan goreus untuk multiple backend support

### Fitur Utama

- ✅ Support 122+ MIME types (images, documents, audio, video, archives, etc.)
- ✅ Custom MIME type registration (thread-safe)
- ✅ File upload dengan sequential dan concurrent processing
- ✅ Per-file error reporting dengan detailed messages
- ✅ Security: path sanitization, filename sanitization, content-type validation
- ✅ Structured logging dengan slog integration
- ✅ Configurable limits (file size, file count, workers)
- ✅ Goreus storage abstraction (S3, R2, Local, Null backends)

---

## MIME Type Detection

### DetectContentType()

Mendeteksi MIME type file berdasarkan extension dengan comprehensive built-in mapping.

#### Signature

```go
func DetectContentType(filename string) string
```

#### Parameter

- `filename` (string) - Nama file (dapat include path, extension di-extract otomatis)

#### Return

- MIME type string dalam format `type/subtype` (misal: `image/jpeg`, `text/plain`)
- `application/octet-stream` sebagai fallback untuk extension yang tidak diketahui

#### Kategori File yang Didukung

- **Images**: JPEG, PNG, GIF, WebP, SVG, ICO, TIFF, BMP
- **Documents**: PDF, Word (DOC/DOCX), Excel (XLS/XLSX), PowerPoint (PPT/PPTX), ODF formats
- **Text & Code**: TXT, CSV, HTML, CSS, JavaScript, JSON, XML, YAML, Markdown, TypeScript, Rust, Go, Python, etc.
- **Archives**: ZIP, RAR, 7Z, TAR, GZIP, BZIP2
- **Video**: MP4, AVI, MOV, WMV, WebM, MPEG, MKV, FLV
- **Audio**: MP3, WAV, OGG, FLAC, M4A, AAC
- **Web Fonts**: WOFF, WOFF2, TTF, OTF, EOT
- **Others**: EPUB, TORRENT, dan lainnya (122+ total)

#### Contoh

```go
// Deteksi dari nama file
mimeType := dim.DetectContentType("photo.jpg")
// Output: "image/jpeg"

mimeType := dim.DetectContentType("document.pdf")
// Output: "application/pdf"

mimeType := dim.DetectContentType("/path/to/report.xlsx")
// Output: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

// Unknown extension
mimeType := dim.DetectContentType("file.unknown")
// Output: "application/octet-stream"
```

### RegisterMIMEType()

Mendaftarkan custom MIME type untuk file extension (thread-safe).

#### Signature

```go
func RegisterMIMEType(ext, mimeType string)
```

#### Parameter

- `ext` (string) - File extension termasuk dot (misal: `.custom`, `.myformat`)
- `mimeType` (string) - MIME type string (misal: `application/x-custom`)

#### Fitur

- Custom MIME types dicek terlebih dahulu (dapat override defaults)
- Thread-safe (RWMutex internal)
- Dapat dipanggil concurrently dari multiple goroutines

#### Contoh

```go
// Register custom types
dim.RegisterMIMEType(".webmanifest", "application/manifest+json")
dim.RegisterMIMEType(".wasm", "application/wasm")
dim.RegisterMIMEType(".protobuf", "application/protobuf")

// Sekarang DetectContentType mengenali custom types
mimeType := dim.DetectContentType("app.webmanifest")
// Output: "application/manifest+json"

// Override default
dim.RegisterMIMEType(".json", "application/json; charset=utf-8")
```

---

## File Upload

### Konsep Dasar

File upload di dim menggunakan **functional options pattern** dan mendukung dua mode:
1. **Sequential** - Proses file satu per satu (predictable, single-threaded)
2. **Concurrent** - Proses multiple files dengan worker pool (fast, parallel)

### UploadFiles()

Upload multiple files dengan validasi, sanitisasi, dan optional concurrent processing.

#### Signature

```go
func UploadFiles(
    ctx context.Context,
    disk storage.Storage,
    files []*multipart.FileHeader,
    opts ...UploadOption,
) ([]string, error)
```

#### Parameter

- `ctx` (context.Context) - Context untuk cancellation dan deadlines
- `disk` (storage.Storage) - Storage backend dari goreus
- `files` ([]*multipart.FileHeader) - Slice file dari multipart form
- `opts` (...UploadOption) - Optional configuration (With* functions)

#### Return

- `[]string` - Paths file yang berhasil di-upload (canonical keys dari storage)
- `error` - Error pertama yang ditemui (lihat Error handling di bawah)

#### Konfigurasi (Options)

Semua options adalah optional dan menggunakan functional options pattern:

```go
WithPath(path string)              // Set direktori upload (default: "/uploads")
WithAllowedExts(exts ...string)    // Set allowed extensions (default: [".jpg", ".jpeg", ".png", ".pdf"])
WithMaxFileSize(size uint64)       // Max file size dalam bytes (default: 10MB)
WithMaxFiles(max uint8)            // Max files untuk upload sekaligus (default: 10)
WithConcurrent(enabled bool)       // Enable concurrent processing (default: false)
WithMaxWorkers(max int)            // Jumlah workers (default: 10, jika concurrent=true)
WithLogger(logger *slog.Logger)    // Optional logger untuk debugging
```

#### Validasi yang Dilakukan

1. **File Count** - Checked against maxFiles limit
2. **Filename** - Sanitized untuk security
3. **File Size** - Checked against maxFileSize limit
4. **Extension** - Validated against allowedExts
5. **Content-Type** - Detected dan divalidasi
6. **MIME Type Spoofing** - Content-type harus match extension

#### Error Handling

```go
result, err := dim.UploadFiles(ctx, disk, files, opts...)
if err != nil {
    // err berisi error pertama
    fmt.Printf("Upload failed: %v\n", err)
}

// Canonical paths (storage keys) berhasil di-upload
for _, path := range result {
    // Simpan paths ini ke database untuk referensi later
    fmt.Printf("Uploaded: %s\n", path)
}
```

#### Contoh - Sequential Upload

```go
package main

import (
    "context"
    "log"
    "log/slog"
    "net/http"

    "github.com/atfromhome/goreus/pkg/storage"
    "yourmodule/dim"
)

func uploadHandler(w http.ResponseWriter, r *http.Request) {
    // Parse form (max 100MB)
    err := r.ParseMultipartForm(100 << 20)
    if err != nil {
        dim.BadRequest(w, "Failed to parse form", nil)
        return
    }
    defer r.MultipartForm.RemoveAll()

    // Get files dari form
    files := r.MultipartForm.File["files"]
    if len(files) == 0 {
        dim.BadRequest(w, "No files provided", nil)
        return
    }

    // Initialize storage (goreus) dari environment config
    disk, err := storage.New()
    if err != nil {
        dim.InternalServerError(w, "Storage initialization failed")
        return
    }
    defer disk.Close()

    // Upload dengan sequential processing (default)
    paths, err := dim.UploadFiles(
        r.Context(),
        disk,
        files,
        dim.WithPath("/documents"),
        dim.WithAllowedExts(".pdf", ".docx", ".xlsx"),
        dim.WithMaxFileSize(50 << 20), // 50MB
        dim.WithMaxFiles(10),
        dim.WithLogger(slog.Default()),
    )

    if err != nil {
        dim.BadRequest(w, err.Error(), nil)
        return
    }

    // Success - return canonical paths
    dim.Created(w, map[string]interface{}{
        "message": "Documents uploaded successfully",
        "count":   len(paths),
        "paths":   paths,
    })
}
```

#### Contoh - Concurrent Upload

```go
// Upload dengan concurrent processing
paths, err := dim.UploadFiles(
    r.Context(),
    disk,
    files,
    dim.WithPath("/images"),
    dim.WithAllowedExts(".jpg", ".jpeg", ".png", ".webp"),
    dim.WithMaxFileSize(20 << 20), // 20MB per file
    dim.WithMaxFiles(50),
    dim.WithConcurrent(true),      // Enable concurrent
    dim.WithMaxWorkers(5),         // 5 workers parallel
    dim.WithLogger(logger),
)

if err != nil {
    log.Printf("Some files failed: %v", err)
    // Partial upload - some files succeeded, some failed
}
```

---

## Goreus Storage Integration

### Storage Architecture (Goreus)

Framework dim menggunakan goreus `pkg/storage` untuk abstraksi file backend. Goreus adalah storage abstraction layer minimal yang fokus pada satu tanggung jawab: upload dan retrieve binary content.

#### Konsep Inti

- **Canonical Paths** - Paths yang dikembalikan dari upload adalah unique identifiers/keys
- **Backend-Agnostic** - Sama logic bekerja dengan semua backend
- **Private by Default** - Files adalah private kecuali explicitly configured public
- **No Business Logic** - Pure storage interface tanpa temporary URLs atau public links

#### Storage Interface (Goreus)

```go
type Storage interface {
    // Upload file dari []byte
    Upload(ctx context.Context, path string, data []byte, opts ...Option) error

    // Upload file dari io.Reader (streaming untuk large files)
    UploadStream(ctx context.Context, path string, reader io.Reader, opts ...Option) error

    // Get file sebagai []byte (small-to-medium files)
    Get(ctx context.Context, path string) ([]byte, error)

    // Get file sebagai io.ReadCloser (streaming untuk large files)
    GetStream(ctx context.Context, path string) (io.ReadCloser, error)

    // Delete file
    Delete(ctx context.Context, path string) error

    // Check file exists
    Has(ctx context.Context, path string) (bool, error)

    // Close storage connection
    Close() error
}
```

#### Upload Options (Goreus)

```go
// WithPublic - override default visibility per upload
// WithContentType - set metadata (driver-dependent, optional)
```

### Supported Backends

#### 1. AWS S3 (Primary)

```bash
# Environment configuration
export STORAGE_DRIVER=s3
export S3_BUCKET=my-bucket
export AWS_REGION=us-east-1
export AWS_ACCESS_KEY_ID=your-key
export AWS_SECRET_ACCESS_KEY=your-secret
export STORAGE_PUBLIC=false  # Files private by default
```

#### 2. Cloudflare R2 (S3-Compatible)

```bash
# Environment configuration
export STORAGE_DRIVER=r2
export S3_BUCKET=my-bucket
export S3_ENDPOINT=https://<account-id>.r2.cloudflarestorage.com
export S3_FORCE_PATH_STYLE=true  # Required for R2
export AWS_ACCESS_KEY_ID=your-token
export AWS_SECRET_ACCESS_KEY=your-secret
export STORAGE_PUBLIC=false
```

#### 3. Local Filesystem (Development)

```bash
# Environment configuration
export STORAGE_DRIVER=local
export LOCAL_BASE_PATH=./storage  # Default: ./storage
export STORAGE_PUBLIC=false
```

#### 4. Null/Mock (Testing)

```bash
# Environment configuration
export STORAGE_DRIVER=null  # Default, in-memory tanpa persistence
export STORAGE_PUBLIC=false
```

### Initialization

```go
import "github.com/atfromhome/goreus/pkg/storage"

// Initialize dari environment variables
disk, err := storage.New()
if err != nil {
    log.Fatal("Storage init failed:", err)
}
defer disk.Close()

// Backend dipilih otomatis berdasarkan STORAGE_DRIVER env var
```

### Contoh - Multiple Backends

```go
// Local development
os.Setenv("STORAGE_DRIVER", "local")
os.Setenv("LOCAL_BASE_PATH", "./uploads")
disk, _ := storage.New()

// Production AWS S3
os.Setenv("STORAGE_DRIVER", "s3")
os.Setenv("S3_BUCKET", "prod-bucket")
disk, _ := storage.New()

// Same code, different backend!
paths, _ := dim.UploadFiles(ctx, disk, files, opts...)
```

---

## Security Features

### Sanitisasi dan Validasi

Semua file upload melalui validasi keamanan berlapis:

#### 1. Filename Sanitization

```go
// Remove path separators, .., dan whitespace
// "/etc/passwd" -> "passwd"
// "../../evil" -> "evil"
// Input: /path/to/file.pdf
// Output: file.pdf
```

#### 2. Path Sanitization

```go
// Normalize path dan prevent directory traversal
// "uploads/../../../etc" -> "/uploads"
// "../uploads" -> "/uploads"
// Input: uploads/./docs -> Output: /uploads/docs
```

#### 3. Extension Validation

```go
// Validate against allowed extensions
dim.UploadFiles(
    ctx, disk, files,
    dim.WithAllowedExts(".pdf", ".docx", ".xlsx"),
    // .exe, .bat, .sh, etc. akan di-reject
)
```

#### 4. Content-Type Validation

```go
// Detect MIME type dari file content
// Validate content-type match extension
// Prevent MIME type spoofing
// Blacklist executables (.exe, .bat, .cmd, .sh, .jar, dll)
```

#### 5. Blocked Extensions

Framework mem-blacklist executable dan system files:
- `.exe`, `.bat`, `.cmd`, `.com`, `.scr`, `.vbs`
- `.jar`, `.app`, `.sh`, `.bash`, `.bin`
- `.dmg`, `.deb`, `.rpm`

---

## File Serving

### ServeFile() - Download

Melayani file sebagai attachment (browser will download).

#### Signature

```go
func ServeFile(
    w http.ResponseWriter,
    filename string,
    filePath string,
    statusCode int,
) error
```

#### Contoh

```go
err := dim.ServeFile(w, "document.pdf", "/var/uploads/docs/file.pdf", http.StatusOK)
if err != nil {
    http.Error(w, "File not found", http.StatusNotFound)
}
// Browser akan download file sebagai "document.pdf"
```

### ServeFileInline() - Display

Melayani file untuk inline display (browser will display jika supported).

#### Signature

```go
func ServeFileInline(
    w http.ResponseWriter,
    filename string,
    filePath string,
    statusCode int,
) error
```

#### Contoh

```go
// Display image di browser
err := dim.ServeFileInline(w, "photo.jpg", "/var/uploads/images/photo.jpg", http.StatusOK)

// Display PDF di browser
err := dim.ServeFileInline(w, "report.pdf", "/var/uploads/reports/report.pdf", http.StatusOK)

// Display video
err := dim.ServeFileInline(w, "video.mp4", "/var/uploads/videos/video.mp4", http.StatusOK)
```

---

## Best Practices

### 1. Always Validate User Input

```go
// ✅ Good
dim.UploadFiles(
    ctx, disk, files,
    dim.WithMaxFileSize(50 << 20),    // Reasonable limit
    dim.WithAllowedExts(".pdf", ".docx"),
    dim.WithMaxFiles(10),              // Prevent DOS
)

// ❌ Bad - No limits
dim.UploadFiles(ctx, disk, files)
```

### 2. Persist Canonical Paths

```go
// ✅ Good - Simpan paths ke database
paths, err := dim.UploadFiles(ctx, disk, files, opts...)
for _, path := range paths {
    // Save path to database for later retrieval
    db.SaveUploadedFile(userID, path)
}

// ❌ Bad - Hanya simpan filename
for _, header := range files {
    db.SaveUploadedFile(userID, header.Filename)
}
```

### 3. Use Concurrent Upload untuk Multiple Files

```go
// ✅ Good untuk multiple files
dim.UploadFiles(
    ctx, disk, files,
    dim.WithConcurrent(true),
    dim.WithMaxWorkers(10),
)

// ❌ Slow untuk banyak files
dim.UploadFiles(ctx, disk, files)  // Sequential
```

### 4. Add Structured Logging

```go
// ✅ Good
logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
dim.UploadFiles(
    ctx, disk, files,
    dim.WithLogger(logger),
)

// ❌ Silent failures
dim.UploadFiles(ctx, disk, files)
```

### 5. Implement Proper Error Handling

```go
// ✅ Good - Handle partial uploads
paths, err := dim.UploadFiles(ctx, disk, files, opts...)
if err != nil {
    log.Printf("Upload error: %v", err)
}
log.Printf("Uploaded: %d files", len(paths))

// ❌ Ignore errors
paths, _ := dim.UploadFiles(ctx, disk, files)
```

### 6. Context Cancellation

```go
// ✅ Good - Respect context cancellation
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

paths, err := dim.UploadFiles(ctx, disk, files, opts...)

// ❌ No timeout protection
dim.UploadFiles(context.Background(), disk, files, opts...)
```

### 7. Handle Goreus Initialization

```go
// ✅ Good - Initialize dengan error handling
disk, err := storage.New()
if err != nil {
    log.Fatal("Storage init failed:", err)
}
defer disk.Close()

// ❌ Ignore init errors
disk, _ := storage.New()
```

---

## Examples

### Complete Upload Handler with Goreus

```go
package handlers

import (
    "context"
    "log/slog"
    "net/http"
    "time"

    "github.com/atfromhome/goreus/pkg/storage"
    "yourmodule/dim"
)

func UploadDocuments(w http.ResponseWriter, r *http.Request) {
    // Parse form
    if err := r.ParseMultipartForm(100 << 20); err != nil {
        dim.BadRequest(w, "Invalid form data", nil)
        return
    }
    defer r.MultipartForm.RemoveAll()

    // Get files
    files := r.MultipartForm.File["documents"]
    if len(files) == 0 {
        dim.BadRequest(w, "No files provided", nil)
        return
    }

    // Initialize storage dari environment
    disk, err := storage.New()
    if err != nil {
        dim.InternalServerError(w, "Storage initialization failed")
        return
    }
    defer disk.Close()

    // Set timeout
    ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
    defer cancel()

    // Upload
    paths, err := dim.UploadFiles(
        ctx,
        disk,
        files,
        dim.WithPath("/documents"),
        dim.WithAllowedExts(".pdf", ".docx", ".xlsx", ".pptx"),
        dim.WithMaxFileSize(50 << 20),  // 50MB
        dim.WithMaxFiles(20),
        dim.WithConcurrent(true),
        dim.WithMaxWorkers(5),
        dim.WithLogger(slog.Default()),
    )

    if err != nil {
        dim.BadRequest(w, err.Error(), nil)
        return
    }

    // Success - Return canonical paths
    dim.Created(w, map[string]interface{}{
        "message": "Documents uploaded successfully",
        "count":   len(paths),
        "paths":   paths,  // Canonical storage keys
    })
}

// Handler untuk retrieve file
func GetDocument(w http.ResponseWriter, r *http.Request) {
    canonicalPath := r.URL.Query().Get("path")
    if canonicalPath == "" {
        dim.BadRequest(w, "Path parameter required", nil)
        return
    }

    // Initialize storage
    disk, err := storage.New()
    if err != nil {
        dim.InternalServerError(w, "Storage error")
        return
    }
    defer disk.Close()

    // Get file content
    content, err := disk.Get(r.Context(), canonicalPath)
    if err != nil {
        dim.NotFound(w, "File not found")
        return
    }

    // Detect content type dan serve
    mimeType := dim.DetectContentType(canonicalPath)
    w.Header().Set("Content-Type", mimeType)
    w.Header().Set("Content-Disposition", "attachment")
    w.Write(content)
}
```

### Image Upload dengan Concurrent Processing

```go
func UploadImages(w http.ResponseWriter, r *http.Request) {
    // Parse form
    if err := r.ParseMultipartForm(50 << 20); err != nil {
        dim.BadRequest(w, "Invalid form data", nil)
        return
    }
    defer r.MultipartForm.RemoveAll()

    files := r.MultipartForm.File["images"]
    if len(files) == 0 {
        dim.BadRequest(w, "No images provided", nil)
        return
    }

    disk, err := storage.New()
    if err != nil {
        dim.InternalServerError(w, "Storage error")
        return
    }
    defer disk.Close()

    ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
    defer cancel()

    // Upload images dengan concurrent processing
    paths, err := dim.UploadFiles(
        ctx,
        disk,
        files,
        dim.WithPath("/images"),
        dim.WithAllowedExts(".jpg", ".jpeg", ".png", ".webp", ".gif"),
        dim.WithMaxFileSize(20 << 20),  // 20MB per image
        dim.WithMaxFiles(50),
        dim.WithConcurrent(true),
        dim.WithMaxWorkers(10),
        dim.WithLogger(slog.Default()),
    )

    if err != nil {
        dim.BadRequest(w, err.Error(), nil)
        return
    }

    // Return canonical paths
    dim.OK(w, map[string]interface{}{
        "message": "Images uploaded successfully",
        "count":   len(paths),
        "paths":   paths,
    })
}
```

---

## Environment Configuration

### Local Development

```bash
export STORAGE_DRIVER=local
export LOCAL_BASE_PATH=./storage
export STORAGE_PUBLIC=false
```

### AWS S3 Production

```bash
export STORAGE_DRIVER=s3
export S3_BUCKET=my-production-bucket
export AWS_REGION=us-east-1
export AWS_ACCESS_KEY_ID=${AWS_KEY}
export AWS_SECRET_ACCESS_KEY=${AWS_SECRET}
export STORAGE_PUBLIC=false
```

### Cloudflare R2 Production

```bash
export STORAGE_DRIVER=r2
export S3_BUCKET=my-bucket
export S3_ENDPOINT=https://${ACCOUNT_ID}.r2.cloudflarestorage.com
export S3_FORCE_PATH_STYLE=true
export AWS_ACCESS_KEY_ID=${R2_TOKEN_ID}
export AWS_SECRET_ACCESS_KEY=${R2_TOKEN_SECRET}
export STORAGE_PUBLIC=false
```

---

## Summary

| Fitur | Description |
|-------|-------------|
| **MIME Types** | 122+ built-in, custom registration, thread-safe |
| **Upload** | Sequential & concurrent, configurable limits, detailed errors |
| **Security** | Filename/path sanitization, extension validation, MIME spoofing prevention |
| **Storage** | Goreus abstraction - Local, S3, R2, Null backends |
| **Logging** | Structured logging dengan slog integration |
| **Error Handling** | Per-file error tracking dengan detailed messages |
| **Serving** | Download (attachment) atau display (inline) dengan proper headers |
| **Backends** | Seamless migration antara Local, AWS S3, Cloudflare R2 |

---

## Related Documentation

- [Security](./14-security.md) - Security best practices
- [Error Handling](./08-error-handling.md) - Error handling patterns
- [Structured Logging](./12-structured-logging.md) - Logging with slog
- [Configuration](./07-configuration.md) - Configuration management
- [Goreus Repository](https://github.com/atfromhome/goreus) - Storage abstraction library
