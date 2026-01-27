package dim

import (
	"context"
	"fmt"
	"time"
)

// LoginRequest merepresentasikan data yang dibutuhkan untuk login.
type LoginRequest struct {
	Email    string
	Password string
}

// TokenResponse merepresentasikan respons token setelah login berhasil.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// AuthUserStore mendefinisikan interface yang dibutuhkan oleh AuthService
// untuk berinteraksi dengan penyimpanan data pengguna.
type AuthUserStore interface {
	FindByEmail(ctx context.Context, email string) (Authenticatable, error)
	FindByID(ctx context.Context, id string) (Authenticatable, error)
	Update(ctx context.Context, user Authenticatable) error
}

// ClaimsProvider adalah fungsi yang mengembalikan custom claims untuk pengguna.
// Gunakan ini untuk menyisipkan data tambahan ke dalam JWT (seperti workspace_id, role, dll).
type ClaimsProvider func(ctx context.Context, user Authenticatable) (map[string]interface{}, error)

// AuthService menangani operasi otentikasi seperti login, register, dan manajemen token.
type AuthService struct {
	userStore      AuthUserStore
	tokenStore     TokenStore
	blocklist      TokenBlocklist
	jwtManager     *JWTManager
	pwValidator    *PasswordValidator
	claimsProvider ClaimsProvider
	logger         *Logger
}

// NewAuthService membuat instance AuthService baru.
func NewAuthService(
	userStore AuthUserStore,
	tokenStore TokenStore,
	blocklist TokenBlocklist,
	jwtConfig *JWTConfig,
) (*AuthService, error) {
	jwtManager, err := NewJWTManager(jwtConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to init jwt manager: %w", err)
	}
	return &AuthService{
		userStore:   userStore,
		tokenStore:  tokenStore,
		blocklist:   blocklist,
		jwtManager:  jwtManager,
		pwValidator: NewPasswordValidator(),
	}, nil
}

// WithClaimsProvider mengatur function provider untuk custom claims dan mengembalikan instance service.
// Method ini menggunakan pola chaining untuk memudahkan konfigurasi.
func (s *AuthService) WithClaimsProvider(provider ClaimsProvider) *AuthService {
	s.claimsProvider = provider
	return s
}

// WithLogger mengatur logger untuk AuthService dan mengembalikan instance service.
// Logger digunakan untuk mencatat internal errors yang tidak dikirim ke client.
// Method ini menggunakan pola chaining untuk memudahkan konfigurasi.
//
// Example:
//
//	authService, _ := dim.NewAuthService(userStore, tokenStore, blocklist, jwtConfig)
//	authService.WithLogger(logger).WithClaimsProvider(claimsProvider)
func (s *AuthService) WithLogger(logger *Logger) *AuthService {
	s.logger = logger
	return s
}

// Login mengotentikasi pengguna menggunakan email dan password.
// Mengembalikan access token dan refresh token jika kredensial valid.
//
// Parameters:
//   - ctx: context request
//   - email: email pengguna
//   - password: password pengguna
//
// Returns:
//   - string: access token
//   - string: refresh token
//   - error: error jika kredensial tidak valid atau terjadi kesalahan server
func (s *AuthService) Login(ctx context.Context, email, password string) (string, string, error) {
	// Validate input
	v := NewValidator().
		Required("email", email).
		Email("email", email).
		Required("password", password)

	if !v.IsValid() {
		err := NewAppError("Kredensial tidak valid", 401)
		return "", "", err
	}

	// Find user by email
	user, err := s.userStore.FindByEmail(ctx, email)
	if err != nil {
		return "", "", NewAppError("Kredensial tidak valid", 401)
	}

	// Verify password
	if err := VerifyPassword(user.GetPassword(), password); err != nil {
		return "", "", NewAppError("Kredensial tidak valid", 401)
	}

	// Get custom claims
	var extraClaims map[string]interface{}
	if s.claimsProvider != nil {
		var err error
		extraClaims, err = s.claimsProvider(ctx, user)
		if err != nil {
			return "", "", NewAppError("Gagal membuat claims", 500)
		}
	}

	// Generate Session ID (UUID)
	sessionID := NewUuid().String()

	// Generate tokens
	accessToken, err := s.jwtManager.GenerateAccessToken(user.GetID(), user.GetEmail(), sessionID, extraClaims)
	if err != nil {
		return "", "", NewAppError("Gagal membuat access token", 500)
	}

	refreshToken, err := s.jwtManager.GenerateRefreshToken(user.GetID(), sessionID)
	if err != nil {
		return "", "", NewAppError("Gagal membuat refresh token", 500)
	}

	// Store refresh token hash
	refreshTokenHash := GenerateTokenHash(refreshToken)
	refreshTokenEntity := &RefreshToken{
		UserID:    user.GetID(),
		TokenHash: refreshTokenHash,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour).UTC().Truncate(time.Second),
	}

	if err := s.tokenStore.SaveRefreshToken(ctx, refreshTokenEntity); err != nil {
		return "", "", NewAppError("Gagal menyimpan refresh token", 500)
	}

	return accessToken, refreshToken, nil
}

