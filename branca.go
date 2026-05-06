package dim

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"golang.org/x/crypto/chacha20poly1305"
)

const (
	brancaVersion     = byte(0xBA)
	brancaHeaderSize  = 29 // 1 (version) + 4 (timestamp) + 24 (nonce)
	brancaBase62Alpha = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	brancaAccessType  = "at+branca"
	brancaRefreshType = "rt+branca"
)

// BrancaConfig holds configuration for BrancaManager.
type BrancaConfig struct {
	// Key is a 32-byte symmetric key encoded as hex (64 chars) or base64.
	// Generate with: openssl rand -hex 32
	Key                string
	AccessTokenExpiry  time.Duration
	RefreshTokenExpiry time.Duration
}

// BrancaManager implements TokenManager using Branca tokens (XChaCha20-Poly1305 encryption).
// Unlike JWT, Branca tokens encrypt the payload — claims are not readable by the client.
type BrancaManager struct {
	config *BrancaConfig
	key    []byte // must be exactly 32 bytes
}

// NewBrancaManager creates a new BrancaManager from the given config.
// The key must decode to exactly 32 bytes (hex or base64 encoded).
func NewBrancaManager(config *BrancaConfig) (*BrancaManager, error) {
	key, err := decodeBrancaKey(config.Key)
	if err != nil {
		return nil, fmt.Errorf("branca: invalid key: %w", err)
	}
	if len(key) != chacha20poly1305.KeySize {
		return nil, fmt.Errorf("branca: key must be %d bytes, got %d", chacha20poly1305.KeySize, len(key))
	}
	return &BrancaManager{config: config, key: key}, nil
}

// GenerateAccessToken creates an encrypted Branca access token with the given claims.
func (m *BrancaManager) GenerateAccessToken(userID, email, sessionID string, extraClaims map[string]interface{}) (string, error) {
	now := time.Now()
	claims := map[string]interface{}{
		"sub":   userID,
		"sid":   sessionID,
		"jti":   NewUuid().String(),
		"email": email,
		"iat":   now.Unix(),
		"exp":   now.Add(m.config.AccessTokenExpiry).Unix(),
		"nbf":   now.Unix(),
		"typ":   brancaAccessType,
	}
	for k, v := range extraClaims {
		claims[k] = v
	}
	return m.encode(claims, now)
}

// GenerateRefreshToken creates an encrypted Branca refresh token.
func (m *BrancaManager) GenerateRefreshToken(userID, sessionID string) (string, error) {
	now := time.Now()
	claims := map[string]interface{}{
		"sub": userID,
		"sid": sessionID,
		"jti": NewUuid().String(),
		"iat": now.Unix(),
		"exp": now.Add(m.config.RefreshTokenExpiry).Unix(),
		"nbf": now.Unix(),
		"typ": brancaRefreshType,
	}
	return m.encode(claims, now)
}

// VerifyToken verifies a Branca access token and returns its claims.
func (m *BrancaManager) VerifyToken(tokenString string) (TokenClaims, error) {
	claims, err := m.decodeAndValidate(tokenString)
	if err != nil {
		return nil, err
	}

	typ, _ := claims["typ"].(string)
	if typ != brancaAccessType {
		return nil, fmt.Errorf("branca: invalid token type: expected access token")
	}

	return claims, nil
}

// VerifyRefreshToken verifies a Branca refresh token and returns userID and sessionID.
func (m *BrancaManager) VerifyRefreshToken(tokenString string) (string, string, error) {
	claims, err := m.decodeAndValidate(tokenString)
	if err != nil {
		return "", "", err
	}

	typ, _ := claims["typ"].(string)
	if typ != brancaRefreshType {
		return "", "", fmt.Errorf("branca: invalid token type: expected refresh token")
	}

	sub, ok := claims["sub"].(string)
	if !ok {
		return "", "", fmt.Errorf("branca: missing sub claim")
	}

	sid, _ := claims["sid"].(string)
	return sub, sid, nil
}

// GetTokenExpiry returns the expiry time embedded in the token's claims.
func (m *BrancaManager) GetTokenExpiry(tokenString string) (time.Time, error) {
	claims, err := m.decrypt(tokenString)
	if err != nil {
		return time.Time{}, err
	}

	exp, ok := claims["exp"].(float64)
	if !ok {
		return time.Time{}, fmt.Errorf("branca: token has no exp claim")
	}
	return time.Unix(int64(exp), 0), nil
}

// IsTokenExpired checks whether a Branca token is expired.
func (m *BrancaManager) IsTokenExpired(tokenString string) (bool, error) {
	expiry, err := m.GetTokenExpiry(tokenString)
	if err != nil {
		return false, err
	}
	return time.Now().After(expiry), nil
}

