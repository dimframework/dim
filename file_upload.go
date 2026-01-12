// Package dim menyediakan utilitas upload file dengan fitur keamanan komprehensif.
// Mendukung pemrosesan sequential dan concurrent dengan validasi, sanitisasi,
// dan deteksi content-type.
package dim

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/atfromhome/goreus/pkg/storage"
)

// UploadConfig menyimpan konfigurasi untuk upload file.
//
// Fields:
//   - path: Path direktori untuk menyimpan file yang di-upload (sanitisasi otomatis)
//   - allowedExts: Daftar ekstensi file yang diizinkan (misalnya ".jpg", ".pdf")
//   - maxFileSize: Ukuran file maksimal dalam bytes (0 = tanpa batas)
//   - maxFiles: Jumlah maksimal file untuk di-upload sekaligus
//   - concurrent: Aktifkan pemrosesan concurrent (false = sequential)
//   - maxWorkers: Jumlah concurrent workers (jika concurrent = true)
//   - logger: Logger opsional untuk debugging (bisa nil)
type UploadConfig struct {
	path        string
	allowedExts []string
	maxFileSize uint64
	maxFiles    uint8
	concurrent  bool
	maxWorkers  int
	logger      *slog.Logger
}

// UploadResult berisi hasil dari operasi upload file.
//
// Fields:
//   - Paths: Daftar path file yang berhasil di-upload
//   - Errors: Map filename -> error untuk upload yang gagal (file -> alasan error)
type UploadResult struct {
	Paths  []string
	Errors map[string]error
}

// UploadOption adalah functional option untuk mengkonfigurasi UploadConfig.
type UploadOption func(*UploadConfig)

// WithPath mengatur path direktori upload.
//
// Contoh:
//
//	WithPath("/uploads/files")
//	WithPath("uploads") // otomatis diprefiks dengan /
//
// Path otomatis disanitisasi terhadap serangan directory traversal.
func WithPath(path string) UploadOption {
	return func(c *UploadConfig) {
		c.path = sanitizePath(path)
	}
}

// WithAllowedExts mengatur daftar ekstensi file yang diizinkan.
//
// Contoh:
//
//	WithAllowedExts(".jpg", ".jpeg", ".png", ".gif")
//
// Ekstensi tidak case-sensitive. Jika tidak diatur, ekstensi default digunakan.
func WithAllowedExts(exts ...string) UploadOption {
	return func(c *UploadConfig) {
		c.allowedExts = exts
	}
}

// WithMaxFileSize mengatur ukuran file maksimal dalam bytes.
//
// Contoh:
//
//	WithMaxFileSize(10 << 20) // 10 MB
//	WithMaxFileSize(100 << 20) // 100 MB
//
// File yang melebihi ukuran ini akan ditolak. Hanya diatur jika size > 0.
func WithMaxFileSize(size uint64) UploadOption {
	return func(c *UploadConfig) {
		if size > 0 {
			c.maxFileSize = size
		}
	}
}

// WithMaxFiles mengatur jumlah maksimal file untuk di-upload sekaligus.
//
// Contoh:
//
//	WithMaxFiles(5)   // Maks 5 file per upload
//	WithMaxFiles(10)  // Maks 10 file per upload
//
// Hanya diatur jika max > 0. Default adalah 10.
func WithMaxFiles(max uint8) UploadOption {
	return func(c *UploadConfig) {
		if max > 0 {
			c.maxFiles = max
		}
	}
}

// WithConcurrent mengaktifkan pemrosesan file concurrent.
//
// Saat diaktifkan, menggunakan worker pool untuk upload paralel. Saat dinonaktifkan,
// memproses file secara sequential (lebih lambat namun lebih dapat diprediksi).
//
// Contoh:
//
//	WithConcurrent(true)   // Gunakan worker pool
//	WithConcurrent(false)  // Pemrosesan sequential
func WithConcurrent(enabled bool) UploadOption {
	return func(c *UploadConfig) {
		c.concurrent = enabled
	}
}

