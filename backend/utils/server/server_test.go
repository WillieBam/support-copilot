package server_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/WillieBam/support_copilot/backend/utils/server"
)

// MockServerConfig satisfies the IServer interface
type MockServerConfig struct {
	port     string
	timeout  time.Duration
	name     string
}

func (m *MockServerConfig) Name() string {
	return m.name
}

func (m *MockServerConfig) Port() string {
	return m.port
}

func (m *MockServerConfig) GetShutdownTimeOutDuration() time.Duration {
	return m.timeout
}

var _ = Describe("Server Utils", func() {
	Context("ServerConfig Options & Behavior", func() {
		It("should return correct Name, Port, and ShutdownTimeout when non-zero values are supplied", func() {
			cfg := server.NewServerConfig("TestApp", 9090, 5*time.Second)
			Expect(cfg.Name()).To(Equal("TestApp"))
			Expect(cfg.Port()).To(Equal("9090"))
			Expect(cfg.GetShutdownTimeOutDuration()).To(Equal(5 * time.Second))
		})

		It("should fall back to defaults when zero values are supplied", func() {
			cfg := server.NewServerConfig("DefaultApp", 0, 0)
			Expect(cfg.Name()).To(Equal("DefaultApp"))
			Expect(cfg.Port()).To(Equal("8080"))
			Expect(cfg.GetShutdownTimeOutDuration()).To(Equal(10 * time.Second))
		})
	})

	Context("Server initialization", func() {
		It("should initialize Server instance with echo framework and provided configuration", func() {
			mockCfg := &MockServerConfig{
				name:    "MockServer",
				port:    "8081",
				timeout: 2 * time.Second,
			}

			srv := server.New(mockCfg)
			Expect(srv).NotTo(BeNil())
			Expect(srv.Echo).NotTo(BeNil())
		})
	})
})
