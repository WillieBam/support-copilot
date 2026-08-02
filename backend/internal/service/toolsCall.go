package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/WillieBam/support_copilot/backend/internal/interfaces"
	"github.com/WillieBam/support_copilot/backend/types/requests"
	"github.com/WillieBam/support_copilot/backend/types/responses"
	"github.com/google/uuid"
)

type orchestratorService struct {
	alertRepo  interfaces.IAlertRepository
	mcpClient1 interfaces.IMCPClient
	mcpClient2 interfaces.IMCP2Client
}

func NewOrchestratorService(
	repo interfaces.IAlertRepository,
	mcpClient1 interfaces.IMCPClient,
	mcpClient2 interfaces.IMCP2Client,
) interfaces.IOrchestratorService {
	return &orchestratorService{
		alertRepo:  repo,
		mcpClient1: mcpClient1,
		mcpClient2: mcpClient2,
	}
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

//Eexecutecreaterunbookraw parses raw args and invokes mcp2 create_runbook tool
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

//Eexecutegetrunbookraw parses raw args and invokes mcp2 get_runbook tool
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

	alertUUID, err := uuid.Parse(strings.TrimSpace(args.AlertID))
	if err != nil {
		return "", fmt.Errorf("invalid alert_id %q: %w", args.AlertID, err)
	}

	incidentUUID, err := uuid.Parse(strings.TrimSpace(args.IncidentID))
	if err != nil {
		return "", fmt.Errorf("invalid incident_id %q: %w", args.IncidentID, err)
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
