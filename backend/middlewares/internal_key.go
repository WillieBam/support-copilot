package middlewares

import (
	"net/http"
	"os"

	"github.com/labstack/echo/v5"
)

// InternalAPIKeyMiddleware validates the x-internal-api-key header for /internal/* routes
// these routes are intended to be called only from within the docker network (eg mcp servers)
func InternalAPIKeyMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			key := c.Request().Header.Get("x-internal-api-key")
			expected := os.Getenv("INTERNAL_API_KEY")
			if expected == "" {
				expected = "dev-internal-key"
			}
			if key != expected {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "unauthorized: invalid internal api key",
				})
			}
			return next(c)
		}
	}
}
