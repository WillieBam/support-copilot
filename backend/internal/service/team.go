package service

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/WillieBam/support_copilot/backend/internal/interfaces"
	"github.com/WillieBam/support_copilot/backend/types/models"
	customErrors "github.com/WillieBam/support_copilot/backend/utils/errors"
	"github.com/google/uuid"
)

func normalizeIncidentStatus(status string) (string, error) {
	upper := strings.ToUpper(strings.TrimSpace(status))
	if upper == "" {
		return "OPEN", nil
	}
	switch upper {
	case "OPEN":
		return "OPEN", nil
	case "IN_PROGRESS", "IN PRORESS", "INPROGRESS":
		return "IN_PROGRESS", nil
	case "RESOLVED":
		return "RESOLVED", nil
	case "CLOSED":
		return "CLOSED", nil
	default:
		return "", customErrors.ErrInvalidIncidentStatus
	}
}

type teamService struct {
	teamRepo  interfaces.ITeamRepository
	alertRepo interfaces.IAlertRepository
}

func NewTeamService(teamRepo interfaces.ITeamRepository, opts ...interface{}) interfaces.ITeamService {
	svc := &teamService{teamRepo: teamRepo}
	for _, opt := range opts {
		if ar, ok := opt.(interfaces.IAlertRepository); ok && ar != nil {
			svc.alertRepo = ar
		}
	}
	return svc
}

func (s *teamService) CreateTeam(ctx context.Context, teamName string, creatorID uuid.UUID) (*models.Team, error) {
	name := strings.TrimSpace(teamName)
	if name == "" {
		return nil, customErrors.ErrTeamNameRequired
	}
	if len(name) > 20 {
		return nil, customErrors.ErrTeamNameTooLong
	}

	team := &models.Team{
		ID:        uuid.New(),
		TeamName:  name,
		CreatedAt: time.Now(),
	}

	slog.InfoContext(ctx, "[team-svc] CreateTeam: persisting team with owner", "team_id", team.ID, "team_name", name, "creator_id", creatorID)
	err := s.teamRepo.CreateTeamWithOwner(ctx, team, creatorID)
	if err != nil {
		slog.ErrorContext(ctx, "[team-svc] CreateTeam: repository error", "team_name", name, "creator_id", creatorID, "error", err)
		return nil, err
	}
	slog.InfoContext(ctx, "[team-svc] CreateTeam: success", "team_id", team.ID)
	return team, nil
}

func (s *teamService) GetTeam(ctx context.Context, teamID uuid.UUID) (*models.Team, error) {
	slog.InfoContext(ctx, "[team-svc] GetTeam: fetching team", "team_id", teamID)
	team, err := s.teamRepo.GetTeamByID(ctx, teamID)
	if err != nil {
		slog.ErrorContext(ctx, "[team-svc] GetTeam: failed", "team_id", teamID, "error", err)
		return nil, err
	}
	slog.InfoContext(ctx, "[team-svc] GetTeam: found", "team_id", teamID, "team_name", team.TeamName, "member_count", len(team.Members))
	return team, nil
}

func (s *teamService) GetUserTeams(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	slog.InfoContext(ctx, "[team-svc] GetUserTeams: loading user with memberships", "user_id", userID)
	user, err := s.teamRepo.GetUserWithTeamsByID(ctx, userID)
	if err != nil {
		slog.ErrorContext(ctx, "[team-svc] GetUserTeams: failed", "user_id", userID, "error", err)
		return nil, err
	}
	slog.InfoContext(ctx, "[team-svc] GetUserTeams: loaded", "user_id", userID, "membership_count", len(user.Memberships))
	return user, nil
}

