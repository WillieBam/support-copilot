package service

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"log/slog"

	"github.com/WillieBam/support_copilot/backend/app/config"
	"github.com/WillieBam/support_copilot/backend/internal/interfaces"
	"github.com/WillieBam/support_copilot/backend/types"
	"github.com/WillieBam/support_copilot/backend/types/models"
	customErrors "github.com/WillieBam/support_copilot/backend/utils/errors"
	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var specialCharRegex = regexp.MustCompile(`[!@#$%^&*(),.?":{}|<>]`)

// ValidatePasswordComplexity validates that password satisfies complexity policy
func ValidatePasswordComplexity(password string) error {
	if len(password) < 6 || len(password) > 30 {
		return customErrors.ErrInvalidPasswordComplexity
	}
	if !specialCharRegex.MatchString(password) {
		return customErrors.ErrInvalidPasswordComplexity
	}
	return nil
}

type authService struct {
	userRepo     interfaces.IUserRepository
	firebaseRepo interfaces.IFirebaseRepository
}

type AuthServiceParam struct {
	UserRepo     interfaces.IUserRepository
	FirebaseRepo interfaces.IFirebaseRepository
}

func New(asp AuthServiceParam) interfaces.IAuthService {
	return &authService{
		userRepo:     asp.UserRepo,
		firebaseRepo: asp.FirebaseRepo,
	}
}

func (s *authService) Register(ctx context.Context, username, email, password string) (*models.User, error) {
	username = strings.TrimSpace(username)
	email = strings.TrimSpace(email)

	if username == "" || email == "" || password == "" {
		return nil, errors.New("username, email, password are required")
	}

	if err := validatePasswordComplexity(password); err != nil {
		return nil, err
	}

	existing, _ := s.userRepo.GetUserByUsernameOrEmail(ctx, username)
	if existing != nil {
		return nil, errors.New("username already taken")
	}

	existingEmail, _ := s.userRepo.GetUserByUsernameOrEmail(ctx, email)
	if existingEmail != nil {
		return nil, errors.New("email is already registered")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		slog.ErrorContext(ctx, "failed to hash password", "error", err)
		return nil, err
	}

	newUser := &models.User{
		Username:     &username,
		Email:        email,
		DisplayName:  username,
		PasswordHash: string(hash),
		Scope:        "engineer",
		CreatedAt:    time.Now(),
	}

	if err := s.userRepo.CreateUser(ctx, newUser); err != nil {
		slog.ErrorContext(ctx, "failed to create user", "error", err)
		return nil, err
	}

	return newUser, nil
}

func (s *authService) LoginWithPassword(ctx context.Context, usernameOrEmail, password, totpCode string) (string, *types.Claims, error) {
	id := strings.TrimSpace(usernameOrEmail)
	if id == "" || password == "" {
		return "", nil, errors.New("username/email and password are required")
	}

	user, err := s.userRepo.GetUserByUsernameOrEmail(ctx, id)
	if err != nil {
		return "", nil, errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", nil, errors.New("invalid credentials")
	}

	if user.TOTPEnabled {
		if totpCode == "" {
			return "", nil, errors.New("mfa required")
		}

		if !validateTOTPCode(user.TOTPSecret, totpCode) {
			return "", nil, errors.New("invalid 2fa code")
		}
	}

	usernameVal := ""

	if user.Username != nil {
		usernameVal = *user.Username
	}

	fbUID := ""
	if user.FirebaseUID != nil {
		fbUID = *user.FirebaseUID
	}

	return s.generateAuthToken(user.ID, fbUID, usernameVal, user.Email, user.TOTPEnabled, "password")

}

func (s *authService) ExchangeToken(ctx context.Context, firebaseToken string, totpCode string) (string, *types.Claims, error) {
	// verify the incoming token with firebase
	verifiedToken, err := s.firebaseRepo.VerifyIDToken(ctx, firebaseToken)
	if err != nil {
		slog.ErrorContext(ctx, "failed to verify firebase id token", "error", err)
		return "", nil, errors.New("invalid or expired firebase token")
	}

	email, _ := verifiedToken.Claims["email"].(string)
	name, _ := verifiedToken.Claims["name"].(string)

	user, err := s.userRepo.GetUserByFirebaseUID(ctx, verifiedToken.UID)
	if err != nil {
		if errors.Is(err, customErrors.ErrUserNotFound) {
			fbUID := verifiedToken.UID
			user = &models.User{
				FirebaseUID: &fbUID,
				Email:       email,
				DisplayName: name,
				CreatedAt:   time.Now(),
				Scope:       "engineer",
			}
			if createErr := s.userRepo.CreateUser(ctx, user); createErr != nil {
				slog.ErrorContext(ctx, "failed to seed user record upon login token exchange", "error", createErr)
				return "", nil, errors.New("internal server registration error")
			}
		} else {
			slog.ErrorContext(ctx, "database repository failure during user sync", "error", err)
			return "", nil, errors.New("internal server database error")
		}
	}

	if user != nil && user.TOTPEnabled {
		totpCode = strings.TrimSpace(totpCode)
		if totpCode == "" {
			return "", nil, errors.New("mfa_required")
		}
		if !validateTOTPCode(user.TOTPSecret, totpCode) {
			return "", nil, errors.New("invalid 2fa code")
		}
	}

	usernameVal := ""
	if user != nil && user.Username != nil {
		usernameVal = *user.Username
	}

	return s.generateAuthToken(user.ID, verifiedToken.UID, usernameVal, email, user.TOTPEnabled, "firebase")
}

