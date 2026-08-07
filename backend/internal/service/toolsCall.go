package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/WillieBam/support_copilot/backend/internal/interfaces"
	"github.com/WillieBam/support_copilot/backend/types"
	"github.com/WillieBam/support_copilot/backend/types/models"
	"github.com/WillieBam/support_copilot/backend/types/requests"
	"github.com/WillieBam/support_copilot/backend/types/responses"
	"github.com/google/uuid"
)

type orchestratorService struct {
	alertRepo  interfaces.IAlertRepository
	mcpClient1 interfaces.IMCPClient
	mcpClient2 interfaces.IMCP2Client
	teamRepo   interfaces.ITeamRepository
}

func NewOrchestratorService(
	repo interfaces.IAlertRepository,
	mcpClient1 interfaces.IMCPClient,
	mcpClient2 interfaces.IMCP2Client,
	opts ...interface{},
) interfaces.IOrchestratorService {
	svc := &orchestratorService{
		alertRepo:  repo,
		mcpClient1: mcpClient1,
		mcpClient2: mcpClient2,
	}
	for _, opt := range opts {
		if tr, ok := opt.(interfaces.ITeamRepository); ok && tr != nil {
			svc.teamRepo = tr
		}
	}
	return svc
}

// ExecuteListAlertsRaw returns a list of alerts from the backend.
func (s *orchestratorService) ExecuteListAlertsRaw(ctx context.Context) (string, error) {
	slog.Info("[ORCHESTRATOR] ExecuteListAlertsRaw triggered")
	alerts, err := s.alertRepo.ListAlerts(ctx, 20)
	if err != nil {
		slog.Error("[ORCHESTRATOR] Failed to list alerts", "err", err)
		return "", fmt.Errorf("failed to list alerts: %w", err)
	}

	items := make([]types.AlertListItem, 0, len(alerts))
	for _, a := range alerts {
		item := types.AlertListItem{
			ID:          a.ID.String(),
			ServiceName: a.ServiceName,
			Severity:    a.Severity,
			ReceivedAt:  a.ReceivedAt.Format(time.RFC3339),
		}
		if a.IncidentID != nil {
			item.IncidentID = a.IncidentID.String()
		}
		items = append(items, item)
	}

	out, err := json.Marshal(items)
	if err != nil {
		slog.Error("[ORCHESTRATOR] Failed to marshal alerts", "err", err)
		return "", fmt.Errorf("failed to marshal alerts: %w", err)
	}
	return string(out), nil
}

// ExecuteValidateAlert fetches alert metrics from Postgres and predicts anomalies via Python MCP.
func (s *orchestratorService) ExecuteValidateAlert(ctx context.Context, alertID uuid.UUID) (*responses.CombinedValidationResult, error) {
	slog.Info("[ORCHESTRATOR] Fetching alert from database", "alert_id", alertID.String())

	// fetch alert from Postgres
	alertRecord, err := s.alertRepo.RetrieveAlertbyID(ctx, alertID)
	if err != nil {
		slog.Error("[ORCHESTRATOR] Failed to fetch alert from DB", "alert_id", alertID.String(), "err", err)
		return nil, fmt.Errorf("failed to fetch alert #%s from database: %w", alertID, err)
	}

	slog.Info("[ORCHESTRATOR] Alert retrieved from DB", "service", alertRecord.ServiceName, "severity", alertRecord.Severity)

	// unmarshal alert JSON metrics string into AnomalyDetectionRequest struct
	var metrics requests.AnomalyDetectionRequest
	if err := json.Unmarshal([]byte(alertRecord.Metrics), &metrics); err != nil {
		slog.Error("[ORCHESTRATOR] Failed to unmarshal alert metrics JSON", "err", err)
		return nil, fmt.Errorf("failed to parse alert metrics JSON: %w", err)
	}

	slog.Info("[ORCHESTRATOR] Invoking Python MCP Server detect_anomalies", "cpu", metrics.CpuUsage, "latency", metrics.ResponseLatency)

	// directly invoke python mcp server
	mcpResp, err := s.mcpClient1.DetectAnomalies(ctx, metrics)
	if err != nil {
		slog.Error("[ORCHESTRATOR] MCP anomaly detection failed", "err", err)
		return nil, fmt.Errorf("failed to analyze metrics via MCP server: %w", err)
	}

	slog.Info("[ORCHESTRATOR] MCP anomaly detection succeeded", "label", mcpResp.Label, "status", mcpResp.Status)

	// assemble combined payload package
	return &responses.CombinedValidationResult{
		AlertID:      alertRecord.ID.String(),
		ServiceName:  alertRecord.ServiceName,
		Severity:     alertRecord.Severity,
		ReceivedAt:   alertRecord.ReceivedAt,
		Metrics:      metrics,
		MLPrediction: *mcpResp,
	}, nil
}

