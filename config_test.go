package dim

import (
	"os"
	"testing"
	"time"
)

func TestLoadServerConfig_Defaults(t *testing.T) {
	cfg, err := loadServerConfig()
	if err != nil {
		t.Fatalf("loadServerConfig() with defaults failed: %v", err)
	}
	if cfg.Port != "8080" {
		t.Errorf("default port = %s, want 8080", cfg.Port)
	}
}

func TestLoadServerConfig_WithEnv(t *testing.T) {
	os.Setenv("SERVER_PORT", "9000")
	defer os.Unsetenv("SERVER_PORT")

	cfg, err := loadServerConfig()
	if err != nil {
		t.Fatalf("loadServerConfig() with env failed: %v", err)
	}
	if cfg.Port != "9000" {
		t.Errorf("port = %s, want 9000", cfg.Port)
	}
}

func TestLoadServerConfig_InvalidDurations(t *testing.T) {
	t.Run("invalid write timeout", func(t *testing.T) {
		os.Setenv("SERVER_WRITE_TIMEOUT", "abc")
		defer os.Unsetenv("SERVER_WRITE_TIMEOUT")
		_, err := loadServerConfig()
		if err == nil {
			t.Error("loadServerConfig() should have returned an error for invalid write timeout")
		}
	})

	t.Run("invalid read timeout", func(t *testing.T) {
		os.Setenv("SERVER_READ_TIMEOUT", "abc")
		defer os.Unsetenv("SERVER_READ_TIMEOUT")
		_, err := loadServerConfig()
		if err == nil {
			t.Error("loadServerConfig() should have returned an error for invalid read timeout")
		}
	})
}

func TestLoadJWTConfig(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret")
	defer os.Unsetenv("JWT_SECRET")

	cfg, err := loadJWTConfig()
	if err != nil {
		t.Fatalf("loadJWTConfig() failed: %v", err)
	}
	if cfg.Secret != "test-secret" {
		t.Error("secret mismatch")
	}
}

func TestLoadDatabaseConfig(t *testing.T) {
	os.Setenv("DB_WRITE_HOST", "localhost")
	os.Setenv("DB_NAME", "testdb")
	os.Setenv("DB_USER", "user")
	defer func() {
		os.Unsetenv("DB_WRITE_HOST")
		os.Unsetenv("DB_NAME")
		os.Unsetenv("DB_USER")
	}()

	cfg, err := loadDatabaseConfig()
	if err != nil {
		t.Fatalf("loadDatabaseConfig() failed: %v", err)
	}
	if cfg.WriteHost != "localhost" {
		t.Error("write host mismatch")
	}
}

func TestLoadDatabaseConfig_InvalidValues(t *testing.T) {
	t.Run("invalid port", func(t *testing.T) {
		os.Setenv("DB_PORT", "not-a-port")
		defer os.Unsetenv("DB_PORT")
		_, err := loadDatabaseConfig()
		if err == nil {
			t.Error("expected an error for invalid port")
		}
	})

	t.Run("invalid max conns", func(t *testing.T) {
		os.Setenv("DB_MAX_CONNS", "not-an-int")
		defer os.Unsetenv("DB_MAX_CONNS")
		_, err := loadDatabaseConfig()
		if err == nil {
			t.Error("expected an error for invalid max conns")
		}
	})
}

func TestLoadRateLimitConfig_Invalid(t *testing.T) {
	os.Setenv("RATE_LIMIT_RESET_PERIOD", "invalid")
	defer os.Unsetenv("RATE_LIMIT_RESET_PERIOD")
	_, err := loadRateLimitConfig()
	if err == nil {
		t.Error("expected an error for invalid reset period")
	}
}

