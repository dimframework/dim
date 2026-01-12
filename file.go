package dim

import (
	"net/http"
	"path/filepath"
	"strings"
	"sync"
)

// CustomMIMETypes memungkinkan pendaftaran custom MIME types di luar daftar built-in.
// Gunakan RegisterMIMEType() untuk menambah custom types. Thread-safe melalui internal mutex.
//
// Contoh:
//
//	RegisterMIMEType(".custom", "application/x-custom")
var customMIMETypes = make(map[string]string)
var mimeTypesMutex sync.RWMutex

// DetectContentType mendeteksi MIME content type file berdasarkan extension-nya.
//
// Fungsi melakukan case-insensitive extension matching terhadap map komprehensif
// dari MIME types yang diketahui. Mengembalikan MIME type string yang sesuai untuk
// extension file, atau "application/octet-stream" jika extension tidak diketahui.
//
// Kategori file yang didukung:
//   - Images: JPEG, PNG, GIF, WebP, SVG, ICO, TIFF, BMP
//   - Documents: PDF, Word (DOC/DOCX), Excel (XLS/XLSX), PowerPoint (PPT/PPTX), ODF formats
//   - Text & Code: TXT, CSV, HTML, CSS, JavaScript, JSON, XML, YAML, Markdown, TypeScript, etc.
//   - Archives: ZIP, RAR, 7Z, TAR, GZIP, BZIP2, etc.
//   - Video: MP4, AVI, MOV, WMV, WebM, MPEG, MKV, FLV
//   - Audio: MP3, WAV, OGG, FLAC, M4A, AAC
//   - Web Fonts: WOFF, WOFF2, TTF, OTF, EOT
//   - Others: EPUB, TORRENT, etc.
//
// Parameter:
//
//	filename - Nama file (dapat include path, extension di-extract otomatis)
//
// Return:
//
//	MIME type string dalam format "type/subtype" (misal: "image/jpeg", "text/plain").
//	Mengembalikan "application/octet-stream" sebagai fallback untuk extension yang tidak diketahui.
//
// Contoh:
//
//	DetectContentType("photo.jpg")       // Mengembalikan "image/jpeg"
//	DetectContentType("document.pdf")    // Mengembalikan "application/pdf"
//	DetectContentType("script.js")       // Mengembalikan "application/javascript"
//	DetectContentType("unknown.xyz")     // Mengembalikan "application/octet-stream"
//	DetectContentType("/path/to/file.png") // Mengembalikan "image/png" (path di-handle otomatis)
func DetectContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))

	// Check custom MIME types first (thread-safe)
	mimeTypesMutex.RLock()
	if contentType, ok := customMIMETypes[ext]; ok {
		mimeTypesMutex.RUnlock()
		return contentType
	}
	mimeTypesMutex.RUnlock()

	contentTypes := map[string]string{
		// ===== IMAGE TYPES =====
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".jpe":  "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".webp": "image/webp",
		".svg":  "image/svg+xml",
		".ico":  "image/x-icon",
		".tiff": "image/tiff",
		".tif":  "image/tiff",
		".bmp":  "image/bmp",
		".dib":  "image/bmp",

		// ===== DOCUMENT TYPES =====
		".pdf":  "application/pdf",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".dot":  "application/msword",
		".dotx": "application/vnd.openxmlformats-officedocument.wordprocessingml.template",
		".xls":  "application/vnd.ms-excel",
		".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		".xlt":  "application/vnd.ms-excel",
		".xltx": "application/vnd.openxmlformats-officedocument.spreadsheetml.template",
		".ppt":  "application/vnd.ms-powerpoint",
		".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
		".pps":  "application/vnd.ms-powerpoint",
		".ppsx": "application/vnd.openxmlformats-officedocument.presentationml.slideshow",
		// ODF formats
		".odt": "application/vnd.oasis.opendocument.text",
		".ods": "application/vnd.oasis.opendocument.spreadsheet",
		".odp": "application/vnd.oasis.opendocument.presentation",
		".odg": "application/vnd.oasis.opendocument.graphics",

		// ===== TEXT & CODE TYPES =====
		".txt":      "text/plain",
		".csv":      "text/csv",
		".html":     "text/html",
		".htm":      "text/html",
		".css":      "text/css",
		".js":       "application/javascript",
		".mjs":      "application/javascript",
		".json":     "application/json",
		".xml":      "application/xml",
		".md":       "text/markdown",
		".markdown": "text/markdown",
		".yaml":     "text/yaml",
		".yml":      "text/yaml",
		".ts":       "application/typescript",
		".tsx":      "application/typescript",
		".jsx":      "application/jsx",
		".go":       "text/plain",
		".py":       "text/plain",
		".rb":       "text/plain",
		".php":      "application/x-php",
		".java":     "text/plain",
		".cpp":      "text/plain",
		".c":        "text/plain",
		".h":        "text/plain",
		".sh":       "application/x-sh",
		".bash":     "application/x-bash",
		".sql":      "application/x-sql",
		".pl":       "application/x-perl",
		".lua":      "text/plain",
		".scala":    "text/plain",
		".kt":       "text/plain",
		".swift":    "text/plain",
		".rs":       "text/plain",
		".asm":      "text/plain",

		// ===== ARCHIVE TYPES =====
		".zip":  "application/zip",
		".rar":  "application/x-rar-compressed",
		".7z":   "application/x-7z-compressed",
		".tar":  "application/x-tar",
		".gz":   "application/gzip",
		".gzip": "application/gzip",
		".bz2":  "application/x-bzip2",
		".xz":   "application/x-xz",
		".iso":  "application/x-iso9660-image",

		// ===== VIDEO TYPES =====
		".mp4":  "video/mp4",
		".m4v":  "video/mp4",
		".avi":  "video/x-msvideo",
		".mov":  "video/quicktime",
		".qt":   "video/quicktime",
		".wmv":  "video/x-ms-wmv",
		".asf":  "video/x-ms-asf",
		".webm": "video/webm",
		".mpeg": "video/mpeg",
		".mpg":  "video/mpeg",
		".mkv":  "video/x-matroska",
		".mka":  "audio/x-matroska",
		".flv":  "video/x-flv",
		".m3u8": "application/vnd.apple.mpegurl",
		".3gp":  "video/3gpp",
		".3g2":  "video/3gpp2",
		".ogv":  "video/ogg",
		".mts":  "video/mp2t",
		".m2ts": "video/mp2t",

		// ===== AUDIO TYPES =====
		".mp3":  "audio/mpeg",
		".wav":  "audio/wav",
		".ogg":  "audio/ogg",
		".oga":  "audio/ogg",
		".flac": "audio/flac",
		".m4a":  "audio/mp4",
		".m4b":  "audio/mp4",
		".aac":  "audio/aac",
		".wma":  "audio/x-ms-wma",
		".aiff": "audio/aiff",
		".aif":  "audio/aiff",
		".au":   "audio/basic",
		".opus": "audio/opus",
		".weba": "audio/webp",

		// ===== WEB FONT TYPES =====
		".woff":  "font/woff",
		".woff2": "font/woff2",
		".ttf":   "font/ttf",
		".otf":   "font/otf",
		".eot":   "application/vnd.ms-fontobject",
		".sfnt":  "font/sfnt",

		// ===== OTHER TYPES =====
		".epub":    "application/epub+zip",
		".torrent": "application/x-bittorrent",
		".swf":     "application/x-shockwave-flash",
		".exe":     "application/x-msdownload",
		".dll":     "application/x-msdownload",
		".msi":     "application/x-msdownload",
		".apk":     "application/vnd.android.package-archive",
		".dmg":     "application/x-apple-diskimage",
		".ipa":     "application/octet-stream",
		".bin":     "application/octet-stream",
		".wasm":    "application/wasm",
	}

	if contentType, ok := contentTypes[ext]; ok {
		return contentType
	}

	return "application/octet-stream"
}

