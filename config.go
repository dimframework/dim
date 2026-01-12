package dim

import (
	"fmt"
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
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// JWTConfig holds JWT configuration
type JWTConfig struct {
	Secret             string
	AccessTokenExpiry  time.Duration
	RefreshTokenExpiry time.Duration
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
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

// EmailConfig holds email configuration
type EmailConfig struct {
	From string
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
	AllowCredentials bool
	MaxAge           int
}

// CSRFConfig holds CSRF configuration
type CSRFConfig struct {
	Enabled     bool
	ExemptPaths []string
	TokenLength int
	CookieName  string
	HeaderName  string
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

	cfg := &Config{
		Server:    serverCfg,
		JWT:       jwtCfg,
		Database:  dbCfg,
		Email:     loadEmailConfig(),
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

	return ServerConfig{
		Port:         GetEnvOrDefault("SERVER_PORT", "8080"),
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
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

	return JWTConfig{
		Secret:             GetEnv("JWT_SECRET"),
		AccessTokenExpiry:  accessTokenExpiry,
		RefreshTokenExpiry: refreshTokenExpiry,
	}, nil
}

// loadDatabaseConfig loads database configuration
func loadDatabaseConfig() (DatabaseConfig, error) {
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
func loadEmailConfig() EmailConfig {
	return EmailConfig{
		From: GetEnv("EMAIL_FROM"),
	}
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

	maxAge, err := ParseEnvInt(GetEnvOrDefault("CORS_MAX_AGE", "3600"))
	if err != nil {
		return CORSConfig{}, fmt.Errorf("invalid CORS_MAX_AGE: %w", err)
	}

	return CORSConfig{
		AllowedOrigins:   origins,
		AllowedMethods:   methods,
		AllowedHeaders:   headers,
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

	return CSRFConfig{
		Enabled:     ParseEnvBool(GetEnvOrDefault("CSRF_ENABLED", "true")),
		ExemptPaths: exemptPaths,
		TokenLength: tokenLength,
		CookieName:  GetEnvOrDefault("CSRF_COOKIE_NAME", "csrf_token"),
		HeaderName:  GetEnvOrDefault("CSRF_HEADER_NAME", "X-CSRF-Token"),
	}, nil
}

// Validate memvalidasi konfigurasi aplikasi untuk memastikan nilai required sudah ada.
// Mengecek JWT_SECRET, DB_WRITE_HOST, DB_NAME, dan DB_USER.
func (c *Config) Validate() error {
	if c.JWT.Secret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}

	if c.Database.WriteHost == "" {
		return fmt.Errorf("DB_WRITE_HOST is required")
	}

	if c.Database.Database == "" {
		return fmt.Errorf("DB_NAME is required")
	}

	if c.Database.Username == "" {
		return fmt.Errorf("DB_USER is required")
	}

	return nil
}
