package server_test

import (
	"context"
	"time"

	echov5 "github.com/labstack/echo/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/WillieBam/support_copilot/backend/utils/server"
)

var _ = Describe("Server Utils", func() {
	Context("Server Config Options & Behavior", func() {
		It("should return correct Addr and Timeout when non-zero values are supplied", func() {
			cfg := server.Config{
				Name:            "TestApp",
				Port:            9090,
				ShutdownTimeout: 5 * time.Second,
			}
			Expect(cfg.Name).To(Equal("TestApp"))
			Expect(cfg.Addr()).To(Equal(":9090"))
			Expect(cfg.Timeout()).To(Equal(5 * time.Second))
		})

		It("should fall back to defaults when zero values are supplied", func() {
			cfg := server.Config{Name: "DefaultApp"}
			Expect(cfg.Name).To(Equal("DefaultApp"))
			Expect(cfg.Addr()).To(Equal(":8080"))
			Expect(cfg.Timeout()).To(Equal(10 * time.Second))
		})
	})

	Context("Server initialization and startup", func() {
		It("should start the server with the provided echo instance", func() {
			cfg := server.Config{Name: "MockServer", Port: 0, ShutdownTimeout: 1 * time.Second}
			e := echov5.New()
			e.GET("/health", func(c *echov5.Context) error {
				return c.String(200, "ok")
			})
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			err := server.Start(ctx, e, cfg)
			Expect(err).To(BeNil())
		})
	})
})