// RegisterMIMEType mendaftarkan custom MIME type untuk file extension.
//
// Ini memungkinkan Anda menambah support untuk custom atau proprietary file types
// di luar daftar built-in. Custom MIME types dicek terlebih dahulu sebelum
// built-in types, jadi Anda dapat override default MIME types jika diperlukan.
//
// Fungsi adalah thread-safe dan dapat dipanggil concurrently dari multiple
// goroutines.
//
// Parameter:
//
//	ext - File extension termasuk dot (misal: ".custom", ".myformat")
//	mimeType - MIME type string (misal: "application/x-custom")
//
// Contoh:
//
//	RegisterMIMEType(".webmanifest", "application/manifest+json")
//	RegisterMIMEType(".wasm", "application/wasm")
//	RegisterMIMEType(".custom", "application/x-custom")
func RegisterMIMEType(ext, mimeType string) {
	ext = strings.ToLower(ext)
	mimeTypesMutex.Lock()
	defer mimeTypesMutex.Unlock()
	customMIMETypes[ext] = mimeType
}

// ServeFile melayani file dari filesystem dengan Content-Type header yang tepat.
//
// Fungsi helper ini mengirim file ke client dengan MIME type yang sesuai
// yang terdeteksi dari file extension. File dilayani sebagai attachment (download),
// berarti browser akan meminta user untuk download file daripada
// menampilkannya secara inline.
//
// Fungsi otomatis mendeteksi Content-Type header yang benar berdasarkan
// file extension menggunakan DetectContentType().
//
// Parameter:
//
//	w - http.ResponseWriter untuk menulis response ke
//	filename - Filename original (digunakan untuk content-type detection)
//	filePath - Path file sebenarnya di filesystem
//	statusCode - HTTP status code (biasanya http.StatusOK atau http.StatusCreated)
//
// Return:
//
//	Error jika file tidak dapat dibuka atau dibaca, atau nil jika sukses.
//
// Contoh:
//
//	ServeFile(w, "document.pdf", "/tmp/document.pdf", http.StatusOK)
//	ServeFile(w, "report.xlsx", "/data/reports/q1.xlsx", http.StatusOK)
//
// Catatan: Gunakan ServeFileInline() jika Anda ingin file ditampilkan di browser
// daripada diunduh.
func ServeFile(w http.ResponseWriter, filename, filePath string, statusCode int) error {
	contentType := DetectContentType(filename)
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.WriteHeader(statusCode)

	http.ServeFile(w, &http.Request{}, filePath)
	return nil
}