func TestLoadConfig_Integration_FailFast(t *testing.T) {
	// Set required values to pass initial validation
	os.Setenv("JWT_SECRET", "secret")
	os.Setenv("DB_WRITE_HOST", "host")
	os.Setenv("DB_NAME", "db")
	os.Setenv("DB_USER", "user")
	defer func() {
		os.Unsetenv("JWT_SECRET")
		os.Unsetenv("DB_WRITE_HOST")
		os.Unsetenv("DB_NAME")
		os.Unsetenv("DB_USER")
	}()

	t.Run("invalid_server_config", func(t *testing.T) {
		os.Setenv("SERVER_READ_TIMEOUT", "abc")
		defer os.Unsetenv("SERVER_READ_TIMEOUT")
		_, err := LoadConfig()
		if err == nil {
			t.Error("LoadConfig should fail on invalid server config")
		}
	})

	t.Run("invalid_jwt_config", func(t *testing.T) {
		os.Setenv("JWT_ACCESS_TOKEN_EXPIRY", "abc")
		defer os.Unsetenv("JWT_ACCESS_TOKEN_EXPIRY")
		_, err := LoadConfig()
		if err == nil {
			t.Error("LoadConfig should fail on invalid jwt config")
		}
	})

	t.Run("invalid_db_config", func(t *testing.T) {
		os.Setenv("DB_MAX_CONNS", "abc")
		defer os.Unsetenv("DB_MAX_CONNS")
		_, err := LoadConfig()
		if err == nil {
			t.Error("LoadConfig should fail on invalid db config")
		}
	})

	t.Run("invalid_ratelimit_config", func(t *testing.T) {
		os.Setenv("RATE_LIMIT_PER_IP", "abc")
		defer os.Unsetenv("RATE_LIMIT_PER_IP")
		_, err := LoadConfig()
		if err == nil {
			t.Error("LoadConfig should fail on invalid rate limit config")
		}
	})

	t.Run("invalid_cors_config", func(t *testing.T) {
		os.Setenv("CORS_MAX_AGE", "abc")
		defer os.Unsetenv("CORS_MAX_AGE")
		_, err := LoadConfig()
		if err == nil {
			t.Error("LoadConfig should fail on invalid cors config")
		}
	})

	t.Run("invalid_csrf_config", func(t *testing.T) {
		os.Setenv("CSRF_TOKEN_LENGTH", "abc")
		defer os.Unsetenv("CSRF_TOKEN_LENGTH")
		_, err := LoadConfig()
		if err == nil {
			t.Error("LoadConfig should fail on invalid csrf config")
		}
	})
}

func TestLoadConfig_ValidateFails(t *testing.T) {
	// Provide valid parsable values, but miss a required one for Validate()
	os.Setenv("DB_WRITE_HOST", "host")
	os.Setenv("DB_NAME", "db")
	os.Setenv("DB_USER", "user")
	// Missing JWT_SECRET
	defer func() {
		os.Unsetenv("DB_WRITE_HOST")
		os.Unsetenv("DB_NAME")
		os.Unsetenv("DB_USER")
	}()

	_, err := LoadConfig()
	if err == nil {
		t.Error("LoadConfig should fail when validation fails, but it did not")
	}
}

func TestValidate_MissingJWTSecret(t *testing.T) {
	cfg := &Config{
		JWT: JWTConfig{Secret: ""},
		Database: DatabaseConfig{
			WriteHost: "localhost",
			Database:  "testdb",
			Username:  "user",
		},
	}
	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() should fail when JWT_SECRET is missing")
	}
	if err.Error() != "JWT_SECRET is required" {
		t.Errorf("Expected 'JWT_SECRET is required', got %v", err)
	}
}

func TestValidate_MissingDBWriteHost(t *testing.T) {
	cfg := &Config{
		JWT: JWTConfig{Secret: "secret"},
		Database: DatabaseConfig{
			WriteHost: "",
			Database:  "testdb",
			Username:  "user",
		},
	}
	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() should fail when DB_WRITE_HOST is missing")
	}
	if err.Error() != "DB_WRITE_HOST is required" {
		t.Errorf("Expected 'DB_WRITE_HOST is required', got %v", err)
	}
}

func TestValidate_MissingDBName(t *testing.T) {
	cfg := &Config{
		JWT: JWTConfig{Secret: "secret"},
		Database: DatabaseConfig{
			WriteHost: "localhost",
			Database:  "",
			Username:  "user",
		},
	}
	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() should fail when DB_NAME is missing")
	}
	if err.Error() != "DB_NAME is required" {
		t.Errorf("Expected 'DB_NAME is required', got %v", err)
	}
}

