package dim

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTManager handles JWT operations
type JWTManager struct {
	config         *JWTConfig
	signingKey     interface{}            // []byte for HMAC, *rsa.PrivateKey for RSA
	validationKeys map[string]interface{} // map[kid]PublicKey (or []byte for HMAC rotation)
}

// NewJWTManager membuat JWT manager baru dengan konfigurasi yang diberikan.
// Membaca konfigurasi Signing Method dan kunci-kunci yang diperlukan.
//
// Parameters:
//   - config: pointer ke struct JWTConfig yang berisi preferensi signing dan kunci
//
// Returns:
//   - *JWTManager: instance manager yang siap digunakan
//   - error: error jika parsing kunci gagal atau konfigurasi tidak valid
func NewJWTManager(config *JWTConfig) (*JWTManager, error) {
	manager := &JWTManager{
		config:         config,
		validationKeys: make(map[string]interface{}),
	}

	// 1. Parse Signing Key based on Method
	switch {
	case strings.HasPrefix(config.SigningMethod, "HS"):
		if config.HMACSecret == "" {
			return nil, fmt.Errorf("HMAC secret is required for %s", config.SigningMethod)
		}
		manager.signingKey = []byte(config.HMACSecret)
		// For HMAC, signing key is also validation key (Symmetric)
		manager.validationKeys["default"] = []byte(config.HMACSecret)

	case strings.HasPrefix(config.SigningMethod, "RS"):
		if config.PrivateKey != "" {
			key, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(config.PrivateKey))
			if err != nil {
				return nil, fmt.Errorf("failed to parse RSA private key: %w", err)
			}
			manager.signingKey = key
			// Extract Public Key for verification
			manager.validationKeys["default"] = &key.PublicKey
		}

	case strings.HasPrefix(config.SigningMethod, "ES"):
		if config.PrivateKey != "" {
			key, err := jwt.ParseECPrivateKeyFromPEM([]byte(config.PrivateKey))
			if err != nil {
				return nil, fmt.Errorf("failed to parse ECDSA private key: %w", err)
			}
			manager.signingKey = key
			manager.validationKeys["default"] = &key.PublicKey
		}
	}

	// 2. Parse Old Public Keys (Rotation)
	for kid, pemStr := range config.PublicKeys {
		if strings.HasPrefix(config.SigningMethod, "RS") {
			key, err := jwt.ParseRSAPublicKeyFromPEM([]byte(pemStr))
			if err != nil {
				return nil, fmt.Errorf("failed to parse public key %s: %w", kid, err)
			}
			manager.validationKeys[kid] = key
		} else if strings.HasPrefix(config.SigningMethod, "ES") {
			key, err := jwt.ParseECPublicKeyFromPEM([]byte(pemStr))
			if err != nil {
				return nil, fmt.Errorf("failed to parse public key %s: %w", kid, err)
			}
			manager.validationKeys[kid] = key
		}
	}

	return manager, nil
}

