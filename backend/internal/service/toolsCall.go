package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/WillieBam/support_copilot/backend/internal/domain/data"
	"github.com/WillieBam/support_copilot/backend/internal/interfaces"
	"github.com/WillieBam/support_copilot/backend/types/models"
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

	out, err := data.MarshalAlerts(alerts)
	if err != nil {
		slog.Error("[ORCHESTRATOR] Failed to marshal alerts", "err", err)
		return "", err
	}
	return out, nil
}

// ExecuteValidateAlert fetches alert metrics from postgres and predicts anomalies via python mcp
func (s *orchestratorService) ExecuteValidateAlert(ctx context.Context, alertID string) (*responses.CombinedValidationResult, error) {
	slog.Info("[ORCHESTRATOR] Fetching alert from database", "alert_id", alertID)

	// fetch alert from postgres
	alertRecord, err := s.alertRepo.RetrieveAlertbyID(ctx, alertID)
	if err != nil {
		slog.Error("[ORCHESTRATOR] Failed to fetch alert from DB", "alert_id", alertID, "err", err)
		return nil, fmt.Errorf("failed to fetch alert #%s from database: %w", alertID, err)
	}

	resSec, _ := data.UnmarshalResourceSection(alertRecord.ResourceInfo)
	alertSec, _ := data.UnmarshalAlertSection(alertRecord.AlertInfo)
	bizSec, _ := data.UnmarshalBusinessContextSection(alertRecord.BusinessContext)
	metaSec, _ := data.UnmarshalMetadataSection(alertRecord.Metadata)

	serviceName := ""
	if resSec != nil {
		serviceName = resSec.Service
	}
	severity := ""
	if alertSec != nil {
		severity = alertSec.Severity
	}

	slog.Info("[ORCHESTRATOR] Alert retrieved from DB", "service", serviceName, "severity", severity)

	// unmarshal alert json metrics string into anomaly detection request struct
	metrics, err := data.ParseAlertMetrics(alertRecord.Metrics)
	if err != nil {
		slog.Error("[ORCHESTRATOR] Failed to unmarshal alert metrics JSON", "err", err)
		return nil, err
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
		AlertID:         alertRecord.ID.String(),
		ServiceName:     serviceName,
		Severity:        severity,
		ReceivedAt:      alertRecord.ReceivedAt,
		Alert:           alertSec,
		Resource:        resSec,
		BusinessContext: bizSec,
		Metadata:        metaSec,
		Metrics:         metrics,
		MLPrediction:    *mcpResp,
	}, nil


}

// ExecuteValidateAlertRaw parses raw llm json arguments and delegates execution
func (s *orchestratorService) ExecuteValidateAlertRaw(ctx context.Context, rawArgs string) (string, error) {
	slog.Info("[ORCHESTRATOR] ExecuteValidateAlertRaw triggered", "rawArgs", rawArgs)

	alertIDStr, err := data.ParseAlertArgs(rawArgs)
	if err != nil {
		slog.Error("[ORCHESTRATOR] Failed to parse raw tool arguments", "rawArgs", rawArgs, "err", err)
		return "", err
	}

	cleanAlertID := strings.TrimSpace(alertIDStr)
	if cleanAlertID == "" || cleanAlertID == "null" || cleanAlertID == "none" || cleanAlertID == "undefined" {
		slog.Warn("[ORCHESTRATOR] Empty or dummy alert_id provided", "alertID", alertIDStr)
		return "", fmt.Errorf("no valid alert_id provided: %q", alertIDStr)
	}

	result, err := s.ExecuteValidateAlert(ctx, cleanAlertID)
	if err != nil {
		return "", err
	}

	resultStr, err := data.MarshalValidationResult(result)
	if err != nil {
		slog.Error("[ORCHESTRATOR] Failed to marshal validation result", "err", err)
		return "", err
	}

	slog.Info("[ORCHESTRATOR] Combined validation package built successfully", "result", resultStr)
	return resultStr, nil
}

