package endpoint

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/WillieBam/support_copilot/backend/types/requests"
	"github.com/labstack/echo/v5"
)

func (h *Handler) RegisterHandler(c *echo.Context) error {
	var req requests.RegisterRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
	}

	user, err := h.authService.Register(c.Request().Context(), req.Username, req.Email, req.Password)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, user)
}

func (h *Handler) LoginHandler(c *echo.Context) error {
	var req requests.LoginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
	}

	token, claims, err := h.authService.LoginWithPassword(c.Request().Context(), req.UsernameOrEmail,
		req.Password,
		req.TOTPCode)
	if err != nil {
		if err.Error() == "mfa_required" {
			return c.JSON(http.StatusForbidden, map[string]string{
				"error":   "mfa_required",
				"message": "TOTP 2FA code is required",
			})
		}
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	setSessionCookie(c, token, claims.ExpiresAt.Time)
	return c.JSON(http.StatusOK, map[string]string{"status": "authenticated"})
}

func (h *Handler) TokenExchangeHandler(c *echo.Context) error {
	var req requests.TokenExchangeRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Missing request payload"})
	}

	if req.FirebaseToken == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Missing firebase request"})
	}

	verified, claims, err := h.authService.ExchangeToken(c.Request().Context(), req.FirebaseToken, req.TOTPCode)
	if err != nil {
		if err.Error() == "mfa_required" {
			return c.JSON(http.StatusForbidden, map[string]string{
				"error":   "mfa_required",
				"message": "TOTP verification required",
			})
		}
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	setSessionCookie(c, verified, claims.ExpiresAt.Time)
	slog.Info("Successfully created and attached HttpOnly session cookie",
		"user_uid", claims.FirebaseUID,
		"expires_at", claims.ExpiresAt.Time.Format(time.RFC3339),
	)
	return c.JSON(http.StatusOK, map[string]string{"status": "authenticated"})
}

func (h *Handler) SetupTOTPHandler(c *echo.Context) error {
	user, err := h.getAuthenticatedUser(c)
	if err != nil || user == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	secret, qrURI, err := h.authService.SetupTOTP(c.Request().Context(), user.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"secret": secret,
		"qr_uri": qrURI,
	})
}

func (h *Handler) VerifyTOTPHandler(c *echo.Context) error {
	user, err := h.getAuthenticatedUser(c)
	if err != nil || user == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	var req requests.TOTPVerifyRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
	}

	if err := h.authService.VerifyAndEnableTOTP(c.Request().Context(), user.ID, req.Code); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "2fa_enabled"})
}

func (h *Handler) DisableTOTPHandler(c *echo.Context) error {
	user, err := h.getAuthenticatedUser(c)
	if err != nil || user == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	if err := h.authService.DisableTOTP(c.Request().Context(), user.ID); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "2fa_disabled"})
}

func setSessionCookie(c *echo.Context, token string, expires time.Time) {
	cookie := &http.Cookie{
		Name:     "support_copilot_session",
		Value:    token,
		Expires:  expires,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	}
	c.SetCookie(cookie)
}