func TestValidate_MissingDBUser(t *testing.T) {
	cfg := &Config{
		JWT: JWTConfig{Secret: "secret"},
		Database: DatabaseConfig{
			WriteHost: "localhost",
			Database:  "testdb",
			Username:  "",
		},
	}
	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() should fail when DB_USER is missing")
	}
	if err.Error() != "DB_USER is required" {
		t.Errorf("Expected 'DB_USER is required', got %v", err)
	}
}

func TestValidate_Success(t *testing.T) {
	cfg := &Config{
		JWT: JWTConfig{Secret: "secret"},
		Database: DatabaseConfig{
			WriteHost: "localhost",
			Database:  "testdb",
			Username:  "user",
		},
	}
	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validate() should succeed with all required fields, got error: %v", err)
	}
}

func TestLoadJWTConfig_InvalidRefreshTokenExpiry(t *testing.T) {
	os.Setenv("JWT_REFRESH_TOKEN_EXPIRY", "invalid")
	defer os.Unsetenv("JWT_REFRESH_TOKEN_EXPIRY")
	_, err := loadJWTConfig()
	if err == nil {
		t.Error("loadJWTConfig() should fail with invalid refresh token expiry")
	}
}

func TestLoadJWTConfig_InvalidAccessTokenExpiry(t *testing.T) {
	os.Setenv("JWT_ACCESS_TOKEN_EXPIRY", "invalid")
	defer os.Unsetenv("JWT_ACCESS_TOKEN_EXPIRY")
	_, err := loadJWTConfig()
	if err == nil {
		t.Error("loadJWTConfig() should fail with invalid access token expiry")
	}
}

func TestLoadDatabaseConfig_ReadHostsParsing(t *testing.T) {
	os.Setenv("DB_READ_HOSTS", "replica1.example.com, replica2.example.com , replica3.example.com")
	defer os.Unsetenv("DB_READ_HOSTS")

	cfg, err := loadDatabaseConfig()
	if err != nil {
		t.Fatalf("loadDatabaseConfig() failed: %v", err)
	}

	if len(cfg.ReadHosts) != 3 {
		t.Errorf("Expected 3 read hosts, got %d", len(cfg.ReadHosts))
	}

	expected := []string{"replica1.example.com", "replica2.example.com", "replica3.example.com"}
	for i, host := range cfg.ReadHosts {
		if host != expected[i] {
			t.Errorf("ReadHosts[%d] = %q, want %q", i, host, expected[i])
		}
	}
}

func TestLoadDatabaseConfig_EmptyReadHosts(t *testing.T) {
	os.Unsetenv("DB_READ_HOSTS")
	cfg, err := loadDatabaseConfig()
	if err != nil {
		t.Fatalf("loadDatabaseConfig() failed: %v", err)
	}

	if len(cfg.ReadHosts) != 0 {
		t.Errorf("Expected 0 read hosts when env not set, got %d", len(cfg.ReadHosts))
	}
}

func TestLoadDatabaseConfig_SSLMode(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		want     string
	}{
		{"default_disable", "", "disable"},
		{"require", "require", "require"},
		{"prefer", "prefer", "prefer"},
		{"verify-ca", "verify-ca", "verify-ca"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv("DB_SSL_MODE", tt.envValue)
				defer os.Unsetenv("DB_SSL_MODE")
			} else {
				os.Unsetenv("DB_SSL_MODE")
			}

			cfg, err := loadDatabaseConfig()
			if err != nil {
				t.Fatalf("loadDatabaseConfig() failed: %v", err)
			}

			if cfg.SSLMode != tt.want {
				t.Errorf("SSLMode = %q, want %q", cfg.SSLMode, tt.want)
			}
		})
	}
}

func TestLoadRateLimitConfig_InvalidPerIP(t *testing.T) {
	os.Setenv("RATE_LIMIT_PER_IP", "not-a-number")
	defer os.Unsetenv("RATE_LIMIT_PER_IP")
	_, err := loadRateLimitConfig()
	if err == nil {
		t.Error("loadRateLimitConfig() should fail with invalid per IP value")
	}
}

func TestLoadRateLimitConfig_InvalidPerUser(t *testing.T) {
	os.Setenv("RATE_LIMIT_PER_USER", "not-a-number")
	defer os.Unsetenv("RATE_LIMIT_PER_USER")
	_, err := loadRateLimitConfig()
	if err == nil {
		t.Error("loadRateLimitConfig() should fail with invalid per user value")
	}
}