// WithMaxWorkers mengatur jumlah concurrent workers.
//
// Hanya berlaku jika pemrosesan concurrent diaktifkan.
// Default adalah 10. Hanya diatur jika max > 0.
//
// Contoh:
//
//	WithMaxWorkers(5)   // 5 parallel upload workers
//	WithMaxWorkers(20)  // 20 parallel upload workers
func WithMaxWorkers(max int) UploadOption {
	return func(c *UploadConfig) {
		if max > 0 {
			c.maxWorkers = max
		}
	}
}

// WithLogger mengatur logger opsional untuk debugging operasi upload.
//
// Jika disediakan, logs akan mencakup progres upload, error, dan detail validasi.
//
// Contoh:
//
//	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
//	WithLogger(logger)
func WithLogger(logger *slog.Logger) UploadOption {
	return func(c *UploadConfig) {
		if logger != nil {
			c.logger = logger
		}
	}
}

// DefaultConfig mengembalikan UploadConfig baru dengan nilai default yang masuk akal.
//
// Nilai default:
//   - path: "/uploads"
//   - allowedExts: [".jpg", ".jpeg", ".png", ".pdf"]
//   - maxFileSize: 10 MB (10 << 20 bytes)
//   - maxFiles: 10
//   - concurrent: false (pemrosesan sequential)
//   - maxWorkers: 10
//   - logger: nil (tanpa logging)
//
// Default ini dapat ditimpa menggunakan fungsi opsi With*.
func DefaultConfig() *UploadConfig {
	return &UploadConfig{
		path:        "/uploads",
		allowedExts: []string{".jpg", ".jpeg", ".png", ".pdf"},
		maxFileSize: 10 << 20,
		maxFiles:    10,
		concurrent:  false,
		maxWorkers:  10,
	}
}

// UploadFiles meng-upload multiple file dengan validasi dan pemrosesan concurrent opsional.
//
// Parameter:
//   - ctx: Context untuk cancellation dan deadlines
//   - disk: Storage backend (mengimplementasikan interface storage.Storage)
//   - files: Slice multipart file headers untuk di-upload
//   - opts: Konfigurasi opsional (WithPath, WithMaxFileSize, dll.)
//
// Return:
//   - []string: Path file yang berhasil di-upload
//   - error: Error pertama yang ditemukan (lihat field Errors untuk complete error map)
//
// Validasi:
//   - Jumlah file dicek terhadap batas maxFiles
//   - Ekstensi file divalidasi terhadap allowedExts
//   - Ukuran file dicek terhadap batas maxFileSize
//   - Content-type divalidasi untuk cocok dengan ekstensi
//   - Nama file dan path disanitisasi terhadap serangan traversal
//
// Saat Error:
//   - File yang berhasil di-upload dibersihkan dari storage
//   - Mengembalikan error dengan jumlah file yang gagal
//   - Gunakan pesan error untuk menentukan file mana yang gagal
//
// Contoh (Sequential):
//
//	paths, err := dim.UploadFiles(
//	    ctx, disk, formFiles,
//	    dim.WithPath("/uploads"),
//	    dim.WithMaxFileSize(10 << 20),
//	    dim.WithAllowedExts(".jpg", ".png", ".pdf"),
//	)
//	if err != nil {
//	    log.Printf("Upload failed: %v", err)
//	    return
//	}
//
// Contoh (Concurrent):
//
//	paths, err := dim.UploadFiles(
//	    ctx, disk, formFiles,
//	    dim.WithConcurrent(true),
//	    dim.WithMaxWorkers(5),
//	    dim.WithLogger(logger),
//	)
func UploadFiles(
	ctx context.Context,
	disk storage.Storage,
	files []*multipart.FileHeader,
	opts ...UploadOption,
) ([]string, error) {
	config := DefaultConfig()
	for _, opt := range opts {
		opt(config)
	}

	if config.maxFiles > 0 && len(files) > int(config.maxFiles) {
		if config.logger != nil {
			config.logger.Error("upload rejected", "reason", "too many files", "count", len(files), "max", config.maxFiles)
		}
		return nil, fmt.Errorf("too many files: got %d, max %d", len(files), config.maxFiles)
	}

	allowedExts := make(map[string]bool)
	for _, ext := range config.allowedExts {
		allowedExts[strings.ToLower(ext)] = true
	}

	if config.concurrent {
		return uploadConcurrent(ctx, disk, files, config, allowedExts)
	}
	return uploadSequential(ctx, disk, files, config, allowedExts)
}