func (s *teamService) AddMember(ctx context.Context, requesterID, teamID, userID uuid.UUID) error {
	slog.InfoContext(ctx, "[team-svc] AddMember: checking requester role", "team_id", teamID, "requester_id", requesterID, "target_user_id", userID)
	reqRole, err := s.checkMemberRoleOrSuperAdmin(ctx, teamID, requesterID)
	if err != nil || reqRole != "owner" {
		slog.WarnContext(ctx, "[team-svc] AddMember: requester is not owner", "team_id", teamID, "requester_id", requesterID, "role", reqRole, "error", err)
		return customErrors.ErrUnauthorizedTeamOp
	}

	member := &models.TeamMember{
		ID:     uuid.New(),
		TeamID: teamID,
		UserID: userID,
		Role:   "member",
	}
	slog.InfoContext(ctx, "[team-svc] AddMember: inserting member", "team_id", teamID, "member_id", member.ID, "target_user_id", userID)
	if err := s.teamRepo.AddTeamMember(ctx, member); err != nil {
		slog.ErrorContext(ctx, "[team-svc] AddMember: failed", "team_id", teamID, "target_user_id", userID, "error", err)
		return err
	}
	slog.InfoContext(ctx, "[team-svc] AddMember: success", "team_id", teamID, "target_user_id", userID)
	return nil
}

func (s *teamService) RemoveMember(ctx context.Context, requesterID, teamID, userID uuid.UUID) error {
	slog.InfoContext(ctx, "[team-svc] RemoveMember: checking requester role", "team_id", teamID, "requester_id", requesterID, "target_user_id", userID)
	reqRole, err := s.checkMemberRoleOrSuperAdmin(ctx, teamID, requesterID)
	if err != nil || reqRole != "owner" {
		slog.WarnContext(ctx, "[team-svc] RemoveMember: requester is not owner", "team_id", teamID, "requester_id", requesterID, "role", reqRole, "error", err)
		return customErrors.ErrUnauthorizedTeamOp
	}

	_, err = s.teamRepo.GetMemberRole(ctx, teamID, userID)
	if err != nil {
		slog.WarnContext(ctx, "[team-svc] RemoveMember: target user not in team", "team_id", teamID, "target_user_id", userID, "error", err)
		return customErrors.ErrUserNotInTeam
	}

	slog.InfoContext(ctx, "[team-svc] RemoveMember: removing", "team_id", teamID, "target_user_id", userID)
	if err := s.teamRepo.RemoveTeamMember(ctx, teamID, userID); err != nil {
		slog.ErrorContext(ctx, "[team-svc] RemoveMember: failed", "team_id", teamID, "target_user_id", userID, "error", err)
		return err
	}
	slog.InfoContext(ctx, "[team-svc] RemoveMember: success", "team_id", teamID, "target_user_id", userID)
	return nil
}

func (s *teamService) DeleteTeam(ctx context.Context, userScope string, teamID uuid.UUID) error {
	if userScope != "super_admin" {
		slog.WarnContext(ctx, "[team-svc] DeleteTeam: forbidden - not super_admin", "team_id", teamID, "scope", userScope)
		return customErrors.ErrSuperAdminRequired
	}
	slog.InfoContext(ctx, "[team-svc] DeleteTeam: deleting team", "team_id", teamID, "scope", userScope)
	if err := s.teamRepo.DeleteTeam(ctx, teamID); err != nil {
		slog.ErrorContext(ctx, "[team-svc] DeleteTeam: failed", "team_id", teamID, "error", err)
		return err
	}
	slog.InfoContext(ctx, "[team-svc] DeleteTeam: success", "team_id", teamID)
	return nil
}

// ListAllTeams returns all system teams for super_admin
func (s *teamService) ListAllTeams(ctx context.Context, userScope string) ([]models.Team, error) {
	if userScope != "super_admin" {
		slog.WarnContext(ctx, "[team-svc] ListAllTeams: forbidden - not super_admin", "scope", userScope)
		return nil, customErrors.ErrSuperAdminRequired
	}
	slog.InfoContext(ctx, "[team-svc] ListAllTeams: listing all teams", "scope", userScope)
	teams, err := s.teamRepo.ListAllTeams(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "[team-svc] ListAllTeams: failed", "error", err)
		return nil, err
	}
	return teams, nil
}

// checkMemberRoleOrSuperAdmin returns owner role for super_admin or checks member role
func (s *teamService) checkMemberRoleOrSuperAdmin(ctx context.Context, teamID, requesterID uuid.UUID) (string, error) {
	role, err := s.teamRepo.GetMemberRole(ctx, teamID, requesterID)
	if err == nil {
		return role, nil
	}
	user, userErr := s.teamRepo.GetUserWithTeamsByID(ctx, requesterID)
	if userErr == nil && user != nil && user.Scope == "super_admin" {
		return "owner", nil
	}
	return "", err
}

