package service

import (
	"context"
	"errors"
	"regexp"
	"time"

	"log/slog"

	"github.com/WillieBam/support_copilot/backend/app/config"
	"github.com/WillieBam/support_copilot/backend/internal/interfaces"
	"github.com/WillieBam/support_copilot/backend/types"
	"github.com/WillieBam/support_copilot/backend/types/models"
	customErrors "github.com/WillieBam/support_copilot/backend/utils/errors"
	jwt "github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

var specialCharRegex = regexp.MustCompile(`[!@#$%^&*(),.?":{}|<>]`)

// ValidatePasswordComplexity validates that password satisfies nfr01 complexity policy
func ValidatePasswordComplexity(password string) error {
	if len(password) < 6 || len(password) > 8 {
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

func (s *authService) ExchangeToken(ctx context.Context, firebaseToken string) (string, *types.Claims, error) {
	// verify the incoming token with firebase
	verifiedToken, err := s.firebaseRepo.VerifyIDToken(ctx, firebaseToken)
	if err != nil {
		slog.ErrorContext(ctx, "failed to verify firebase id token", "error", err)
		return "", nil, errors.New("invalid or expired firebase token")
	}

	cfg := config.Get()
	isMfaVerified := s.validateMFAClaims(verifiedToken.Claims)

	// ensure user exists in local database schema
	email, _ := verifiedToken.Claims["email"].(string)
	name, _ := verifiedToken.Claims["name"].(string)

	newUser := &models.User{
		FirebaseUID: verifiedToken.UID,
		Email:       email,
		DisplayName: name,
		CreatedAt:   time.Now(),
		Scope:       "engineer",
	}

	if err := s.userRepo.UpsertUser(ctx, newUser); err != nil {
		slog.ErrorContext(ctx, "failed to cleanly sync user record upon token exchange", "error", err)
		return "", nil, errors.New("internal server database synchronization error")
	}

	hasEnrolledMFA := isMfaVerified || s.checkIfUserHasEnrolledMFA(verifiedToken.Claims)
	if cfg.Auth.TOTPRequired && hasEnrolledMFA && !isMfaVerified {
		slog.WarnContext(ctx, "User has registered MFA but skipped the validation challenge", "uid", verifiedToken.UID)
		return "", nil, errors.New("mfa_required")
	}

	_, err = s.userRepo.GetUserByFirebaseUID(ctx, verifiedToken.UID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			slog.InfoContext(ctx, "registering new user record in database", "uid", verifiedToken.UID)

			// extract basic user info
			email, _ := verifiedToken.Claims["email"].(string)
			name, _ := verifiedToken.Claims["name"].(string)

			newUser := &models.User{
				FirebaseUID: verifiedToken.UID,
				Email:       email,
				DisplayName: name,
				CreatedAt:   time.Now(),
				Scope:       "engineer",
			}
			if createErr := s.userRepo.CreateUser(ctx, newUser); createErr != nil {
				slog.ErrorContext(ctx, "failed to seed user record upon login token exchange", "error", createErr)
				return "", nil, errors.New("internal server registration error")
			}
		} else {
			slog.ErrorContext(ctx, "database repository failure during user sync", "error", err)
			return "", nil, errors.New("internal server database error")
		}
	}

	// generate and return backend signed session token
	backendToken, claims, err := s.generateAuthToken(verifiedToken.UID, email, isMfaVerified)
	if err != nil {
		return "", nil, err
	}

	return backendToken, claims, nil
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

// validateMFAClaims isolates the unexported dictionary verification checkout in middleware layer
func (s *authService) validateMFAClaims(claims map[string]any) bool {
	v, ok := claims["firebase"]
	if !ok {
		return false
	}
	firebaseClaims, ok := v.(map[string]any)
	if !ok {
		return false
	}

	methodRaw, ok := firebaseClaims["sign_in_second_factor"]
	if !ok {
		return false
	}
	method, ok := methodRaw.(string)
	if !ok {
		return false
	}

	// string match against expected 'totp' identifier value case-insensitively
	return len(method) > 0 && (method == "totp" || method == "TOTP")
}

// generateAuthToken handles generating the cryptographic HS256 JWT signature
func (s *authService) generateAuthToken(uid string, email string, mfaVerified bool) (string, *types.Claims, error) {
	cfg := config.Get()

	// create 1 hour expiration duration
	expirationTime := time.Now().Add(1 * time.Hour)

	claims := &types.Claims{
		FirebaseUID: uid,
		Email:       email,
		MfaVerified: mfaVerified,
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