// ExecuteValidateAlertRaw parses raw LLM JSON arguments and delegates execution.
func (s *orchestratorService) ExecuteValidateAlertRaw(ctx context.Context, rawArgs string) (string, error) {
	slog.Info("[ORCHESTRATOR] ExecuteValidateAlertRaw triggered", "rawArgs", rawArgs)

	var args struct {
		AlertID string `json:"alert_id"`
	}
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		slog.Error("[ORCHESTRATOR] Failed to parse raw tool arguments", "rawArgs", rawArgs, "err", err)
		return "", fmt.Errorf("invalid tool arguments: %w", err)
	}

	cleanAlertID := strings.TrimSpace(args.AlertID)
	if cleanAlertID == "" || cleanAlertID == "null" || cleanAlertID == "none" || cleanAlertID == "undefined" {
		slog.Warn("[ORCHESTRATOR] Empty or dummy alert_id provided", "alertID", args.AlertID)
		return "", fmt.Errorf("no valid alert_id provided: %q", args.AlertID)
	}

	alertUUID, err := uuid.Parse(cleanAlertID)
	if err != nil {
		slog.Error("[ORCHESTRATOR] Invalid alert UUID", "alertID", args.AlertID, "err", err)
		return "", fmt.Errorf("invalid alert id %q: %w", args.AlertID, err)
	}

	result, err := s.ExecuteValidateAlert(ctx, alertUUID)
	if err != nil {
		return "", err
	}

	resultBytes, err := json.Marshal(result)
	if err != nil {
		slog.Error("[ORCHESTRATOR] Failed to marshal validation result", "err", err)
		return "", fmt.Errorf("failed to marshal validation result: %w", err)
	}

	slog.Info("[ORCHESTRATOR] Combined validation package built successfully", "result", string(resultBytes))
	return string(resultBytes), nil
}

// Executegetincidentraw parses raw args and invokes mcp2 get_incident tool
func (s *orchestratorService) ExecuteGetIncidentRaw(ctx context.Context, rawArgs string) (string, error) {
	slog.Info("[ORCHESTRATOR] ExecuteGetIncidentRaw triggered", "rawArgs", rawArgs)
	if s.mcpClient2 == nil {
		return "", fmt.Errorf("mcp2 client is not configured")
	}

	var args requests.MCP2GetIncidentArgs
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		slog.Error("[ORCHESTRATOR] Failed to parse get_incident raw args", "err", err)
		return "", fmt.Errorf("invalid get_incident arguments: %w", err)
	}
	if strings.TrimSpace(args.IncidentID) == "" {
		return "", fmt.Errorf("incident_id is required")
	}
	return s.mcpClient2.GetIncident(ctx, args)
}

// Executelistincidentsraw parses raw args and invokes mcp2 list_incidents tool
func (s *orchestratorService) ExecuteListIncidentsRaw(ctx context.Context, rawArgs string) (string, error) {
	slog.Info("[ORCHESTRATOR] ExecuteListIncidentsRaw triggered", "rawArgs", rawArgs)
	if s.mcpClient2 == nil {
		return "", fmt.Errorf("mcp2 client is not configured")
	}

	var args requests.MCP2ListIncidentsArgs
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		slog.Error("[ORCHESTRATOR] Failed to parse list_incidents raw args", "err", err)
		return "", fmt.Errorf("invalid list_incidents arguments: %w", err)
	}
	if strings.TrimSpace(args.TeamID) == "" {
		return "", fmt.Errorf("team_id is required")
	}
	return s.mcpClient2.ListIncidents(ctx, args)
}

