package dim

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGetEnv(t *testing.T) {
	os.Setenv("TEST_VAR", "test_value")
	defer os.Unsetenv("TEST_VAR")

	result := GetEnv("TEST_VAR")
	if result != "test_value" {
		t.Errorf("GetEnv() = %q, want %q", result, "test_value")
	}
}

func TestGetEnvNotSet(t *testing.T) {
	result := GetEnv("NON_EXISTENT_VAR_12345")
	if result != "" {
		t.Errorf("GetEnv() = %q, want empty string", result)
	}
}

func TestGetEnvOrDefault(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		value        string
		defaultValue string
		want         string
	}{
		{"env_var_exists", "TEST_VAR_EXISTS", "actual_value", "default_value", "actual_value"},
		{"env_var_not_exists", "TEST_VAR_NOT_EXISTS_12345", "", "default_value", "default_value"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != "" {
				os.Setenv(tt.key, tt.value)
				defer os.Unsetenv(tt.key)
			}
			result := GetEnvOrDefault(tt.key, tt.defaultValue)
			if result != tt.want {
				t.Errorf("GetEnvOrDefault() = %q, want %q", result, tt.want)
			}
		})
	}
}

func TestParseEnvDuration(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    time.Duration
		wantErr bool
	}{
		{"valid_15m", "15m", 15 * time.Minute, false},
		{"valid_1h", "1h", time.Hour, false},
		{"empty_is_zero", "", 0, false},
		{"invalid_duration", "abc", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseEnvDuration(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseEnvDuration() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseEnvDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseEnvBool(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"true_lowercase", "true", true},
		{"true_uppercase", "TRUE", true},
		{"yes", "yes", true},
		{"1", "1", true},
		{"on", "on", true},
		{"false", "false", false},
		{"no", "no", false},
		{"0", "0", false},
		{"off", "off", false},
		{"empty", "", false},
		{"with_spaces", "  true  ", true},
		{"random_string", "anything_else", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseEnvBool(tt.input); got != tt.want {
				t.Errorf("ParseEnvBool() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseEnvInt(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int
		wantErr bool
	}{
		{"valid_100", "100", 100, false},
		{"valid_with_spaces", "  50  ", 50, false},
		{"empty_returns_zero", "", 0, false},
		{"invalid_returns_error", "invalid", 0, true},
		{"negative_number", "-25", -25, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseEnvInt(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseEnvInt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseEnvInt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoadEnvFile(t *testing.T) {
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".env")
	envContent := `
# Comment line
TEST_KEY1=value1
TEST_KEY2="quoted value"
TEST_KEY3='single quoted'
EMPTY_VALUE=
TEST_KEY_EXISTS=new_value
`
	if err := os.WriteFile(envFile, []byte(envContent), 0644); err != nil {
		t.Fatalf("failed to create temp .env file: %v", err)
	}

	os.Setenv("TEST_KEY_EXISTS", "original_value")
	defer os.Unsetenv("TEST_KEY_EXISTS")

	if err := LoadEnvFile(envFile); err != nil {
		t.Fatalf("LoadEnvFile() error = %v", err)
	}

	if val := os.Getenv("TEST_KEY1"); val != "value1" {
		t.Errorf("Getenv(TEST_KEY1) = %q, want %q", val, "value1")
	}
	if val := os.Getenv("TEST_KEY2"); val != "quoted value" {
		t.Errorf("Getenv(TEST_KEY2) = %q, want %q", val, "quoted value")
	}
	if val := os.Getenv("TEST_KEY_EXISTS"); val != "original_value" {
		t.Errorf("LoadEnvFile should not overwrite existing env var, got %q", val)
	}
}

func TestLoadEnvFile_MalformedLine(t *testing.T) {
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".env")
	// File with a malformed line and a valid line
	envContent := "MALFORMED_LINE\nVALID_LINE=correct"
	if err := os.WriteFile(envFile, []byte(envContent), 0644); err != nil {
		t.Fatalf("failed to create temp .env file: %v", err)
	}
	defer os.Unsetenv("VALID_LINE")

	// LoadEnvFile should log a warning but not return an error
	if err := LoadEnvFile(envFile); err != nil {
		t.Fatalf("LoadEnvFile() returned an error for a malformed line: %v", err)
	}

	// The valid variable should still be loaded
	if val := os.Getenv("VALID_LINE"); val != "correct" {
		t.Errorf("Getenv(VALID_LINE) = %q, want %q", val, "correct")
	}
}

func TestLoadEnvFileNotFound(t *testing.T) {
	err := LoadEnvFile("/nonexistent/path/.env")
	if err != nil {
		t.Errorf("LoadEnvFile() should not error on missing file, got %v", err)
	}
}

func TestLoadEnvFileFromPath(t *testing.T) {
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".env")
	envContent := `FROMPATH_KEY=frompath_value`
	if err := os.WriteFile(envFile, []byte(envContent), 0644); err != nil {
		t.Fatalf("failed to create temp .env file: %v", err)
	}
	defer os.Unsetenv("FROMPATH_KEY")

	if err := LoadEnvFileFromPath(tmpDir); err != nil {
		t.Fatalf("LoadEnvFileFromPath() error = %v", err)
	}

	if val := os.Getenv("FROMPATH_KEY"); val != "frompath_value" {
		t.Errorf("Getenv(FROMPATH_KEY) = %q, want %q", val, "frompath_value")
	}
}
