package service_test

import (
	"context"
	"errors"

	firebaseAuth "firebase.google.com/go/v4/auth"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

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

	Context("Register", func() {
		It("should fail if required fields are missing", func() {
			user, err := authSvc.Register(ctx, "", "email@test.com", "Pass!1")
			Expect(err).To(HaveOccurred())
			Expect(user).To(BeNil())
		})

		It("should fail if password complexity is invalid", func() {
			user, err := authSvc.Register(ctx, "john", "email@test.com", "short")
			Expect(err).To(Equal(customErrors.ErrInvalidPasswordComplexity))
			Expect(user).To(BeNil())
		})

		It("should fail if username is already taken", func() {
			existing := &models.User{Email: "existing@test.com"}
			userRepo.On("GetUserByUsernameOrEmail", ctx, "john").Return(existing, nil)

			user, err := authSvc.Register(ctx, "john", "new@test.com", "Pass!1")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("username already taken"))
			Expect(user).To(BeNil())
		})

		It("should fail if email is already registered", func() {
			userRepo.On("GetUserByUsernameOrEmail", ctx, "john").Return(nil, customErrors.ErrUserNotFound)
			existingEmail := &models.User{Email: "email@test.com"}
			userRepo.On("GetUserByUsernameOrEmail", ctx, "email@test.com").Return(existingEmail, nil)

			user, err := authSvc.Register(ctx, "john", "email@test.com", "Pass!1")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("email is already registered"))
			Expect(user).To(BeNil())
		})

		It("should fail if CreateUser repository fails", func() {
			userRepo.On("GetUserByUsernameOrEmail", ctx, "john").Return(nil, customErrors.ErrUserNotFound)
			userRepo.On("GetUserByUsernameOrEmail", ctx, "email@test.com").Return(nil, customErrors.ErrUserNotFound)
			userRepo.On("CreateUser", ctx, mock.Anything).Return(errors.New("db error"))

			user, err := authSvc.Register(ctx, "john", "email@test.com", "Pass!1")
			Expect(err).To(HaveOccurred())
			Expect(user).To(BeNil())
		})

		It("should successfully register a new user", func() {
			userRepo.On("GetUserByUsernameOrEmail", ctx, "john2").Return(nil, customErrors.ErrUserNotFound)
			userRepo.On("GetUserByUsernameOrEmail", ctx, "john2@test.com").Return(nil, customErrors.ErrUserNotFound)
			userRepo.On("CreateUser", ctx, mock.Anything).Return(nil)

			user, err := authSvc.Register(ctx, "john2", "john2@test.com", "Pass!1")
			Expect(err).NotTo(HaveOccurred())
			Expect(user).NotTo(BeNil())
			Expect(user.Email).To(Equal("john2@test.com"))
		})
	})

	Context("LoginWithPassword", func() {
		It("should fail if credentials are blank", func() {
			token, claims, err := authSvc.LoginWithPassword(ctx, "", "Pass!1234", "")
			Expect(err).To(HaveOccurred())
			Expect(token).To(BeEmpty())
			Expect(claims).To(BeNil())
		})

		It("should fail if user is not found", func() {
			userRepo.On("GetUserByUsernameOrEmail", ctx, "unknown").Return(nil, customErrors.ErrUserNotFound)

			token, claims, err := authSvc.LoginWithPassword(ctx, "unknown", "Pass!1234", "")
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, customErrors.ErrUserNotFound)).To(BeTrue())
			Expect(token).To(BeEmpty())
			Expect(claims).To(BeNil())
		})

		It("should fail if password does not match", func() {
			hash, _ := bcrypt.GenerateFromPassword([]byte("CorrectPass!12"), bcrypt.DefaultCost)
			existingUser := &models.User{
				Email:        "user@test.com",
				PasswordHash: string(hash),
			}
			userRepo.On("GetUserByUsernameOrEmail", ctx, "user@test.com").Return(existingUser, nil)

			token, claims, err := authSvc.LoginWithPassword(ctx, "user@test.com", "WrongPass!12", "")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("invalid credentials"))
			Expect(token).To(BeEmpty())
			Expect(claims).To(BeNil())
		})

		It("should demand 2fa code if TOTP is enabled", func() {
			hash, _ := bcrypt.GenerateFromPassword([]byte("CorrectPass!12"), bcrypt.DefaultCost)
			existingUser := &models.User{
				Email:        "user@test.com",
				PasswordHash: string(hash),
				TOTPEnabled:  true,
				TOTPSecret:   "JBSWY3DPEHPK3PXP",
			}
			userRepo.On("GetUserByUsernameOrEmail", ctx, "user@test.com").Return(existingUser, nil)

			token, claims, err := authSvc.LoginWithPassword(ctx, "user@test.com", "CorrectPass!12", "")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("mfa required"))
			Expect(token).To(BeEmpty())
			Expect(claims).To(BeNil())
		})

		It("should fail if 2fa code is invalid", func() {
			hash, _ := bcrypt.GenerateFromPassword([]byte("CorrectPass!12"), bcrypt.DefaultCost)
			existingUser := &models.User{
				Email:        "user@test.com",
				PasswordHash: string(hash),
				TOTPEnabled:  true,
				TOTPSecret:   "JBSWY3DPEHPK3PXP",
			}
			userRepo.On("GetUserByUsernameOrEmail", ctx, "user@test.com").Return(existingUser, nil)

			token, claims, err := authSvc.LoginWithPassword(ctx, "user@test.com", "CorrectPass!12", "000000")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("invalid 2fa code"))
			Expect(token).To(BeEmpty())
			Expect(claims).To(BeNil())
		})

		It("should successfully login when password matches and MFA is not enabled", func() {
			hash, _ := bcrypt.GenerateFromPassword([]byte("CorrectPass!12"), bcrypt.DefaultCost)
			uname := "john"
			fbUID := "fb-123"
			existingUser := &models.User{
				ID:           uuid.New(),
				Username:     &uname,
				FirebaseUID:  &fbUID,
				Email:        "user@test.com",
				PasswordHash: string(hash),
			}
			userRepo.On("GetUserByUsernameOrEmail", ctx, "user@test.com").Return(existingUser, nil)

			token, claims, err := authSvc.LoginWithPassword(ctx, "user@test.com", "CorrectPass!12", "")
			Expect(err).NotTo(HaveOccurred())
			Expect(token).NotTo(BeEmpty())
			Expect(claims.Email).To(Equal("user@test.com"))
		})
	})

	Context("SetupTOTP, VerifyAndEnableTOTP, DisableTOTP", func() {
		testID := uuid.New()

		It("should fail SetupTOTP if user is not found", func() {
			userRepo.On("GetUserByID", ctx, testID).Return(nil, customErrors.ErrUserNotFound)

			secret, qrURI, err := authSvc.SetupTOTP(ctx, testID)
			Expect(err).To(HaveOccurred())
			Expect(secret).To(BeEmpty())
			Expect(qrURI).To(BeEmpty())
		})

		It("should generate TOTP secret and QR URI on SetupTOTP", func() {
			user := &models.User{ID: testID, Email: "user@test.com"}
			userRepo.On("GetUserByID", ctx, testID).Return(user, nil)
			userRepo.On("UpdateUser", ctx, mock.Anything).Return(nil)

			secret, qrURI, err := authSvc.SetupTOTP(ctx, testID)
			Expect(err).NotTo(HaveOccurred())
			Expect(secret).NotTo(BeEmpty())
			Expect(qrURI).To(ContainSubstring("otpauth://totp/SupportCopilot"))
		})

		It("should fail SetupTOTP if UpdateUser fails", func() {
			user := &models.User{ID: testID, Email: "user@test.com"}
			userRepo.On("GetUserByID", ctx, testID).Return(user, nil)
			userRepo.On("UpdateUser", ctx, mock.Anything).Return(errors.New("db error"))

			secret, qrURI, err := authSvc.SetupTOTP(ctx, testID)
			Expect(err).To(HaveOccurred())
			Expect(secret).To(BeEmpty())
			Expect(qrURI).To(BeEmpty())
		})

		It("should fail VerifyAndEnableTOTP if user is not found", func() {
			userRepo.On("GetUserByID", ctx, testID).Return(nil, customErrors.ErrUserNotFound)

			err := authSvc.VerifyAndEnableTOTP(ctx, testID, "123456")
			Expect(err).To(HaveOccurred())
		})

		It("should fail VerifyAndEnableTOTP if TOTP secret was not initialized", func() {
			user := &models.User{ID: testID, TOTPSecret: ""}
			userRepo.On("GetUserByID", ctx, testID).Return(user, nil)

			err := authSvc.VerifyAndEnableTOTP(ctx, testID, "123456")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("totp setup has not been initialized"))
		})

		It("should fail VerifyAndEnableTOTP if TOTP code is invalid", func() {
			user := &models.User{ID: testID, TOTPSecret: "JBSWY3DPEHPK3PXP"}
			userRepo.On("GetUserByID", ctx, testID).Return(user, nil)

			err := authSvc.VerifyAndEnableTOTP(ctx, testID, "000000")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("invalid totp code"))
		})

		It("should fail DisableTOTP if user is not found", func() {
			userRepo.On("GetUserByID", ctx, testID).Return(nil, customErrors.ErrUserNotFound)

			err := authSvc.DisableTOTP(ctx, testID)
			Expect(err).To(HaveOccurred())
		})

		It("should successfully DisableTOTP for valid user", func() {
			user := &models.User{ID: testID, TOTPEnabled: true, TOTPSecret: "JBSWY3DPEHPK3PXP"}
			userRepo.On("GetUserByID", ctx, testID).Return(user, nil)
			userRepo.On("UpdateUser", ctx, mock.Anything).Return(nil)

			err := authSvc.DisableTOTP(ctx, testID)
			Expect(err).NotTo(HaveOccurred())
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
			err := service.ValidatePasswordComplexity("Pass!12")
			Expect(err).NotTo(HaveOccurred())
		})
	})
})

func strPtr(s string) *string {
	return &s
}
