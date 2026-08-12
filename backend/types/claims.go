package types

import (
	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claims struct {
	UserID      uuid.UUID `json:"user_id"`
	FirebaseUID string    `json:"firebase_uid,omitempty"`
	Username    string    `json:"username,omitempty"`
	Email       string    `json:"email"`
	MfaVerified bool      `json:"mfa_verified"`
	AuthMeTHOD  string    `json:"auth_method"` // "password" ||  "firebase"
	jwt.RegisteredClaims
}
