package dim

import (
	"context"
	"time"
)

// AuthService handles authentication operations
type AuthService struct {
	userStore   UserStore
	tokenStore  TokenStore
	jwtManager  *JWTManager
	pwValidator *PasswordValidator
}

// NewAuthService creates a new auth service
func NewAuthService(
	userStore UserStore,
	tokenStore TokenStore,
	jwtConfig *JWTConfig,
) *AuthService {
	return &AuthService{
		userStore:   userStore,
		tokenStore:  tokenStore,
		jwtManager:  NewJWTManager(jwtConfig),
		pwValidator: NewPasswordValidator(),
	}
}

// RegisterRequest represents a registration request
type RegisterRequest struct {
	Email    string
	Name     string
	Password string
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string
	Password string
}

// TokenResponse represents a token response
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// Register mendaftarkan pengguna baru dengan validasi email dan password strength.
//
// Parameters:
//   - ctx: context untuk membatalkan operasi
//   - email: alamat email pengguna yang akan didaftarkan
//   - name: nama lengkap pengguna
//   - password: kata sandi pengguna (harus memenuhi strength requirements)
//
// Returns:
//   - *User: data pengguna yang baru dibuat
//   - error: error jika validasi gagal, email sudah terdaftar, atau ada error saat create
//
// Example:
//
//	user, err := authService.Register(ctx, "user@example.com", "John Doe", "SecurePass123!")
func (s *AuthService) Register(ctx context.Context, email, name, password string) (*User, error) {
	// Validate input
	v := NewValidator().
		Required("email", email).
		Email("email", email).
		Required("name", name).
		Required("password", password)

	if !v.IsValid() {
		err := NewAppError("Validasi gagal", 400)
		err.Errors = v.ErrorMap()
		return nil, err
	}

	// Validate password strength
	if err := s.pwValidator.Validate(password); err != nil {
		return nil, err
	}

	// Check if email already exists
	exists, err := s.userStore.Exists(ctx, email)
	if err != nil {
		return nil, NewAppError("Gagal memeriksa keberadaan pengguna", 500)
	}

	if exists {
		return nil, NewAppError("Email sudah terdaftar", 409).
			WithFieldError("email", "Email ini sudah terdaftar")
	}

	// Hash password
	passwordHash, err := HashPassword(password)
	if err != nil {
		return nil, NewAppError("Gagal memproses kata sandi", 500)
	}

	// Create user
	user := &User{
		Email:    email,
		Name:     name,
		Password: passwordHash,
	}

	if err := s.userStore.Create(ctx, user); err != nil {
		return nil, NewAppError("Gagal membuat pengguna", 500)
	}

	return user, nil
}

// Login mengotentikasi pengguna dan mengembalikan access token dan refresh token.
//
// Parameters:
//   - ctx: context untuk membatalkan operasi
//   - email: alamat email pengguna
//   - password: kata sandi pengguna
//
// Returns:
//   - string: access token
//   - string: refresh token
//   - error: error jika kredensial tidak valid atau ada masalah saat membuat token
//
// Example:
//
//	accessToken, refreshToken, err := authService.Login(ctx, "user@example.com", "password123")
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
	if err := VerifyPassword(user.Password, password); err != nil {
		return "", "", NewAppError("Kredensial tidak valid", 401)
	}

	// Generate tokens
	accessToken, err := s.jwtManager.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		return "", "", NewAppError("Gagal membuat token akses", 500)
	}

	refreshToken, err := s.jwtManager.GenerateRefreshToken(user.ID)
	if err != nil {
		return "", "", NewAppError("Gagal membuat token refresh", 500)
	}

	// Store refresh token hash
	refreshTokenHash := GenerateTokenHash(refreshToken)
	refreshTokenEntity := &RefreshToken{
		UserID:    user.ID,
		TokenHash: refreshTokenHash,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	if err := s.tokenStore.SaveRefreshToken(ctx, refreshTokenEntity); err != nil {
		return "", "", NewAppError("Gagal menyimpan token refresh", 500)
	}

	return accessToken, refreshToken, nil
}