// normalizeUUID cleans malformed uuid strings with extra hyphens
func normalizeUUID(input string) string {
	cleaned := strings.TrimSpace(input)
	cleaned = strings.Trim(cleaned, `"'`)
	digits := strings.ReplaceAll(cleaned, "-", "")
	if len(digits) == 32 {
		return fmt.Sprintf("%s-%s-%s-%s-%s", digits[:8], digits[8:12], digits[12:16], digits[16:20], digits[20:])
	}
	return cleaned
}

// Executegetincidentraw parses raw args and invokes mcp2 get_incident tool
func (s *orchestratorService) ExecuteGetIncidentRaw(ctx context.Context, rawArgs string) (string, error) {
	slog.Info("[ORCHESTRATOR] ExecuteGetIncidentRaw triggered", "rawArgs", rawArgs)
	if s.mcpClient2 == nil {
		return "", fmt.Errorf("mcp2 client is not configured")
	}

	args, err := data.ParseGetIncidentArgs(rawArgs)
	if err != nil {
		slog.Error("[ORCHESTRATOR] Failed to parse get_incident raw args", "err", err)
		return "", err
	}
	args.IncidentID = normalizeUUID(args.IncidentID)
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

	args, err := data.ParseListIncidentsArgs(rawArgs)
	if err != nil {
		slog.Error("[ORCHESTRATOR] Failed to parse list_incidents raw args", "err", err)
		return "", err
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

	args, err := data.ParseCreateRunbookArgs(rawArgs)
	if err != nil {
		slog.Error("[ORCHESTRATOR] Failed to parse create_runbook raw args", "err", err)
		return "", err
	}
	// If team_id is empty, nil, or refers to a non-existent team (like placeholder a0000000-...), auto-resolve team_id from incident
	parsedTeamID, _ := uuid.Parse(strings.TrimSpace(args.TeamID))
	teamExists := false
	if parsedTeamID != uuid.Nil && s.teamRepo != nil {
		if _, err := s.teamRepo.GetTeamByID(ctx, parsedTeamID); err == nil {
			teamExists = true
		}
	}

	if (!teamExists || strings.TrimSpace(args.TeamID) == "" || parsedTeamID == uuid.Nil) && s.teamRepo != nil {
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

	args, err := data.ParseUpdateRunbookArgs(rawArgs)
	if err != nil {
		slog.Error("[ORCHESTRATOR] Failed to parse update_runbook raw args", "err", err)
		return "", err
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

	args, err := data.ParseDeprecateRunbookArgs(rawArgs)
	if err != nil {
		slog.Error("[ORCHESTRATOR] Failed to parse deprecate_runbook raw args", "err", err)
		return "", err
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

	args, err := data.ParseGetRunbookArgs(rawArgs)
	if err != nil {
		slog.Error("[ORCHESTRATOR] Failed to parse get_runbook raw args", "err", err)
		return "", err
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

	args, err := data.ParseListRunbooksArgs(rawArgs)
	if err != nil {
		slog.Error("[ORCHESTRATOR] Failed to parse list_runbooks raw args", "err", err)
		return "", err
	}
	if strings.TrimSpace(args.TeamID) == "" {
		return "", fmt.Errorf("team_id is required")
	}
	return s.mcpClient2.ListRunbooks(ctx, args)
}

// Executelinkalerttoincidentraw links an alert to a specific incident id
func (s *orchestratorService) ExecuteLinkAlertToIncidentRaw(ctx context.Context, rawArgs string) (string, error) {
	slog.Info("[ORCHESTRATOR] ExecuteLinkAlertToIncidentRaw triggered", "rawArgs", rawArgs)

	args, err := data.ParseLinkAlertArgs(rawArgs)
	if err != nil {
		slog.Error("[ORCHESTRATOR] Failed to parse link_alert_to_incident raw args", "err", err)
		return "", err
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
	return data.MarshalLinkResult(alertUUID.String(), incidentUUID.String())
}
