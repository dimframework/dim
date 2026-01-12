package dim

import (
	"io"
	"log/slog"
	"os"
)

// Logger is a wrapper around slog.Logger for structured logging
type Logger struct {
	*slog.Logger
}

// NewLogger membuat logger baru dengan JSON output format dan specified log level.
// Output dikirim ke stdout (os.Stdout).
// Gunakan untuk structured logging dalam JSON format yang bisa di-parse oleh log aggregation tools.
//
// Parameters:
//   - level: slog.Level untuk minimum log level (LevelDebug, LevelInfo, LevelWarn, LevelError)
//
// Returns:
//   - *Logger: logger instance dengan JSON handler
//
// Example:
//
//	logger := NewLogger(slog.LevelInfo)
//	logger.Info("User login", "user_id", 123, "email", "user@example.com")
func NewLogger(level slog.Level) *Logger {
	opts := &slog.HandlerOptions{
		Level: level,
	}

	handler := slog.NewJSONHandler(os.Stdout, opts)
	logger := slog.New(handler)

	return &Logger{logger}
}

// NewLoggerWithWriter membuat logger baru dengan JSON output format dan custom writer.
// Berguna untuk logging ke file, buffer, atau custom destinations.
// Output di-encode sebagai JSON untuk structured logging.
//
// Parameters:
//   - w: io.Writer untuk output destination (file, buffer, etc)
//   - level: slog.Level untuk minimum log level
//
// Returns:
//   - *Logger: logger instance dengan JSON handler dan custom writer
//
// Example:
//
//	file, _ := os.Create("app.log")
//	logger := NewLoggerWithWriter(file, slog.LevelInfo)
//	logger.Info("Application started")
func NewLoggerWithWriter(w io.Writer, level slog.Level) *Logger {
	opts := &slog.HandlerOptions{
		Level: level,
	}

	handler := slog.NewJSONHandler(w, opts)
	logger := slog.New(handler)

	return &Logger{logger}
}

// NewTextLogger membuat logger baru dengan text output format dan specified log level.
// Output dikirim ke stdout (os.Stdout) dalam human-readable text format.
// Gunakan untuk development environment atau ketika structured JSON tidak diperlukan.
//
// Parameters:
//   - level: slog.Level untuk minimum log level
//
// Returns:
//   - *Logger: logger instance dengan text handler
//
// Example:
//
//	logger := NewTextLogger(slog.LevelDebug)
//	logger.Debug("Debug information", "key", "value")
func NewTextLogger(level slog.Level) *Logger {
	opts := &slog.HandlerOptions{
		Level: level,
	}

	handler := slog.NewTextHandler(os.Stdout, opts)
	logger := slog.New(handler)

	return &Logger{logger}
}

// NewTextLoggerWithWriter membuat logger baru dengan text output format dan custom writer.
// Output dalam human-readable text format ke specified writer.
// Berguna untuk logging ke file dalam text format.
//
// Parameters:
//   - w: io.Writer untuk output destination
//   - level: slog.Level untuk minimum log level
//
// Returns:
//   - *Logger: logger instance dengan text handler dan custom writer
//
// Example:
//
//	file, _ := os.Create("app.log")
//	logger := NewTextLoggerWithWriter(file, slog.LevelInfo)
//	logger.Info("User registered", "email", "user@example.com")
func NewTextLoggerWithWriter(w io.Writer, level slog.Level) *Logger {
	opts := &slog.HandlerOptions{
		Level: level,
	}

	handler := slog.NewTextHandler(w, opts)
	logger := slog.New(handler)

	return &Logger{logger}
}

// Info menulis info-level log message dengan optional key-value attributes.
// Info level digunakan untuk general informational messages (login, request, etc).
// Arguments adalah variadic key-value pairs (key1, value1, key2, value2, ...).
//
// Parameters:
//   - msg: log message string
//   - args: variadic arguments (key-value pairs)
//
// Example:
//
//	logger.Info("User login successful", "user_id", 123, "ip", "192.168.1.1")
func (l *Logger) Info(msg string, args ...any) {
	l.Logger.Info(msg, args...)
}

// Error menulis error-level log message dengan optional key-value attributes.
// Error level digunakan untuk error messages yang perlu attention.
// Arguments adalah variadic key-value pairs untuk additional context.
//
// Parameters:
//   - msg: error message string
//   - args: variadic arguments (key-value pairs)
//
// Example:
//
//	logger.Error("Database connection failed", "error", err, "retry_count", 3)
func (l *Logger) Error(msg string, args ...any) {
	l.Logger.Error(msg, args...)
}

// Warn menulis warn-level log message dengan optional key-value attributes.
// Warn level digunakan untuk warning messages tentang non-critical issues.
// Arguments adalah variadic key-value pairs untuk additional context.
//
// Parameters:
//   - msg: warning message string
//   - args: variadic arguments (key-value pairs)
//
// Example:
//
//	logger.Warn("High memory usage detected", "usage_percent", 85.5)
func (l *Logger) Warn(msg string, args ...any) {
	l.Logger.Warn(msg, args...)
}

// Debug menulis debug-level log message dengan optional key-value attributes.
// Debug level digunakan untuk detailed debugging information.
// Hanya ditampilkan jika logger level di-set ke LevelDebug.
// Arguments adalah variadic key-value pairs untuk debugging context.
//
// Parameters:
//   - msg: debug message string
//   - args: variadic arguments (key-value pairs)
//
// Example:
//
//	logger.Debug("Processing request", "method", "GET", "path", "/users/123")
func (l *Logger) Debug(msg string, args ...any) {
	l.Logger.Debug(msg, args...)
}

// WithGroup returns a new logger with a group added to attributes
func (l *Logger) WithGroup(name string) *Logger {
	return &Logger{l.Logger.WithGroup(name)}
}

// WithAttrs returns a new logger with the attributes added
func (l *Logger) WithAttrs(attrs ...slog.Attr) *Logger {
	return &Logger{l.Logger.With(slogAttrsToAny(attrs)...)}
}

// slogAttrsToAny converts slog.Attr slice to any slice
func slogAttrsToAny(attrs []slog.Attr) []any {
	result := make([]any, len(attrs)*2)
	for i, attr := range attrs {
		result[i*2] = attr.Key
		result[i*2+1] = attr.Value
	}
	return result
}
