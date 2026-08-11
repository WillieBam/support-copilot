package cmd

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"

	"github.com/labstack/echo/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Cmd Package", func() {
	Context("Cobra Commands", func() {
		It("should configure root command and subcommands correctly", func() {
			Expect(rootCmd).NotTo(BeNil())
			Expect(rootCmd.Use).To(Equal("support-copilot"))

			Expect(migrateCmd).NotTo(BeNil())
			Expect(migrateCmd.Use).To(Equal("migrate"))

			Expect(supportCopilotCmd).NotTo(BeNil())
			Expect(supportCopilotCmd.Use).To(Equal("server"))
		})
	})

	Context("SPA Static Handler", func() {
		var (
			tempDir   string
			e         *echo.Echo
			indexFile string
			assetFile string
		)

		BeforeEach(func() {
			var err error
			tempDir = "testdata_spa"
			err = os.MkdirAll(tempDir, 0755)
			Expect(err).NotTo(HaveOccurred())

			indexFile = filepath.Join(tempDir, "index.html")
			err = os.WriteFile(indexFile, []byte("<html>index</html>"), 0644)
			Expect(err).NotTo(HaveOccurred())

			assetFile = filepath.Join(tempDir, "asset.js")
			err = os.WriteFile(assetFile, []byte("console.log('test')"), 0644)
			Expect(err).NotTo(HaveOccurred())

			e = echo.New()
		})

		AfterEach(func() {
			os.RemoveAll(tempDir)
		})

		It("should return 404 for non GET requests", func() {
			handler := spaFallbackHandler(tempDir)

			req := httptest.NewRequest(http.MethodPost, "/asset.js", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPathValues(echo.PathValues{{Name: "*", Value: "asset.js"}})

			err := handler(c)
			Expect(err).To(Equal(echo.ErrNotFound))
		})

		It("should serve static file if it exists", func() {
			handler := spaFallbackHandler(tempDir)

			req := httptest.NewRequest(http.MethodGet, "/asset.js", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPathValues(echo.PathValues{{Name: "*", Value: "asset.js"}})

			err := handler(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusOK))
			Expect(rec.Body.String()).To(Equal("console.log('test')"))
		})

		It("should fallback to index html if file does not exist", func() {
			handler := spaFallbackHandler(tempDir)

			req := httptest.NewRequest(http.MethodGet, "/unknown-page", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPathValues(echo.PathValues{{Name: "*", Value: "unknown-page"}})

			err := handler(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusOK))
			Expect(rec.Body.String()).To(Equal("<html>index</html>"))
		})

		It("should register static SPA routes on Echo instance", func() {
			echoInst := echo.New()
			registerSPAStatic(echoInst, tempDir)

			req := httptest.NewRequest(http.MethodGet, "/static/asset.js", nil)
			rec := httptest.NewRecorder()
			echoInst.ServeHTTP(rec, req)
			Expect(rec).NotTo(BeNil())
		})
	})
})


