package dim

// Authenticatable merepresentasikan entitas pengguna yang dapat diotentikasi.
// Interface ini memungkinkan framework untuk berinteraksi dengan model User apa pun.
type Authenticatable interface {
	GetID() string
	GetEmail() string
	GetPassword() string
	SetPassword(string)
}

// TokenUser represents a minimal user entity derived from a token
type TokenUser struct {
	ID       string
	Email    string
	Password string // Usually empty for token-derived users
	Claims   map[string]interface{}
}

func (u *TokenUser) GetID() string {
	return u.ID
}

func (u *TokenUser) GetEmail() string {
	return u.Email
}

func (u *TokenUser) GetPassword() string {
	return u.Password
}

func (u *TokenUser) SetPassword(password string) {
	u.Password = password
}

func (u *TokenUser) GetClaims() map[string]interface{} {
	return u.Claims
}
