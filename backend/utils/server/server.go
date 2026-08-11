package server

import (
	"context"
	"net/http"
	"time"

	"github.com/labstack/echo/v5"
)

// Start runs echo instance with graceful shutdown
func Start(ctx context.Context, e *echo.Echo, cfg Config) error {
	sc := echo.StartConfig{
		Address:         cfg.Addr(),
		HideBanner:      true,
		GracefulTimeout: cfg.Timeout(),
		BeforeServeFunc: func(s *http.Server) error {
			s.WriteTimeout = 0
			s.ReadTimeout = 5 * time.Minute
			s.IdleTimeout = 120 * time.Second
			return nil
		},
	}

	return sc.Start(ctx, e)
}

