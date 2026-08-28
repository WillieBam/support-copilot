package endpoint

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/WillieBam/support_copilot/backend/types/requests"
	customErrors "github.com/WillieBam/support_copilot/backend/utils/errors"
	"github.com/google/uuid"
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
		if err.Error() == "mfa_required" || err.Error() == "mfa required" {
			return c.JSON(http.StatusForbidden, map[string]string{
				"error":   "mfa_required",
				"message": "TOTP 2FA code is required",
			})
		}
		if errors.Is(err, customErrors.ErrUserNotFound) || err.Error() == "User not found" || err.Error() == "user not found" {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
		}
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	setSessionCookie(c, token, claims.ExpiresAt.Time)
	return c.JSON(http.StatusOK, map[string]string{"status": "authenticated"})
}

// LogoutHandler clears the session cookie
func (h *Handler) LogoutHandler(c *echo.Context) error {
	cookie := &http.Cookie{
		Name:     "support_copilot_session",
		Value:    "",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		Path:     "/",
		HttpOnly: true,
		Secure:   isSecureRequest(c),
		SameSite: http.SameSiteLaxMode,
	}
	c.SetCookie(cookie)
	return c.JSON(http.StatusOK, map[string]string{"status": "logged_out"})
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

// RefreshTokenHandler renews the active or recently expired session cookie
func (h *Handler) RefreshTokenHandler(c *echo.Context) error {
	cookie, err := c.Request().Cookie("support_copilot_session")
	if err != nil || cookie.Value == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "no active session cookie"})
	}

	newToken, claims, err := h.authService.RefreshToken(c.Request().Context(), cookie.Value)
	if err != nil {
		slog.Warn("Session refresh failed", "error", err)
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	setSessionCookie(c, newToken, claims.ExpiresAt.Time)
	slog.Info("Successfully refreshed and attached HttpOnly session cookie",
		"user_id", claims.UserID,
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

func (h *Handler) DeactivateUserHandler(c *echo.Context) error {
	user, err := h.getAuthenticatedUser(c)
	if err != nil || user == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	targetIDStr := c.Param("id")
	targetID, err := uuid.Parse(targetIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid user ID"})
	}

	if err := h.userService.DeactivateUser(c.Request().Context(), user.ID, targetID); err != nil {
		if errors.Is(err, customErrors.ErrSuperAdminRequired) || errors.Is(err, customErrors.ErrSelfDeactivationNotAllowed) {
			return c.JSON(http.StatusForbidden, map[string]string{"error": err.Error()})
		}
		if errors.Is(err, customErrors.ErrUserNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "deactivated", "message": "User account deactivated successfully"})
}

func setSessionCookie(c *echo.Context, token string, expires time.Time) {
	cookie := &http.Cookie{
		Name:     "support_copilot_session",
		Value:    token,
		Expires:  expires,
		Path:     "/",
		HttpOnly: true,
		Secure:   isSecureRequest(c),
		SameSite: http.SameSiteLaxMode,
	}
	c.SetCookie(cookie)
}

func isSecureRequest(c *echo.Context) bool {
	if c.Scheme() == "https" || c.Request().TLS != nil {
		return true
	}
	proto := c.Request().Header.Get("X-Forwarded-Proto")
	return proto == "https"
}
