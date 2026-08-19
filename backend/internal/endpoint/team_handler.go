package endpoint

import (
	"errors"
	"log/slog"
	"net/http"

	"strings"

	"github.com/WillieBam/support_copilot/backend/types/models"
	"github.com/WillieBam/support_copilot/backend/types/requests"
	customErrors "github.com/WillieBam/support_copilot/backend/utils/errors"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"gorm.io/gorm"
)

func (h *Handler) getAuthenticatedUser(c *echo.Context) (*models.User, error) {
	if uidVal := c.Get("user_id"); uidVal != nil {
		if uID, ok := uidVal.(uuid.UUID); ok && uID != uuid.Nil {
			if h.userService != nil {
				user, err := h.userService.GetUserByID(c.Request().Context(), uID)
				if err == nil && user != nil {
					slog.Info("[team] getAuthenticatedUser: resolved user by user_id", "user_id", user.ID, "email", user.Email)
					return user, nil
				}
			}
		}
	}

	uidVal := c.Get("user_uid")
	appUID, ok := uidVal.(string)
	if !ok || appUID == "" {
		slog.Warn("[team] getAuthenticatedUser: missing or invalid user_uid in context")
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "unauthorized session")
	}

	if h.userService == nil {
		slog.Error("[team] getAuthenticatedUser: user service is nil")
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "user service unavailable")
	}

	if parsedUUID, err := uuid.Parse(appUID); err == nil {
		user, err := h.userService.GetUserByID(c.Request().Context(), parsedUUID)
		if err == nil && user != nil {
			slog.Info("[team] getAuthenticatedUser: resolved user by parsed uuid", "user_id", user.ID, "email", user.Email)
			return user, nil
		}
	}

	user, err := h.userService.GetUserByFirebaseUID(c.Request().Context(), appUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) || errors.Is(err, customErrors.ErrUserNotFound) {
			slog.Warn("[team] getAuthenticatedUser: no user record found", "user_uid", appUID)
			return nil, echo.NewHTTPError(http.StatusNotFound, "user record not found")
		}
		slog.Error("[team] getAuthenticatedUser: database lookup failed", "user_uid", appUID, "error", err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "failed to retrieve user context")
	}
	slog.Info("[team] getAuthenticatedUser: resolved user", "user_id", user.ID, "email", user.Email)
	return user, nil
}

// CreateTeam handles POST /api/teams
func (h *Handler) CreateTeam(c *echo.Context) error {
	user, err := h.getAuthenticatedUser(c)
	if err != nil {
		return err
	}

	var req requests.CreateTeamRequest
	if err := c.Bind(&req); err != nil {
		slog.Warn("[team] CreateTeam: invalid payload", "user_id", user.ID, "error", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid payload"})
	}

	if h.teamService == nil {
		slog.Error("[team] CreateTeam: team service is nil")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "team service unavailable"})
	}

	slog.Info("[team] CreateTeam: creating team", "user_id", user.ID, "team_name", req.TeamName)
	team, err := h.teamService.CreateTeam(c.Request().Context(), req.TeamName, user.ID)
	if err != nil {
		if errors.Is(err, customErrors.ErrTeamNameRequired) || errors.Is(err, customErrors.ErrTeamNameTooLong) {
			slog.Warn("[team] CreateTeam: validation failed", "user_id", user.ID, "error", err)
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		slog.Error("[team] CreateTeam: failed to create team", "user_id", user.ID, "team_name", req.TeamName, "error", err)
		return c.JSON(http.StatusConflict, map[string]string{"error": "failed to create team: team name may already exist"})
	}

	slog.Info("[team] CreateTeam: team created successfully", "team_id", team.ID, "team_name", team.TeamName, "owner_id", user.ID)
	return c.JSON(http.StatusCreated, team)
}

