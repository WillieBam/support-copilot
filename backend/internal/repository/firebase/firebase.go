package firebase

import (
	"context"
	"errors"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"github.com/WillieBam/support_copilot/backend/app/config"
	"github.com/WillieBam/support_copilot/backend/internal/interfaces"
	"google.golang.org/api/option"
)

type FirebaseRepository struct {
	authClient *auth.Client
}

// NewFirebaseRepository initializes the Firebase Admin SDK
func NewFirebaseRepository(cfg *config.Config) (interfaces.IFirebaseRepository, error) {
	ctx := context.Background()

	opt := option.WithCredentialsFile(cfg.Firebase.ServiceAccountPath)

	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		return nil, err
	}

	authClient, err := app.Auth(ctx)
	if err != nil {
		return nil, err
	}

	return &FirebaseRepository{
		authClient: authClient,
	}, nil
}

// VerifyIDToken contacts Firebase to decode and validate the incoming JWT token
func (r *FirebaseRepository) VerifyIDToken(ctx context.Context, idToken string) (*auth.Token, error) {
	if r == nil || r.authClient == nil {
		return nil, errors.New("firebase auth client is not initialized")
	}

	token, err := r.authClient.VerifyIDToken(ctx, idToken)
	if err != nil {
		return nil, err
	}

	// email, _ := token.Claims["email"].(string)
	// name, _ := token.Claims["name"].(string)

	// return token.UID, email, name, nil
	return token, nil
}
