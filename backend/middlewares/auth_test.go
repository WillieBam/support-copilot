package middlewares_test

import (
	"errors"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	"github.com/WillieBam/support_copilot/backend/app/config"
	"github.com/WillieBam/support_copilot/backend/internal/mocks"
	"github.com/WillieBam/support_copilot/backend/middlewares"
	"github.com/WillieBam/support_copilot/backend/types"
	"github.com/labstack/echo/v5"
)

var _ = Describe("Middlewares", func() {
	var (
		e           *echo.Echo
		mockAuthSvc *mocks.IAuthService
	)

	BeforeEach(func() {
		e = echo.New()
		mockAuthSvc = &mocks.IAuthService{}
	})

	Context("AuthMiddleware", func() {
		var nextCalled bool
		var nextHandler echo.HandlerFunc

		BeforeEach(func() {
			nextCalled = false
			nextHandler = func(c *echo.Context) error {
				nextCalled = true
				return c.NoContent(http.StatusOK)
			}
		})

		It("should bypass auth check when auth is disabled", func() {
			config.Get().Auth.Enabled = false

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			mw := middlewares.AuthMiddleware(mockAuthSvc)
			err := mw(nextHandler)(c)

			Expect(err).NotTo(HaveOccurred())
			Expect(nextCalled).To(BeTrue())
		})

		It("should return 401 missing session if cookie is not present when auth is enabled", func() {
			config.Get().Auth.Enabled = true

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			mw := middlewares.AuthMiddleware(mockAuthSvc)
			err := mw(nextHandler)(c)

			Expect(err).To(HaveOccurred())
			he, ok := err.(*echo.HTTPError)
			Expect(ok).To(BeTrue())
			Expect(he.Code).To(Equal(http.StatusUnauthorized))
			Expect(he.Message).To(Equal("missing session"))
			Expect(nextCalled).To(BeFalse())
		})

		It("should return 401 empty session token if cookie value is empty", func() {
			config.Get().Auth.Enabled = true

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.AddCookie(&http.Cookie{
				Name:  "support_copilot_session",
				Value: "",
			})
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			mw := middlewares.AuthMiddleware(mockAuthSvc)
			err := mw(nextHandler)(c)

			Expect(err).To(HaveOccurred())
			he, ok := err.(*echo.HTTPError)
			Expect(ok).To(BeTrue())
			Expect(he.Code).To(Equal(http.StatusUnauthorized))
			Expect(he.Message).To(Equal("empty session token"))
			Expect(nextCalled).To(BeFalse())
		})

		It("should return 401 unauthorized session context if token is invalid", func() {
			config.Get().Auth.Enabled = true
			mockAuthSvc.On("ParseAndValidateAuthToken", mock.Anything, "invalid-token").Return(nil, errors.New("invalid signature"))

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.AddCookie(&http.Cookie{
				Name:  "support_copilot_session",
				Value: "invalid-token",
			})
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			mw := middlewares.AuthMiddleware(mockAuthSvc)
			err := mw(nextHandler)(c)

			Expect(err).To(HaveOccurred())
			he, ok := err.(*echo.HTTPError)
			Expect(ok).To(BeTrue())
			Expect(he.Code).To(Equal(http.StatusUnauthorized))
			Expect(he.Message).To(Equal("unauthorized session context"))
			Expect(nextCalled).To(BeFalse())
		})

		It("should populate context and proceed to next handler if token is valid", func() {
			config.Get().Auth.Enabled = true
			mockAuthSvc.On("ParseAndValidateAuthToken", mock.Anything, "valid-token").Return(&types.Claims{
				FirebaseUID: "uid-123",
				Email:       "user@example.com",
			}, nil)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.AddCookie(&http.Cookie{
				Name:  "support_copilot_session",
				Value: "valid-token",
			})
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			mw := middlewares.AuthMiddleware(mockAuthSvc)
			err := mw(nextHandler)(c)

			Expect(err).NotTo(HaveOccurred())
			Expect(nextCalled).To(BeTrue())
			Expect(c.Get("user_uid")).To(Equal("uid-123"))
			Expect(c.Get("user_email")).To(Equal("user@example.com"))
		})
	})
})
