package tools

import (
	"context"

	"github.com/WillieBam/support_copilot/backend/internal/interfaces"
	"github.com/WillieBam/support_copilot/backend/types/requests"
)

// ValidateAlertTool retrieves telemetry and predicts anomalies via ML
type ValidateAlertTool struct {
	orchestrator interfaces.IOrchestratorService
}

func NewValidateAlertTool(orchestrator interfaces.IOrchestratorService) interfaces.ITool {
	return &ValidateAlertTool{orchestrator: orchestrator}
}

func (t *ValidateAlertTool) Name() string { return "validate_alert" }

func (t *ValidateAlertTool) Definition() requests.LLMTool {
	return requests.LLMTool{
		Type: "function",
		Function: requests.LLMFunction{
			Name:        "validate_alert",
			Description: "Retrieves telemetry metrics for a given alert_id from Postgres and predicts whether the system state is Anomaly or Normal using IsolationForest ML. Call 'validate_alert' ONLY when an alert ID is provided or explicit alert validation is requested. Do NOT call this tool for general conversational input, greetings, or acknowledgments.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"alert_id": map[string]interface{}{
						"type":        "string",
						"description": "The alert identifier string (e.g. '165028917' or UUID). Only call this tool when the user has explicitly provided an alert ID. Do not guess, generate, or infer the alert_id.",
					},
				},
				"required":             []string{"alert_id"},
				"additionalProperties": false,
			},
		},
	}
}

func (t *ValidateAlertTool) Execute(ctx context.Context, rawArgs string) (string, error) {
	return t.orchestrator.ExecuteValidateAlertRaw(ctx, rawArgs)
}

// GetIncidentTool retrieves enriched incident context
type GetIncidentTool struct {
	orchestrator interfaces.IOrchestratorService
}

func NewGetIncidentTool(orchestrator interfaces.IOrchestratorService) interfaces.ITool {
	return &GetIncidentTool{orchestrator: orchestrator}
}

func (t *GetIncidentTool) Name() string { return "get_incident" }

func (t *GetIncidentTool) Definition() requests.LLMTool {
	return requests.LLMTool{
		Type: "function",
		Function: requests.LLMFunction{
			Name:        "get_incident",
			Description: "Retrieves enriched incident context — summary, affected services with key metrics, status timeline, and existing runbooks. Call this before creating a runbook.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"incident_id": map[string]interface{}{
						"type":        "string",
						"description": "An incident surrogate key (e.g. INC-101) or UUIDv4 identifier.",
					},
				},
				"required":             []string{"incident_id"},
				"additionalProperties": false,
			},
		},
	}
}

func (t *GetIncidentTool) Execute(ctx context.Context, rawArgs string) (string, error) {
	return t.orchestrator.ExecuteGetIncidentRaw(ctx, rawArgs)
}

// CreateRunbookTool creates a runbook in the Knowledge Base
type CreateRunbookTool struct {
	orchestrator interfaces.IOrchestratorService
}

func NewCreateRunbookTool(orchestrator interfaces.IOrchestratorService) interfaces.ITool {
	return &CreateRunbookTool{orchestrator: orchestrator}
}

func (t *CreateRunbookTool) Name() string { return "create_runbook" }

func (t *CreateRunbookTool) Definition() requests.LLMTool {
	return requests.LLMTool{
		Type: "function",
		Function: requests.LLMFunction{
			Name:        "create_runbook",
			Description: "Creates a runbook in the Knowledge Base for an incident. Content must follow: ## Root Cause, ## Diagnostic Steps, ## Resolution, ## Prevention.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"team_id": map[string]interface{}{
						"type":        "string",
						"description": "UUIDv4 of the team.",
					},
					"incident_id": map[string]interface{}{
						"type":        "string",
						"description": "UUIDv4 of the incident.",
					},
					"title": map[string]interface{}{
						"type":        "string",
						"description": "Short descriptive title for the runbook.",
					},
					"content": map[string]interface{}{
						"type":        "string",
						"description": "Full runbook markdown content.",
					},
				},
				"required":             []string{"team_id", "incident_id", "title", "content"},
				"additionalProperties": false,
			},
		},
	}
}

func (t *CreateRunbookTool) Execute(ctx context.Context, rawArgs string) (string, error) {
	return t.orchestrator.ExecuteCreateRunbookRaw(ctx, rawArgs)
}

// UpdateRunbookTool updates an existing runbook
type UpdateRunbookTool struct {
	orchestrator interfaces.IOrchestratorService
}

func NewUpdateRunbookTool(orchestrator interfaces.IOrchestratorService) interfaces.ITool {
	return &UpdateRunbookTool{orchestrator: orchestrator}
}

func (t *UpdateRunbookTool) Name() string { return "update_runbook" }

func (t *UpdateRunbookTool) Definition() requests.LLMTool {
	return requests.LLMTool{
		Type: "function",
		Function: requests.LLMFunction{
			Name:        "update_runbook",
			Description: "Updates the title and/or content of an existing active runbook in the Knowledge Base.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"runbook_id": map[string]interface{}{
						"type":        "string",
						"description": "UUIDv4 of the runbook to update.",
					},
					"title": map[string]interface{}{
						"type":        "string",
						"description": "Updated runbook title.",
					},
					"content": map[string]interface{}{
						"type":        "string",
						"description": "Updated runbook content in markdown.",
					},
				},
				"required":             []string{"runbook_id"},
				"additionalProperties": false,
			},
		},
	}
}

