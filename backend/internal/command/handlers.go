package command

import (
	"context"
	"encoding/json"
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
			return &types.CommandResult{Handled: true, Message: formatIncidentDetail(details)}, nil
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
			return &types.CommandResult{Handled: true, Message: formatIncidentDetail(details)}, nil
		}
	} else if _, err := uuid.Parse(arg); err == nil {
		rawArgs := fmt.Sprintf(`{"incident_id": "%s"}`, arg)
		details, err := h.orchestrator.ExecuteGetIncidentRaw(ctx, rawArgs)
		if err == nil {
			return &types.CommandResult{Handled: true, Message: formatIncidentDetail(details)}, nil
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
func (h *AlertCommandHandler) Description() string { return "List recent alerts. Usage: /alert [service|severity]" }

func (h *AlertCommandHandler) Handle(ctx context.Context, prompt string) (*types.CommandResult, error) {
	if h.orchestrator == nil {
		return &types.CommandResult{
			Handled: true,
			Message: "Orchestrator service is unavailable.",
		}, nil
	}

	arg := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(prompt), "/alert"))

	alertJSON, err := h.orchestrator.ExecuteListAlertsRaw(ctx)
	if err != nil {
		return &types.CommandResult{
			Handled: true,
			Message: fmt.Sprintf("Failed to list alerts: %v", err),
		}, nil
	}

	return &types.CommandResult{
		Handled: true,
		Message: formatAlertList(alertJSON, arg),
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

// formatIncidentDetail formats a single incident context JSON (from get_incident / GetIncidentContext)
// into a human-readable Markdown summary.
func formatIncidentDetail(jsonStr string) string {
	var ctx struct {
		IncidentID string `json:"incident_id"`
		Title      string `json:"title"`
		Status     string `json:"status"`
		Age        string `json:"age"`
		Details    string `json:"details"`
		Alerts     []struct {
			Service  string `json:"service"`
			Severity string `json:"severity"`
			Received string `json:"received"`
		} `json:"alerts"`
		Timeline []struct {
			At   string `json:"at"`
			From string `json:"from"`
			To   string `json:"to"`
			Note string `json:"note"`
		} `json:"timeline"`
		ExistingRunbooks []struct {
			ID     string `json:"id"`
			Title  string `json:"title"`
			Status string `json:"status"`
		} `json:"existing_runbooks"`
	}
	if err := unmarshalJSON(jsonStr, &ctx); err != nil {
		return jsonStr // safe fallback
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("### 🚨 Incident: %s\n\n", ctx.Title))
	sb.WriteString(fmt.Sprintf("| Field | Value |\n|---|---|\n"))
	sb.WriteString(fmt.Sprintf("| **Status** | `%s` |\n", ctx.Status))
	sb.WriteString(fmt.Sprintf("| **Age** | %s |\n", ctx.Age))
	if ctx.Details != "" {
		sb.WriteString(fmt.Sprintf("| **Details** | %s |\n", ctx.Details))
	}

	if len(ctx.Alerts) > 0 {
		sb.WriteString("\n**Linked Alerts:**\n")
		for _, a := range ctx.Alerts {
			sb.WriteString(fmt.Sprintf("- `%s` [%s] — received %s\n", a.Service, a.Severity, a.Received))
		}
	} else {
		sb.WriteString("\n**Linked Alerts:** none\n")
	}

	if len(ctx.Timeline) > 0 {
		sb.WriteString("\n**Timeline:**\n")
		for _, t := range ctx.Timeline {
			if t.At == "" {
				sb.WriteString(fmt.Sprintf("- _%s_\n", t.Note))
			} else {
				sb.WriteString(fmt.Sprintf("- `%s` → `%s` (%s)", t.From, t.To, t.At))
				if t.Note != "" {
					sb.WriteString(fmt.Sprintf(" — %s", t.Note))
				}
				sb.WriteString("\n")
			}
		}
	}

	if len(ctx.ExistingRunbooks) > 0 {
		sb.WriteString("\n**Runbooks:**\n")
		for _, rb := range ctx.ExistingRunbooks {
			sb.WriteString(fmt.Sprintf("- **%s** (`%s`) — status: %s\n", rb.Title, rb.ID, rb.Status))
		}
	} else {
		sb.WriteString("\n**Runbooks:** none yet\n")
	}

	return sb.String()
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

// formatAlertList formats the alert list into a readable grouped Markdown summary.
// Shows up to 10 alerts by default, grouped by service; supports keyword filter.
func formatAlertList(jsonStr string, filter string) string {
	alerts, err := data.UnmarshalAlerts(jsonStr)
	if err != nil {
		return jsonStr
	}

	if len(alerts) == 0 {
		return "no alerts found"
	}

	// apply keyword filter if provided
	filterLower := strings.ToLower(strings.TrimSpace(filter))
	if filterLower != "" {
		var filtered []types.AlertRecord
		for _, a := range alerts {
			if strings.Contains(strings.ToLower(a.ServiceName), filterLower) ||
				strings.Contains(strings.ToLower(a.Severity), filterLower) {
				filtered = append(filtered, a)
			}
		}
		if len(filtered) == 0 {
			return fmt.Sprintf("no alerts matched filter: `%s`", filter)
		}
		alerts = filtered
	}

	total := len(alerts)
	const pageSize = 10
	if total > pageSize {
		alerts = alerts[:pageSize]
	}

	// group by service
	grouped := make(map[string][]types.AlertRecord)
	order := []string{}
	for _, a := range alerts {
		svc := a.ServiceName
		if svc == "" {
			svc = "unknown-service"
		}
		if _, exists := grouped[svc]; !exists {
			order = append(order, svc)
		}
		grouped[svc] = append(grouped[svc], a)
	}

	var sb strings.Builder
	shown := len(alerts)
	if filter != "" {
		sb.WriteString(fmt.Sprintf("**%d** alert(s) matching `%s`", total, filter))
	} else {
		sb.WriteString(fmt.Sprintf("**%d** recent alert(s) total", total))
	}
	if total > pageSize {
		sb.WriteString(fmt.Sprintf(" — showing first %d. Use `/alert <service>` to filter.\n\n", shown))
	} else {
		sb.WriteString("\n\n")
	}

	for _, svc := range order {
		group := grouped[svc]
		sb.WriteString(fmt.Sprintf("**%s** (%d alert(s))\n", svc, len(group)))
		for _, a := range group {
			link := "🔗 unlinked"
			if a.IncidentID != "" {
				link = fmt.Sprintf("🔗 linked to incident")
			}
			sb.WriteString(fmt.Sprintf("  - [%s] %s — received %s\n", a.Severity, link, a.ReceivedAt))
		}
		sb.WriteString("\n")
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

// unmarshalJSON is a thin helper to avoid importing encoding/json directly in helpers
func unmarshalJSON(s string, v any) error {
	return json.Unmarshal([]byte(s), v)
}
