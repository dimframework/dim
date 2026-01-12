package dim

import (
	"testing"
)

// ============================================================================
// Sanitization Tests
// ============================================================================

func TestSanitizePath_BasicPath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"uploads", "/uploads"},
		{"/uploads", "/uploads"},
		{"uploads/files", "/uploads/files"},
		{"/uploads/files", "/uploads/files"},
		{"./uploads", "/uploads"},
		{"/./uploads", "/uploads"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizePath(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizePath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizePath_DirectoryTraversal(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		desc     string
	}{
		{"../uploads", "/uploads", "parent reference"},
		{"../../etc", "/etc", "multiple parent refs"},
		{"/uploads/../../../etc", "/etc", "mixed paths"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			result := sanitizePath(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizePath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizeFilename_Basic(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"document.pdf", "document.pdf"},
		{"my-file.txt", "my-file.txt"},
		{"file_name.jpg", "file_name.jpg"},
		{"photo (1).png", "photo (1).png"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeFilename(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizeFilename_Dangerous(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		desc     string
	}{
		{"/etc/passwd", "passwd", "absolute path"},
		{"../../../etc/passwd", "passwd", "relative path with traversal"},
		{"file..name.txt", "filename.txt", "double dots removed"},
		{"/path/to/file.pdf", "file.pdf", "full path extracted"},
		{"C:\\Windows\\System32", "C:\\Windows\\System32", "windows path (preserved on unix)"},
		{"..\\..\\windows", "\\\\windows", "windows backslash (preserved on unix)"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			result := sanitizeFilename(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizeFilename_Invalid(t *testing.T) {
	tests := []struct {
		input string
		desc  string
	}{
		{"", "empty string"},
		{".", "dot only"},
		{"..", "double dot"},
		{"   ", "whitespace only"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			result := sanitizeFilename(tt.input)
			if result != "" {
				t.Errorf("sanitizeFilename(%q) should return empty, got %q", tt.input, result)
			}
		})
	}
}

// ============================================================================
// Content-Type Validation Tests
// ============================================================================

func TestIsContentTypeValid_AllowedTypes(t *testing.T) {
	tests := []struct {
		ext         string
		contentType string
		expected    bool
	}{
		{".jpg", "image/jpeg", true},
		{".png", "image/png", true},
		{".pdf", "application/pdf", true},
		{".txt", "text/plain", true},
		{".csv", "text/csv", true},
		{".csv", "text/plain", true}, // CSV can be text/plain
		{".json", "application/json", true},
	}

	for _, tt := range tests {
		t.Run(tt.ext+":"+tt.contentType, func(t *testing.T) {
			result := isContentTypeValid(tt.contentType, tt.ext)
			if result != tt.expected {
				t.Errorf("isContentTypeValid(%q, %q) = %v, want %v", tt.contentType, tt.ext, result, tt.expected)
			}
		})
	}
}

func TestIsContentTypeValid_BlockedExtensions(t *testing.T) {
	tests := []struct {
		ext  string
		desc string
	}{
		{".exe", "Windows executable"},
		{".bat", "Batch file"},
		{".sh", "Shell script"},
		{".bash", "Bash script"},
		{".jar", "Java archive"},
		{".dmg", "macOS image"},
		{".deb", "Debian package"},
		{".rpm", "RPM package"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			result := isContentTypeValid("application/octet-stream", tt.ext)
			if result {
				t.Errorf("isContentTypeValid(_, %q) should reject blocked extension", tt.ext)
			}
		})
	}
}

func TestIsContentTypeValid_MismatchedTypes(t *testing.T) {
	tests := []struct {
		ext         string
		contentType string
		expected    bool
		desc        string
	}{
		{".jpg", "application/pdf", false, "image as PDF"},
		{".pdf", "image/jpeg", false, "PDF as image"},
		{".txt", "image/png", false, "text as image"},
		{".png", "application/json", false, "image as JSON"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			result := isContentTypeValid(tt.contentType, tt.ext)
			if result != tt.expected {
				t.Errorf("isContentTypeValid(%q, %q) = %v, want %v", tt.contentType, tt.ext, result, tt.expected)
			}
		})
	}
}

// ============================================================================
// Default Config Tests
// ============================================================================

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.path != "/uploads" {
		t.Errorf("Default path = %q, want /uploads", config.path)
	}

	if config.maxFileSize != 10<<20 {
		t.Errorf("Default maxFileSize = %d, want %d", config.maxFileSize, 10<<20)
	}

	if config.maxFiles != 10 {
		t.Errorf("Default maxFiles = %d, want 10", config.maxFiles)
	}

	if config.maxWorkers != 10 {
		t.Errorf("Default maxWorkers = %d, want 10", config.maxWorkers)
	}

	if len(config.allowedExts) == 0 {
		t.Errorf("Default allowedExts should not be empty")
	}
}

// ============================================================================
// Option Pattern Tests
// ============================================================================

func TestWithPath(t *testing.T) {
	config := DefaultConfig()
	opt := WithPath("/custom/uploads")
	opt(config)

	if config.path != "/custom/uploads" {
		t.Errorf("WithPath() = %q, want /custom/uploads", config.path)
	}
}

func TestWithAllowedExts(t *testing.T) {
	config := DefaultConfig()
	opt := WithAllowedExts(".jpg", ".png", ".gif")
	opt(config)

	if len(config.allowedExts) != 3 {
		t.Errorf("WithAllowedExts() length = %d, want 3", len(config.allowedExts))
	}
}

func TestWithMaxFileSize_Valid(t *testing.T) {
	config := DefaultConfig()
	opt := WithMaxFileSize(50 << 20) // 50 MB
	opt(config)

	if config.maxFileSize != 50<<20 {
		t.Errorf("WithMaxFileSize() = %d, want %d", config.maxFileSize, 50<<20)
	}
}

func TestWithMaxFileSize_Zero(t *testing.T) {
	config := DefaultConfig()
	original := config.maxFileSize
	opt := WithMaxFileSize(0)
	opt(config)

	if config.maxFileSize != original {
		t.Errorf("WithMaxFileSize(0) should not change config")
	}
}

func TestWithMaxFiles_Valid(t *testing.T) {
	config := DefaultConfig()
	opt := WithMaxFiles(5)
	opt(config)

	if config.maxFiles != 5 {
		t.Errorf("WithMaxFiles() = %d, want 5", config.maxFiles)
	}
}

func TestWithMaxFiles_Zero(t *testing.T) {
	config := DefaultConfig()
	original := config.maxFiles
	opt := WithMaxFiles(0)
	opt(config)

	if config.maxFiles != original {
		t.Errorf("WithMaxFiles(0) should not change config")
	}
}

func TestWithConcurrent(t *testing.T) {
	config := DefaultConfig()
	opt := WithConcurrent(true)
	opt(config)

	if !config.concurrent {
		t.Errorf("WithConcurrent(true) failed")
	}
}

func TestWithMaxWorkers_Valid(t *testing.T) {
	config := DefaultConfig()
	opt := WithMaxWorkers(20)
	opt(config)

	if config.maxWorkers != 20 {
		t.Errorf("WithMaxWorkers() = %d, want 20", config.maxWorkers)
	}
}

func TestWithMaxWorkers_Zero(t *testing.T) {
	config := DefaultConfig()
	original := config.maxWorkers
	opt := WithMaxWorkers(0)
	opt(config)

	if config.maxWorkers != original {
		t.Errorf("WithMaxWorkers(0) should not change config")
	}
}

// ============================================================================
// Benchmark Tests
// ============================================================================

func BenchmarkSanitizePath(b *testing.B) {
	paths := []string{
		"/uploads",
		"../../../etc/passwd",
		"/uploads/files/../../../etc",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, path := range paths {
			sanitizePath(path)
		}
	}
}

func BenchmarkSanitizeFilename(b *testing.B) {
	filenames := []string{
		"document.pdf",
		"/etc/passwd",
		"../../../windows",
		"file..name.txt",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, fn := range filenames {
			sanitizeFilename(fn)
		}
	}
}

func BenchmarkIsContentTypeValid(b *testing.B) {
	tests := []struct {
		ext         string
		contentType string
	}{
		{".jpg", "image/jpeg"},
		{".pdf", "application/pdf"},
		{".exe", "application/octet-stream"},
		{".txt", "text/plain"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, tt := range tests {
			isContentTypeValid(tt.contentType, tt.ext)
		}
	}
}