func (t *UpdateRunbookTool) Execute(ctx context.Context, rawArgs string) (string, error) {
	return t.orchestrator.ExecuteUpdateRunbookRaw(ctx, rawArgs)
}

// DeprecateRunbookTool deprecates a runbook
type DeprecateRunbookTool struct {
	orchestrator interfaces.IOrchestratorService
}

func NewDeprecateRunbookTool(orchestrator interfaces.IOrchestratorService) interfaces.ITool {
	return &DeprecateRunbookTool{orchestrator: orchestrator}
}

func (t *DeprecateRunbookTool) Name() string { return "deprecate_runbook" }

func (t *DeprecateRunbookTool) Definition() requests.LLMTool {
	return requests.LLMTool{
		Type: "function",
		Function: requests.LLMFunction{
			Name:        "deprecate_runbook",
			Description: "Marks a runbook as deprecated so it is retained for audit but excluded from active listings.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"runbook_id": map[string]interface{}{
						"type":        "string",
						"description": "UUIDv4 of the runbook to deprecate.",
					},
				},
				"required":             []string{"runbook_id"},
				"additionalProperties": false,
			},
		},
	}
}

func (t *DeprecateRunbookTool) Execute(ctx context.Context, rawArgs string) (string, error) {
	return t.orchestrator.ExecuteDeprecateRunbookRaw(ctx, rawArgs)
}

// GetRunbookTool retrieves a runbook by ID
type GetRunbookTool struct {
	orchestrator interfaces.IOrchestratorService
}

func NewGetRunbookTool(orchestrator interfaces.IOrchestratorService) interfaces.ITool {
	return &GetRunbookTool{orchestrator: orchestrator}
}

func (t *GetRunbookTool) Name() string { return "get_runbook" }

func (t *GetRunbookTool) Definition() requests.LLMTool {
	return requests.LLMTool{
		Type: "function",
		Function: requests.LLMFunction{
			Name:        "get_runbook",
			Description: "Retrieves a single runbook by ID including its full content.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"runbook_id": map[string]interface{}{
						"type":        "string",
						"description": "UUIDv4 of the runbook to retrieve.",
					},
				},
				"required":             []string{"runbook_id"},
				"additionalProperties": false,
			},
		},
	}
}

func (t *GetRunbookTool) Execute(ctx context.Context, rawArgs string) (string, error) {
	return t.orchestrator.ExecuteGetRunbookRaw(ctx, rawArgs)
}

// ListRunbooksTool lists runbooks for a team
type ListRunbooksTool struct {
	orchestrator interfaces.IOrchestratorService
}

func NewListRunbooksTool(orchestrator interfaces.IOrchestratorService) interfaces.ITool {
	return &ListRunbooksTool{orchestrator: orchestrator}
}

func (t *ListRunbooksTool) Name() string { return "list_runbooks" }

func (t *ListRunbooksTool) Definition() requests.LLMTool {
	return requests.LLMTool{
		Type: "function",
		Function: requests.LLMFunction{
			Name:        "list_runbooks",
			Description: "Lists runbooks for a team filtered by status ('active' or 'deprecated').",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"team_id": map[string]interface{}{
						"type":        "string",
						"description": "UUIDv4 of the team. This is automatically injected by the system if omitted; do NOT ask the user for team_id.",
					},
					"status": map[string]interface{}{
						"type":        "string",
						"description": "Status filter.",
						"enum":        []string{"active", "deprecated"},
					},
				},
				"required":             []string{"team_id"},
				"additionalProperties": false,
			},
		},
	}
}

func (t *ListRunbooksTool) Execute(ctx context.Context, rawArgs string) (string, error) {
	return t.orchestrator.ExecuteListRunbooksRaw(ctx, rawArgs)
}

// LinkAlertToIncidentTool associates an alert with an incident
type LinkAlertToIncidentTool struct {
	orchestrator interfaces.IOrchestratorService
}

func NewLinkAlertToIncidentTool(orchestrator interfaces.IOrchestratorService) interfaces.ITool {
	return &LinkAlertToIncidentTool{orchestrator: orchestrator}
}

func (t *LinkAlertToIncidentTool) Name() string { return "link_alert_to_incident" }

func (t *LinkAlertToIncidentTool) Definition() requests.LLMTool {
	return requests.LLMTool{
		Type: "function",
		Function: requests.LLMFunction{
			Name:        "link_alert_to_incident",
			Description: "Associates an existing alert with an incident by surrogate key INC-xxx, UUID, or title.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"alert_id": map[string]interface{}{
						"type":        "string",
						"description": "UUIDv4 of the target alert.",
					},
					"incident_id": map[string]interface{}{
						"type":        "string",
						"description": "Surrogate key INC-xxx (e.g. INC-101) or UUIDv4 of the target incident (if known).",
					},
					"incident_title": map[string]interface{}{
						"type":        "string",
						"description": "Human-readable title or name of the target incident (e.g. 'report-download-service CPU Spike').",
					},
				},
				"required":             []string{"alert_id"},
				"additionalProperties": false,
			},
		},
	}
}

func (t *LinkAlertToIncidentTool) Execute(ctx context.Context, rawArgs string) (string, error) {
	return t.orchestrator.ExecuteLinkAlertToIncidentRaw(ctx, rawArgs)
}
