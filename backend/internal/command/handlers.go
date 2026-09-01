package command

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/WillieBam/support_copilot/backend/internal/domain/data"
	"github.com/WillieBam/support_copilot/backend/internal/interfaces"
	"github.com/WillieBam/support_copilot/backend/types"
	"github.com/google/uuid"
)

// FuncCommandHandler adapts a function into an ICommandHandler
type FuncCommandHandler struct {
	cmd         string
	description string
	handler     CommandHandler
}

func NewFuncCommandHandler(cmd, description string, handler CommandHandler) interfaces.ICommandHandler {
	return &FuncCommandHandler{
		cmd:         cmd,
		description: description,
		handler:     handler,
	}
}

func (h *FuncCommandHandler) Command() string     { return h.cmd }
func (h *FuncCommandHandler) Description() string { return h.description }
func (h *FuncCommandHandler) Handle(ctx context.Context, prompt string) (*types.CommandResult, error) {
	return h.handler(ctx, prompt)
}

// QuitCommandHandler handles /quit
type QuitCommandHandler struct{}

func NewQuitCommandHandler() interfaces.ICommandHandler {
	return &QuitCommandHandler{}
}

func (h *QuitCommandHandler) Command() string     { return "/quit" }
func (h *QuitCommandHandler) Description() string { return "Stops LLM processing immediately." }
func (h *QuitCommandHandler) Handle(ctx context.Context, prompt string) (*types.CommandResult, error) {
	return &types.CommandResult{
		Handled: true,
		Message: "LLM processing stopped by /quit command.",
	}, nil
}

// IncidentCommandHandler handles /incident
type IncidentCommandHandler struct {
	orchestrator interfaces.IOrchestratorService
}

func NewIncidentCommandHandler(orchestrator interfaces.IOrchestratorService) interfaces.ICommandHandler {
	return &IncidentCommandHandler{orchestrator: orchestrator}
}

func (h *IncidentCommandHandler) Command() string { return "/incident" }
func (h *IncidentCommandHandler) Description() string {
	return "View active incident details or search team incidents."
}

func (h *IncidentCommandHandler) Handle(ctx context.Context, prompt string) (*types.CommandResult, error) {
	arg := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(prompt), "/incident"))

	if h.orchestrator == nil {
		return &types.CommandResult{
			Handled: true,
			Message: "orchestrator service is unavailable",
		}, nil
	}

	if arg == "" {
		incidentID, ok := GetActiveIncidentID(ctx)
		if ok && incidentID != uuid.Nil {
			rawArgs := fmt.Sprintf(`{"incident_id": "%s"}`, incidentID.String())
			details, err := h.orchestrator.ExecuteGetIncidentRaw(ctx, rawArgs)
			if err != nil {
				return &types.CommandResult{Handled: true, Message: fmt.Sprintf("failed to fetch incident: %v", err)}, nil
			}
			return &types.CommandResult{Handled: true, Message: details}, nil
		}

		// Fallback: list all incidents for active team context
		teamID, hasTeam := GetTeamID(ctx)
		if hasTeam && teamID != uuid.Nil {
			rawArgs := fmt.Sprintf(`{"team_id": "%s"}`, teamID.String())
			incidentsJSON, err := h.orchestrator.ExecuteListIncidentsRaw(ctx, rawArgs)
			if err != nil {
				return &types.CommandResult{Handled: true, Message: fmt.Sprintf("failed to list incidents: %v", err)}, nil
			}
			return &types.CommandResult{Handled: true, Message: formatIncidentList(incidentsJSON)}, nil
		}

		return &types.CommandResult{
			Handled: true,
			Message: "no active incident found in session context, provide a search query like /incident redis latency",
		}, nil
	}

	// Direct lookup for surrogate key (e.g. INC-101) or UUID
	if strings.HasPrefix(strings.ToUpper(arg), "INC-") {
		rawArgs := fmt.Sprintf(`{"incident_id": "%s"}`, arg)
		details, err := h.orchestrator.ExecuteGetIncidentRaw(ctx, rawArgs)
		if err == nil {
			return &types.CommandResult{Handled: true, Message: details}, nil
		}
	} else if _, err := uuid.Parse(arg); err == nil {
		rawArgs := fmt.Sprintf(`{"incident_id": "%s"}`, arg)
		details, err := h.orchestrator.ExecuteGetIncidentRaw(ctx, rawArgs)
		if err == nil {
			return &types.CommandResult{Handled: true, Message: details}, nil
		}
	}

	// Argument provided: search team incidents
	teamID, ok := GetTeamID(ctx)
	if !ok {
		return &types.CommandResult{
			Handled: true,
			Message: "no active team context associated with session",
		}, nil
	}

	rawArgs := fmt.Sprintf(`{"team_id": "%s"}`, teamID.String())
	incidentsJSON, err := h.orchestrator.ExecuteListIncidentsRaw(ctx, rawArgs)
	if err != nil {
		return &types.CommandResult{Handled: true, Message: fmt.Sprintf("failed to list incidents: %v", err)}, nil
	}

	filteredResult := filterIncidentsByQuery(incidentsJSON, arg)
	return &types.CommandResult{Handled: true, Message: filteredResult}, nil
}

// RunbookCommandHandler handles /runbook
type RunbookCommandHandler struct {
	orchestrator interfaces.IOrchestratorService
}

func NewRunbookCommandHandler(orchestrator interfaces.IOrchestratorService) interfaces.ICommandHandler {
	return &RunbookCommandHandler{orchestrator: orchestrator}
}