func (s *teamService) AssignIncident(ctx context.Context, requesterID, teamID uuid.UUID, title, status, details string) (*models.TeamIncident, error) {
	slog.InfoContext(ctx, "[team-svc] AssignIncident: checking membership", "team_id", teamID, "requester_id", requesterID)
	_, err := s.checkMemberRoleOrSuperAdmin(ctx, teamID, requesterID)
	if err != nil {
		slog.WarnContext(ctx, "[team-svc] AssignIncident: requester not in team", "team_id", teamID, "requester_id", requesterID, "error", err)
		return nil, customErrors.ErrUnauthorizedTeamOp
	}

	validStatus, err := normalizeIncidentStatus(status)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	inc := &models.TeamIncident{
		ID:         uuid.New(),
		TeamID:     teamID,
		CreatedBy:  requesterID,
		Title:      strings.TrimSpace(title),
		Status:     validStatus,
		Details:    details,
		CreatedAt:  now,
		AssignedAt: now,
	}

	slog.InfoContext(ctx, "[team-svc] AssignIncident: persisting", "team_incident_id", inc.ID, "team_id", teamID)
	err = s.teamRepo.AssignTeamIncident(ctx, inc)
	if err != nil {
		slog.ErrorContext(ctx, "[team-svc] AssignIncident: failed", "team_id", teamID, "error", err)
		return nil, err
	}
	slog.InfoContext(ctx, "[team-svc] AssignIncident: success", "team_incident_id", inc.ID)
	return inc, nil
}

func (s *teamService) ListIncidents(ctx context.Context, requesterID, teamID uuid.UUID) ([]models.TeamIncident, error) {
	slog.InfoContext(ctx, "[team-svc] ListIncidents: checking membership", "team_id", teamID, "requester_id", requesterID)
	_, err := s.checkMemberRoleOrSuperAdmin(ctx, teamID, requesterID)
	if err != nil {
		slog.WarnContext(ctx, "[team-svc] ListIncidents: requester not in team", "team_id", teamID, "requester_id", requesterID, "error", err)
		return nil, customErrors.ErrUnauthorizedTeamOp
	}
	incidents, err := s.teamRepo.ListTeamIncidents(ctx, teamID)
	if err != nil {
		slog.ErrorContext(ctx, "[team-svc] ListIncidents: failed", "team_id", teamID, "error", err)
		return nil, err
	}
	slog.InfoContext(ctx, "[team-svc] ListIncidents: returning", "team_id", teamID, "count", len(incidents))
	return incidents, nil
}

func (s *teamService) ListTeamIncidents(ctx context.Context, teamID uuid.UUID) ([]models.TeamIncident, error) {
	slog.InfoContext(ctx, "[team-svc] ListTeamIncidents internal call", "team_id", teamID)
	return s.teamRepo.ListTeamIncidents(ctx, teamID)
}

func (s *teamService) ListMembers(ctx context.Context, requesterID, teamID uuid.UUID) ([]models.TeamMember, error) {
	slog.InfoContext(ctx, "[team-svc] ListMembers: checking membership", "team_id", teamID, "requester_id", requesterID)
	_, err := s.checkMemberRoleOrSuperAdmin(ctx, teamID, requesterID)
	if err != nil {
		slog.WarnContext(ctx, "[team-svc] ListMembers: requester not in team", "team_id", teamID, "requester_id", requesterID, "error", err)
		return nil, customErrors.ErrUnauthorizedTeamOp
	}
	members, err := s.teamRepo.ListTeamMembers(ctx, teamID)
	if err != nil {
		slog.ErrorContext(ctx, "[team-svc] ListMembers: failed", "team_id", teamID, "error", err)
		return nil, err
	}
	slog.InfoContext(ctx, "[team-svc] ListMembers: returning", "team_id", teamID, "count", len(members))
	return members, nil
}

