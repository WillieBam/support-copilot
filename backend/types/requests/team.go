package requests

import "github.com/google/uuid"

type CreateTeamRequest struct {
	TeamName string `json:"team_name" binding:"required"`
}

type AddTeamMemberRequest struct {
	UserID uuid.UUID `json:"user_id" binding:"required"`
}

type AssignTeamIncidentRequest struct {
	IncidentID uuid.UUID `json:"incident_id"`
	Title      string    `json:"title" binding:"required"`
	Status     string    `json:"status"`
	Details    string    `json:"details"`
	AlertID    string    `json:"alert_id,omitempty"`
	AlertIDs   []string  `json:"alert_ids,omitempty"`
}

type UpdateIncidentStatusRequest struct {
	Status  string `json:"status" binding:"required"`
	Title   string `json:"title"`
	Details string `json:"details"`
}

type SaveTeamInstructionRequest struct {
	InstructionDetails string `json:"instruction_details" binding:"required"`
}
