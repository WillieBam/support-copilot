package middlewares

import (
	"net/http"

	"github.com/WillieBam/support_copilot/backend/app/config"
	"github.com/WillieBam/support_copilot/backend/internal/interfaces"
	"github.com/labstack/echo/v5"
)

// AuthMiddleware handles token inspection and hooks securely into Echo's context engine
func AuthMiddleware(authSvc interfaces.IAuthService) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			cfg := config.Get()

			if !cfg.Auth.Enabled {
				return next(c)
			}
			cookie, err := c.Request().Cookie("support_copilot_session")
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing session")
			}

			// token := getBearerToken(c.Request().Header.Get("Authorization"))
			token := cookie.Value
			if token == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "empty session token")
			}

			claims, err := authSvc.ParseAndValidateAuthToken(c.Request().Context(), token)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized session context")
			}

			c.Set("user_uid", claims.FirebaseUID)
			c.Set("user_email", claims.Email)
			return next(c)

		}
	}
}
