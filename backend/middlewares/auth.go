package middlewares

import (
	"net/http"

	"github.com/WillieBam/support_copilot/backend/app/config"
	"github.com/WillieBam/support_copilot/backend/internal/interfaces"
	"github.com/google/uuid"
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

			token := cookie.Value
			if token == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "empty session token")
			}

			claims, err := authSvc.ParseAndValidateAuthToken(c.Request().Context(), token)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized session context")
			}

			userUID := claims.FirebaseUID
			if userUID == "" && claims.UserID != uuid.Nil {
				userUID = claims.UserID.String()
			}

			c.Set("user_uid", userUID)
			c.Set("user_id", claims.UserID)
			c.Set("user_email", claims.Email)
			c.Set("username", claims.Username)
			return next(c)
		}
	}
}

