package command

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/WillieBam/support_copilot/backend/internal/interfaces"
	"github.com/WillieBam/support_copilot/backend/types"
)

// CommandHandler is the function signature for slash command handlers.
type CommandHandler func(ctx context.Context, prompt string) (*types.CommandResult, error)

type CommandInterceptor struct {
	handlers     map[string]CommandHandler
	orchestrator interfaces.IOrchestratorService
}



func NewCommandInterceptor(orchestrator ...interfaces.IOrchestratorService) interfaces.ICommandInterceptor {
	ci := &CommandInterceptor{
		handlers: make(map[string]CommandHandler),
	}
	if len(orchestrator) > 0 {
		ci.orchestrator = orchestrator[0]
	}
	ci.RegisterCommand("/quit", handleQuitCommand) // register command here, add on when needed
	ci.RegisterCommand("/incident", ci.handleIncidentCommand)
	ci.RegisterCommand("/runbook", ci.handleRunbookCommand)
	return ci
}

func (ci *CommandInterceptor) RegisterCommand(command string, handler CommandHandler) {
	ci.handlers[strings.ToLower(command)] = handler
}

// Intercept is the function to check prompt against all registered command
func (ci *CommandInterceptor) Intercept(ctx context.Context, prompt string) (*types.CommandResult, error) {
	trimmed := strings.ToLower(strings.TrimSpace(prompt))
	for cmd, handler := range ci.handlers {
		if trimmed == cmd || strings.HasPrefix(trimmed, cmd+" ") || strings.HasPrefix(trimmed, cmd) {
			return handler(ctx, prompt)
		}
	}
	return &types.CommandResult{Handled: false}, nil
}

// function to bind with the RegisterCommand
func handleQuitCommand(ctx context.Context, prompt string) (*types.CommandResult, error) {
	return &types.CommandResult{
		Handled: true,
		Message: "LLM processing stopped by /quit command.",
	}, nil
}

func (ci *CommandInterceptor) handleIncidentCommand(ctx context.Context, prompt string) (*types.CommandResult, error) {
	arg := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(prompt),
		"/incident"))

	if ci.orchestrator == nil {
		return &types.CommandResult{
			Handled: true,
			Message: "orchestrator service is unavailable",
		}, nil
	}

	// case 1 no argument provided retrieve active session incident
	if arg == "" {
		incidentID, ok := GetActiveIncidentID(ctx)
		if !ok {
			return &types.CommandResult{
				Handled: true,
				Message: "no active incident found in session context, provide a search query like /incident redis latency"}, nil
		}

		rawArgs := fmt.Sprintf(`{"incident_id": "%s"}`, incidentID.String())
		details, err := ci.orchestrator.ExecuteGetIncidentRaw(ctx, rawArgs)
		if err != nil {
			return &types.CommandResult{Handled: true, Message: fmt.Sprintf("failed to fetch incident: %v", err)}, nil
		}
		return &types.CommandResult{Handled: true, Message: details}, nil
	}

	// argument provided search team incidents
	teamID, ok := GetTeamID(ctx)
	if !ok {
		return &types.CommandResult{
			Handled: true,
			Message: "no active team context associated with session",
		}, nil
	}

	rawArgs := fmt.Sprintf(`{"team_id": "%s"}`, teamID.String())
	incidentsJSON, err := ci.orchestrator.ExecuteListIncidentsRaw(ctx, rawArgs)
	if err != nil {
		return &types.CommandResult{Handled: true, Message: fmt.Sprintf("failed to list incidents: %v", err)}, nil
	}

	filteredResult := filterIncidentsByQuery(incidentsJSON, arg)
	return &types.CommandResult{Handled: true, Message: filteredResult}, nil
}

func (ci *CommandInterceptor) handleRunbookCommand(ctx context.Context, prompt string) (*types.CommandResult,
	error) {
	arg := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(prompt), "/runbook"))

	if ci.orchestrator == nil {
		return &types.CommandResult{
			Handled: true,
			Message: "Orchestrator service is unavailable.",
		}, nil
	}

	teamID, ok := GetTeamID(ctx)
	if !ok {
		return &types.CommandResult{
			Handled: true,
			Message: "No active team context associated with this session.",
		}, nil
	}

	rawArgs := fmt.Sprintf(`{"team_id": "%s", "status": "active"}`, teamID.String())
	runbooksJSON, err := ci.orchestrator.ExecuteListRunbooksRaw(ctx, rawArgs)
	if err != nil {
		return &types.CommandResult{Handled: true, Message: fmt.Sprintf("Failed to list runbooks: %v", err)}, nil
	}

	// no argument provided list all active runbooks
	if arg == "" {
		return &types.CommandResult{Handled: true, Message: formatRunbookList(runbooksJSON)}, nil
	}

	// argument provided perform keyword search on title and content
	filtered := filterRunbooksByQuery(runbooksJSON, arg)
	return &types.CommandResult{Handled: true, Message: filtered}, nil
}

func formatRunbookList(jsonStr string) string {
	var runbooks []types.RunbookRecord
	if err := json.Unmarshal([]byte(jsonStr), &runbooks); err != nil {
		return jsonStr
	}

	if len(runbooks) == 0 {
		return "no active runbooks found for team"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("found %d active runbook(s) for your team:\n\n", len(runbooks)))
	for _, rb := range runbooks {
		sb.WriteString(fmt.Sprintf("- **%s** (status: %s)\n", rb.Title, rb.Status))
	}
	return sb.String()
}

func filterIncidentsByQuery(jsonStr string, query string) string {
	var incidents []types.IncidentRecord
	if err := json.Unmarshal([]byte(jsonStr), &incidents); err != nil {
		return jsonStr // return raw fallback if parsing fails
	}

	queryTokens := strings.Fields(strings.ToLower(query))
	var matches []types.IncidentRecord

	for _, inc := range incidents {
		targetText := strings.ToLower(inc.Title + " " + inc.Summary + " " + inc.Status)
		matchCount := 0
		for _, token := range queryTokens {
			if strings.Contains(targetText, token) {
				matchCount++
			}
		}
		if matchCount > 0 {
			matches = append(matches, inc)
		}
	}

	if len(matches) == 0 {
		return fmt.Sprintf("no incidents matched query: '%s'", query)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("found %d matching incident(s) for '%s':\n\n", len(matches), query))
	for _, inc := range matches {
		sb.WriteString(fmt.Sprintf("- **%s** [%s] — %s\n", inc.Title, inc.Status, inc.ID))
	}
	return sb.String()
}

func filterRunbooksByQuery(jsonStr string, query string) string {
	var runbooks []types.RunbookRecord
	if err := json.Unmarshal([]byte(jsonStr), &runbooks); err != nil {
		return jsonStr
	}

	tokens := strings.Fields(strings.ToLower(query))
	var match []types.RunbookRecord
	for _, rb := range runbooks {
		searchSpace := strings.ToLower(rb.Title + " " + rb.Content)
		matched := true
		for _, token := range tokens {
			if !strings.Contains(searchSpace, token) {
				matched = false
				break
			}
		}
		if matched {
			match = append(match, rb)
		}
	}

	if len(match) == 0 {
		return fmt.Sprintf("no runbooks matched query: '%s'", query)
	}
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("found %d matching runbook(s) for '%s':\n\n", len(match), query))
	for _, rb := range match {
		sb.WriteString(fmt.Sprintf("- **%s** (status: %s)\n", rb.Title, rb.Status))
	}
	return sb.String()
}
