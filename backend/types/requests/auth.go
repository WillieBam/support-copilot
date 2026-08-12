package requests

type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	UsernameOrEmail string `json:"username_or_email"`
	Password        string `json:"password"`
	TOTPCode        string `json:"totp_code,omitempty"`
}

type TOTPVerifyRequest struct {
	Code string `json:"code"`
}

type TokenExchangeRequest struct {
	FirebaseToken string `json:"firebase_token"`
	TOTPCode      string `json:"totp_code,omitempty"`
}
