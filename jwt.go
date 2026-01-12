package dim

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTManager handles JWT operations
type JWTManager struct {
	secret             string
	accessTokenExpiry  time.Duration
	refreshTokenExpiry time.Duration
}

// NewJWTManager membuat JWT manager baru dengan konfigurasi yang diberikan.
//
// Parameters:
//   - config: JWTConfig berisi secret key dan expiry times untuk tokens
//
// Returns:
//   - *JWTManager: instance manager yang siap generate dan verify JWT tokens
//
// Example:
//
//	manager := NewJWTManager(config)
func NewJWTManager(config *JWTConfig) *JWTManager {
	return &JWTManager{
		secret:             config.Secret,
		accessTokenExpiry:  config.AccessTokenExpiry,
		refreshTokenExpiry: config.RefreshTokenExpiry,
	}
}

// GenerateAccessToken membuat access token JWT baru untuk user dengan expiry yang sudah dikonfigurasi.
// Token berisi userID, email, dan claims tambahan.
//
// Parameters:
//   - userID: ID unik dari pengguna
//   - email: alamat email pengguna
//   - extraClaims: claims tambahan yang ingin disertakan (opsional)
//
// Returns:
//   - string: signed JWT access token
//   - error: error jika gagal generate atau sign token
//
// Example:
//
//	token, err := manager.GenerateAccessToken("123", "user@example.com", map[string]interface{}{"role": "admin"})
func (m *JWTManager) GenerateAccessToken(userID string, email string, extraClaims map[string]interface{}) (string, error) {
	now := time.Now()
	expiresAt := now.Add(m.accessTokenExpiry)

	claims := jwt.MapClaims{
		"sub":   userID,
		"email": email,
		"iat":   now.Unix(),
		"exp":   expiresAt.Unix(),
		"nbf":   now.Unix(),
	}

	// Add extra claims
	for k, v := range extraClaims {
		claims[k] = v
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(m.secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// GenerateRefreshToken membuat refresh token JWT baru untuk user dengan expiry lebih panjang.
// Token hanya berisi userID sebagai subject claim.
//
// Parameters:
//   - userID: ID unik dari pengguna
//
// Returns:
//   - string: signed JWT refresh token
//   - error: error jika gagal generate atau sign token
//
// Example:
//
//	token, err := manager.GenerateRefreshToken("123")
func (m *JWTManager) GenerateRefreshToken(userID string) (string, error) {
	now := time.Now()
	expiresAt := now.Add(m.refreshTokenExpiry)

	claims := jwt.RegisteredClaims{
		Subject:   userID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
		NotBefore: jwt.NewNumericDate(now),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(m.secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// VerifyToken memverifikasi access token dan mengembalikan claims di dalamnya.
// Melakukan validasi signature, expiry, dan token validity.
//
// Parameters:
//   - tokenString: JWT token string yang akan diverifikasi
//
// Returns:
//   - jwt.MapClaims: claims dari token key-value map
//   - error: error jika token tidak valid, expired, atau signature tidak cocok
//
// Example:
//
//	claims, err := manager.VerifyToken(tokenString)
//	if err != nil {
//	  return err
//	}
func (m *JWTManager) VerifyToken(tokenString string) (jwt.MapClaims, error) {
	claims := jwt.MapClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(m.secret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

// VerifyRefreshToken memverifikasi refresh token dan mengembalikan userID.
// Melakukan validasi signature, expiry, dan token validity.
//
// Parameters:
//   - tokenString: JWT refresh token string yang akan diverifikasi
//
// Returns:
//   - string: user ID dari subject claim
//   - error: error jika token tidak valid, expired, signature tidak cocok, atau parse userID gagal
//
// Example:
//
//	userID, err := manager.VerifyRefreshToken(tokenString)
func (m *JWTManager) VerifyRefreshToken(tokenString string) (string, error) {
	var claims jwt.RegisteredClaims

	token, err := jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(m.secret), nil
	})

	if err != nil {
		return "", fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return "", fmt.Errorf("invalid token")
	}

	return claims.Subject, nil
}

// GetTokenExpiry mengembalikan waktu expiry dari token.
// Bisa menangani token yang expired dengan mengambil expiry time dari claims.
//
// Parameters:
//   - tokenString: JWT token string
//
// Returns:
//   - time.Time: waktu expiry dari token
//   - error: error jika parse token gagal atau token tidak memiliki expiry
//
// Example:
//
//	expiryTime, err := manager.GetTokenExpiry(tokenString)
func (m *JWTManager) GetTokenExpiry(tokenString string) (time.Time, error) {
	claims := &jwt.RegisteredClaims{}

	_, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(m.secret), nil
	})

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
// Menggunakan bcrypt hashing untuk keamanan, bukan actual token.
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
	hash, _ := HashPassword(token)
	return hash
}

// VerifyTokenHash memverifikasi token terhadap hash yang tersimpan di database.
// Menggunakan bcrypt compare untuk aman.
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
	return VerifyPassword(hash, token)
}