func (s *teamService) GetIncident(ctx context.Context, requesterID, incidentID uuid.UUID) (*models.TeamIncident, error) {
	slog.InfoContext(ctx, "[team-svc] GetIncident: fetching incident", "incident_id", incidentID, "requester_id", requesterID)
	inc, err := s.teamRepo.GetTeamIncidentByID(ctx, incidentID)
	if err != nil {
		slog.ErrorContext(ctx, "[team-svc] GetIncident: failed", "incident_id", incidentID, "error", err)
		return nil, err
	}

	_, err = s.checkMemberRoleOrSuperAdmin(ctx, inc.TeamID, requesterID)
	if err != nil {
		slog.WarnContext(ctx, "[team-svc] GetIncident: requester not in team", "team_id", inc.TeamID, "requester_id", requesterID, "error", err)
		return nil, customErrors.ErrUnauthorizedTeamOp
	}

	return inc, nil
}

func (s *teamService) UpdateIncidentStatus(ctx context.Context, requesterID, incidentID uuid.UUID, newStatus, title, details string) (*models.TeamIncident, error) {
	slog.InfoContext(ctx, "[team-svc] UpdateIncidentStatus: updating incident status", "incident_id", incidentID, "requester_id", requesterID, "new_status", newStatus)

	validStatus, err := normalizeIncidentStatus(newStatus)
	if err != nil {
		return nil, err
	}

	inc, err := s.teamRepo.GetTeamIncidentByID(ctx, incidentID)
	if err != nil {
		slog.ErrorContext(ctx, "[team-svc] UpdateIncidentStatus: failed to get incident", "incident_id", incidentID, "error", err)
		return nil, err
	}

	_, err = s.checkMemberRoleOrSuperAdmin(ctx, inc.TeamID, requesterID)
	if err != nil {
		slog.WarnContext(ctx, "[team-svc] UpdateIncidentStatus: requester not in team", "team_id", inc.TeamID, "requester_id", requesterID, "error", err)
		return nil, customErrors.ErrUnauthorizedTeamOp
	}

	previousStatus := inc.Status
	inc.Status = validStatus
	if strings.TrimSpace(title) != "" {
		inc.Title = strings.TrimSpace(title)
	}
	if details != "" {
		inc.Details = details
	}
	// only capture resolved_at on the first transition to RESOLVED
	if validStatus == "RESOLVED" && inc.ResolvedAt == nil {
		now := time.Now()
		inc.ResolvedAt = &now
	}

	history := &models.IncidentStatusHistory{
		ID:             uuid.New(),
		TeamIncidentID: inc.ID,
		UpdatedBy:      requesterID,
		Title:          inc.Title,
		NewStatus:      validStatus,
		PreviousStatus: previousStatus,
		Details:        details,
		UpdatedAt:      time.Now(),
	}

	err = s.teamRepo.UpdateTeamIncidentStatus(ctx, history, inc)
	if err != nil {
		slog.ErrorContext(ctx, "[team-svc] UpdateIncidentStatus: failed to update", "incident_id", incidentID, "error", err)
		return nil, err
	}

	slog.InfoContext(ctx, "[team-svc] UpdateIncidentStatus: success", "incident_id", incidentID, "new_status", validStatus)
	return s.teamRepo.GetTeamIncidentByID(ctx, incidentID)
}

func (s *teamService) GetTeamInstruction(ctx context.Context, requesterID, teamID uuid.UUID) (*models.Instruction, []models.InstructionLog, error) {
	slog.InfoContext(ctx, "[team-svc] GetTeamInstruction: checking membership", "team_id", teamID, "requester_id", requesterID)
	_, err := s.checkMemberRoleOrSuperAdmin(ctx, teamID, requesterID)
	if err != nil {
		slog.WarnContext(ctx, "[team-svc] GetTeamInstruction: requester not in team", "team_id", teamID, "requester_id", requesterID, "error", err)
		return nil, nil, customErrors.ErrUnauthorizedTeamOp
	}

	inst, logs, err := s.teamRepo.GetTeamInstruction(ctx, teamID)
	if err != nil {
		slog.ErrorContext(ctx, "[team-svc] GetTeamInstruction: failed", "team_id", teamID, "error", err)
		return nil, nil, err
	}
	return inst, logs, nil
}

