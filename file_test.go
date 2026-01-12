package dim

import (
	"testing"
)

// ============================================================================
// Image File Tests
// ============================================================================

func TestDetectContentType_ImagesCommon(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"photo.jpg", "image/jpeg"},
		{"photo.jpeg", "image/jpeg"},
		{"screenshot.png", "image/png"},
		{"animated.gif", "image/gif"},
		{"modern.webp", "image/webp"},
		{"icon.svg", "image/svg+xml"},
		{"favicon.ico", "image/x-icon"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := DetectContentType(tt.filename)
			if result != tt.expected {
				t.Errorf("DetectContentType(%q) = %q, want %q", tt.filename, result, tt.expected)
			}
		})
	}
}

// ============================================================================
// Document File Tests
// ============================================================================

func TestDetectContentType_DocumentsOffice(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"report.pdf", "application/pdf"},
		{"document.doc", "application/msword"},
		{"document.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
		{"spreadsheet.xls", "application/vnd.ms-excel"},
		{"spreadsheet.xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"},
		{"presentation.ppt", "application/vnd.ms-powerpoint"},
		{"presentation.pptx", "application/vnd.openxmlformats-officedocument.presentationml.presentation"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := DetectContentType(tt.filename)
			if result != tt.expected {
				t.Errorf("DetectContentType(%q) = %q, want %q", tt.filename, result, tt.expected)
			}
		})
	}
}

// ============================================================================
// Text and Code Files Tests
// ============================================================================

func TestDetectContentType_TextAndCode(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"readme.txt", "text/plain"},
		{"data.csv", "text/csv"},
		{"index.html", "text/html"},
		{"style.css", "text/css"},
		{"script.js", "application/javascript"},
		{"config.json", "application/json"},
		{"data.xml", "application/xml"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := DetectContentType(tt.filename)
			if result != tt.expected {
				t.Errorf("DetectContentType(%q) = %q, want %q", tt.filename, result, tt.expected)
			}
		})
	}
}

// ============================================================================
// Archive File Tests
// ============================================================================

func TestDetectContentType_Archives(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"archive.zip", "application/zip"},
		{"archive.rar", "application/x-rar-compressed"},
		{"archive.7z", "application/x-7z-compressed"},
		{"archive.tar", "application/x-tar"},
		{"archive.tar.gz", "application/gzip"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := DetectContentType(tt.filename)
			if result != tt.expected {
				t.Errorf("DetectContentType(%q) = %q, want %q", tt.filename, result, tt.expected)
			}
		})
	}
}

// ============================================================================
// Video File Tests
// ============================================================================

func TestDetectContentType_Videos(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"movie.mp4", "video/mp4"},
		{"video.avi", "video/x-msvideo"},
		{"video.mov", "video/quicktime"},
		{"video.wmv", "video/x-ms-wmv"},
		{"video.webm", "video/webm"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := DetectContentType(tt.filename)
			if result != tt.expected {
				t.Errorf("DetectContentType(%q) = %q, want %q", tt.filename, result, tt.expected)
			}
		})
	}
}

// ============================================================================
// Audio File Tests
// ============================================================================

func TestDetectContentType_Audio(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"song.mp3", "audio/mpeg"},
		{"sound.wav", "audio/wav"},
		{"audio.ogg", "audio/ogg"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := DetectContentType(tt.filename)
			if result != tt.expected {
				t.Errorf("DetectContentType(%q) = %q, want %q", tt.filename, result, tt.expected)
			}
		})
	}
}

// ============================================================================
// Case Sensitivity Tests
// ============================================================================

func TestDetectContentType_Uppercase(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"PHOTO.JPG", "image/jpeg"},
		{"DOCUMENT.PDF", "application/pdf"},
		{"SCRIPT.JS", "application/javascript"},
		{"DATA.CSV", "text/csv"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := DetectContentType(tt.filename)
			if result != tt.expected {
				t.Errorf("DetectContentType(%q) = %q, want %q", tt.filename, result, tt.expected)
			}
		})
	}
}

func TestDetectContentType_MixedCase(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"Photo.Jpg", "image/jpeg"},
		{"Document.Pdf", "application/pdf"},
		{"Script.Js", "application/javascript"},
		{"Data.Csv", "text/csv"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := DetectContentType(tt.filename)
			if result != tt.expected {
				t.Errorf("DetectContentType(%q) = %q, want %q", tt.filename, result, tt.expected)
			}
		})
	}
}