// uploadSequential memproses file secara sequential (satu per satu).
// Lebih sederhana daripada concurrent namun lebih lambat untuk batch besar.
func uploadSequential(
	ctx context.Context,
	disk storage.Storage,
	fileHeaders []*multipart.FileHeader,
	config *UploadConfig,
	allowedExts map[string]bool,
) ([]string, error) {
	var uploadedPaths []string

	for i, fileHeader := range fileHeaders {
		if ctx.Err() != nil {
			cleanupFiles(ctx, disk, uploadedPaths)
			if config.logger != nil {
				config.logger.Error("sequential upload cancelled",
					"processed_count", i,
					"total_files", len(fileHeaders))
			}
			return nil, ctx.Err()
		}

		if config.logger != nil {
			config.logger.Debug("uploading file",
				"index", i+1,
				"total", len(fileHeaders),
				"filename", fileHeader.Filename)
		}

		path, err := processFile(ctx, disk, fileHeader, config, allowedExts)
		if err != nil {
			cleanupFiles(ctx, disk, uploadedPaths)
			if config.logger != nil {
				config.logger.Error("sequential upload failed",
					"filename", fileHeader.Filename,
					"error", err.Error(),
					"processed_count", i+1,
					"total_files", len(fileHeaders))
			}
			return nil, fmt.Errorf("failed to upload file '%s': %w", fileHeader.Filename, err)
		}
		uploadedPaths = append(uploadedPaths, path)
	}

	if config.logger != nil {
		config.logger.Info("sequential upload successful",
			"file_count", len(uploadedPaths))
	}

	return uploadedPaths, nil
}

type uploadResult struct {
	path string
	err  error
}

// uploadJob represents a file upload job for concurrent processing
type uploadJob struct {
	index      int    // Position in original file list
	filename   string // Original filename for error reporting
	fileHeader *multipart.FileHeader
}

// uploadResultJob includes index for proper result ordering
type uploadResultJob struct {
	index    int
	path     string
	err      error
	filename string
}

