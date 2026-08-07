package endpoint

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	customErrors "github.com/WillieBam/support_copilot/backend/utils/errors"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

// defaultSLATarget is the default sla threshold used when the query param is absent
const defaultSLATarget = 30

// parseSLATarget reads sla_target_minutes from query params, returns defaultSLATarget if absent
func parseSLATarget(c *echo.Context) (int, error) {
	raw := c.QueryParam("sla_target_minutes")
	if raw == "" {
		return defaultSLATarget, nil
	}
	val, err := strconv.Atoi(raw)
	if err != nil || val < 0 {
		return 0, errors.New("sla_target_minutes must be a non-negative integer")
	}
	return val, nil
}

// GetIncidentTrend handles GET /api/dashboard/incidents/trend
func (h *Handler) GetIncidentTrend(c *echo.Context) error {
	user, err := h.getAuthenticatedUser(c)
	if err != nil {
		return err
	}

	teamID, err := uuid.Parse(c.QueryParam("team_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid or missing team_id"})
	}

	timeframe := c.QueryParam("timeframe")
	if timeframe == "" {
		timeframe = "month"
	}

	slog.Info("[dashboard] GetIncidentTrend", "team_id", teamID, "timeframe", timeframe, "requester_id", user.ID)
	results, err := h.dashboardService.GetIncidentTrend(c.Request().Context(), user.ID, teamID, user.Scope, timeframe)
	if err != nil {
		if errors.Is(err, customErrors.ErrInvalidTimeframe) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		if errors.Is(err, customErrors.ErrDashboardUnauthorized) {
			return c.JSON(http.StatusForbidden, map[string]string{"error": err.Error()})
		}
		slog.Error("[dashboard] GetIncidentTrend failed", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, results)
}

// GetMTTR handles GET /api/dashboard/mttr
func (h *Handler) GetMTTR(c *echo.Context) error {
	user, err := h.getAuthenticatedUser(c)
	if err != nil {
		return err
	}

	teamID, err := uuid.Parse(c.QueryParam("team_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid or missing team_id"})
	}

	slaTarget, err := parseSLATarget(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	slog.Info("[dashboard] GetMTTR", "team_id", teamID, "sla_target_minutes", slaTarget, "requester_id", user.ID)
	result, err := h.dashboardService.GetMTTR(c.Request().Context(), user.ID, teamID, user.Scope, slaTarget)
	if err != nil {
		if errors.Is(err, customErrors.ErrInvalidSLATarget) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		if errors.Is(err, customErrors.ErrDashboardUnauthorized) {
			return c.JSON(http.StatusForbidden, map[string]string{"error": err.Error()})
		}
		slog.Error("[dashboard] GetMTTR failed", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, result)
}

// GetBreachedIncidents handles GET /api/dashboard/incidents/breached
func (h *Handler) GetBreachedIncidents(c *echo.Context) error {
	user, err := h.getAuthenticatedUser(c)
	if err != nil {
		return err
	}

	teamID, err := uuid.Parse(c.QueryParam("team_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid or missing team_id"})
	}

	slaTarget, err := parseSLATarget(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	limit := 50
	if l := c.QueryParam("limit"); l != "" {
		if parsed, parseErr := strconv.Atoi(l); parseErr == nil && parsed > 0 {
			limit = parsed
		}
	}

	offset := 0
	if o := c.QueryParam("offset"); o != "" {
		if parsed, parseErr := strconv.Atoi(o); parseErr == nil && parsed >= 0 {
			offset = parsed
		}
	}

	slog.Info("[dashboard] GetBreachedIncidents", "team_id", teamID, "sla_target_minutes", slaTarget, "requester_id", user.ID)
	results, err := h.dashboardService.GetBreachedIncidents(c.Request().Context(), user.ID, teamID, user.Scope, slaTarget, limit, offset)
	if err != nil {
		if errors.Is(err, customErrors.ErrInvalidSLATarget) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		if errors.Is(err, customErrors.ErrDashboardUnauthorized) {
			return c.JSON(http.StatusForbidden, map[string]string{"error": err.Error()})
		}
		slog.Error("[dashboard] GetBreachedIncidents failed", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, results)
}
