package config_test

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/WillieBam/support_copilot/backend/app/config"
)

var _ = Describe("Config", func() {
	BeforeEach(func() {
		// Unset any environment variables that might interfere with defaults
		os.Unsetenv("AUTH_TOTP_REQUIRED")
		os.Unsetenv("DATABASE_HOST")
		os.Unsetenv("DATABASE_PORT")
		os.Unsetenv("HTTP_PORT")
		os.Unsetenv("OLLAMA_MODEL")
	})

	Context("Retrieving config", func() {
		It("should successfully return a config object with default settings", func() {
			cfg := config.Get()
			Expect(cfg).NotTo(BeNil())

			// Verify some basic default properties
			Expect(cfg.Http.Port).To(Equal(8080))
			Expect(cfg.Database.Host).To(Equal("localhost"))
			Expect(cfg.Database.Port).To(Equal(5432))
			Expect(cfg.Auth.TOTPRequired).To(BeFalse())
			Expect(cfg.Ollama.Model).To(Equal("llama3.2:latest"))
		})
	})
})
