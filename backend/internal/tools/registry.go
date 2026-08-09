package tools

import (
	"context"
	"fmt"
	"sync"

	"github.com/WillieBam/support_copilot/backend/internal/interfaces"
	"github.com/WillieBam/support_copilot/backend/types/requests"
)

type ToolHandler func(ctx context.Context, rawArgs string) (string, error)

type ToolDefinition struct {
	Tool    requests.LLMTool
	Handler ToolHandler
}

type ToolRegistry struct {
	mu    sync.RWMutex
	tools map[string]ToolDefinition
}

func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]ToolDefinition),
	}
}

func (r *ToolRegistry) Register(name string, tool requests.LLMTool, handler func(ctx context.Context, rawArgs string) (string, error)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[name] = ToolDefinition{
		Tool:    tool,
		Handler: handler,
	}
}

func (r *ToolRegistry) GetTools() []requests.LLMTool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]requests.LLMTool, 0, len(r.tools))
	for _, def := range r.tools {
		result = append(result, def.Tool)
	}
	return result
}

func (r *ToolRegistry) Execute(ctx context.Context, name string, rawArgs string) (string, error) {
	r.mu.RLock()
	def, exists := r.tools[name]
	r.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("tool %q not registered in ToolRegistry", name)
	}
	return def.Handler(ctx, rawArgs)
}

// registerdefaulttools registers standard backend tools for anomaly validation and mcp2 knowledge base
func RegisterDefaultTools(registry interfaces.IToolRegistry, orchestrator interfaces.IOrchestratorService) {
	registry.Register("validate_alert", requests.LLMTool{
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
	}, func(ctx context.Context, rawArgs string) (string, error) {
		return orchestrator.ExecuteValidateAlertRaw(ctx, rawArgs)
	})

	registry.Register("get_incident", requests.LLMTool{
		Type: "function",
		Function: requests.LLMFunction{
			Name:        "get_incident",
			Description: "Retrieves enriched incident context — summary, affected services with key metrics, status timeline, and existing runbooks. Call this before creating a runbook.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"incident_id": map[string]interface{}{
						"type":        "string",
						"description": "A valid UUIDv4 incident identifier.",
					},
				},
				"required":             []string{"incident_id"},
				"additionalProperties": false,
			},
		},
	}, func(ctx context.Context, rawArgs string) (string, error) {
		return orchestrator.ExecuteGetIncidentRaw(ctx, rawArgs)
	})

	registry.Register("list_incidents", requests.LLMTool{
		Type: "function",
		Function: requests.LLMFunction{
			Name:        "list_incidents",
			Description: "Lists all team incidents with summary info including id, title, status, and age.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"team_id": map[string]interface{}{
						"type":        "string",
						"description": "A valid UUIDv4 team identifier.",
					},
				},
				"required":             []string{"team_id"},
				"additionalProperties": false,
			},
		},
	}, func(ctx context.Context, rawArgs string) (string, error) {
		return orchestrator.ExecuteListIncidentsRaw(ctx, rawArgs)
	})

	registry.Register("create_runbook", requests.LLMTool{
		Type: "function",
		Function: requests.LLMFunction{
			Name:        "create_runbook",
			Description: "Creates a runbook in the Knowledge Base for an incident. Use after get_incident. Content must follow: ## Root Cause, ## Diagnostic Steps, ## Resolution, ## Prevention.",
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
	}, func(ctx context.Context, rawArgs string) (string, error) {
		return orchestrator.ExecuteCreateRunbookRaw(ctx, rawArgs)
	})

	registry.Register("update_runbook", requests.LLMTool{
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
	}, func(ctx context.Context, rawArgs string) (string, error) {
		return orchestrator.ExecuteUpdateRunbookRaw(ctx, rawArgs)
	})

	registry.Register("deprecate_runbook", requests.LLMTool{
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
	}, func(ctx context.Context, rawArgs string) (string, error) {
		return orchestrator.ExecuteDeprecateRunbookRaw(ctx, rawArgs)
	})

	registry.Register("get_runbook", requests.LLMTool{
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
	}, func(ctx context.Context, rawArgs string) (string, error) {
		return orchestrator.ExecuteGetRunbookRaw(ctx, rawArgs)
	})

	registry.Register("list_runbooks", requests.LLMTool{
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
	}, func(ctx context.Context, rawArgs string) (string, error) {
		return orchestrator.ExecuteListRunbooksRaw(ctx, rawArgs)
	})

	registry.Register("link_alert_to_incident", requests.LLMTool{
		Type: "function",
		Function: requests.LLMFunction{
			Name:        "link_alert_to_incident",
			Description: "Associates an existing alert with an incident by UUID or title. If you only have the incident title or service name, call list_incidents first to get the exact incident_id, or pass the title in incident_title.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"alert_id": map[string]interface{}{
						"type":        "string",
						"description": "UUIDv4 of the target alert.",
					},
					"incident_id": map[string]interface{}{
						"type":        "string",
						"description": "UUIDv4 of the target incident (if known).",
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
	}, func(ctx context.Context, rawArgs string) (string, error) {
		return orchestrator.ExecuteLinkAlertToIncidentRaw(ctx, rawArgs)
	})
}