// ServeFileInline melayani file dari filesystem untuk inline display.
//
// Fungsi helper ini mengirim file ke client dengan MIME type yang sesuai
// yang terdeteksi dari file extension. File dilayani sebagai inline content, artinya
// browser akan mencoba menampilkan file (jika tipe yang didukung seperti
// images, PDFs, atau videos) daripada meminta download.
//
// Fungsi otomatis mendeteksi Content-Type header yang benar berdasarkan
// file extension menggunakan DetectContentType().
//
// Parameter:
//
//	w - http.ResponseWriter untuk menulis response ke
//	filename - Filename original (digunakan untuk content-type detection)
//	filePath - Path file sebenarnya di filesystem
//	statusCode - HTTP status code (biasanya http.StatusOK)
//
// Return:
//
//	Error jika file tidak dapat dibuka atau dibaca, atau nil jika sukses.
//
// Contoh:
//
//	ServeFileInline(w, "image.jpg", "/images/photo.jpg", http.StatusOK)
//	ServeFileInline(w, "document.pdf", "/pdfs/report.pdf", http.StatusOK)
//	ServeFileInline(w, "video.mp4", "/videos/tutorial.mp4", http.StatusOK)
//
// Catatan: Gunakan ServeFile() jika Anda ingin file diunduh daripada ditampilkan.
func ServeFileInline(w http.ResponseWriter, filename, filePath string, statusCode int) error {
	contentType := DetectContentType(filename)
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", `inline; filename="`+filename+`"`)
	w.WriteHeader(statusCode)

	http.ServeFile(w, &http.Request{}, filePath)
	return nil
}