// GenerateAccessToken membuat access token JWT baru untuk user dengan expiry yang sudah dikonfigurasi.
// Token ditandatangani menggunakan metode dan kunci yang aktif saat ini.
//
// Parameters:
//   - userID: ID unik pengguna (disimpan dalam claim 'sub')
//   - email: email pengguna (disimpan dalam claim 'email')
//   - sessionID: ID unik sesi (disimpan dalam claim 'sid')
//   - extraClaims: map tambahan claims custom yang ingin dimasukkan
//
// Returns:
//   - string: signed JWT string
//   - error: error jika signing gagal
func (m *JWTManager) GenerateAccessToken(userID string, email string, sessionID string, extraClaims map[string]interface{}) (string, error) {
	now := time.Now()
	expiresAt := now.Add(m.config.AccessTokenExpiry)

	claims := jwt.MapClaims{
		"sub":   userID,
		"sid":   sessionID,
		"jti":   NewUuid().String(),
		"email": email,
		"iat":   now.Unix(),
		"exp":   expiresAt.Unix(),
		"nbf":   now.Unix(),
	}

	// Add extra claims
	for k, v := range extraClaims {
		claims[k] = v
	}

	// Determine Signing Method
	method := jwt.GetSigningMethod(m.config.SigningMethod)
	if method == nil {
		return "", fmt.Errorf("invalid signing method: %s", m.config.SigningMethod)
	}

	token := jwt.NewWithClaims(method, claims)

	// Jika menggunakan Asymmetric, tambahkan 'kid' ke header jika logika sistem membutuhkannya.
	// Logika saat ini menggunakan "default" yang berasal dari PrivateKey. Implementasi nyata mungkin menggunakan KID tertentu.
	// Demi kesederhanaan, kita tidak menyetel KID di header untuk saat ini kecuali kita mengimplementasikan manajemen rotasi penuh untuk Signing Key.
	// Namun untuk pencocokan validasi, jika kita memiliki beberapa kunci validasi, kita memerlukan KID.
	// Ini hanya jika kita berada dalam fase rotasi. Untuk saat ini, mari kita buat sederhana.
	// Standar JWT menggunakan header 'kid'.
	// Kita asumsikan kunci aktif saat ini tidak memiliki KID tertentu atau "default".

	return token.SignedString(m.signingKey)
}

// GenerateRefreshToken membuat refresh token JWT baru untuk user dengan expiry lebih panjang.
// Digunakan untuk mendapatkan access token baru tanpa login ulang.
//
// Parameters:
//   - userID: ID unik pengguna (disimpan dalam claim 'sub')
//   - sessionID: ID unik sesi (disimpan dalam claim 'sid')
//
// Returns:
//   - string: signed JWT string
//   - error: error jika signing gagal
func (m *JWTManager) GenerateRefreshToken(userID, sessionID string) (string, error) {
	now := time.Now()
	expiresAt := now.Add(m.config.RefreshTokenExpiry)

	// Gunakan MapClaims agar bisa menambahkan custom claim 'sid'
	claims := jwt.MapClaims{
		"sub": userID,
		"sid": sessionID,
		"jti": NewUuid().String(),
		"iat": now.Unix(),
		"exp": expiresAt.Unix(),
		"nbf": now.Unix(),
	}

	// Determine Signing Method
	method := jwt.GetSigningMethod(m.config.SigningMethod)
	if method == nil {
		return "", fmt.Errorf("invalid signing method: %s", m.config.SigningMethod)
	}

	token := jwt.NewWithClaims(method, claims)

	return token.SignedString(m.signingKey)
}

// verifyKeyFunc validates the token method and selects the correct key.
func (m *JWTManager) verifyKeyFunc(token *jwt.Token) (interface{}, error) {
	// 1. Validate Algorithm family
	switch {
	case strings.HasPrefix(m.config.SigningMethod, "HS"):
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
	case strings.HasPrefix(m.config.SigningMethod, "RS"):
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
	case strings.HasPrefix(m.config.SigningMethod, "ES"):
		if _, ok := token.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
	default:
		return nil, fmt.Errorf("unsupported signing method config: %s", m.config.SigningMethod)
	}

	// 2. Select Key (Support Rotation)
	if kid, ok := token.Header["kid"].(string); ok {
		if key, ok := m.validationKeys[kid]; ok {
			return key, nil
		}
		// Usually if KID is specified, one SHOULD match.
		// But if we only have default key and no headers, we fallback.
	}

	// Fallback to default key (current active key)
	if key, ok := m.validationKeys["default"]; ok {
		return key, nil
	}

	return nil, fmt.Errorf("no verification key available")
}

// VerifyToken memverifikasi access token dan mengembalikan claims di dalamnya.
// Mendukung rotasi kunci melalui header 'kid'.
//
// Parameters:
//   - tokenString: raw JWT string yang diterima dari client
//
// Returns:
//   - jwt.MapClaims: klaim-klaim yang ada di dalam token jika valid
//   - error: error jika signature tidak valid, token kedaluwarsa, atau format salah
func (m *JWTManager) VerifyToken(tokenString string) (jwt.MapClaims, error) {
	claims := jwt.MapClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, m.verifyKeyFunc)

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