// RefreshToken memperbarui access token menggunakan refresh token yang valid.
// Method ini akan membatalkan refresh token lama dan mengeluarkan pasangan token baru (Token Rotation).
//
// Parameters:
//   - ctx: context request
//   - refreshTokenStr: string refresh token yang dikirim oleh client
//
// Returns:
//   - string: access token baru
//   - string: refresh token baru
//   - error: error jika token tidak valid, kadaluarsa, atau sudah dibatalkan
func (s *AuthService) RefreshToken(ctx context.Context, refreshTokenStr string) (string, string, error) {
	// Verify refresh token
	userID, sessionID, err := s.jwtManager.VerifyRefreshToken(refreshTokenStr)
	if err != nil {
		// Log internal error jika logger tersedia
		if s.logger != nil {
			s.logger.Warn("Refresh token verification failed", "error", err.Error())
		}
		return "", "", NewAppError("Refresh token tidak valid", 401)
	}

	// Check if token is in the database and not revoked
	refreshTokenHash := GenerateTokenHash(refreshTokenStr)
	storedToken, err := s.tokenStore.FindRefreshToken(ctx, refreshTokenHash)
	if err != nil {
		return "", "", NewAppError("Refresh token tidak valid", 401)
	}

	// Check if token is revoked
	if storedToken.RevokedAt != nil {
		return "", "", NewAppError("Token telah dibatalkan (revoked)", 401)
	}

	// Check if token has expired
	if time.Now().After(storedToken.ExpiresAt) {
		return "", "", NewAppError("Token telah kadaluarsa", 401)
	}

	// Get user info
	user, err := s.userStore.FindByID(ctx, userID)
	if err != nil {
		return "", "", NewAppError("Pengguna tidak ditemukan", 404)
	}

	// Get custom claims
	var extraClaims map[string]interface{}
	if s.claimsProvider != nil {
		extraClaims, err = s.claimsProvider(ctx, user)
		if err != nil {
			return "", "", NewAppError("Gagal membuat claims", 500)
		}
	}

	// Generate new access token
	newAccessToken, err := s.jwtManager.GenerateAccessToken(user.GetID(), user.GetEmail(), sessionID, extraClaims)
	if err != nil {
		return "", "", NewAppError("Gagal membuat access token", 500)
	}

	// Generate new refresh token
	newRefreshToken, err := s.jwtManager.GenerateRefreshToken(user.GetID(), sessionID)
	if err != nil {
		return "", "", NewAppError("Gagal membuat refresh token", 500)
	}

	// Revoke old refresh token
	_ = s.tokenStore.RevokeRefreshToken(ctx, refreshTokenHash)

	// Store new refresh token hash
	newRefreshTokenHash := GenerateTokenHash(newRefreshToken)
	newRefreshTokenEntity := &RefreshToken{
		UserID:    user.GetID(),
		TokenHash: newRefreshTokenHash,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour).UTC().Truncate(time.Second),
	}

	if err := s.tokenStore.SaveRefreshToken(ctx, newRefreshTokenEntity); err != nil {
		return "", "", NewAppError("Gagal menyimpan refresh token", 500)
	}

	return newAccessToken, newRefreshToken, nil
}