// uploadConcurrent memproses file secara concurrent dengan synchronization yang tepat.
// Menggunakan result map untuk mempertahankan urutan dan melacak semua error.
func uploadConcurrent(
	ctx context.Context,
	disk storage.Storage,
	fileHeaders []*multipart.FileHeader,
	config *UploadConfig,
	allowedExts map[string]bool,
) ([]string, error) {
	numWorkers := config.maxWorkers
	if numWorkers <= 0 {
		numWorkers = 10
	}
	if numWorkers > len(fileHeaders) {
		numWorkers = len(fileHeaders)
	}

	jobs := make(chan uploadJob, len(fileHeaders))
	results := make(chan uploadResultJob, len(fileHeaders))

	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				if ctx.Err() != nil {
					results <- uploadResultJob{
						index:    job.index,
						filename: job.filename,
						err:      ctx.Err(),
					}
					continue
				}

				path, err := processFile(ctx, disk, job.fileHeader, config, allowedExts)

				if err != nil && config.logger != nil {
					config.logger.Error("file upload failed",
						"index", job.index,
						"filename", job.filename,
						"error", err.Error())
				}

				results <- uploadResultJob{
					index:    job.index,
					filename: job.filename,
					path:     path,
					err:      err,
				}
			}
		}()
	}

	// Send jobs
	for i, fh := range fileHeaders {
		jobs <- uploadJob{
			index:      i,
			filename:   fh.Filename,
			fileHeader: fh,
		}
	}
	close(jobs)

	// Wait for all workers to complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results in order to maintain file ordering
	resultMap := make(map[int]uploadResultJob)
	for result := range results {
		resultMap[result.index] = result
	}

	// Process results in original order
	var uploadedPaths []string
	var fileErrors map[string]error
	fileErrors = make(map[string]error)

	for i := 0; i < len(fileHeaders); i++ {
		result, exists := resultMap[i]
		if !exists {
			continue
		}

		if result.err != nil {
			fileErrors[result.filename] = result.err
			continue
		}

		uploadedPaths = append(uploadedPaths, result.path)
	}

	// If any errors, cleanup and return
	if len(fileErrors) > 0 {
		cleanupFiles(ctx, disk, uploadedPaths)

		var errorMsg strings.Builder
		fmt.Fprintf(&errorMsg, "upload failed: %d of %d files had errors: ", len(fileErrors), len(fileHeaders))
		for filename, err := range fileErrors {
			fmt.Fprintf(&errorMsg, "[%s: %v] ", filename, err)
		}

		if config.logger != nil {
			config.logger.Error("concurrent upload failed",
				"total_files", len(fileHeaders),
				"failed_count", len(fileErrors),
				"successful_count", len(uploadedPaths))
		}

		return uploadedPaths, fmt.Errorf("%s", errorMsg.String())
	}

	if config.logger != nil {
		config.logger.Info("concurrent upload successful",
			"file_count", len(uploadedPaths))
	}

	return uploadedPaths, nil
}

// cleanupFiles menghapus file yang di-upload dari storage saat operasi upload gagal.
// Digunakan untuk membersihkan file yang partially uploaded untuk menghindari orphaned files.
// Error saat penghapusan diabaikan diam-diam untuk memastikan semua upaya cleanup dilakukan.
//
// Parameter:
//   - ctx: Context untuk cancellation dan deadlines
//   - disk: Storage backend untuk menghapus file
//   - paths: Daftar path file untuk dihapus
func cleanupFiles(ctx context.Context, disk storage.Storage, paths []string) {
	for _, path := range paths {
		_ = disk.Delete(ctx, path)
	}
}

// processFile menangani upload file individual dengan validasi lengkap dan penyimpanan.
// Melakukan sanitisasi nama file, validasi ukuran, pengecekan ekstensi, dan
// verifikasi content-type sebelum menyimpan file.
//
// Parameter:
//   - ctx: Context untuk cancellation dan deadlines
//   - disk: Storage backend untuk menyimpan file
//   - fileHeader: Multipart file header dengan metadata file
//   - config: Konfigurasi upload dengan batas size/extension
//   - allowedExts: Map ekstensi file yang diizinkan (kosong = semua diizinkan)
//
// Return:
//   - string: Path file yang di-upload saat sukses
//   - error: Error validasi atau storage dengan pesan detail
//
// Langkah validasi:
//   - Sanitisasi nama file
//   - Pengecekan ukuran file terhadap maxFileSize
//   - Validasi ekstensi terhadap allowedExts
//   - Validasi dan verifikasi content-type
func processFile(ctx context.Context, disk storage.Storage, fileHeader *multipart.FileHeader, config *UploadConfig, allowedExts map[string]bool) (string, error) {
	sanitizedFilename := sanitizeFilename(fileHeader.Filename)
	if sanitizedFilename == "" {
		return "", fmt.Errorf("invalid filename")
	}

	if config.maxFileSize > 0 && fileHeader.Size > int64(config.maxFileSize) {
		return "", fmt.Errorf(
			"file exceeds max size: %d bytes (max: %d bytes)",
			fileHeader.Size,
			config.maxFileSize,
		)
	}

	ext := strings.ToLower(filepath.Ext(sanitizedFilename))
	if len(allowedExts) > 0 && !allowedExts[ext] {
		return "", fmt.Errorf("invalid file extension: %s", ext)
	}

	file, err := fileHeader.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	contentType, needReopen, err := detectContentTypeFromFile(file, sanitizedFilename)
	if err != nil {
		return "", fmt.Errorf("failed to detect content type: %w", err)
	}

	if !isContentTypeValid(contentType, ext) {
		return "", fmt.Errorf("content type mismatch: detected %s for extension %s", contentType, ext)
	}

	if needReopen {
		if err := file.Close(); err != nil {
			return "", fmt.Errorf("failed to close file: %w", err)
		}

		file, err = fileHeader.Open()
		if err != nil {
			return "", fmt.Errorf("failed to reopen file: %w", err)
		}
		defer file.Close()
	}

	filename := fmt.Sprintf("%s/%s%s", config.path, NewUuid().String(), ext)
	path, err := disk.UploadStream(ctx, filename, file, storage.WithContentType(contentType))
	if err != nil {
		return "", fmt.Errorf("failed to save file: %w", err)
	}

	return path, nil
}

