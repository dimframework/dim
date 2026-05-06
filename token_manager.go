package dim

import "time"

// TokenClaims represents decoded claims from any token type (JWT, Branca, etc.).
// Defined as a type alias so it is directly interchangeable with map[string]interface{}.
type TokenClaims = map[string]interface{}

// TokenManager defines the interface for token generation and verification.
// Both JWTManager and BrancaManager implement this interface, enabling
// token provider selection without changing application code.
type TokenManager interface {
	GenerateAccessToken(userID, email, sessionID string, extraClaims map[string]interface{}) (string, error)
	GenerateRefreshToken(userID, sessionID string) (string, error)

	// VerifyToken verifies an access token and returns its claims.
	VerifyToken(tokenString string) (TokenClaims, error)

	// VerifyRefreshToken verifies a refresh token and returns userID and sessionID.
	VerifyRefreshToken(tokenString string) (userID, sessionID string, err error)

	GetTokenExpiry(tokenString string) (time.Time, error)
	IsTokenExpired(tokenString string) (bool, error)
}
