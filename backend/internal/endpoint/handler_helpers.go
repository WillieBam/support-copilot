package endpoint

import (
	"errors"
	"net/http"
	"strings"

	"github.com/WillieBam/support_copilot/backend/types/models"
	customErrors "github.com/WillieBam/support_copilot/backend/utils/errors"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

// runbookNotFoundError returns a 404 for runbook-not-found errors and 500 for others
func runbookNotFoundError(c *echo.Context, err error) error {
	if errors.Is(err, customErrors.ErrRunbookNotFound) || strings.Contains(strings.ToLower(err.Error()), "not found") {
		return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
}

// checkTeamMembership verifies that the given user is a member of teamID
// writes 403 forbidden and returns false if access is denied
func (h *Handler) checkTeamMembership(c *echo.Context, user *models.User, teamID uuid.UUID) bool {
	if user.Scope == "super_admin" || h.teamService == nil {
		return true
	}
	if _, err := h.teamService.GetMemberRole(c.Request().Context(), teamID, user.ID); err != nil {
		_ = c.JSON(http.StatusForbidden, map[string]string{"error": "forbidden: not a member of this team"})
		return false
	}
	return true
}