// ============================================================================
// Edge Cases and Unknown Types
// ============================================================================

func TestDetectContentType_EdgeCases(t *testing.T) {
	tests := []struct {
		filename string
		expected string
		desc     string
	}{
		{"file", "application/octet-stream", "no extension"},
		{"file.unknown", "application/octet-stream", "unknown extension"},
		{"file.xyz", "application/octet-stream", "random extension"},
		{"file.UNKNOWN", "application/octet-stream", "unknown uppercase"},
		{".hiddenfile", "application/octet-stream", "hidden file no extension"},
		{"path/to/file.pdf", "application/pdf", "with path"},
		{"file.multiple.dots.pdf", "application/pdf", "multiple dots in name"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			result := DetectContentType(tt.filename)
			if result != tt.expected {
				t.Errorf("DetectContentType(%q) = %q, want %q", tt.filename, result, tt.expected)
			}
		})
	}
}

func TestDetectContentType_EmptyFilename(t *testing.T) {
	result := DetectContentType("")
	expected := "application/octet-stream"
	if result != expected {
		t.Errorf("DetectContentType(\"\") = %q, want %q", result, expected)
	}
}

// ============================================================================
// Benchmark Tests
// ============================================================================

func BenchmarkDetectContentType_Common(b *testing.B) {
	filenames := []string{
		"photo.jpg",
		"document.pdf",
		"script.js",
		"data.json",
		"archive.zip",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, filename := range filenames {
			DetectContentType(filename)
		}
	}
}

func BenchmarkDetectContentType_Unknown(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DetectContentType("file.unknown")
	}
}

func BenchmarkDetectContentType_WithPath(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DetectContentType("/path/to/very/deep/file.pdf")
	}
}

// ============================================================================
// Custom MIME Type Registration Tests
// ============================================================================

func TestRegisterMIMEType_SingleCustomType(t *testing.T) {
	// Register custom type
	RegisterMIMEType(".custom", "application/x-custom")

	result := DetectContentType("file.custom")
	expected := "application/x-custom"

	if result != expected {
		t.Errorf("DetectContentType with custom type = %q, want %q", result, expected)
	}
}

func TestRegisterMIMEType_MultipleCustomTypes(t *testing.T) {
	tests := []struct {
		ext      string
		mimeType string
	}{
		{".myformat", "application/x-myformat"},
		{".proprietary", "application/vnd.company.proprietary"},
		{".newformat", "application/x-newformat"},
	}

	for _, tt := range tests {
		RegisterMIMEType(tt.ext, tt.mimeType)
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			result := DetectContentType("file" + tt.ext)
			if result != tt.mimeType {
				t.Errorf("DetectContentType(%q) = %q, want %q", "file"+tt.ext, result, tt.mimeType)
			}
		})
	}
}

func TestRegisterMIMEType_CaseInsensitive(t *testing.T) {
	RegisterMIMEType(".TEST", "application/x-test")

	tests := []string{
		"file.test",
		"file.TEST",
		"file.Test",
	}

	for _, filename := range tests {
		result := DetectContentType(filename)
		expected := "application/x-test"
		if result != expected {
			t.Errorf("DetectContentType(%q) = %q, want %q", filename, result, expected)
		}
	}
}

func TestRegisterMIMEType_OverrideBuiltin(t *testing.T) {
	// Override built-in PDF type
	originalType := DetectContentType("file.pdf")
	expected := "application/pdf"
	if originalType != expected {
		t.Errorf("Original PDF type = %q, want %q", originalType, expected)
	}

	// Register custom PDF handler
	RegisterMIMEType(".pdf", "application/x-custom-pdf")

	result := DetectContentType("file.pdf")
	if result != "application/x-custom-pdf" {
		t.Errorf("After override, DetectContentType(file.pdf) = %q, want %q", result, "application/x-custom-pdf")
	}
}

func TestRegisterMIMEType_EmptyExtension(t *testing.T) {
	RegisterMIMEType("", "application/empty")

	result := DetectContentType("filename")
	expected := "application/empty"

	if result != expected {
		t.Errorf("DetectContentType with empty ext = %q, want %q", result, expected)
	}
}

// ============================================================================
// Benchmark Tests for RegisterMIMEType
// ============================================================================

func BenchmarkRegisterMIMEType(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		RegisterMIMEType(".benchmark", "application/benchmark")
	}
}

func BenchmarkDetectContentType_CustomType(b *testing.B) {
	RegisterMIMEType(".custom", "application/x-custom")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DetectContentType("file.custom")
	}
}