// Eexecutecreaterunbookraw parses raw args and invokes mcp2 create_runbook tool
func (s *orchestratorService) ExecuteCreateRunbookRaw(ctx context.Context, rawArgs string) (string, error) {
	slog.Info("[ORCHESTRATOR] ExecuteCreateRunbookRaw triggered", "rawArgs", rawArgs)
	if s.mcpClient2 == nil {
		return "", fmt.Errorf("mcp2 client is not configured")
	}

	var args requests.MCP2CreateRunbookArgs
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		slog.Error("[ORCHESTRATOR] Failed to parse create_runbook raw args", "err", err)
		return "", fmt.Errorf("invalid create_runbook arguments: %w", err)
	}
	// If team_id is empty or uuid.Nil, auto-resolve team_id from the specified incident_id
	parsedTeamID, _ := uuid.Parse(strings.TrimSpace(args.TeamID))
	if (strings.TrimSpace(args.TeamID) == "" || parsedTeamID == uuid.Nil) && s.teamRepo != nil {
		if incID, err := uuid.Parse(strings.TrimSpace(args.IncidentID)); err == nil && incID != uuid.Nil {
			if inc, _, err := s.teamRepo.GetIncidentContext(ctx, incID); err == nil && inc != nil {
				args.TeamID = inc.TeamID.String()
				slog.Info("[ORCHESTRATOR] Auto-resolved team_id from incident for create_runbook", "incident_id", incID.String(), "team_id", args.TeamID)
			}
		}
	}

	if strings.TrimSpace(args.TeamID) == "" || strings.TrimSpace(args.IncidentID) == "" || strings.TrimSpace(args.Title) == "" || strings.TrimSpace(args.Content) == "" {
		return "", fmt.Errorf("team_id, incident_id, title, and content are required")
	}
	return s.mcpClient2.CreateRunbook(ctx, args)
}

// Executeupdaterunbookraw parses raw args and invokes mcp2 update_runbook tool
func (s *orchestratorService) ExecuteUpdateRunbookRaw(ctx context.Context, rawArgs string) (string, error) {
	slog.Info("[ORCHESTRATOR] ExecuteUpdateRunbookRaw triggered", "rawArgs", rawArgs)
	if s.mcpClient2 == nil {
		return "", fmt.Errorf("mcp2 client is not configured")
	}

	var args requests.MCP2UpdateRunbookArgs
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		slog.Error("[ORCHESTRATOR] Failed to parse update_runbook raw args", "err", err)
		return "", fmt.Errorf("invalid update_runbook arguments: %w", err)
	}
	if strings.TrimSpace(args.RunbookID) == "" {
		return "", fmt.Errorf("runbook_id is required")
	}
	return s.mcpClient2.UpdateRunbook(ctx, args)
}

// Executedeprecaterunbookraw parses raw args and invokes mcp2 deprecate_runbook tool
func (s *orchestratorService) ExecuteDeprecateRunbookRaw(ctx context.Context, rawArgs string) (string, error) {
	slog.Info("[ORCHESTRATOR] ExecuteDeprecateRunbookRaw triggered", "rawArgs", rawArgs)
	if s.mcpClient2 == nil {
		return "", fmt.Errorf("mcp2 client is not configured")
	}

	var args requests.MCP2DeprecateRunbookArgs
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		slog.Error("[ORCHESTRATOR] Failed to parse deprecate_runbook raw args", "err", err)
		return "", fmt.Errorf("invalid deprecate_runbook arguments: %w", err)
	}
	if strings.TrimSpace(args.RunbookID) == "" {
		return "", fmt.Errorf("runbook_id is required")
	}
	return s.mcpClient2.DeprecateRunbook(ctx, args)
}

// Eexecutegetrunbookraw parses raw args and invokes mcp2 get_runbook tool
func (s *orchestratorService) ExecuteGetRunbookRaw(ctx context.Context, rawArgs string) (string, error) {
	slog.Info("[ORCHESTRATOR] ExecuteGetRunbookRaw triggered", "rawArgs", rawArgs)
	if s.mcpClient2 == nil {
		return "", fmt.Errorf("mcp2 client is not configured")
	}

	var args requests.MCP2GetRunbookArgs
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		slog.Error("[ORCHESTRATOR] Failed to parse get_runbook raw args", "err", err)
		return "", fmt.Errorf("invalid get_runbook arguments: %w", err)
	}
	if strings.TrimSpace(args.RunbookID) == "" {
		return "", fmt.Errorf("runbook_id is required")
	}
	return s.mcpClient2.GetRunbook(ctx, args)
}

// Executelistrunbooksraw parses raw args and invokes mcp2 list_runbooks tool
func (s *orchestratorService) ExecuteListRunbooksRaw(ctx context.Context, rawArgs string) (string, error) {
	slog.Info("[ORCHESTRATOR] ExecuteListRunbooksRaw triggered", "rawArgs", rawArgs)
	if s.mcpClient2 == nil {
		return "", fmt.Errorf("mcp2 client is not configured")
	}

	var args requests.MCP2ListRunbooksArgs
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		slog.Error("[ORCHESTRATOR] Failed to parse list_runbooks raw args", "err", err)
		return "", fmt.Errorf("invalid list_runbooks arguments: %w", err)
	}
	if strings.TrimSpace(args.TeamID) == "" {
		return "", fmt.Errorf("team_id is required")
	}
	return s.mcpClient2.ListRunbooks(ctx, args)
}

