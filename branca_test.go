package dim

import (
	"testing"
	"time"
)

// testBrancaKey returns a valid 32-byte hex key for testing.
func testBrancaKey() string {
	return "0000000000000000000000000000000000000000000000000000000000000001"
}

func testBrancaManager(t *testing.T) *BrancaManager {
	t.Helper()
	cfg := &BrancaConfig{
		Key:                testBrancaKey(),
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	}
	m, err := NewBrancaManager(cfg)
	if err != nil {
		t.Fatalf("NewBrancaManager failed: %v", err)
	}
	return m
}

func TestNewBrancaManager_ValidHexKey(t *testing.T) {
	_, err := NewBrancaManager(&BrancaConfig{
		Key:                testBrancaKey(),
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 168 * time.Hour,
	})
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestNewBrancaManager_InvalidKey(t *testing.T) {
	_, err := NewBrancaManager(&BrancaConfig{Key: "tooshort"})
	if err == nil {
		t.Error("expected error for invalid key, got nil")
	}
}

func TestNewBrancaManager_EmptyKey(t *testing.T) {
	_, err := NewBrancaManager(&BrancaConfig{Key: ""})
	if err == nil {
		t.Error("expected error for empty key, got nil")
	}
}

func TestBrancaManager_ImplementsTokenManager(t *testing.T) {
	m := testBrancaManager(t)
	var _ TokenManager = m
}

func TestBrancaManager_GenerateAndVerifyAccessToken(t *testing.T) {
	m := testBrancaManager(t)

	token, err := m.GenerateAccessToken("user123", "user@example.com", "session456", nil)
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	claims, err := m.VerifyToken(token)
	if err != nil {
		t.Fatalf("VerifyToken failed: %v", err)
	}

	if claims["sub"] != "user123" {
		t.Errorf("sub = %v, want user123", claims["sub"])
	}
	if claims["email"] != "user@example.com" {
		t.Errorf("email = %v, want user@example.com", claims["email"])
	}
	if claims["sid"] != "session456" {
		t.Errorf("sid = %v, want session456", claims["sid"])
	}
	if claims["typ"] != brancaAccessType {
		t.Errorf("typ = %v, want %s", claims["typ"], brancaAccessType)
	}
}

func TestBrancaManager_GenerateAndVerifyRefreshToken(t *testing.T) {
	m := testBrancaManager(t)

	token, err := m.GenerateRefreshToken("user123", "session456")
	if err != nil {
		t.Fatalf("GenerateRefreshToken failed: %v", err)
	}

	userID, sessionID, err := m.VerifyRefreshToken(token)
	if err != nil {
		t.Fatalf("VerifyRefreshToken failed: %v", err)
	}
	if userID != "user123" {
		t.Errorf("userID = %s, want user123", userID)
	}
	if sessionID != "session456" {
		t.Errorf("sessionID = %s, want session456", sessionID)
	}
}

func TestBrancaManager_VerifyToken_RejectsRefreshToken(t *testing.T) {
	m := testBrancaManager(t)

	refresh, _ := m.GenerateRefreshToken("user123", "session456")
	_, err := m.VerifyToken(refresh)
	if err == nil {
		t.Error("VerifyToken should reject a refresh token")
	}
}

func TestBrancaManager_VerifyRefreshToken_RejectsAccessToken(t *testing.T) {
	m := testBrancaManager(t)

	access, _ := m.GenerateAccessToken("user123", "email@example.com", "session456", nil)
	_, _, err := m.VerifyRefreshToken(access)
	if err == nil {
		t.Error("VerifyRefreshToken should reject an access token")
	}
}

func TestBrancaManager_VerifyToken_InvalidToken(t *testing.T) {
	m := testBrancaManager(t)
	_, err := m.VerifyToken("notavalidtoken")
	if err == nil {
		t.Error("expected error for invalid token")
	}
}

func TestBrancaManager_VerifyToken_WrongKey(t *testing.T) {
	m1 := testBrancaManager(t)
	m2, _ := NewBrancaManager(&BrancaConfig{
		Key:                "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 168 * time.Hour,
	})

	token, _ := m1.GenerateAccessToken("user123", "email@example.com", "sid", nil)
	_, err := m2.VerifyToken(token)
	if err == nil {
		t.Error("expected decryption failure with wrong key")
	}
}

func TestBrancaManager_ExtraClaimsIncluded(t *testing.T) {
	m := testBrancaManager(t)

	extra := map[string]interface{}{"role": "admin", "workspace": "ws-1"}
	token, _ := m.GenerateAccessToken("user123", "email@example.com", "sid", extra)

	claims, err := m.VerifyToken(token)
	if err != nil {
		t.Fatalf("VerifyToken failed: %v", err)
	}
	if claims["role"] != "admin" {
		t.Errorf("role = %v, want admin", claims["role"])
	}
	if claims["workspace"] != "ws-1" {
		t.Errorf("workspace = %v, want ws-1", claims["workspace"])
	}
}

func TestBrancaManager_GetTokenExpiry(t *testing.T) {
	m := testBrancaManager(t)

	// exp is stored as Unix seconds so truncate to second precision for comparison
	before := time.Now().Add(15 * time.Minute).Truncate(time.Second)
	token, _ := m.GenerateAccessToken("user123", "email@example.com", "sid", nil)
	after := time.Now().Add(15 * time.Minute).Add(time.Second)

	expiry, err := m.GetTokenExpiry(token)
	if err != nil {
		t.Fatalf("GetTokenExpiry failed: %v", err)
	}
	if expiry.Before(before) || expiry.After(after) {
		t.Errorf("expiry %v out of expected range [%v, %v]", expiry, before, after)
	}
}

func TestBrancaManager_IsTokenExpired_NotExpired(t *testing.T) {
	m := testBrancaManager(t)
	token, _ := m.GenerateAccessToken("user123", "email@example.com", "sid", nil)

	expired, err := m.IsTokenExpired(token)
	if err != nil {
		t.Fatalf("IsTokenExpired failed: %v", err)
	}
	if expired {
		t.Error("token should not be expired")
	}
}

func TestBrancaManager_ExpiredToken(t *testing.T) {
	cfg := &BrancaConfig{
		Key:                testBrancaKey(),
		AccessTokenExpiry:  -1 * time.Second, // already expired
		RefreshTokenExpiry: 168 * time.Hour,
	}
	m, _ := NewBrancaManager(cfg)

	token, _ := m.GenerateAccessToken("user123", "email@example.com", "sid", nil)
	_, err := m.VerifyToken(token)
	if err == nil {
		t.Error("expected error for expired token")
	}
}

func TestDecodeBrancaKey_Hex(t *testing.T) {
	key, err := decodeBrancaKey(testBrancaKey())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(key) != 32 {
		t.Errorf("key length = %d, want 32", len(key))
	}
}

func TestDecodeBrancaKey_Empty(t *testing.T) {
	_, err := decodeBrancaKey("")
	if err == nil {
		t.Error("expected error for empty key")
	}
}

func TestLoadBrancaConfig_Defaults(t *testing.T) {
	cfg, err := loadBrancaConfig()
	if err != nil {
		t.Fatalf("loadBrancaConfig failed: %v", err)
	}
	if cfg.AccessTokenExpiry != 15*time.Minute {
		t.Errorf("AccessTokenExpiry = %v, want 15m", cfg.AccessTokenExpiry)
	}
	if cfg.RefreshTokenExpiry != 168*time.Hour {
		t.Errorf("RefreshTokenExpiry = %v, want 168h", cfg.RefreshTokenExpiry)
	}
}
