package middlewares_test

import (
	"net/http"
	"net/http/httptest"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/WillieBam/support_copilot/backend/middlewares"
	"github.com/labstack/echo/v5"
)

var _ = Describe("InternalAPIKeyMiddleware", func() {
	var (
		e           *echo.Echo
		nextCalled  bool
		nextHandler echo.HandlerFunc
	)

	BeforeEach(func() {
		e = echo.New()
		nextCalled = false
		nextHandler = func(c *echo.Context) error {
			nextCalled = true
			return c.NoContent(http.StatusOK)
		}
	})

	Context("when INTERNAL_API_KEY environment variable is not set", func() {
		BeforeEach(func() {
			os.Unsetenv("INTERNAL_API_KEY")
		})

		It("should reject requests without x-internal-api-key header", func() {
			req := httptest.NewRequest(http.MethodGet, "/internal/status", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			mw := middlewares.InternalAPIKeyMiddleware()
			err := mw(nextHandler)(c)

			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusUnauthorized))
			Expect(rec.Body.String()).To(ContainSubstring("unauthorized: invalid internal api key"))
			Expect(nextCalled).To(BeFalse())
		})

		It("should reject requests with an incorrect key", func() {
			req := httptest.NewRequest(http.MethodGet, "/internal/status", nil)
			req.Header.Set("x-internal-api-key", "wrong-key")
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			mw := middlewares.InternalAPIKeyMiddleware()
			err := mw(nextHandler)(c)

			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusUnauthorized))
			Expect(rec.Body.String()).To(ContainSubstring("unauthorized: invalid internal api key"))
			Expect(nextCalled).To(BeFalse())
		})

		It("should allow requests with default dev-internal-key", func() {
			req := httptest.NewRequest(http.MethodGet, "/internal/status", nil)
			req.Header.Set("x-internal-api-key", "dev-internal-key")
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			mw := middlewares.InternalAPIKeyMiddleware()
			err := mw(nextHandler)(c)

			Expect(err).NotTo(HaveOccurred())
			Expect(nextCalled).To(BeTrue())
		})
	})

	Context("when INTERNAL_API_KEY environment variable is set", func() {
		const customKey = "secret-production-key"

		BeforeEach(func() {
			os.Setenv("INTERNAL_API_KEY", customKey)
		})

		AfterEach(func() {
			os.Unsetenv("INTERNAL_API_KEY")
		})

		It("should reject default key when custom key is configured", func() {
			req := httptest.NewRequest(http.MethodGet, "/internal/status", nil)
			req.Header.Set("x-internal-api-key", "dev-internal-key")
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			mw := middlewares.InternalAPIKeyMiddleware()
			err := mw(nextHandler)(c)

			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusUnauthorized))
			Expect(nextCalled).To(BeFalse())
		})

		It("should allow requests matching custom INTERNAL_API_KEY", func() {
			req := httptest.NewRequest(http.MethodGet, "/internal/status", nil)
			req.Header.Set("x-internal-api-key", customKey)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			mw := middlewares.InternalAPIKeyMiddleware()
			err := mw(nextHandler)(c)

			Expect(err).NotTo(HaveOccurred())
			Expect(nextCalled).To(BeTrue())
		})
	})
})