func (h *RunbookCommandHandler) Command() string { return "/runbook" }
func (h *RunbookCommandHandler) Description() string {
	return "List active runbooks or search runbooks by keywords."
}

func (h *RunbookCommandHandler) Handle(ctx context.Context, prompt string) (*types.CommandResult, error) {
	arg := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(prompt), "/runbook"))

	if h.orchestrator == nil {
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
	runbooksJSON, err := h.orchestrator.ExecuteListRunbooksRaw(ctx, rawArgs)
	if err != nil {
		return &types.CommandResult{Handled: true, Message: fmt.Sprintf("Failed to list runbooks: %v", err)}, nil
	}

	// no argument provided: list all active runbooks
	if arg == "" {
		return &types.CommandResult{Handled: true, Message: formatRunbookList(runbooksJSON)}, nil
	}

	// argument provided: perform keyword search on title and content
	filtered := filterRunbooksByQuery(runbooksJSON, arg)
	return &types.CommandResult{Handled: true, Message: filtered}, nil
}

// AlertCommandHandler handles /alert
type AlertCommandHandler struct {
	orchestrator interfaces.IOrchestratorService
}

func NewAlertCommandHandler(orchestrator interfaces.IOrchestratorService) interfaces.ICommandHandler {
	return &AlertCommandHandler{orchestrator: orchestrator}
}

func (h *AlertCommandHandler) Command() string     { return "/alert" }
func (h *AlertCommandHandler) Description() string { return "List recent alerts." }

func (h *AlertCommandHandler) Handle(ctx context.Context, prompt string) (*types.CommandResult, error) {
	if h.orchestrator == nil {
		return &types.CommandResult{
			Handled: true,
			Message: "Orchestrator service is unavailable.",
		}, nil
	}

	alertJSON, err := h.orchestrator.ExecuteListAlertsRaw(ctx)
	if err != nil {
		return &types.CommandResult{
			Handled: true,
			Message: fmt.Sprintf("Failed to list alerts: %v", err),
		}, nil
	}

	return &types.CommandResult{
		Handled: true,
		Message: formatAlertList(alertJSON),
	}, nil
}

// HelpCommandHandler handles /help and enumerates all registered commands
type HelpCommandHandler struct {
	interceptor *CommandInterceptor
}

func NewHelpCommandHandler(interceptor *CommandInterceptor) interfaces.ICommandHandler {
	return &HelpCommandHandler{interceptor: interceptor}
}

func (h *HelpCommandHandler) Command() string     { return "/help" }
func (h *HelpCommandHandler) Description() string { return "List all available slash commands." }

func (h *HelpCommandHandler) Handle(ctx context.Context, prompt string) (*types.CommandResult, error) {
	if h.interceptor == nil {
		return &types.CommandResult{Handled: true, Message: "No command interceptor available."}, nil
	}

	handlers := h.interceptor.GetHandlers()
	commands := make([]string, 0, len(handlers))
	for cmd := range handlers {
		commands = append(commands, cmd)
	}
	sort.Strings(commands)

	var sb strings.Builder
	sb.WriteString("Available slash commands:\n\n")
	for _, cmd := range commands {
		handler := handlers[cmd]
		desc := handler.Description()
		if desc == "" {
			desc = "Custom command"
		}
		sb.WriteString(fmt.Sprintf("- **%s**: %s\n", handler.Command(), desc))
	}

	return &types.CommandResult{
		Handled: true,
		Message: sb.String(),
	}, nil
}

func formatIncidentList(jsonStr string) string {
	incidents, err := data.UnmarshalIncidents(jsonStr)
	if err != nil {
		return jsonStr
	}

	if len(incidents) == 0 {
		return "no incidents found for team"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("found %d incident(s) for your team:\n\n", len(incidents)))
	for _, inc := range incidents {
		key := inc.IncidentNumber
		if key == "" {
			key = inc.ID
		}
		sb.WriteString(fmt.Sprintf("- **%s** [%s] — `%s`\n", inc.Title, inc.Status, key))
	}
	return sb.String()
}

func formatRunbookList(jsonStr string) string {
	runbooks, err := data.UnmarshalRunbooks(jsonStr)
	if err != nil {
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

func formatAlertList(jsonStr string) string {
	alerts, err := data.UnmarshalAlerts(jsonStr)
	if err != nil {
		return jsonStr
	}

	if len(alerts) == 0 {
		return "no alerts found"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("found %d alert(s):\n\n", len(alerts)))
	for _, a := range alerts {
		link := "unlinked"
		if a.IncidentID != "" {
			link = fmt.Sprintf("linked to incident `%s`", a.IncidentID)
		}
		sb.WriteString(fmt.Sprintf("- **%s** [%s] %s — `%s`\n", a.ServiceName, a.Severity, link, a.ID))
	}
	return sb.String()
}

func filterIncidentsByQuery(jsonStr string, query string) string {
	incidents, err := data.UnmarshalIncidents(jsonStr)
	if err != nil {
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
		key := inc.IncidentNumber
		if key == "" {
			key = inc.ID
		}
		sb.WriteString(fmt.Sprintf("- **%s** [%s] — %s\n", inc.Title, inc.Status, key))
	}
	return sb.String()
}

func filterRunbooksByQuery(jsonStr string, query string) string {
	runbooks, err := data.UnmarshalRunbooks(jsonStr)
	if err != nil {
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
