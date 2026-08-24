package cmd

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/labstack/echo/v5"
)

// registerSPAStatic registers static file serving with a SPA fallback for the
// given Echo instance
func registerSPAStatic(e *echo.Echo, clientDir string) {
	// serve hashed static assets e.g. JS files directly
	e.Static("/static", filepath.Join(clientDir, "static"))

	// try the requested file first, fall back to index.html
	e.GET("/*", spaFallbackHandler(clientDir))
}

// spaFallbackHandler returns an echo.HandlerFunc that serves a real file when
// it exists, and falls back to index.html otherwise
func spaFallbackHandler(clientDir string) echo.HandlerFunc {
	indexHTML := filepath.Join(clientDir, "index.html")
	return func(c *echo.Context) error {
		requestedPath := filepath.Join(clientDir, filepath.Clean("/"+c.Param("*")))

		// only serve GET requests; let other methods fall through to a 404
		if c.Request().Method != http.MethodGet {
			return echo.ErrNotFound
		}

		// if the file exists on disk, serve it directly
		if _, err := os.Stat(requestedPath); err == nil {
			return c.File(requestedPath)
		}

		return c.File(indexHTML) // else fall back to index.html

	}
}
