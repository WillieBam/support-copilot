package service_test

import (
	"context"
	"errors"

	firebaseAuth "firebase.google.com/go/v4/auth"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	"github.com/WillieBam/support_copilot/backend/app/config"
	"github.com/WillieBam/support_copilot/backend/internal/interfaces"
	"github.com/WillieBam/support_copilot/backend/internal/mocks"
	"github.com/WillieBam/support_copilot/backend/internal/service"
	"github.com/WillieBam/support_copilot/backend/types/models"
	customErrors "github.com/WillieBam/support_copilot/backend/utils/errors"
)

var _ = Describe("AuthService", func() {
	var (
		authSvc      interfaces.IAuthService
		userRepo     *mocks.IUserRepository
		firebaseRepo *mocks.IFirebaseRepository
		ctx          context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		userRepo = &mocks.IUserRepository{}
		firebaseRepo = &mocks.IFirebaseRepository{}
		authSvc = service.New(service.AuthServiceParam{
			UserRepo:     userRepo,
			FirebaseRepo: firebaseRepo,
		})

		// Configure test JWT secret
		config.Get().Auth.JWTSecret = "test_jwt_secret_must_be_long_enough_32_bytes"
	})

	Context("ExchangeToken", func() {
		It("should fail if firebase token verification fails", func() {
			firebaseRepo.On("VerifyIDToken", ctx, "invalid-token").Return(nil, errors.New("firebase error"))

			token, claims, err := authSvc.ExchangeToken(ctx, "invalid-token", "")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("invalid or expired firebase token"))
			Expect(token).To(BeEmpty())
			Expect(claims).To(BeNil())
		})

		It("should register new user if they do not exist locally", func() {
			config.Get().Auth.TOTPRequired = false

			fbToken := &firebaseAuth.Token{
				UID: "uid-123",
				Claims: map[string]interface{}{
					"email": "user@test.com",
					"name":  "Test User",
				},
			}
			firebaseRepo.On("VerifyIDToken", ctx, "valid-token").Return(fbToken, nil)
			userRepo.On("GetUserByFirebaseUID", ctx, "uid-123").Return(nil, customErrors.ErrUserNotFound)
			userRepo.On("CreateUser", ctx, mock.Anything).Return(nil)

			token, claims, err := authSvc.ExchangeToken(ctx, "valid-token", "")
			Expect(err).NotTo(HaveOccurred())
			Expect(token).NotTo(BeEmpty())
			Expect(claims).NotTo(BeNil())
			Expect(claims.FirebaseUID).To(Equal("uid-123"))
		})

		It("should fail when TOTP MFA is enabled for user but code is missing", func() {
			fbToken := &firebaseAuth.Token{
				UID: "uid-123",
				Claims: map[string]interface{}{
					"email": "user@test.com",
				},
			}
			existingUser := &models.User{
				FirebaseUID: strPtr("uid-123"),
				Email:       "user@test.com",
				TOTPEnabled: true,
				TOTPSecret:  "JBSWY3DPEHPK3PXP",
			}
			firebaseRepo.On("VerifyIDToken", ctx, "mfa-token").Return(fbToken, nil)
			userRepo.On("GetUserByFirebaseUID", ctx, "uid-123").Return(existingUser, nil)

			token, claims, err := authSvc.ExchangeToken(ctx, "mfa-token", "")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("mfa_required"))
			Expect(token).To(BeEmpty())
			Expect(claims).To(BeNil())
		})

		It("should succeed when user is found and token exchange succeeds", func() {
			config.Get().Auth.TOTPRequired = false

			fbToken := &firebaseAuth.Token{
				UID: "uid-123",
				Claims: map[string]interface{}{
					"email": "user@test.com",
				},
			}
			existingUser := &models.User{
				FirebaseUID: strPtr("uid-123"),
				Email:       "user@test.com",
			}
			firebaseRepo.On("VerifyIDToken", ctx, "exist-token").Return(fbToken, nil)
			userRepo.On("GetUserByFirebaseUID", ctx, "uid-123").Return(existingUser, nil)

			token, claims, err := authSvc.ExchangeToken(ctx, "exist-token", "")
			Expect(err).NotTo(HaveOccurred())
			Expect(token).NotTo(BeEmpty())
			Expect(claims.Email).To(Equal("user@test.com"))
		})

		It("should handle database error when retrieving existing user", func() {
			config.Get().Auth.TOTPRequired = false

			fbToken := &firebaseAuth.Token{
				UID: "uid-123",
				Claims: map[string]interface{}{
					"email": "user@test.com",
				},
			}
			firebaseRepo.On("VerifyIDToken", ctx, "err-user-token").Return(fbToken, nil)
			userRepo.On("GetUserByFirebaseUID", ctx, "uid-123").Return(nil, errors.New("db error"))

			token, claims, err := authSvc.ExchangeToken(ctx, "err-user-token", "")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("internal server database error"))
			Expect(token).To(BeEmpty())
			Expect(claims).To(BeNil())
		})

		It("should fail when user creation returns error", func() {
			config.Get().Auth.TOTPRequired = false

			fbToken := &firebaseAuth.Token{
				UID: "uid-123",
				Claims: map[string]interface{}{
					"email": "user@test.com",
				},
			}
			firebaseRepo.On("VerifyIDToken", ctx, "create-err-token").Return(fbToken, nil)
			userRepo.On("GetUserByFirebaseUID", ctx, "uid-123").Return(nil, customErrors.ErrUserNotFound)
			userRepo.On("CreateUser", ctx, mock.Anything).Return(errors.New("create err"))

			token, claims, err := authSvc.ExchangeToken(ctx, "create-err-token", "")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("internal server registration error"))
			Expect(token).To(BeEmpty())
			Expect(claims).To(BeNil())
		})
	})

	Context("ParseAndValidateAuthToken", func() {
		It("should fail on invalid token string", func() {
			claims, err := authSvc.ParseAndValidateAuthToken(ctx, "invalid.token.string")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("invalid signature or expired session"))
			Expect(claims).To(BeNil())
		})

		It("should successfully parse a valid generated token", func() {
			config.Get().Auth.TOTPRequired = false

			fbToken := &firebaseAuth.Token{
				UID: "uid-123",
				Claims: map[string]interface{}{
					"email": "user@test.com",
				},
			}
			firebaseRepo.On("VerifyIDToken", ctx, "valid-token").Return(fbToken, nil)
			userRepo.On("UpsertUser", ctx, mock.Anything).Return(nil)
			userRepo.On("GetUserByFirebaseUID", ctx, "uid-123").Return(nil, customErrors.ErrUserNotFound)
			userRepo.On("CreateUser", ctx, mock.Anything).Return(nil)

			validTokenString, _, err := authSvc.ExchangeToken(ctx, "valid-token", "")
			Expect(err).NotTo(HaveOccurred())

			parsedClaims, err := authSvc.ParseAndValidateAuthToken(ctx, validTokenString)
			Expect(err).NotTo(HaveOccurred())
			Expect(parsedClaims).NotTo(BeNil())
			Expect(parsedClaims.FirebaseUID).To(Equal("uid-123"))
		})
	})

	Context("ValidatePasswordComplexity", func() {
		It("should fail if password is too short", func() {
			err := service.ValidatePasswordComplexity("short")
			Expect(err).To(Equal(customErrors.ErrInvalidPasswordComplexity))
		})

		It("should fail if password is too long", func() {
			err := service.ValidatePasswordComplexity("toolongpasswordtoolongpasswordtoolongpassword")
			Expect(err).To(Equal(customErrors.ErrInvalidPasswordComplexity))
		})

		It("should fail if password missing special character", func() {
			err := service.ValidatePasswordComplexity("nospec1234")
			Expect(err).To(Equal(customErrors.ErrInvalidPasswordComplexity))
		})

		It("should succeed when password satisfies complexity policy", func() {
			err := service.ValidatePasswordComplexity("Pass!1234")
			Expect(err).NotTo(HaveOccurred())
		})
	})
})

func strPtr(s string) *string {
	return &s
}