// GetTeams handles GET /api/teams/me
func (h *Handler) GetTeams(c *echo.Context) error {
	user, err := h.getAuthenticatedUser(c)
	if err != nil {
		return err
	}

	if h.teamService == nil {
		slog.Error("[team] GetTeams: team service is nil")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "team service unavailable"})
	}

	slog.Info("[team] GetTeams: fetching teams for user", "user_id", user.ID, "email", user.Email)
	userWithTeams, err := h.teamService.GetUserTeams(c.Request().Context(), user.ID)
	if err != nil {
		slog.Error("[team] GetTeams: failed to fetch user teams", "user_id", user.ID, "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	membershipCount := 0
	if userWithTeams != nil {
		membershipCount = len(userWithTeams.Memberships)
	}
	slog.Info("[team] GetTeams: returning user teams", "user_id", user.ID, "membership_count", membershipCount)
	return c.JSON(http.StatusOK, userWithTeams)
}

// AddTeamMember handles POST /api/teams/:team_id/members
func (h *Handler) AddTeamMember(c *echo.Context) error {
	user, err := h.getAuthenticatedUser(c)
	if err != nil {
		return err
	}

	teamIDStr := c.Param("team_id")
	teamID, err := uuid.Parse(teamIDStr)
	if err != nil {
		slog.Warn("[team] AddTeamMember: invalid team_id param", "team_id_raw", teamIDStr, "error", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid team ID"})
	}

	var req requests.AddTeamMemberRequest
	if err := c.Bind(&req); err != nil {
		slog.Warn("[team] AddTeamMember: invalid payload", "team_id", teamID, "error", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid payload"})
	}

	if req.UserID == uuid.Nil {
		slog.Warn("[team] AddTeamMember: missing user_id in request", "team_id", teamID, "requester_id", user.ID)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "user_id is required"})
	}

	if h.teamService == nil {
		slog.Error("[team] AddTeamMember: team service is nil")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "team service unavailable"})
	}

	slog.Info("[team] AddTeamMember: adding member", "team_id", teamID, "target_user_id", req.UserID, "requester_id", user.ID)
	err = h.teamService.AddMember(c.Request().Context(), user.ID, teamID, req.UserID)
	if err != nil {
		if errors.Is(err, customErrors.ErrUnauthorizedTeamOp) {
			slog.Warn("[team] AddTeamMember: unauthorized", "team_id", teamID, "requester_id", user.ID, "error", err)
			return c.JSON(http.StatusForbidden, map[string]string{"error": err.Error()})
		}
		slog.Error("[team] AddTeamMember: failed", "team_id", teamID, "target_user_id", req.UserID, "error", err)
		return c.JSON(http.StatusConflict, map[string]string{"error": "failed to add member: member may already exist in team"})
	}

	slog.Info("[team] AddTeamMember: member added successfully", "team_id", teamID, "target_user_id", req.UserID)
	return c.JSON(http.StatusCreated, map[string]string{"status": "member added successfully"})
}