func TestLoadRateLimitConfig_FullConfig(t *testing.T) {
	os.Setenv("RATE_LIMIT_ENABLED", "false")
	os.Setenv("RATE_LIMIT_PER_IP", "50")
	os.Setenv("RATE_LIMIT_PER_USER", "100")
	os.Setenv("RATE_LIMIT_RESET_PERIOD", "30m")
	defer func() {
		os.Unsetenv("RATE_LIMIT_ENABLED")
		os.Unsetenv("RATE_LIMIT_PER_IP")
		os.Unsetenv("RATE_LIMIT_PER_USER")
		os.Unsetenv("RATE_LIMIT_RESET_PERIOD")
	}()

	cfg, err := loadRateLimitConfig()
	if err != nil {
		t.Fatalf("loadRateLimitConfig() failed: %v", err)
	}

	if cfg.Enabled != false {
		t.Errorf("Enabled = %v, want false", cfg.Enabled)
	}
	if cfg.PerIP != 50 {
		t.Errorf("PerIP = %d, want 50", cfg.PerIP)
	}
	if cfg.PerUser != 100 {
		t.Errorf("PerUser = %d, want 100", cfg.PerUser)
	}
	if cfg.ResetPeriod != 30*time.Minute {
		t.Errorf("ResetPeriod = %v, want 30m", cfg.ResetPeriod)
	}
}

func TestLoadCSRFConfig_ExemptPathsParsing(t *testing.T) {
	os.Setenv("CSRF_EXEMPT_PATHS", "/api/webhook, /api/public , /health")
	defer os.Unsetenv("CSRF_EXEMPT_PATHS")

	cfg, err := loadCSRFConfig()
	if err != nil {
		t.Fatalf("loadCSRFConfig() failed: %v", err)
	}

	if len(cfg.ExemptPaths) != 3 {
		t.Errorf("Expected 3 exempt paths, got %d", len(cfg.ExemptPaths))
	}

	expected := []string{"/api/webhook", "/api/public", "/health"}
	for i, path := range cfg.ExemptPaths {
		if path != expected[i] {
			t.Errorf("ExemptPaths[%d] = %q, want %q", i, path, expected[i])
		}
	}
}

func TestLoadCSRFConfig_EmptyExemptPaths(t *testing.T) {
	os.Unsetenv("CSRF_EXEMPT_PATHS")
	cfg, err := loadCSRFConfig()
	if err != nil {
		t.Fatalf("loadCSRFConfig() failed: %v", err)
	}

	if len(cfg.ExemptPaths) != 0 {
		t.Errorf("Expected 0 exempt paths when env not set, got %d", len(cfg.ExemptPaths))
	}
}

func TestLoadCSRFConfig_FullConfig(t *testing.T) {
	os.Setenv("CSRF_ENABLED", "false")
	os.Setenv("CSRF_TOKEN_LENGTH", "64")
	os.Setenv("CSRF_COOKIE_NAME", "custom_csrf")
	os.Setenv("CSRF_HEADER_NAME", "X-Custom-CSRF")
	defer func() {
		os.Unsetenv("CSRF_ENABLED")
		os.Unsetenv("CSRF_TOKEN_LENGTH")
		os.Unsetenv("CSRF_COOKIE_NAME")
		os.Unsetenv("CSRF_HEADER_NAME")
	}()

	cfg, err := loadCSRFConfig()
	if err != nil {
		t.Fatalf("loadCSRFConfig() failed: %v", err)
	}

	if cfg.Enabled != false {
		t.Errorf("Enabled = %v, want false", cfg.Enabled)
	}
	if cfg.TokenLength != 64 {
		t.Errorf("TokenLength = %d, want 64", cfg.TokenLength)
	}
	if cfg.CookieName != "custom_csrf" {
		t.Errorf("CookieName = %q, want 'custom_csrf'", cfg.CookieName)
	}
	if cfg.HeaderName != "X-Custom-CSRF" {
		t.Errorf("HeaderName = %q, want 'X-Custom-CSRF'", cfg.HeaderName)
	}
}