func (s *authService) SetupTOTP(ctx context.Context, userID uuid.UUID) (string, string, error) {
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil || user == nil {
		return "", "", errors.New("user not found")
	}

	secret := generateTOTPSecret()
	qrURI := fmt.Sprintf("otpauth://totp/SupportCopilot:%s?secret=%s&issuer=SupportCopilot",
		url.QueryEscape(user.
			Email), secret)

	user.TOTPSecret = secret
	if err := s.userRepo.UpdateUser(ctx, user); err != nil {
		return "", "", errors.New("failed to save totp setup")
	}

	return secret, qrURI, nil
}

func (s *authService) VerifyAndEnableTOTP(ctx context.Context, userID uuid.UUID, code string) error {
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil || user == nil {
		return errors.New("user not found")
	}

	if user.TOTPSecret == "" {
		return errors.New("totp setup has not been initialized")
	}

	if !validateTOTPCode(user.TOTPSecret, code) {
		return errors.New("invalid totp code")
	}

	user.TOTPEnabled = true
	return s.userRepo.UpdateUser(ctx, user)
}

func (s *authService) DisableTOTP(ctx context.Context, userID uuid.UUID) error {
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil || user == nil {
		return errors.New("user not found")
	}

	user.TOTPEnabled = false
	user.TOTPSecret = ""
	return s.userRepo.UpdateUser(ctx, user)
}

// ParseAndValidateAuthToken decrypts and validates application tokens passed on subsequent HTTP calls.
func (s *authService) ParseAndValidateAuthToken(ctx context.Context, tokenString string) (*types.Claims, error) {
	cfg := config.Get()

	token, err := jwt.ParseWithClaims(tokenString, &types.Claims{}, func(t *jwt.Token) (interface{}, error) {
		// confirm the signing method is expected (HMAC-SHA256)
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected token signing algorithm")
		}
		return []byte(cfg.Auth.JWTSecret), nil
	})

	if err != nil {
		slog.WarnContext(ctx, "failed to parse jwt", "error", err)
		return nil, errors.New("invalid signature or expired session")
	}

	if claims, ok := token.Claims.(*types.Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token payload claims")
}

// // validateMFAClaims isolates the unexported dictionary verification checkout in middleware layer
// func (s *authService) validateMFAClaims(claims map[string]any) bool {
// 	v, ok := claims["firebase"]
// 	if !ok {
// 		return false
// 	}
// 	firebaseClaims, ok := v.(map[string]any)
// 	if !ok {
// 		return false
// 	}

// 	methodRaw, ok := firebaseClaims["sign_in_second_factor"]
// 	if !ok {
// 		return false
// 	}
// 	method, ok := methodRaw.(string)
// 	if !ok {
// 		return false
// 	}

// 	// string match against expected 'totp' identifier value case-insensitively
// 	return len(method) > 0 && (method == "totp" || method == "TOTP")
// }

// generateAuthToken handles generating the cryptographic HS256 JWT signature
func (s *authService) generateAuthToken(userID uuid.UUID, firebaseUID string, username string, email string, mfaVerified bool, method string) (string, *types.Claims, error) {
	cfg := config.Get()

	// create 1 hour expiration duration
	expirationTime := time.Now().Add(1 * time.Hour)

	claims := &types.Claims{
		UserID:      userID,
		FirebaseUID: firebaseUID,
		Username:    username,
		Email:       email,
		MfaVerified: mfaVerified,
		AuthMeTHOD:  method,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "support-copilot-backend",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(cfg.Auth.JWTSecret))
	if err != nil {
		slog.Error("failed to generate system cryptographic signature", "error", err)
		return "", nil, errors.New("failed to sign backend session credentials")
	}

	return tokenString, claims, nil
}

// ValidatePasswordComplexity validates that password satisfies nfr01 complexity policy
func validatePasswordComplexity(password string) error {
	if len(password) < 6 || len(password) > 8 {
		return customErrors.ErrInvalidPasswordComplexity
	}
	if !specialCharRegex.MatchString(password) {
		return customErrors.ErrInvalidPasswordComplexity
	}
	return nil
}

// generateTOTPSecret generates a secure, random 10-byte secret key
// and returns it as a padding-free Base32 encoded string
func generateTOTPSecret() string {
	bytes := make([]byte, 10)
	_, _ = rand.Read(bytes)
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(bytes)
}

func validateTOTPCode(secret, code string) bool {
	secret = strings.ToUpper(strings.TrimSpace(secret))
	code = strings.TrimSpace(code)

	if len(code) != 6 || secret == "" {
		return false
	}

	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil {
		return false
	}

	currentTime := time.Now().Unix() / 30
	for window := -1; window <= 1; window++ {
		t := uint64(currentTime + int64(window))
		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, t)
		mac := hmac.New(sha1.New, key)

		mac.Write(buf)
		hash := mac.Sum(nil)
		offset := hash[len(hash)-1] & 0x0f
		trunc := binary.BigEndian.Uint32(hash[offset:offset+4]) & 0x7fffffff
		expectedCode := fmt.Sprintf("%06d", trunc%1000000)

		if hmac.Equal([]byte(expectedCode), []byte(code)) {
			return true
		}

	}
	return false
}
func (s *authService) checkIfUserHasEnrolledMFA(claims map[string]any) bool {
	v, ok := claims["firebase"]
	if !ok {
		return false
	}
	fbClaims, ok := v.(map[string]any)
	if !ok {
		return false
	}

	// If enrolled_factors list exists and isn't empty, they have configured MFA
	_, hasFactors := fbClaims["enrolled_factors"]
	return hasFactors
}