// RemoveTeamMember handles DELETE /api/teams/:team_id/members/:user_id
func (h *Handler) RemoveTeamMember(c *echo.Context) error {
	user, err := h.getAuthenticatedUser(c)
	if err != nil {
		return err
	}

	teamID, err := uuid.Parse(c.Param("team_id"))
	if err != nil {
		slog.Warn("[team] RemoveTeamMember: invalid team_id param", "error", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid team ID"})
	}

	targetUserID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		slog.Warn("[team] RemoveTeamMember: invalid user_id param", "error", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid user ID"})
	}

	if h.teamService == nil {
		slog.Error("[team] RemoveTeamMember: team service is nil")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "team service unavailable"})
	}

	slog.Info("[team] RemoveTeamMember: removing member", "team_id", teamID, "target_user_id", targetUserID, "requester_id", user.ID)
	err = h.teamService.RemoveMember(c.Request().Context(), user.ID, teamID, targetUserID)
	if err != nil {
		if errors.Is(err, customErrors.ErrUnauthorizedTeamOp) {
			slog.Warn("[team] RemoveTeamMember: unauthorized", "team_id", teamID, "requester_id", user.ID, "error", err)
			return c.JSON(http.StatusForbidden, map[string]string{"error": err.Error()})
		}
		if errors.Is(err, customErrors.ErrUserNotInTeam) || errors.Is(err, gorm.ErrRecordNotFound) {
			slog.Warn("[team] RemoveTeamMember: member not found", "team_id", teamID, "target_user_id", targetUserID)
			return c.JSON(http.StatusNotFound, map[string]string{"error": "member not found in team"})
		}
		slog.Error("[team] RemoveTeamMember: failed", "team_id", teamID, "target_user_id", targetUserID, "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	slog.Info("[team] RemoveTeamMember: member removed successfully", "team_id", teamID, "target_user_id", targetUserID)
	return c.JSON(http.StatusOK, map[string]string{"status": "member removed successfully"})
}

// DeleteTeam handles DELETE /api/teams/:team_id (super_admin only)
func (h *Handler) DeleteTeam(c *echo.Context) error {
	user, err := h.getAuthenticatedUser(c)
	if err != nil {
		return err
	}

	teamID, err := uuid.Parse(c.Param("team_id"))
	if err != nil {
		slog.Warn("[team] DeleteTeam: invalid team_id param", "error", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid team ID"})
	}

	if h.teamService == nil {
		slog.Error("[team] DeleteTeam: team service is nil")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "team service unavailable"})
	}

	slog.Info("[team] DeleteTeam: attempting delete", "team_id", teamID, "requester_id", user.ID, "scope", user.Scope)
	err = h.teamService.DeleteTeam(c.Request().Context(), user.Scope, teamID)
	if err != nil {
		if errors.Is(err, customErrors.ErrSuperAdminRequired) {
			slog.Warn("[team] DeleteTeam: forbidden - not super_admin", "team_id", teamID, "requester_id", user.ID, "scope", user.Scope)
			return c.JSON(http.StatusForbidden, map[string]string{"error": err.Error()})
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			slog.Warn("[team] DeleteTeam: team not found", "team_id", teamID)
			return c.JSON(http.StatusNotFound, map[string]string{"error": "team not found"})
		}
		slog.Error("[team] DeleteTeam: failed", "team_id", teamID, "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	slog.Info("[team] DeleteTeam: team deleted successfully", "team_id", teamID, "deleted_by", user.ID)
	return c.JSON(http.StatusOK, map[string]string{"status": "team deleted successfully"})
}

// ListAllTeams handles GET /api/admin/teams
func (h *Handler) ListAllTeams(c *echo.Context) error {
	user, err := h.getAuthenticatedUser(c)
	if err != nil {
		return err
	}

	if h.teamService == nil {
		slog.Error("[team] ListAllTeams: team service is nil")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "team service unavailable"})
	}

	slog.Info("[team] ListAllTeams: listing all teams", "requester_id", user.ID, "scope", user.Scope)
	teams, err := h.teamService.ListAllTeams(c.Request().Context(), user.Scope)
	if err != nil {
		if errors.Is(err, customErrors.ErrSuperAdminRequired) {
			slog.Warn("[team] ListAllTeams: forbidden - not super_admin", "requester_id", user.ID, "scope", user.Scope)
			return c.JSON(http.StatusForbidden, map[string]string{"error": err.Error()})
		}
		slog.Error("[team] ListAllTeams: failed", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, teams)
}

// AssignTeamIncident handles POST /api/teams/:team_id/incidents
func (h *Handler) AssignTeamIncident(c *echo.Context) error {
	user, err := h.getAuthenticatedUser(c)
	if err != nil {
		return err
	}

	teamID, err := uuid.Parse(c.Param("team_id"))
	if err != nil {
		slog.Warn("[team] AssignTeamIncident: invalid team_id param", "error", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid team ID"})
	}

	var req requests.AssignTeamIncidentRequest
	if err := c.Bind(&req); err != nil {
		slog.Warn("[team] AssignTeamIncident: invalid payload", "team_id", teamID, "error", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid payload"})
	}

	if h.teamService == nil {
		slog.Error("[team] AssignTeamIncident: team service is nil")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "team service unavailable"})
	}

	slog.Info("[team] AssignTeamIncident: assigning incident", "team_id", teamID, "title", req.Title, "requester_id", user.ID)
	inc, err := h.teamService.AssignIncident(c.Request().Context(), user.ID, teamID, req.Title, req.Status, req.Details)
	if err != nil {
		if errors.Is(err, customErrors.ErrUnauthorizedTeamOp) {
			slog.Warn("[team] AssignTeamIncident: unauthorized", "team_id", teamID, "requester_id", user.ID, "error", err)
			return c.JSON(http.StatusForbidden, map[string]string{"error": err.Error()})
		}
		slog.Error("[team] AssignTeamIncident: failed", "team_id", teamID, "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	slog.Info("[team] AssignTeamIncident: incident assigned successfully", "team_incident_id", inc.ID, "team_id", teamID)

	var alertsToLink []string
	if strings.TrimSpace(req.AlertID) != "" {
		alertsToLink = append(alertsToLink, req.AlertID)
	}
	if len(req.AlertIDs) > 0 {
		alertsToLink = append(alertsToLink, req.AlertIDs...)
	}
	if len(alertsToLink) > 0 {
		_ = h.teamService.LinkAlertsToIncident(c.Request().Context(), alertsToLink, inc.ID)
	}

	return c.JSON(http.StatusCreated, inc)
}

// GetTeamIncidents handles GET /api/teams/:team_id/incidents
func (h *Handler) GetTeamIncidents(c *echo.Context) error {
	user, err := h.getAuthenticatedUser(c)
	if err != nil {
		return err
	}

	teamID, err := uuid.Parse(c.Param("team_id"))
	if err != nil {
		slog.Warn("[team] GetTeamIncidents: invalid team_id param", "error", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid team ID"})
	}

	if h.teamService == nil {
		slog.Error("[team] GetTeamIncidents: team service is nil")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "team service unavailable"})
	}

	slog.Info("[team] GetTeamIncidents: fetching incidents", "team_id", teamID, "requester_id", user.ID)
	incidents, err := h.teamService.ListIncidents(c.Request().Context(), user.ID, teamID)
	if err != nil {
		if errors.Is(err, customErrors.ErrUnauthorizedTeamOp) {
			slog.Warn("[team] GetTeamIncidents: unauthorized", "team_id", teamID, "requester_id", user.ID, "error", err)
			return c.JSON(http.StatusForbidden, map[string]string{"error": err.Error()})
		}
		slog.Error("[team] GetTeamIncidents: failed", "team_id", teamID, "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	slog.Info("[team] GetTeamIncidents: returning incidents", "team_id", teamID, "incident_count", len(incidents))
	return c.JSON(http.StatusOK, incidents)
}

// GetTeamMembers handles GET /api/teams/:team_id/members
func (h *Handler) GetTeamMembers(c *echo.Context) error {
	user, err := h.getAuthenticatedUser(c)
	if err != nil {
		return err
	}

	teamID, err := uuid.Parse(c.Param("team_id"))
	if err != nil {
		slog.Warn("[team] GetTeamMembers: invalid team_id param", "error", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid team ID"})
	}

	if h.teamService == nil {
		slog.Error("[team] GetTeamMembers: team service is nil")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "team service unavailable"})
	}

	slog.Info("[team] GetTeamMembers: fetching members", "team_id", teamID, "requester_id", user.ID)
	members, err := h.teamService.ListMembers(c.Request().Context(), user.ID, teamID)
	if err != nil {
		if errors.Is(err, customErrors.ErrUnauthorizedTeamOp) {
			slog.Warn("[team] GetTeamMembers: unauthorized", "team_id", teamID, "requester_id", user.ID, "error", err)
			return c.JSON(http.StatusForbidden, map[string]string{"error": err.Error()})
		}
		slog.Error("[team] GetTeamMembers: failed", "team_id", teamID, "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	slog.Info("[team] GetTeamMembers: returning members", "team_id", teamID, "member_count", len(members))
	return c.JSON(http.StatusOK, members)
}

// GetTeamIncident handles GET /api/incidents/:id
func (h *Handler) GetTeamIncident(c *echo.Context) error {
	user, err := h.getAuthenticatedUser(c)
	if err != nil {
		return err
	}

	incID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		slog.Warn("[team] GetTeamIncident: invalid incident id param", "error", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid incident ID"})
	}

	if h.teamService == nil {
		slog.Error("[team] GetTeamIncident: team service is nil")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "team service unavailable"})
	}

	slog.Info("[team] GetTeamIncident: fetching incident", "incident_id", incID, "requester_id", user.ID)
	inc, err := h.teamService.GetIncident(c.Request().Context(), user.ID, incID)
	if err != nil {
		if errors.Is(err, customErrors.ErrUnauthorizedTeamOp) {
			slog.Warn("[team] GetTeamIncident: unauthorized", "incident_id", incID, "requester_id", user.ID, "error", err)
			return c.JSON(http.StatusForbidden, map[string]string{"error": err.Error()})
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			slog.Warn("[team] GetTeamIncident: incident not found", "incident_id", incID)
			return c.JSON(http.StatusNotFound, map[string]string{"error": "incident not found"})
		}
		slog.Error("[team] GetTeamIncident: failed", "incident_id", incID, "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, inc)
}

// UpdateTeamIncidentStatus handles PUT /api/incidents/:id
func (h *Handler) UpdateTeamIncidentStatus(c *echo.Context) error {
	user, err := h.getAuthenticatedUser(c)
	if err != nil {
		return err
	}

	incID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		slog.Warn("[team] UpdateTeamIncidentStatus: invalid incident id param", "error", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid incident ID"})
	}

	var req requests.UpdateIncidentStatusRequest
	if err := c.Bind(&req); err != nil {
		slog.Warn("[team] UpdateTeamIncidentStatus: invalid payload", "incident_id", incID, "error", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid payload"})
	}

	if h.teamService == nil {
		slog.Error("[team] UpdateTeamIncidentStatus: team service is nil")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "team service unavailable"})
	}

	slog.Info("[team] UpdateTeamIncidentStatus: updating status", "incident_id", incID, "status", req.Status, "requester_id", user.ID)
	inc, err := h.teamService.UpdateIncidentStatus(c.Request().Context(), user.ID, incID, req.Status, req.Title, req.Details)
	if err != nil {
		if errors.Is(err, customErrors.ErrInvalidIncidentStatus) {
			slog.Warn("[team] UpdateTeamIncidentStatus: invalid status", "incident_id", incID, "status", req.Status, "error", err)
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		if errors.Is(err, customErrors.ErrUnauthorizedTeamOp) {
			slog.Warn("[team] UpdateTeamIncidentStatus: unauthorized", "incident_id", incID, "requester_id", user.ID, "error", err)
			return c.JSON(http.StatusForbidden, map[string]string{"error": err.Error()})
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			slog.Warn("[team] UpdateTeamIncidentStatus: incident not found", "incident_id", incID)
			return c.JSON(http.StatusNotFound, map[string]string{"error": "incident not found"})
		}
		slog.Error("[team] UpdateTeamIncidentStatus: failed", "incident_id", incID, "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, inc)
}

type TeamInstructionResponse struct {
	Instruction *models.Instruction     `json:"instruction"`
	Logs        []models.InstructionLog `json:"logs"`
}

// GetTeamInstruction handles GET /api/teams/:team_id/instruction
func (h *Handler) GetTeamInstruction(c *echo.Context) error {
	user, err := h.getAuthenticatedUser(c)
	if err != nil {
		return err
	}

	teamID, err := uuid.Parse(c.Param("team_id"))
	if err != nil {
		slog.Warn("[team] GetTeamInstruction: invalid team_id param", "error", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid team ID"})
	}

	if h.teamService == nil {
		slog.Error("[team] GetTeamInstruction: team service is nil")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "team service unavailable"})
	}

	slog.Info("[team] GetTeamInstruction: fetching instruction", "team_id", teamID, "requester_id", user.ID)
	inst, logs, err := h.teamService.GetTeamInstruction(c.Request().Context(), user.ID, teamID)
	if err != nil {
		if errors.Is(err, customErrors.ErrUnauthorizedTeamOp) {
			slog.Warn("[team] GetTeamInstruction: unauthorized", "team_id", teamID, "requester_id", user.ID)
			return c.JSON(http.StatusForbidden, map[string]string{"error": err.Error()})
		}
		slog.Error("[team] GetTeamInstruction: failed", "team_id", teamID, "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if logs == nil {
		logs = []models.InstructionLog{}
	}

	return c.JSON(http.StatusOK, TeamInstructionResponse{
		Instruction: inst,
		Logs:        logs,
	})
}

// SaveTeamInstruction handles POST /api/teams/:team_id/instruction
func (h *Handler) SaveTeamInstruction(c *echo.Context) error {
	user, err := h.getAuthenticatedUser(c)
	if err != nil {
		return err
	}

	teamID, err := uuid.Parse(c.Param("team_id"))
	if err != nil {
		slog.Warn("[team] SaveTeamInstruction: invalid team_id param", "error", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid team ID"})
	}

	var req requests.SaveTeamInstructionRequest
	if err := c.Bind(&req); err != nil {
		slog.Warn("[team] SaveTeamInstruction: invalid payload", "team_id", teamID, "error", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid payload"})
	}

	if h.teamService == nil {
		slog.Error("[team] SaveTeamInstruction: team service is nil")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "team service unavailable"})
	}

	slog.Info("[team] SaveTeamInstruction: saving instruction", "team_id", teamID, "requester_id", user.ID)
	inst, err := h.teamService.SaveTeamInstruction(c.Request().Context(), user.ID, teamID, req.InstructionDetails)
	if err != nil {
		if errors.Is(err, customErrors.ErrInstructionTooShort) {
			slog.Warn("[team] SaveTeamInstruction: instruction too short", "team_id", teamID, "error", err)
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		if errors.Is(err, customErrors.ErrUnauthorizedTeamOp) {
			slog.Warn("[team] SaveTeamInstruction: unauthorized", "team_id", teamID, "requester_id", user.ID)
			return c.JSON(http.StatusForbidden, map[string]string{"error": err.Error()})
		}
		slog.Error("[team] SaveTeamInstruction: failed", "team_id", teamID, "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, inst)
}