// Executelinkalerttoincidentraw links an alert to a specific incident id
func (s *orchestratorService) ExecuteLinkAlertToIncidentRaw(ctx context.Context, rawArgs string) (string, error) {
	slog.Info("[ORCHESTRATOR] ExecuteLinkAlertToIncidentRaw triggered", "rawArgs", rawArgs)

	var args requests.MCP2LinkAlertIncidentArgs
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		slog.Error("[ORCHESTRATOR] Failed to parse link_alert_to_incident raw args", "err", err)
		return "", fmt.Errorf("invalid link_alert_to_incident arguments: %w", err)
	}

	alertID := strings.TrimSpace(args.AlertID)
	incidentID := strings.TrimSpace(args.IncidentID)
	incidentTitle := strings.TrimSpace(args.IncidentTitle)

	if alertID == "" && incidentID != "" {
		// Recover from a common model mistake where the alert UUID is placed in incident_id.
		if _, err := uuid.Parse(incidentID); err == nil {
			alertID = incidentID
			incidentID = ""
		}
	}

	alertUUID, err := uuid.Parse(alertID)
	if err != nil {
		return "", fmt.Errorf("invalid alert_id %q: %w", args.AlertID, err)
	}

	var incidentUUID uuid.UUID
	var parseErr error

	if incidentID != "" {
		incidentUUID, parseErr = uuid.Parse(incidentID)
	}

	// if incident_id is not a valid UUID, treat it or incident_title as a human readable title to resolve
	if (parseErr != nil || incidentUUID == uuid.Nil) && s.teamRepo != nil {
		titleToSearch := incidentTitle
		if titleToSearch == "" {
			titleToSearch = incidentID
		}
		if titleToSearch != "" {
			if incidents, err := s.teamRepo.ListTeamIncidents(ctx, uuid.Nil); err == nil {
				cleanSearch := strings.TrimSpace(titleToSearch)

				// 1. Strict exact match first (case-insensitive)
				for _, inc := range incidents {
					if strings.EqualFold(strings.TrimSpace(inc.Title), cleanSearch) {
						incidentUUID = inc.ID
						slog.Info("[ORCHESTRATOR] Resolved incident by exact title match", "title", cleanSearch, "incident_id", incidentUUID)
						break
					}
				}

				// 2. Substring match ONLY if search string contains full service/incident title
				if incidentUUID == uuid.Nil {
					lowSearch := strings.ToLower(cleanSearch)
					var matches []models.TeamIncident
					for _, inc := range incidents {
						lowTitle := strings.ToLower(strings.TrimSpace(inc.Title))
						if strings.Contains(lowSearch, lowTitle) || strings.Contains(lowTitle, lowSearch) {
							matches = append(matches, inc)
						}
					}
					// Only link if exactly 1 unambiguous match was found
					if len(matches) == 1 {
						incidentUUID = matches[0].ID
						slog.Info("[ORCHESTRATOR] Resolved incident by single unambiguous title substring", "search", cleanSearch, "matched_title", matches[0].Title, "incident_id", incidentUUID)
					} else if len(matches) > 1 {
						slog.Warn("[ORCHESTRATOR] Multiple matching incidents found for title, aborting to prevent mislinking", "count", len(matches), "title", cleanSearch)
					}
				}
			}
		}
	}

	if incidentUUID == uuid.Nil {
		return "", fmt.Errorf("could not resolve valid incident UUID from arguments (incident_id=%q, incident_title=%q)", args.IncidentID, args.IncidentTitle)
	}

	if err := s.alertRepo.UpdateAlertIncidentID(ctx, alertUUID, incidentUUID); err != nil {
		slog.Error("[ORCHESTRATOR] Failed to link alert to incident", "alert_id", alertUUID, "incident_id", incidentUUID, "err", err)
		return "", fmt.Errorf("failed to link alert to incident: %w", err)
	}

	slog.Info("[ORCHESTRATOR] Successfully linked alert to incident", "alert_id", alertUUID, "incident_id", incidentUUID)
	result := map[string]string{
		"status":      "success",
		"alert_id":    alertUUID.String(),
		"incident_id": incidentUUID.String(),
	}
	resultBytes, _ := json.Marshal(result)
	return string(resultBytes), nil
}