// VerifyRefreshToken memverifikasi refresh token dan mengembalikan userID dan sessionID.
// Memastikan token valid dan belum kedaluwarsa.
//
// Parameters:
//   - tokenString: raw JWT string
//
// Returns:
//   - string: userID yang tersimpan dalam claim 'sub'
//   - string: sessionID yang tersimpan dalam claim 'sid'
//   - error: error jika token tidak valid
func (m *JWTManager) VerifyRefreshToken(tokenString string) (string, string, error) {
	// Gunakan MapClaims karena kita menggunakan sid (custom claim)
	claims := jwt.MapClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, m.verifyKeyFunc)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return "", "", fmt.Errorf("invalid token")
	}

	sub, ok := claims["sub"].(string)
	if !ok {
		return "", "", fmt.Errorf("invalid token claims: missing sub")
	}

	sid, ok := claims["sid"].(string)
	if !ok {
		sid = ""
	}

	return sub, sid, nil
}

// GetTokenExpiry mengembalikan waktu expiry dari token.
// Berguna untuk pengecekan sisi client atau logika refresh otomatis.
//
// Parameters:
//   - tokenString: raw JWT string
//
// Returns:
//   - time.Time: waktu kapan token tersebut expired
//   - error: error jika parsing token gagal
func (m *JWTManager) GetTokenExpiry(tokenString string) (time.Time, error) {
	claims := &jwt.RegisteredClaims{}

	_, err := jwt.ParseWithClaims(tokenString, claims, m.verifyKeyFunc)

	// Handle both parsing errors and expired tokens
	if err != nil {
		// Check if token is expired (claims will still be populated for expired tokens in v5)
		if claims.ExpiresAt != nil && strings.Contains(err.Error(), "expired") {
			// Token is expired, return the expiry time
			return claims.ExpiresAt.Time, nil
		}
		return time.Time{}, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims.ExpiresAt == nil {
		return time.Time{}, fmt.Errorf("token has no expiry")
	}

	return claims.ExpiresAt.Time, nil
}

// IsTokenExpired mengecek apakah token sudah expired atau tidak.
//
// Parameters:
//   - tokenString: JWT token string
//
// Returns:
//   - bool: true jika token sudah expired, false jika masih valid
//   - error: error jika parse token gagal
//
// Example:
//
//	isExpired, err := manager.IsTokenExpired(tokenString)
//	if isExpired {
//	  // get new token
//	}
func (m *JWTManager) IsTokenExpired(tokenString string) (bool, error) {
	expiry, err := m.GetTokenExpiry(tokenString)
	if err != nil {
		return false, err
	}

	return time.Now().After(expiry), nil
}

// GenerateTokenHash membuat hash dari token untuk disimpan di database.
// Menggunakan SHA256 hashing (deterministik) agar bisa di-lookup di database.
//
// Parameters:
//   - token: token string yang akan di-hash
//
// Returns:
//   - string: hashed token yang bisa disimpan di database
//
// Example:
//
//	tokenHash := GenerateTokenHash(refreshToken)
//	// store tokenHash in database instead of actual token
func GenerateTokenHash(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// VerifyTokenHash memverifikasi token terhadap hash yang tersimpan di database.
//
// Parameters:
//   - hash: hashed token dari database
//   - token: actual token string untuk diverifikasi
//
// Returns:
//   - error: error jika token tidak cocok dengan hash
//
// Example:
//
//	err := VerifyTokenHash(storedHash, token)
//	if err != nil {
//	  return "token tidak valid"
//	}
func VerifyTokenHash(hash, token string) error {
	expected := GenerateTokenHash(token)
	if hash != expected {
		return fmt.Errorf("token invalid")
	}
	return nil
}
