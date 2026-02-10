package dim

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// Config holds all application configuration
type Config struct {
	Server    ServerConfig
	JWT       JWTConfig
	Database  DatabaseConfig
	Email     EmailConfig
	RateLimit RateLimitConfig
	CORS      CORSConfig
	CSRF      CSRFConfig
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Port            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

// JWTConfig holds JWT configuration
type JWTConfig struct {
	AccessTokenExpiry  time.Duration
	RefreshTokenExpiry time.Duration

	// Algorithm configuration
	SigningMethod string // "HS256" (default), "RS256", "ES256"

	// Symmetric Config (HMAC: HS256, HS384, HS512)
	HMACSecret string

	// Asymmetric Config (RSA/ECDSA: RS256, ES256)
	PrivateKey string            // PEM content for Signing
	PublicKeys map[string]string // Key ID (kid) -> PEM content Public Key (for rotation)

	// Remote Verification (JWKS)
	JWKSURL string
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Driver        string // "postgres" or "sqlite"
	WriteHost     string
	ReadHosts     []string
	Port          int
	Database      string
	Username      string
	Password      string
	MaxConns      int
	SSLMode       string            // SSL mode: "disable", "require", "prefer", "allow", "verify-ca", "verify-full" (default: "disable")
	RuntimeParams map[string]string // Custom runtime parameters (search_path, standard_conforming_strings, etc)
	QueryExecMode string            // Query execution mode: "simple" or "" (default)
}

// EmailConfig holds email configuration and branding settings.
type EmailConfig struct {
	// From is the default sender email address.
	From string

	// Transport is the mail delivery method: "smtp", "ses", or "null" (default: "null").
	Transport string

	// SMTP Configuration (required if Transport is "smtp")
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string

	// SES Configuration (required if Transport is "ses")
	SESRegion           string
	SESAccessKeyID      string
	SESSecretAccessKey  string
	SESConfigurationSet string

	// Branding settings for email templates
	AppName      string // Application name shown in emails (default: "App")
	LogoURL      string // URL to application logo (optional)
	PrimaryColor string // Primary brand color in hex (default: "#007bff")
	SupportEmail string // Support contact email (optional)
	SupportURL   string // Support website URL (optional)
	CompanyName  string // Company name for footer (optional)
	SocialLinks  string // JSON array of SocialLink objects (optional)

	// BaseURL is the application root URL, required for generating action links (e.g. password reset).
	BaseURL string
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	Enabled     bool
	PerIP       int
	PerUser     int
	ResetPeriod time.Duration
}

// CORSConfig holds CORS configuration
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

// CSRFConfig holds CSRF configuration
type CSRFConfig struct {
	Enabled      bool
	ExemptPaths  []string
	TokenLength  int
	CookieName   string
	HeaderName   string
	CookieMaxAge int
}

// LoadConfig memuat konfigurasi aplikasi dari environment variables.
// Menggabungkan konfigurasi dari semua bagian (Server, JWT, Database, Email, RateLimit, CORS, CSRF).
//
// Returns:
//   - *Config: struktur konfigurasi lengkap aplikasi
//   - error: error jika validasi konfigurasi gagal
//
// Example:
//
//	config, err := LoadConfig()
//	if err != nil {
//	  log.Fatal(err)
//	}
func LoadConfig() (*Config, error) {
	serverCfg, err := loadServerConfig()
	if err != nil {
		return nil, err
	}

	jwtCfg, err := loadJWTConfig()
	if err != nil {
		return nil, err
	}

	dbCfg, err := loadDatabaseConfig()
	if err != nil {
		return nil, err
	}

	rateLimitCfg, err := loadRateLimitConfig()
	if err != nil {
		return nil, err
	}

	corsCfg, err := loadCORSConfig()
	if err != nil {
		return nil, err
	}

	csrfCfg, err := loadCSRFConfig()
	if err != nil {
		return nil, err
	}

	emailCfg, err := loadEmailConfig()
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		Server:    serverCfg,
		JWT:       jwtCfg,
		Database:  dbCfg,
		Email:     emailCfg,
		RateLimit: rateLimitCfg,
		CORS:      corsCfg,
		CSRF:      csrfCfg,
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// loadServerConfig loads server configuration
func loadServerConfig() (ServerConfig, error) {
	readTimeout, err := ParseEnvDuration(GetEnvOrDefault("SERVER_READ_TIMEOUT", "30s"))
	if err != nil {
		return ServerConfig{}, fmt.Errorf("invalid SERVER_READ_TIMEOUT: %w", err)
	}

	writeTimeout, err := ParseEnvDuration(GetEnvOrDefault("SERVER_WRITE_TIMEOUT", "30s"))
	if err != nil {
		return ServerConfig{}, fmt.Errorf("invalid SERVER_WRITE_TIMEOUT: %w", err)
	}

	idleTimeout, err := ParseEnvDuration(GetEnvOrDefault("SERVER_IDLE_TIMEOUT", "120s"))
	if err != nil {
		return ServerConfig{}, fmt.Errorf("invalid SERVER_IDLE_TIMEOUT: %w", err)
	}

	shutdownTimeout, err := ParseEnvDuration(GetEnvOrDefault("SERVER_SHUTDOWN_TIMEOUT", "10s"))
	if err != nil {
		return ServerConfig{}, fmt.Errorf("invalid SERVER_SHUTDOWN_TIMEOUT: %w", err)
	}

	return ServerConfig{
		Port:            GetEnvOrDefault("SERVER_PORT", "8080"),
		ReadTimeout:     readTimeout,
		WriteTimeout:    writeTimeout,
		IdleTimeout:     idleTimeout,
		ShutdownTimeout: shutdownTimeout,
	}, nil
}

// loadJWTConfig loads JWT configuration
func loadJWTConfig() (JWTConfig, error) {
	accessTokenExpiry, err := ParseEnvDuration(GetEnvOrDefault("JWT_ACCESS_TOKEN_EXPIRY", "15m"))
	if err != nil {
		return JWTConfig{}, fmt.Errorf("invalid JWT_ACCESS_TOKEN_EXPIRY: %w", err)
	}

	refreshTokenExpiry, err := ParseEnvDuration(GetEnvOrDefault("JWT_REFRESH_TOKEN_EXPIRY", "168h"))
	if err != nil {
		return JWTConfig{}, fmt.Errorf("invalid JWT_REFRESH_TOKEN_EXPIRY: %w", err)
	}

	signingMethod := GetEnvOrDefault("JWT_SIGNING_METHOD", "HS256")
	hmacSecret := GetEnv("JWT_SECRET")
	privateKey := resolveKeyContent(GetEnv("JWT_PRIVATE_KEY"))
	jwksURL := GetEnv("JWT_JWKS_URL")

	// Parse Public Keys (JSON format: {"kid1": "pem1", "kid2": "pem2"})
	publicKeys := make(map[string]string)
	publicKeysStr := GetEnv("JWT_PUBLIC_KEYS")
	if publicKeysStr != "" {
		if err := json.Unmarshal([]byte(publicKeysStr), &publicKeys); err != nil {
			return JWTConfig{}, fmt.Errorf("invalid JWT_PUBLIC_KEYS format (expected JSON): %w", err)
		}
		// Resolve file paths for public keys if necessary
		for k, v := range publicKeys {
			publicKeys[k] = resolveKeyContent(v)
		}
	}

	return JWTConfig{
		AccessTokenExpiry:  accessTokenExpiry,
		RefreshTokenExpiry: refreshTokenExpiry,
		SigningMethod:      signingMethod,
		HMACSecret:         hmacSecret,
		PrivateKey:         privateKey,
		PublicKeys:         publicKeys,
		JWKSURL:            jwksURL,
	}, nil
}

// resolveKeyContent checks if the value is a file path and reads it,
// otherwise returns the value as is (assuming it's PEM content).
func resolveKeyContent(val string) string {
	if val == "" {
		return ""
	}
	// Check if it's already a PEM string (starts with -----BEGIN)
	if strings.HasPrefix(strings.TrimSpace(val), "-----BEGIN") {
		return val
	}

	// Try to read as file
	b, err := os.ReadFile(val)
	if err == nil {
		return string(b)
	}

	// If failed to read file, assume it's the content (or invalid path)
	return val
}

// loadDatabaseConfig loads database configuration
func loadDatabaseConfig() (DatabaseConfig, error) {
	driver := GetEnvOrDefault("DB_DRIVER", "postgres")

	readHostsStr := GetEnv("DB_READ_HOSTS")
	readHosts := []string{}
	if readHostsStr != "" {
		readHosts = strings.Split(readHostsStr, ",")
		for i := range readHosts {
			readHosts[i] = strings.TrimSpace(readHosts[i])
		}
	}

	port, err := ParseEnvInt(GetEnvOrDefault("DB_PORT", "5432"))
	if err != nil {
		return DatabaseConfig{}, fmt.Errorf("invalid DB_PORT: %w", err)
	}

	maxConns, err := ParseEnvInt(GetEnvOrDefault("DB_MAX_CONNS", "25"))
	if err != nil {
		return DatabaseConfig{}, fmt.Errorf("invalid DB_MAX_CONNS: %w", err)
	}

	return DatabaseConfig{
		Driver:        driver,
		WriteHost:     GetEnv("DB_WRITE_HOST"),
		ReadHosts:     readHosts,
		Port:          port,
		Database:      GetEnv("DB_NAME"),
		Username:      GetEnv("DB_USER"),
		Password:      GetEnv("DB_PASSWORD"),
		MaxConns:      maxConns,
		SSLMode:       GetEnvOrDefault("DB_SSL_MODE", "disable"),
		RuntimeParams: make(map[string]string),
		QueryExecMode: "",
	}, nil
}

// loadEmailConfig loads email configuration
func loadEmailConfig() (EmailConfig, error) {
	smtpPort, err := ParseEnvInt(GetEnvOrDefault("MAIL_SMTP_PORT", "587"))
	if err != nil {
		return EmailConfig{}, fmt.Errorf("invalid MAIL_SMTP_PORT: %w", err)
	}

	// SES Config Loading with Fallbacks
	sesRegion := GetEnv("AWS_REGION")
	if sesRegion == "" {
		sesRegion = GetEnv("SES_REGION")
	}

	sesAccessKey := GetEnv("AWS_ACCESS_KEY_ID")
	if sesAccessKey == "" {
		sesAccessKey = GetEnv("SES_ACCESS_KEY_ID")
	}

	sesSecretKey := GetEnv("AWS_SECRET_ACCESS_KEY")
	if sesSecretKey == "" {
		sesSecretKey = GetEnv("SES_SECRET_ACCESS_KEY")
	}

	return EmailConfig{
		From:                GetEnv("MAIL_FROM"),
		Transport:           GetEnvOrDefault("MAIL_TRANSPORT", "null"),
		SMTPHost:            GetEnv("MAIL_SMTP_HOST"),
		SMTPPort:            smtpPort,
		SMTPUsername:        GetEnv("MAIL_SMTP_USERNAME"),
		SMTPPassword:        GetEnv("MAIL_SMTP_PASSWORD"),
		SESRegion:           sesRegion,
		SESAccessKeyID:      sesAccessKey,
		SESSecretAccessKey:  sesSecretKey,
		SESConfigurationSet: GetEnv("SES_CONFIGURATION_SET"),
		AppName:             GetEnvOrDefault("MAIL_APP_NAME", "App"),
		LogoURL:             GetEnv("MAIL_LOGO_URL"),
		PrimaryColor:        GetEnvOrDefault("MAIL_PRIMARY_COLOR", "#007bff"),
		SupportEmail:        GetEnv("MAIL_SUPPORT_EMAIL"),
		SupportURL:          GetEnv("MAIL_SUPPORT_URL"),
		CompanyName:         GetEnv("MAIL_COMPANY_NAME"),
		SocialLinks:         GetEnv("MAIL_SOCIAL_LINKS"),
		BaseURL:             GetEnv("APP_BASE_URL"),
	}, nil
}

// loadRateLimitConfig loads rate limiting configuration
func loadRateLimitConfig() (RateLimitConfig, error) {
	perIP, err := ParseEnvInt(GetEnvOrDefault("RATE_LIMIT_PER_IP", "100"))
	if err != nil {
		return RateLimitConfig{}, fmt.Errorf("invalid RATE_LIMIT_PER_IP: %w", err)
	}

	perUser, err := ParseEnvInt(GetEnvOrDefault("RATE_LIMIT_PER_USER", "200"))
	if err != nil {
		return RateLimitConfig{}, fmt.Errorf("invalid RATE_LIMIT_PER_USER: %w", err)
	}

	resetPeriod, err := ParseEnvDuration(GetEnvOrDefault("RATE_LIMIT_RESET_PERIOD", "1h"))
	if err != nil {
		return RateLimitConfig{}, fmt.Errorf("invalid RATE_LIMIT_RESET_PERIOD: %w", err)
	}

	return RateLimitConfig{
		Enabled:     ParseEnvBool(GetEnvOrDefault("RATE_LIMIT_ENABLED", "true")),
		PerIP:       perIP,
		PerUser:     perUser,
		ResetPeriod: resetPeriod,
	}, nil
}

// loadCORSConfig loads CORS configuration
func loadCORSConfig() (CORSConfig, error) {
	originsStr := GetEnvOrDefault("CORS_ALLOWED_ORIGINS", "http://localhost:3000")
	origins := strings.Split(originsStr, ",")
	for i := range origins {
		origins[i] = strings.TrimSpace(origins[i])
	}

	methodsStr := GetEnvOrDefault("CORS_ALLOWED_METHODS", "GET,POST,PUT,DELETE,PATCH,OPTIONS")
	methods := strings.Split(methodsStr, ",")
	for i := range methods {
		methods[i] = strings.TrimSpace(methods[i])
	}

	headersStr := GetEnvOrDefault("CORS_ALLOWED_HEADERS", "Content-Type,Authorization,X-CSRF-Token")
	headers := strings.Split(headersStr, ",")
	for i := range headers {
		headers[i] = strings.TrimSpace(headers[i])
	}

	exposedHeadersStr := GetEnvOrDefault("CORS_EXPOSED_HEADERS", "")
	exposedHeaders := []string{}
	if exposedHeadersStr != "" {
		parts := strings.Split(exposedHeadersStr, ",")
		for _, p := range parts {
			trimmed := strings.TrimSpace(p)
			if trimmed != "" {
				exposedHeaders = append(exposedHeaders, trimmed)
			}
		}
	}

	maxAge, err := ParseEnvInt(GetEnvOrDefault("CORS_MAX_AGE", "3600"))
	if err != nil {
		return CORSConfig{}, fmt.Errorf("invalid CORS_MAX_AGE: %w", err)
	}

	return CORSConfig{
		AllowedOrigins:   origins,
		AllowedMethods:   methods,
		AllowedHeaders:   headers,
		ExposedHeaders:   exposedHeaders,
		AllowCredentials: ParseEnvBool(GetEnvOrDefault("CORS_ALLOW_CREDENTIALS", "true")),
		MaxAge:           maxAge,
	}, nil
}

// loadCSRFConfig loads CSRF configuration
func loadCSRFConfig() (CSRFConfig, error) {
	exemptPathsStr := GetEnv("CSRF_EXEMPT_PATHS")
	exemptPaths := []string{}
	if exemptPathsStr != "" {
		exemptPaths = strings.Split(exemptPathsStr, ",")
		for i := range exemptPaths {
			exemptPaths[i] = strings.TrimSpace(exemptPaths[i])
		}
	}

	tokenLength, err := ParseEnvInt(GetEnvOrDefault("CSRF_TOKEN_LENGTH", "32"))
	if err != nil {
		return CSRFConfig{}, fmt.Errorf("invalid CSRF_TOKEN_LENGTH: %w", err)
	}

	cookieMaxAge, err := ParseEnvInt(GetEnvOrDefault("CSRF_COOKIE_MAX_AGE", "43200")) // Default 12 jam
	if err != nil {
		return CSRFConfig{}, fmt.Errorf("invalid CSRF_COOKIE_MAX_AGE: %w", err)
	}

	return CSRFConfig{
		Enabled:      ParseEnvBool(GetEnvOrDefault("CSRF_ENABLED", "true")),
		ExemptPaths:  exemptPaths,
		TokenLength:  tokenLength,
		CookieName:   GetEnvOrDefault("CSRF_COOKIE_NAME", "csrf_token"),
		HeaderName:   GetEnvOrDefault("CSRF_HEADER_NAME", "X-CSRF-Token"),
		CookieMaxAge: cookieMaxAge,
	}, nil
}

// Validate memvalidasi konfigurasi aplikasi untuk memastikan nilai required sudah ada.
// Mengecek JWT_SECRET, DB_WRITE_HOST, DB_NAME, dan DB_USER.
func (c *Config) Validate() error {
	if strings.HasPrefix(c.JWT.SigningMethod, "HS") {
		if c.JWT.HMACSecret == "" {
			return fmt.Errorf("JWT_SECRET is required for HMAC signing method")
		}
	} else if strings.HasPrefix(c.JWT.SigningMethod, "RS") || strings.HasPrefix(c.JWT.SigningMethod, "ES") {
		if c.JWT.PrivateKey == "" && c.JWT.JWKSURL == "" {
			// Jika pakai asymmetric, minimal butuh Private Key (untuk sign) ATAU JWKS (untuk verify saja)
			return fmt.Errorf("JWT_PRIVATE_KEY is required for RSA/ECDSA signing method")
		}
	}

	if c.Database.Database == "" {
		return fmt.Errorf("DB_NAME is required")
	}

	// Validation specific to Postgres
	if c.Database.Driver == "postgres" {
		if c.Database.WriteHost == "" {
			return fmt.Errorf("DB_WRITE_HOST is required for postgres")
		}
		if c.Database.Username == "" {
			return fmt.Errorf("DB_USER is required for postgres")
		}
	}

	// Email Validation
	if c.Email.Transport != "null" && c.Email.Transport != "" {
		if c.Email.From == "" {
			return fmt.Errorf("MAIL_FROM is required when mail transport is enabled")
		}
		if c.Email.Transport == "smtp" {
			if c.Email.SMTPHost == "" {
				return fmt.Errorf("MAIL_SMTP_HOST is required for SMTP transport")
			}
		}
		if c.Email.Transport == "ses" {
			if c.Email.SESRegion == "" {
				return fmt.Errorf("AWS_REGION (or SES_REGION) is required for SES transport")
			}
		}
	}

	return nil
}
