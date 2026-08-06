package service_test

import (
	"context"
	"errors"

	firebaseAuth "firebase.google.com/go/v4/auth"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"

	"github.com/WillieBam/support_copilot/backend/app/config"
	"github.com/WillieBam/support_copilot/backend/internal/interfaces"
	"github.com/WillieBam/support_copilot/backend/internal/mocks"
	"github.com/WillieBam/support_copilot/backend/internal/service"
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

			token, claims, err := authSvc.ExchangeToken(ctx, "invalid-token")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("invalid or expired firebase token"))
			Expect(token).To(BeEmpty())
			Expect(claims).To(BeNil())
		})

		It("should succeed when user exists and MFA is not required", func() {
			config.Get().Auth.TOTPRequired = false

			fbToken := &firebaseAuth.Token{
				UID: "uid-123",
				Claims: map[string]interface{}{
					"email": "user@test.com",
					"name":  "Test User",
				},
			}
			firebaseRepo.On("VerifyIDToken", ctx, "valid-token").Return(fbToken, nil)
			userRepo.On("UpsertUser", ctx, mock.Anything).Return(nil)
			userRepo.On("GetUserByFirebaseUID", ctx, "uid-123").Return(nil, nil)

			token, claims, err := authSvc.ExchangeToken(ctx, "valid-token")
			Expect(err).NotTo(HaveOccurred())
			Expect(token).NotTo(BeEmpty())
			Expect(claims).NotTo(BeNil())
			Expect(claims.FirebaseUID).To(Equal("uid-123"))
			Expect(claims.Email).To(Equal("user@test.com"))
		})

		It("should register new user if they do not exist locally", func() {
			config.Get().Auth.TOTPRequired = false

			fbToken := &firebaseAuth.Token{
				UID: "uid-new",
				Claims: map[string]interface{}{
					"email": "new@test.com",
					"name":  "New User",
				},
			}
			firebaseRepo.On("VerifyIDToken", ctx, "valid-token").Return(fbToken, nil)
			userRepo.On("UpsertUser", ctx, mock.Anything).Return(nil)
			userRepo.On("GetUserByFirebaseUID", ctx, "uid-new").Return(nil, gorm.ErrRecordNotFound)
			userRepo.On("CreateUser", ctx, mock.Anything).Return(nil)

			token, claims, err := authSvc.ExchangeToken(ctx, "valid-token")
			Expect(err).NotTo(HaveOccurred())
			Expect(token).NotTo(BeEmpty())
			Expect(claims).NotTo(BeNil())
			Expect(claims.FirebaseUID).To(Equal("uid-new"))
		})

		It("should fail local DB sync if UpsertUser fails", func() {
			fbToken := &firebaseAuth.Token{
				UID: "uid-123",
				Claims: map[string]interface{}{
					"email": "user@test.com",
				},
			}
			firebaseRepo.On("VerifyIDToken", ctx, "valid-token").Return(fbToken, nil)
			userRepo.On("UpsertUser", ctx, mock.Anything).Return(errors.New("db write failed"))

			token, claims, err := authSvc.ExchangeToken(ctx, "valid-token")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("internal server database synchronization error"))
			Expect(token).To(BeEmpty())
			Expect(claims).To(BeNil())
		})

		It("should demand MFA challenge when TOTP is required, user has enrolled MFA, but hasn't completed second factor check", func() {
			config.Get().Auth.TOTPRequired = true

			fbToken := &firebaseAuth.Token{
				UID: "uid-mfa",
				Claims: map[string]interface{}{
					"email": "mfa@test.com",
					"firebase": map[string]interface{}{
						"enrolled_factors": []interface{}{"totp-factor-metadata"},
					},
				},
			}
			firebaseRepo.On("VerifyIDToken", ctx, "valid-token").Return(fbToken, nil)
			userRepo.On("UpsertUser", ctx, mock.Anything).Return(nil)

			token, claims, err := authSvc.ExchangeToken(ctx, "valid-token")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("mfa_required"))
			Expect(token).To(BeEmpty())
			Expect(claims).To(BeNil())
		})

		It("should bypass MFA requirements check when TOTP is required but user has not enrolled MFA", func() {
			config.Get().Auth.TOTPRequired = true

			fbToken := &firebaseAuth.Token{
				UID: "uid-no-mfa",
				Claims: map[string]interface{}{
					"email": "nomfa@test.com",
					// No "firebase.enrolled_factors" claim here
				},
			}
			firebaseRepo.On("VerifyIDToken", ctx, "valid-token").Return(fbToken, nil)
			userRepo.On("UpsertUser", ctx, mock.Anything).Return(nil)
			userRepo.On("GetUserByFirebaseUID", ctx, "uid-no-mfa").Return(nil, nil)

			token, claims, err := authSvc.ExchangeToken(ctx, "valid-token")
			Expect(err).NotTo(HaveOccurred())
			Expect(token).NotTo(BeEmpty())
			Expect(claims.MfaVerified).To(BeFalse())
		})

		It("should succeed when TOTP is required and user has successfully completed second factor totp validation", func() {
			config.Get().Auth.TOTPRequired = true

			fbToken := &firebaseAuth.Token{
				UID: "uid-mfa-ok",
				Claims: map[string]interface{}{
					"email": "mfa-ok@test.com",
					"firebase": map[string]interface{}{
						"enrolled_factors":      []interface{}{"totp-factor-metadata"},
						"sign_in_second_factor": "totp",
					},
				},
			}
			firebaseRepo.On("VerifyIDToken", ctx, "valid-token").Return(fbToken, nil)
			userRepo.On("UpsertUser", ctx, mock.Anything).Return(nil)
			userRepo.On("GetUserByFirebaseUID", ctx, "uid-mfa-ok").Return(nil, nil)

			token, claims, err := authSvc.ExchangeToken(ctx, "valid-token")
			Expect(err).NotTo(HaveOccurred())
			Expect(token).NotTo(BeEmpty())
			Expect(claims.MfaVerified).To(BeTrue())
		})
	})

	Context("ParseAndValidateAuthToken", func() {
		It("should fail if token is invalid or expired", func() {
			claims, err := authSvc.ParseAndValidateAuthToken(ctx, "invalid-token-string")
			Expect(err).To(HaveOccurred())
			Expect(claims).To(BeNil())
		})

		It("should parse and return claims for a valid token", func() {
			// First exchange token to get a valid signed token
			fbToken := &firebaseAuth.Token{
				UID: "uid-123",
				Claims: map[string]interface{}{
					"email": "user@test.com",
				},
			}
			firebaseRepo.On("VerifyIDToken", ctx, "valid-token").Return(fbToken, nil)
			userRepo.On("UpsertUser", ctx, mock.Anything).Return(nil)
			userRepo.On("GetUserByFirebaseUID", ctx, "uid-123").Return(nil, nil)

			validTokenString, _, err := authSvc.ExchangeToken(ctx, "valid-token")
			Expect(err).NotTo(HaveOccurred())

			parsedClaims, err := authSvc.ParseAndValidateAuthToken(ctx, validTokenString)
			Expect(err).NotTo(HaveOccurred())
			Expect(parsedClaims).NotTo(BeNil())
			Expect(parsedClaims.FirebaseUID).To(Equal("uid-123"))
			Expect(parsedClaims.Email).To(Equal("user@test.com"))
		})
	})

	Context("ValidatePasswordComplexity", func() {
		It("should fail if password length is less than 6 characters", func() {
			err := service.ValidatePasswordComplexity("P!12")
			Expect(err).To(Equal(service.ErrInvalidPasswordComplexity))
		})

		It("should fail if password length is greater than 8 characters", func() {
			err := service.ValidatePasswordComplexity("LongPass!1")
			Expect(err).To(Equal(service.ErrInvalidPasswordComplexity))
		})

		It("should fail if password lacks special character", func() {
			err := service.ValidatePasswordComplexity("Pass1234")
			Expect(err).To(Equal(service.ErrInvalidPasswordComplexity))
		})

		It("should succeed when password satisfies 6-8 chars and special char requirement", func() {
			err := service.ValidatePasswordComplexity("Pass!12")
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
