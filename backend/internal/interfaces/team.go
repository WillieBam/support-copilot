package interfaces

import (
	"context"

	"github.com/WillieBam/support_copilot/backend/types/models"
	"github.com/google/uuid"
)

type ITeamRepository interface {
	CreateTeamWithOwner(ctx context.Context, team *models.Team, ownerID uuid.UUID) error
	GetTeamByID(ctx context.Context, teamID uuid.UUID) (*models.Team, error)
	GetUserWithTeamsByID(ctx context.Context, userID uuid.UUID) (*models.User, error)
	AddTeamMember(ctx context.Context, member *models.TeamMember) error
	RemoveTeamMember(ctx context.Context, teamID, userID uuid.UUID) error
	DeleteTeam(ctx context.Context, teamID uuid.UUID) error
	GetMemberRole(ctx context.Context, teamID, userID uuid.UUID) (string, error)
	ListTeamMembers(ctx context.Context, teamID uuid.UUID) ([]models.TeamMember, error)
	AssignTeamIncident(ctx context.Context, incident *models.TeamIncident) error
	ListTeamIncidents(ctx context.Context, teamID uuid.UUID) ([]models.TeamIncident, error)
	GetTeamIncidentByID(ctx context.Context, incidentID uuid.UUID) (*models.TeamIncident, error)
	GetTeamIncidentByIDOrNumber(ctx context.Context, idOrNumber string) (*models.TeamIncident, error)
	UpdateTeamIncidentStatus(ctx context.Context, history *models.IncidentStatusHistory, updatedInc *models.TeamIncident) error
	GetTeamInstruction(ctx context.Context, teamID uuid.UUID) (*models.Instruction, []models.InstructionLog, error)
	SaveTeamInstruction(ctx context.Context, instruction *models.Instruction, log *models.InstructionLog) error
	// Runbook operations
	CreateRunbook(ctx context.Context, runbook *models.Runbook) error
	UpdateRunbook(ctx context.Context, runbookID uuid.UUID, title, content string, log *models.RunbookLog) (*models.Runbook, error)
	DeprecateRunbook(ctx context.Context, runbookID uuid.UUID) (*models.Runbook, error)
	GetRunbookByID(ctx context.Context, runbookID uuid.UUID) (*models.Runbook, error)
	ListRunbooks(ctx context.Context, teamID uuid.UUID, status string) ([]models.Runbook, error)
	GetRunbooksByIncidentID(ctx context.Context, incidentID uuid.UUID) ([]models.Runbook, error)
	GetRunbookLogs(ctx context.Context, runbookID uuid.UUID) ([]models.RunbookLog, error)
	// Enriched incident context for MCP KB tools
	GetIncidentContext(ctx context.Context, teamIncidentID uuid.UUID) (*models.TeamIncident, []models.Alert, error)
	GetIncidentContextByIDOrNumber(ctx context.Context, idOrNumber string) (*models.TeamIncident, []models.Alert, error)
	// ListAllTeams returns all teams in database
	ListAllTeams(ctx context.Context) ([]models.Team, error)
}

type ITeamService interface {
	CreateTeam(ctx context.Context, teamName string, creatorID uuid.UUID) (*models.Team, error)
	GetTeam(ctx context.Context, teamID uuid.UUID) (*models.Team, error)
	GetUserTeams(ctx context.Context, userID uuid.UUID) (*models.User, error)
	AddMember(ctx context.Context, requesterID, teamID, userID uuid.UUID) error
	RemoveMember(ctx context.Context, requesterID, teamID, userID uuid.UUID) error
	DeleteTeam(ctx context.Context, userScope string, teamID uuid.UUID) error
	AssignIncident(ctx context.Context, requesterID, teamID uuid.UUID, title, status, details string) (*models.TeamIncident, error)
	ListIncidents(ctx context.Context, requesterID, teamID uuid.UUID) ([]models.TeamIncident, error)
	ListMembers(ctx context.Context, requesterID, teamID uuid.UUID) ([]models.TeamMember, error)
	GetIncident(ctx context.Context, requesterID, incidentID uuid.UUID) (*models.TeamIncident, error)
	UpdateIncidentStatus(ctx context.Context, requesterID, incidentID uuid.UUID, newStatus, title, details string) (*models.TeamIncident, error)
	GetTeamInstruction(ctx context.Context, requesterID, teamID uuid.UUID) (*models.Instruction, []models.InstructionLog, error)
	SaveTeamInstruction(ctx context.Context, requesterID, teamID uuid.UUID, details string) (*models.Instruction, error)
	// Runbook operations (used by MCP Server 2 KB tools and REST API)
	CreateRunbook(ctx context.Context, creatorID, teamID uuid.UUID, incidentIDOrNumber, title, content string) (*models.Runbook, error)
	UpdateRunbook(ctx context.Context, updaterID, runbookID uuid.UUID, title, content string) (*models.Runbook, error)
	DeprecateRunbook(ctx context.Context, runbookID uuid.UUID) (*models.Runbook, error)
	GetRunbook(ctx context.Context, runbookID uuid.UUID) (*models.Runbook, error)
	ListRunbooks(ctx context.Context, teamID uuid.UUID, status string) ([]models.Runbook, error)
	GetRunbookLogs(ctx context.Context, runbookID uuid.UUID) ([]models.RunbookLog, error)
	ListTeamIncidents(ctx context.Context, teamID uuid.UUID) ([]models.TeamIncident, error)
	GetIncidentContext(ctx context.Context, teamIncidentID uuid.UUID) (*models.TeamIncident, []models.Alert, error)
	GetIncidentContextByIDOrNumber(ctx context.Context, idOrNumber string) (*models.TeamIncident, []models.Alert, error)
	LinkAlertsToIncident(ctx context.Context, alertIDStrings []string, incidentID uuid.UUID) error
	GetMemberRole(ctx context.Context, teamID, userID uuid.UUID) (string, error)
	// ListAllTeams returns all teams for super_admin
	ListAllTeams(ctx context.Context, userScope string) ([]models.Team, error)
}