// encode serializes claims as JSON and encrypts them into a Branca token string.
func (m *BrancaManager) encode(claims map[string]interface{}, ts time.Time) (string, error) {
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("branca: failed to marshal claims: %w", err)
	}

	var nonce [24]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return "", fmt.Errorf("branca: failed to generate nonce: %w", err)
	}

	// Header: version(1) + timestamp(4, big-endian) + nonce(24)
	header := make([]byte, brancaHeaderSize)
	header[0] = brancaVersion
	binary.BigEndian.PutUint32(header[1:5], uint32(ts.Unix()))
	copy(header[5:], nonce[:])

	aead, err := chacha20poly1305.NewX(m.key)
	if err != nil {
		return "", fmt.Errorf("branca: cipher init failed: %w", err)
	}

	// Header is the additional authenticated data
	ciphertext := aead.Seal(nil, nonce[:], payload, header)

	token := make([]byte, 0, brancaHeaderSize+len(ciphertext))
	token = append(token, header...)
	token = append(token, ciphertext...)

	return brancaBase62Encode(token), nil
}

// decrypt decrypts a Branca token and returns raw claims without expiry validation.
func (m *BrancaManager) decrypt(tokenString string) (map[string]interface{}, error) {
	raw, err := brancaBase62Decode(tokenString)
	if err != nil {
		return nil, fmt.Errorf("branca: invalid token encoding: %w", err)
	}
	if len(raw) < brancaHeaderSize {
		return nil, fmt.Errorf("branca: token too short")
	}
	if raw[0] != brancaVersion {
		return nil, fmt.Errorf("branca: unsupported version: %x", raw[0])
	}

	header := raw[:brancaHeaderSize]
	nonce := raw[5:brancaHeaderSize]
	ciphertext := raw[brancaHeaderSize:]

	aead, err := chacha20poly1305.NewX(m.key)
	if err != nil {
		return nil, fmt.Errorf("branca: cipher init failed: %w", err)
	}

	plaintext, err := aead.Open(nil, nonce, ciphertext, header)
	if err != nil {
		return nil, fmt.Errorf("branca: decryption failed (invalid token or wrong key)")
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(plaintext, &claims); err != nil {
		return nil, fmt.Errorf("branca: invalid token payload: %w", err)
	}

	return claims, nil
}

// decodeAndValidate decrypts the token and validates exp/nbf claims.
func (m *BrancaManager) decodeAndValidate(tokenString string) (map[string]interface{}, error) {
	claims, err := m.decrypt(tokenString)
	if err != nil {
		return nil, err
	}

	now := time.Now().Unix()

	if exp, ok := claims["exp"].(float64); ok {
		if now > int64(exp) {
			return nil, fmt.Errorf("branca: token has expired")
		}
	}

	if nbf, ok := claims["nbf"].(float64); ok {
		if now < int64(nbf) {
			return nil, fmt.Errorf("branca: token not yet valid")
		}
	}

	return claims, nil
}

// decodeBrancaKey accepts hex (64 chars → 32 bytes), base64, or a raw 32-byte string.
func decodeBrancaKey(s string) ([]byte, error) {
	if s == "" {
		return nil, fmt.Errorf("key is empty")
	}

	// Try hex (64 hex chars = 32 bytes)
	if len(s) == 64 {
		if b, err := hex.DecodeString(s); err == nil {
			return b, nil
		}
	}

	// Try standard base64
	if b, err := base64.StdEncoding.DecodeString(s); err == nil {
		return b, nil
	}

	// Try base64 URL encoding (no padding)
	if b, err := base64.RawURLEncoding.DecodeString(s); err == nil {
		return b, nil
	}

	// Use raw bytes if exactly 32
	if len(s) == 32 {
		return []byte(s), nil
	}

	return nil, fmt.Errorf("key must decode to 32 bytes (provide hex, base64, or 32-char string)")
}

// brancaBase62Encode encodes bytes to base62 using the Branca alphabet.
func brancaBase62Encode(data []byte) string {
	n := new(big.Int).SetBytes(data)
	base := big.NewInt(62)
	zero := big.NewInt(0)
	mod := new(big.Int)

	var result []byte
	for n.Cmp(zero) > 0 {
		n.DivMod(n, base, mod)
		result = append(result, brancaBase62Alpha[mod.Int64()])
	}

	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return string(result)
}

// brancaBase62Decode decodes a base62 string back to bytes.
func brancaBase62Decode(s string) ([]byte, error) {
	n := new(big.Int)
	base := big.NewInt(62)

	for _, c := range s {
		idx := strings.IndexRune(brancaBase62Alpha, c)
		if idx < 0 {
			return nil, fmt.Errorf("invalid character: %c", c)
		}
		n.Mul(n, base)
		n.Add(n, big.NewInt(int64(idx)))
	}

	return n.Bytes(), nil
}