func (s *teamService) SaveTeamInstruction(ctx context.Context, requesterID, teamID uuid.UUID, details string) (*models.Instruction, error) {
	slog.InfoContext(ctx, "[team-svc] SaveTeamInstruction: checking membership", "team_id", teamID, "requester_id", requesterID)
	_, err := s.checkMemberRoleOrSuperAdmin(ctx, teamID, requesterID)
	if err != nil {
		slog.WarnContext(ctx, "[team-svc] SaveTeamInstruction: requester not in team", "team_id", teamID, "requester_id", requesterID, "error", err)
		return nil, customErrors.ErrUnauthorizedTeamOp
	}

	// validate instruction details minimum length requirement of 30 characters
	trimmed := strings.TrimSpace(details)
	if len(trimmed) < 30 {
		slog.WarnContext(ctx, "[team-svc] SaveTeamInstruction: instruction details too short", "team_id", teamID, "length", len(trimmed))
		return nil, customErrors.ErrInstructionTooShort
	}

	existingInst, existingLogs, err := s.teamRepo.GetTeamInstruction(ctx, teamID)
	if err != nil {
		slog.ErrorContext(ctx, "[team-svc] SaveTeamInstruction: failed to check existing", "team_id", teamID, "error", err)
		return nil, err
	}

	newInst := &models.Instruction{
		ID:                 uuid.New(),
		CreatedBy:          requesterID,
		TeamID:             teamID,
		InstructionDetails: strings.TrimSpace(details),
		CreatedAt:          time.Now(),
	}

	var logEntry *models.InstructionLog
	if existingInst != nil {
		nextVersion := len(existingLogs) + 1
		prevTime := existingInst.CreatedAt
		if prevTime.IsZero() && len(existingLogs) > 0 {
			prevTime = existingLogs[0].UpdatedAt
		}
		if prevTime.IsZero() {
			prevTime = time.Now()
		}
		logEntry = &models.InstructionLog{
			ID:               uuid.New(),
			InstructionID:    existingInst.ID,
			UpdatedBy:        requesterID,
			OlderInstruction: existingInst.InstructionDetails,
			Version:          nextVersion,
			UpdatedAt:        prevTime,
		}
	}

	err = s.teamRepo.SaveTeamInstruction(ctx, newInst, logEntry)
	if err != nil {
		slog.ErrorContext(ctx, "[team-svc] SaveTeamInstruction: failed to save", "team_id", teamID, "error", err)
		return nil, err
	}

	updatedInst, _, err := s.teamRepo.GetTeamInstruction(ctx, teamID)
	return updatedInst, err
}