// RequestPasswordReset memproses permintaan reset password.
// Akan membuat token reset dan menyimpannya (pengiriman email dilakukan oleh pemanggil).
// Mengembalikan token reset yang belum di-hash agar bisa dikirim ke user.
func (s *AuthService) RequestPasswordReset(ctx context.Context, email string) (string, error) {
	// Validate email
	v := NewValidator().
		Required("email", email).
		Email("email", email)

	if !v.IsValid() {
		err := NewAppError("Validasi gagal", 400)
		err.Errors = v.ErrorMap()
		return "", err
	}

	// Find user by email
	user, err := s.userStore.FindByEmail(ctx, email)
	if err != nil {
		// Don't reveal if email exists (security best practice)
		return "", nil
	}

	// Generate reset token
	resetToken, err := GenerateSecureToken(32)
	if err != nil {
		return "", NewAppError("Gagal membuat token reset", 500)
	}

	// Store reset token hash
	resetTokenHash := GenerateTokenHash(resetToken)
	resetTokenEntity := &PasswordResetToken{
		UserID:    user.GetID(),
		TokenHash: resetTokenHash,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	if err := s.tokenStore.SavePasswordResetToken(ctx, resetTokenEntity); err != nil {
		return "", NewAppError("Gagal menyimpan token reset", 500)
	}

	return resetToken, nil
}

// ResetPassword mereset password pengguna menggunakan token reset yang valid.
// Setelah berhasil, semua refresh token pengguna akan dihapus untuk alasan keamanan.
func (s *AuthService) ResetPassword(ctx context.Context, resetTokenStr, newPassword string) error {
	// Validate input
	v := NewValidator().
		Required("password", newPassword)

	if !v.IsValid() {
		err := NewAppError("Validasi gagal", 400)
		err.Errors = v.ErrorMap()
		return err
	}

	// Validate password strength
	if err := s.pwValidator.Validate(newPassword); err != nil {
		return err
	}

	// Find reset token
	resetTokenHash := GenerateTokenHash(resetTokenStr)
	resetToken, err := s.tokenStore.FindPasswordResetToken(ctx, resetTokenHash)
	if err != nil {
		return NewAppError("Token reset tidak valid atau kadaluarsa", 400)
	}

	// Check if token is expired
	if time.Now().After(resetToken.ExpiresAt) {
		return NewAppError("Token reset telah kadaluarsa", 400)
	}

	// Check if token was already used
	if resetToken.UsedAt != nil {
		return NewAppError("Token reset sudah pernah digunakan", 400)
	}

	// Get user
	user, err := s.userStore.FindByID(ctx, resetToken.UserID)
	if err != nil {
		return NewAppError("Pengguna tidak ditemukan", 404)
	}

	// Hash new password
	passwordHash, err := HashPassword(newPassword)
	if err != nil {
		return NewAppError("Gagal memproses password hash", 500)
	}

	// Update user password
	user.SetPassword(passwordHash)
	if err := s.userStore.Update(ctx, user); err != nil {
		return NewAppError("Gagal memperbarui password", 500)
	}

	// Mark reset token as used
	if err := s.tokenStore.MarkPasswordResetUsed(ctx, resetTokenHash); err != nil {
		return NewAppError("Gagal menandai token reset", 500)
	}

	// Revoke all user's refresh tokens for security
	_ = s.tokenStore.RevokeAllUserTokens(ctx, user.GetID())

	return nil
}

// Logout mengeluarkan pengguna dengan membatalkan (revoke) refresh token mereka.
// Karena kita menggunakan Session ID (sid), kita juga mem-blacklist sid tersebut
// agar Access Token yang masih hidup (yang memiliki sid sama) ikut tidak valid.
func (s *AuthService) Logout(ctx context.Context, refreshTokenStr string) error {
	if refreshTokenStr == "" {
		return NewAppError("Refresh token diperlukan", 400)
	}

	// 1. Dapatkan Session ID dari Refresh Token
	// Kita abaikan userID karena tidak digunakan disini
	_, sid, err := s.jwtManager.VerifyRefreshToken(refreshTokenStr)
	if err != nil {
		// Log internal error jika logger tersedia
		if s.logger != nil {
			s.logger.Warn("Logout: refresh token verification failed", "error", err.Error())
		}
		return NewAppError("Refresh token tidak valid atau expired", 400)
	}

	// 2. Blacklist Session ID jika ada
	if sid != "" && s.blocklist != nil {
		// Kita set expiry blocklist sama dengan durasi Access Token (misal 15 menit),
		// atau cukup lama. Karena kita tidak tahu persis sisa umur akses tokennya,
		// kita gunakan default expiry yang aman, misal durasi access token default system.
		// Untuk amannya, kita set 1 jam.
		if err := s.blocklist.Invalidate(ctx, sid, 1*time.Hour); err != nil {
			// Log error tapi jangan gagalkan logout, lanjut ke revoke refresh token
			if s.logger != nil {
				s.logger.Warn("Failed to blacklist session", "session_id", sid, "error", err.Error())
			}
		}
	}

	// 3. Revoke refresh token (Standard Procedure)
	refreshTokenHash := GenerateTokenHash(refreshTokenStr)
	if err := s.tokenStore.RevokeRefreshToken(ctx, refreshTokenHash); err != nil {
		return NewAppError("Gagal logout", 500)
	}

	return nil
}