// RefreshToken merefresh access token menggunakan refresh token yang valid.
// Membatalkan token lama dan membuat token baru.
//
// Parameters:
//   - ctx: context untuk membatalkan operasi
//   - refreshTokenStr: refresh token string yang akan digunakan untuk mendapatkan access token baru
//
// Returns:
//   - string: access token baru
//   - string: refresh token baru
//   - error: error jika token tidak valid, sudah di-revoke, atau kadaluarsa
//
// Example:
//
//	newAccessToken, newRefreshToken, err := authService.RefreshToken(ctx, oldRefreshToken)
func (s *AuthService) RefreshToken(ctx context.Context, refreshTokenStr string) (string, string, error) {
	// Verify refresh token
	userID, err := s.jwtManager.VerifyRefreshToken(refreshTokenStr)
	if err != nil {
		return "", "", NewAppError("Token refresh tidak valid", 401)
	}

	// Check if token is in the database and not revoked
	refreshTokenHash := GenerateTokenHash(refreshTokenStr)
	storedToken, err := s.tokenStore.FindRefreshToken(ctx, refreshTokenHash)
	if err != nil {
		return "", "", NewAppError("Token refresh tidak valid", 401)
	}

	// Check if token is revoked
	if storedToken.RevokedAt != nil {
		return "", "", NewAppError("Token refresh telah dibatalkan", 401)
	}

	// Check if token has expired
	if time.Now().After(storedToken.ExpiresAt) {
		return "", "", NewAppError("Token refresh telah kadaluarsa", 401)
	}

	// Get user info
	user, err := s.userStore.FindByID(ctx, userID)
	if err != nil {
		return "", "", NewAppError("Pengguna tidak ditemukan", 404)
	}

	// Generate new access token
	newAccessToken, err := s.jwtManager.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		return "", "", NewAppError("Gagal membuat token akses", 500)
	}

	// Generate new refresh token
	newRefreshToken, err := s.jwtManager.GenerateRefreshToken(user.ID)
	if err != nil {
		return "", "", NewAppError("Gagal membuat token refresh", 500)
	}

	// Revoke old refresh token
	_ = s.tokenStore.RevokeRefreshToken(ctx, refreshTokenHash)

	// Store new refresh token hash
	newRefreshTokenHash := GenerateTokenHash(newRefreshToken)
	newRefreshTokenEntity := &RefreshToken{
		UserID:    user.ID,
		TokenHash: newRefreshTokenHash,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	if err := s.tokenStore.SaveRefreshToken(ctx, newRefreshTokenEntity); err != nil {
		return "", "", NewAppError("Gagal menyimpan token refresh baru", 500)
	}

	return newAccessToken, newRefreshToken, nil
}

// RequestPasswordReset membuat request reset password untuk email pengguna.
// Tidak mengungkapkan apakah email terdaftar atau tidak untuk keamanan.
//
// Parameters:
//   - ctx: context untuk membatalkan operasi
//   - email: alamat email pengguna yang ingin melakukan reset password
//
// Returns:
//   - error: error jika ada masalah saat generate token atau menyimpan ke database
//
// Example:
//
//	err := authService.RequestPasswordReset(ctx, "user@example.com")
func (s *AuthService) RequestPasswordReset(ctx context.Context, email string) error {
	// Validate email
	v := NewValidator().
		Required("email", email).
		Email("email", email)

	if !v.IsValid() {
		err := NewAppError("Validasi gagal", 400)
		err.Errors = v.ErrorMap()
		return err
	}

	// Find user by email
	user, err := s.userStore.FindByEmail(ctx, email)
	if err != nil {
		// Don't reveal if email exists
		return nil
	}

	// Generate reset token
	resetToken, err := GenerateSecureToken(32)
	if err != nil {
		return NewAppError("Gagal membuat token reset", 500)
	}

	// Store reset token hash
	resetTokenHash := GenerateTokenHash(resetToken)
	resetTokenEntity := &PasswordResetToken{
		UserID:    user.ID,
		TokenHash: resetTokenHash,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	if err := s.tokenStore.SavePasswordResetToken(ctx, resetTokenEntity); err != nil {
		return NewAppError("Gagal menyimpan token reset", 500)
	}

	return nil
}

// ResetPassword mereset password pengguna menggunakan reset token yang valid.
// Membatalkan semua refresh token pengguna untuk keamanan setelah reset.
//
// Parameters:
//   - ctx: context untuk membatalkan operasi
//   - resetTokenStr: token reset password yang dikirim via email
//   - newPassword: password baru yang harus memenuhi strength requirements
//
// Returns:
//   - error: error jika token tidak valid, sudah digunakan, kadaluarsa, atau password tidak memenuhi requirements
//
// Example:
//
//	err := authService.ResetPassword(ctx, resetToken, "NewSecurePass123!")
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
		return NewAppError("Token reset tidak valid atau telah kadaluarsa", 400)
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
		return NewAppError("Gagal memproses kata sandi", 500)
	}

	// Update user password
	user.Password = passwordHash
	if err := s.userStore.Update(ctx, user); err != nil {
		return NewAppError("Gagal memperbarui kata sandi", 500)
	}

	// Mark reset token as used
	if err := s.tokenStore.MarkPasswordResetUsed(ctx, resetTokenHash); err != nil {
		return NewAppError("Gagal menandai token reset sebagai sudah digunakan", 500)
	}

	// Revoke all user's refresh tokens for security
	_ = s.tokenStore.RevokeAllUserTokens(ctx, user.ID)

	return nil
}

// Logout mengeluarkan pengguna dengan membatalkan refresh token mereka.
//
// Parameters:
//   - ctx: context untuk membatalkan operasi
//   - refreshTokenStr: refresh token yang akan dibatalkan
//
// Returns:
//   - error: error jika ada masalah saat melakukan revoke token
//
// Example:
//
//	err := authService.Logout(ctx, refreshToken)
func (s *AuthService) Logout(ctx context.Context, refreshTokenStr string) error {
	if refreshTokenStr == "" {
		return NewAppError("Token refresh diperlukan", 400)
	}

	// Revoke refresh token
	refreshTokenHash := GenerateTokenHash(refreshTokenStr)
	if err := s.tokenStore.RevokeRefreshToken(ctx, refreshTokenHash); err != nil {
		return NewAppError("Gagal keluar", 500)
	}

	return nil
}