// detectContentTypeFromFile mendeteksi content type menggunakan magic numbers dan ekstensi.
//
// Strategi:
// 1. Pertama, manfaatkan DetectContentType() framework untuk coverage komprehensif
// 2. Fall back ke http.DetectContentType() untuk deteksi magic number
// 3. Penanganan khusus untuk format yang perlu reopening
//
// Return:
//   - contentType: MIME type yang terdeteksi
//   - needReopen: Apakah file perlu reopening setelah deteksi
//   - error: Error deteksi apapun
func detectContentTypeFromFile(file multipart.File, filename string) (string, bool, error) {
	needReopen := false

	if seeker, ok := file.(io.Seeker); ok {
		if _, err := seeker.Seek(0, io.SeekStart); err != nil {
			return "", false, fmt.Errorf("failed to seek to start: %w", err)
		}
	} else {
		needReopen = true
	}

	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return "", needReopen, fmt.Errorf("failed to read file for content type detection: %w", err)
	}

	if seeker, ok := file.(io.Seeker); ok {
		if _, err := seeker.Seek(0, io.SeekStart); err != nil {
			return "", false, fmt.Errorf("failed to seek back to start: %w", err)
		}
	}

	// Phase 1: Try framework's comprehensive detection (122 MIME types)
	contentType := DetectContentType(filename)

	// Phase 2: If got fallback, use magic number detection for better accuracy
	if contentType == "application/octet-stream" {
		contentType = http.DetectContentType(buffer[:n])
	}

	// Phase 3: Special handling for specific formats that need more detection
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".pdf":
		// Verify PDF magic number even if detection failed
		if n >= 4 && string(buffer[:4]) == "%PDF" {
			return "application/pdf", needReopen, nil
		}
	case ".svg":
		// SVG might be detected as text/xml, ensure svg+xml
		if contentType == "text/xml" || contentType == "application/xml" {
			return "image/svg+xml", needReopen, nil
		}
	}

	return contentType, needReopen, nil
}

// sanitizeFilename menghapus karakter dan pola yang berpotensi berbahaya dari nama file.
// Ini mencegah serangan directory traversal dan null byte injection.
//
// Langkah sanitisasi:
//   - Ekstrak base filename (menghapus path separators)
//   - Hapus sequence ".." (mencegah directory traversal)
//   - Trim whitespace
//   - Tolak nama file kosong dan "."
//
// Parameter:
//   - filename: Nama file asli dari multipart file header
//
// Return:
//   - Nama file yang disanitisasi aman untuk operasi filesystem
//   - String kosong jika nama file tidak valid
func sanitizeFilename(filename string) string {
	filename = filepath.Base(filename)
	filename = strings.ReplaceAll(filename, "..", "")
	filename = strings.TrimSpace(filename)

	if filename == "" || filename == "." {
		return ""
	}

	return filename
}

