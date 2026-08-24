package interfaces

import (
	"context"

	"github.com/WillieBam/support_copilot/backend/types"
	"github.com/WillieBam/support_copilot/backend/types/models"
	"github.com/google/uuid"
)

type IAuthService interface {
	Register(ctx context.Context, username, email, password string) (*models.User, error)
	LoginWithPassword(ctx context.Context, usernameOrEmail, password, totpCode string) (string, *types.Claims, error)
	SetupTOTP(ctx context.Context, userID uuid.UUID) (secret string, qrURI string, err error)
	VerifyAndEnableTOTP(ctx context.Context, userID uuid.UUID, code string) error
	DisableTOTP(ctx context.Context, userID uuid.UUID) error
	ExchangeToken(ctx context.Context, firebaseToken string, totpCode string) (string, *types.Claims, error)
	RefreshToken(ctx context.Context, tokenString string) (string, *types.Claims, error)
	ParseAndValidateAuthToken(ctx context.Context, tokenString string) (*types.Claims, error)
}