func (s *teamService) CreateRunbook(ctx context.Context, creatorID, teamID uuid.UUID, incidentIDOrNumber, title, content string) (*models.Runbook, error) {
	slog.InfoContext(ctx, "[team-svc] CreateRunbook: persisting", "team_id", teamID, "incident_id_or_number", incidentIDOrNumber, "creator_id", creatorID)

	var incidentIDPtr *uuid.UUID

	cleanIncID := strings.TrimSpace(incidentIDOrNumber)
	if cleanIncID != "" && cleanIncID != "null" && cleanIncID != "none" && cleanIncID != "undefined" && cleanIncID != "00000000-0000-0000-0000-000000000000" {
		inc, err := s.teamRepo.GetTeamIncidentByIDOrNumber(ctx, cleanIncID)
		if err != nil || inc == nil {
			slog.WarnContext(ctx, "[team-svc] CreateRunbook: incident not found", "incident_id_or_number", cleanIncID, "error", err)
			return nil, customErrors.ErrIncidentNotFound
		}
		incidentIDPtr = &inc.ID
		if _, teamErr := s.teamRepo.GetTeamByID(ctx, teamID); teamErr != nil || teamID == uuid.Nil {
			teamID = inc.TeamID
		}
		if creatorID == uuid.Nil {
			creatorID = inc.CreatedBy
		}
	} else {
		// auto-link to latest team incident if any exists
		if incidents, err := s.teamRepo.ListTeamIncidents(ctx, teamID); err == nil && len(incidents) > 0 {
			incidentIDPtr = &incidents[0].ID
		}
	}

	// fallback creatorID to system boss if creatorID is still nil
	if creatorID == uuid.Nil {
		creatorID = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	}

	// verify team exists in database
	if _, err := s.teamRepo.GetTeamByID(ctx, teamID); err != nil {
		slog.WarnContext(ctx, "[team-svc] CreateRunbook: team not found", "team_id", teamID, "error", err)
		return nil, customErrors.ErrTeamNotFound
	}

	rb := &models.Runbook{
		ID:         uuid.New(),
		TeamID:     teamID,
		IncidentID: incidentIDPtr,
		CreatedBy:  creatorID,
		Title:      strings.TrimSpace(title),
		Status:     "active",
		Content:    content,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if err := s.teamRepo.CreateRunbook(ctx, rb); err != nil {
		slog.ErrorContext(ctx, "[team-svc] CreateRunbook: failed", "error", err)
		return nil, err
	}
	slog.InfoContext(ctx, "[team-svc] CreateRunbook: success", "runbook_id", rb.ID)
	return rb, nil
}

func (s *teamService) UpdateRunbook(ctx context.Context, updaterID, runbookID uuid.UUID, title, content string) (*models.Runbook, error) {
	slog.InfoContext(ctx, "[team-svc] UpdateRunbook: updating", "runbook_id", runbookID, "updater_id", updaterID)
	existingLogs, _ := s.teamRepo.GetRunbookLogs(ctx, runbookID)
	existingRb, err := s.teamRepo.GetRunbookByID(ctx, runbookID)
	if err != nil || existingRb == nil {
		slog.WarnContext(ctx, "[team-svc] UpdateRunbook: runbook not found", "runbook_id", runbookID, "error", err)
		return nil, customErrors.ErrRunbookNotFound
	}
	if existingRb.Status == "deprecated" {
		slog.WarnContext(ctx, "[team-svc] UpdateRunbook: runbook is deprecated", "runbook_id", runbookID)
		return nil, customErrors.ErrRunbookDeprecated
	}

	if updaterID == uuid.Nil {
		if existingRb.CreatedBy != uuid.Nil {
			updaterID = existingRb.CreatedBy
		} else {
			updaterID = uuid.MustParse("11111111-1111-1111-1111-111111111111")
		}
	}

	var logEntry *models.RunbookLog
	if existingRb != nil {
		prevTime := existingRb.UpdatedAt
		if prevTime.IsZero() || len(existingLogs) == 0 {
			prevTime = existingRb.CreatedAt
		}
		if prevTime.IsZero() {
			prevTime = time.Now()
		}
		logEntry = &models.RunbookLog{
			ID:           uuid.New(),
			RunbookID:    runbookID,
			IncidentID:   existingRb.IncidentID,
			TeamID:       existingRb.TeamID,
			UpdatedBy:    updaterID,
			OlderTitle:   existingRb.Title,
			OlderContent: existingRb.Content,
			Version:      len(existingLogs) + 1,
			UpdatedAt:    prevTime,
		}
	}


	rb, err := s.teamRepo.UpdateRunbook(ctx, runbookID, strings.TrimSpace(title), content, logEntry)
	if err != nil {
		slog.ErrorContext(ctx, "[team-svc] UpdateRunbook: failed", "runbook_id", runbookID, "error", err)
		return nil, err
	}
	slog.InfoContext(ctx, "[team-svc] UpdateRunbook: success", "runbook_id", runbookID)
	return rb, nil
}

func (s *teamService) DeprecateRunbook(ctx context.Context, runbookID uuid.UUID) (*models.Runbook, error) {
	slog.InfoContext(ctx, "[team-svc] DeprecateRunbook: deprecating", "runbook_id", runbookID)
	rb, err := s.teamRepo.DeprecateRunbook(ctx, runbookID)
	if err != nil {
		slog.ErrorContext(ctx, "[team-svc] DeprecateRunbook: failed", "runbook_id", runbookID, "error", err)
		return nil, err
	}
	slog.InfoContext(ctx, "[team-svc] DeprecateRunbook: success", "runbook_id", runbookID)
	return rb, nil
}

func (s *teamService) GetRunbook(ctx context.Context, runbookID uuid.UUID) (*models.Runbook, error) {
	slog.InfoContext(ctx, "[team-svc] GetRunbook: fetching", "runbook_id", runbookID)
	return s.teamRepo.GetRunbookByID(ctx, runbookID)
}

func (s *teamService) ListRunbooks(ctx context.Context, teamID uuid.UUID, status string) ([]models.Runbook, error) {
	if status == "" {
		status = "active"
	}
	slog.InfoContext(ctx, "[team-svc] ListRunbooks: listing", "team_id", teamID, "status", status)
	return s.teamRepo.ListRunbooks(ctx, teamID, status)
}

func (s *teamService) GetRunbookLogs(ctx context.Context, runbookID uuid.UUID) ([]models.RunbookLog, error) {
	slog.InfoContext(ctx, "[team-svc] GetRunbookLogs: fetching logs", "runbook_id", runbookID)
	return s.teamRepo.GetRunbookLogs(ctx, runbookID)
}

func (s *teamService) GetIncidentContext(ctx context.Context, teamIncidentID uuid.UUID) (*models.TeamIncident, []models.Alert, error) {
	slog.InfoContext(ctx, "[team-svc] GetIncidentContext: fetching enriched context", "team_incident_id", teamIncidentID)
	inc, alerts, err := s.teamRepo.GetIncidentContext(ctx, teamIncidentID)
	if err != nil {
		slog.ErrorContext(ctx, "[team-svc] GetIncidentContext: failed", "team_incident_id", teamIncidentID, "error", err)
		return nil, nil, err
	}
	slog.InfoContext(ctx, "[team-svc] GetIncidentContext: success", "team_incident_id", teamIncidentID, "alert_count", len(alerts))
	return inc, alerts, nil
}

func (s *teamService) LinkAlertsToIncident(ctx context.Context, alertIDStrings []string, incidentID uuid.UUID) error {
	if s.alertRepo == nil {
		slog.WarnContext(ctx, "[team-svc] LinkAlertsToIncident: alertRepo is nil")
		return nil
	}
	slog.InfoContext(ctx, "[team-svc] LinkAlertsToIncident: linking alerts", "count", len(alertIDStrings), "incident_id", incidentID)
	for _, rawID := range alertIDStrings {
		rawID = strings.TrimSpace(rawID)
		if rawID == "" {
			continue
		}
		alertRecord, err := s.alertRepo.RetrieveAlertbyID(ctx, rawID)
		if err != nil || alertRecord == nil {
			slog.WarnContext(ctx, "[team-svc] LinkAlertsToIncident: failed to resolve alert ID", "raw_id", rawID, "err", err)
			continue
		}
		if err := s.alertRepo.UpdateAlertIncidentID(ctx, alertRecord.ID, incidentID); err != nil {
			slog.ErrorContext(ctx, "[team-svc] LinkAlertsToIncident: failed to update alert incident_id", "alert_db_id", alertRecord.ID, "incident_id", incidentID, "err", err)
		} else {
			slog.InfoContext(ctx, "[team-svc] LinkAlertsToIncident: successfully linked alert", "raw_id", rawID, "alert_db_id", alertRecord.ID, "incident_id", incidentID)
		}
	}
	return nil
}

func (s *teamService) GetIncidentContextByIDOrNumber(ctx context.Context, idOrNumber string) (*models.TeamIncident, []models.Alert, error) {
	slog.InfoContext(ctx, "[team-svc] GetIncidentContextByIDOrNumber: fetching context", "id_or_number", idOrNumber)
	inc, alerts, err := s.teamRepo.GetIncidentContextByIDOrNumber(ctx, idOrNumber)
	if err != nil {
		slog.ErrorContext(ctx, "[team-svc] GetIncidentContextByIDOrNumber: failed", "id_or_number", idOrNumber, "error", err)
		return nil, nil, err
	}
	slog.InfoContext(ctx, "[team-svc] GetIncidentContextByIDOrNumber: success", "id_or_number", idOrNumber, "alert_count", len(alerts))
	return inc, alerts, nil
}

func (s *teamService) GetMemberRole(ctx context.Context, teamID, userID uuid.UUID) (string, error) {
	return s.teamRepo.GetMemberRole(ctx, teamID, userID)
}