// sanitizePath menormalisasi dan mengamankan path direktori terhadap serangan directory traversal.
// Ini memastikan path direktori upload aman dan properly formatted.
//
// Langkah sanitisasi:
//   - Bersihkan path (normalisasi separators, hapus . dan ..)
//   - Hapus semua sequence ".." secara iteratif
//   - Consolidate double slashes
//   - Trim whitespace
//   - Pastikan absolute path (prefiks dengan /)
//
// Parameter:
//   - path: Path direktori upload (relative atau absolute)
//
// Return:
//   - Absolute path aman untuk digunakan sebagai direktori upload
//   - Format: /absolute/path/to/uploads
func sanitizePath(path string) string {
	path = filepath.Clean(path)

	// Remove all .. sequences iteratively to handle cases like "../../etc"
	for strings.Contains(path, "..") {
		path = strings.ReplaceAll(path, "..", "")
	}

	// Clean up any double slashes that may have resulted
	path = strings.ReplaceAll(path, "//", "/")
	path = strings.TrimSpace(path)

	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return path
}

// isContentTypeValid memvalidasi content-type file terhadap ekstensinya.
// Ini mencegah MIME type spoofing dimana file disamar dengan ekstensi yang salah.
//
// Logika validasi:
//   - Ekstensi yang tidak dikenal diizinkan (content-type dianggap valid)
//   - Ekstensi yang dikenal harus cocok dengan content-type yang diharapkan
//   - Ekstensi yang blacklist (exe, sh, dll.) selalu gagal validasi
//   - Multiple valid content-types per ekstensi didukung (misalnya csv sebagai text/csv atau text/plain)
//
// Parameter:
//   - contentType: MIME type yang terdeteksi dari konten file
//   - ext: Ekstensi file (misalnya ".jpg", ".pdf")
//
// Return:
//   - true: Content-type cocok dengan ekstensi atau ekstensi tidak dikenal
//   - false: Content-type tidak cocok atau ekstensi berbahaya
func isContentTypeValid(contentType, ext string) bool {
	validMappings := map[string][]string{
		".jpg":  {"image/jpeg"},
		".jpeg": {"image/jpeg"},
		".png":  {"image/png"},
		".gif":  {"image/gif"},
		".webp": {"image/webp"},
		".bmp":  {"image/bmp", "image/x-ms-bmp"},
		".svg":  {"image/svg+xml", "text/xml"},
		".pdf":  {"application/pdf"},
		".doc":  {"application/msword"},
		".docx": {"application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
		".xls":  {"application/vnd.ms-excel"},
		".xlsx": {"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"},
		".ppt":  {"application/vnd.ms-powerpoint"},
		".pptx": {"application/vnd.openxmlformats-officedocument.presentationml.presentation"},
		".txt":  {"text/plain"},
		".csv":  {"text/csv", "text/plain"},
		".json": {"application/json", "text/plain"},
		".xml":  {"application/xml", "text/xml"},
		".zip":  {"application/zip"},
		".rar":  {"application/x-rar-compressed"},
		".7z":   {"application/x-7z-compressed"},
		".tar":  {"application/x-tar"},
		".gz":   {"application/gzip"},
		".mp3":  {"audio/mpeg"},
		".mp4":  {"video/mp4"},
		".avi":  {"video/x-msvideo"},
		".mov":  {"video/quicktime"},
		".wmv":  {"video/x-ms-wmv"},
		".css":  {"text/css"},
		".js":   {"application/javascript", "text/javascript"},
		".html": {"text/html"},
		// Dangerous executable extensions - blacklisted (empty list means no valid content type)
		".exe":  {},
		".bat":  {},
		".cmd":  {},
		".com":  {},
		".scr":  {},
		".vbs":  {},
		".jar":  {},
		".app":  {},
		".sh":   {},
		".bash": {},
		".bin":  {},
		".dmg":  {},
		".deb":  {},
		".rpm":  {},
	}

	allowedTypes, exists := validMappings[strings.ToLower(ext)]
	if !exists {
		return true
	}

	// If extension exists in map but has no allowed types (blacklisted), reject it
	if len(allowedTypes) == 0 {
		return false
	}

	return slices.Contains(allowedTypes, contentType)
}
