package data

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/WillieBam/support_copilot/backend/types"
	"github.com/WillieBam/support_copilot/backend/types/models"
	"github.com/WillieBam/support_copilot/backend/types/requests"
	"github.com/WillieBam/support_copilot/backend/types/responses"
)

// ParseAlertArgs decodes alert id from a raw llm tool call json string
func ParseAlertArgs(rawArgs string) (alertID string, err error) {
	var args struct {
		AlertID string `json:"alert_id"`
		ID      string `json:"id"`
		Alert   struct {
			ID string `json:"id"`
		} `json:"alert"`
		AlertInfo struct {
			ID string `json:"id"`
		} `json:"alert_info"`
	}
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		return "", fmt.Errorf("invalid alert arguments: %w", err)
	}
	if args.AlertID != "" {
		return args.AlertID, nil
	}
	if args.Alert.ID != "" {
		return args.Alert.ID, nil
	}
	if args.AlertInfo.ID != "" {
		return args.AlertInfo.ID, nil
	}
	return args.ID, nil
}

// ParseAlertMetrics decodes metrics json into anomaly detection request
func ParseAlertMetrics(metricsJSON string) (requests.AnomalyDetectionRequest, error) {
	var metrics requests.AnomalyDetectionRequest
	if err := json.Unmarshal([]byte(metricsJSON), &metrics); err != nil {
		return metrics, fmt.Errorf("failed to parse alert metrics JSON: %w", err)
	}
	return metrics, nil
}

// MarshalAlerts maps a slice of alert models to alert list item dtos and returns json
func MarshalAlerts(alerts []*models.Alert) (string, error) {
	items := make([]types.AlertListItem, 0, len(alerts))
	for _, a := range alerts {
		resSec, _ := UnmarshalResourceSection(a.ResourceInfo)
		alertSec, _ := UnmarshalAlertSection(a.AlertInfo)

		serviceName := ""
		if resSec != nil {
			serviceName = resSec.Service
		}
		severity := ""
		if alertSec != nil {
			severity = alertSec.Severity
		}

		item := types.AlertListItem{
			ID:          a.ID.String(),
			ServiceName: serviceName,
			Severity:    severity,
			ReceivedAt:  a.ReceivedAt.Format(time.RFC3339),
		}
		if a.IncidentID != nil {
			item.IncidentID = a.IncidentID.String()
		}
		items = append(items, item)
	}
	out, err := json.Marshal(items)
	if err != nil {
		return "", fmt.Errorf("failed to marshal alerts: %w", err)
	}
	return string(out), nil
}

// UnmarshalAlertSection decodes raw json string into alert section dto
func UnmarshalAlertSection(jsonStr string) (*types.AlertSection, error) {
	if jsonStr == "" {
		return nil, nil
	}
	var data types.AlertSection
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal alert section: %w", err)
	}
	return &data, nil
}

// UnmarshalResourceSection decodes raw json string into resource section dto
func UnmarshalResourceSection(jsonStr string) (*types.ResourceSection, error) {
	if jsonStr == "" {
		return nil, nil
	}
	var data types.ResourceSection
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal resource section: %w", err)
	}
	return &data, nil
}

// UnmarshalMetricsSection decodes raw json string into alert metrics dto
func UnmarshalMetricsSection(jsonStr string) (*types.AlertMetrics, error) {
	if jsonStr == "" {
		return nil, nil
	}
	var data types.AlertMetrics
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metrics section: %w", err)
	}
	return &data, nil
}

// UnmarshalBusinessContextSection decodes raw json string into business context section dto
func UnmarshalBusinessContextSection(jsonStr string) (*types.BusinessContextSection, error) {
	if jsonStr == "" {
		return nil, nil
	}
	var data types.BusinessContextSection
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal business context section: %w", err)
	}
	return &data, nil
}

// UnmarshalMetadataSection decodes raw json string into metadata section dto
func UnmarshalMetadataSection(jsonStr string) (*types.MetadataSection, error) {
	if jsonStr == "" {
		return nil, nil
	}
	var data types.MetadataSection
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata section: %w", err)
	}
	return &data, nil
}

// UnmarshalAlertRecord decodes all 5 json sections of a models alert record
func UnmarshalAlertRecord(a *models.Alert) (*types.ParsedAlertRecord, error) {
	if a == nil {
		return nil, nil
	}
	alertSec, _ := UnmarshalAlertSection(a.AlertInfo)
	resSec, _ := UnmarshalResourceSection(a.ResourceInfo)
	metricsSec, _ := UnmarshalMetricsSection(a.Metrics)
	bizSec, _ := UnmarshalBusinessContextSection(a.BusinessContext)
	metaSec, _ := UnmarshalMetadataSection(a.Metadata)

	rec := &types.ParsedAlertRecord{
		ID:              a.ID.String(),
		ReceivedAt:      a.ReceivedAt.Format(time.RFC3339),
		Alert:           alertSec,
		Resource:        resSec,
		Metrics:         metricsSec,
		BusinessContext: bizSec,
		Metadata:        metaSec,
	}
	if a.IncidentID != nil {
		incID := a.IncidentID.String()
		rec.IncidentID = &incID
	}
	return rec, nil
}

// MarshalValidationResult serialises a combined validation result to a clean structured summary string
func MarshalValidationResult(result *responses.CombinedValidationResult) (string, error) {
	if result == nil {
		return "[]", nil
	}

	var sb strings.Builder
	sb.WriteString("[Alert Validation Package]\n")
	sb.WriteString(fmt.Sprintf("• Alert ID: %s\n", result.AlertID))
	sb.WriteString(fmt.Sprintf("• Service: %s (Severity: %s)\n", result.ServiceName, result.Severity))
	if !result.ReceivedAt.IsZero() {
		sb.WriteString(fmt.Sprintf("• Received At: %s\n", result.ReceivedAt.UTC().Format("2006-01-02 15:04:05 MST")))
	}

	// ML Prediction Verdict
	pred := result.MLPrediction
	verdict := pred.Label
	if verdict == "" {
		if pred.Status == 0 {
			verdict = "Real Alert (Anomaly)"
		} else {
			verdict = "False Alarm (Normal)"
		}
	}
	sb.WriteString(fmt.Sprintf("• ML Verdict: %s | Risk: %s | Confidence: %.0f%% (Score: %.4f)\n",
		verdict, pred.RiskLevel, pred.Confidence*100, pred.AnomalyScore))
	if pred.Summary != "" {
		sb.WriteString(fmt.Sprintf("• Assessment: %s\n", pred.Summary))
	}

	// Telemetry Metrics
	var metricParts []string
	if result.Metrics.CpuUsage > 0 {
		metricParts = append(metricParts, fmt.Sprintf("cpu_usage=%.1f%%", result.Metrics.CpuUsage))
	}
	if result.Metrics.MemoryUsage > 0 {
		metricParts = append(metricParts, fmt.Sprintf("memory_usage=%.1f%%", result.Metrics.MemoryUsage))
	}
	if result.Metrics.ResponseLatency > 0 {
		metricParts = append(metricParts, fmt.Sprintf("response_latency=%.1fms", result.Metrics.ResponseLatency))
	}
	if result.Metrics.ErrorRate > 0 {
		metricParts = append(metricParts, fmt.Sprintf("error_rate=%.2f%%", result.Metrics.ErrorRate))
	}
	if result.Metrics.IncomingTraffic > 0 {
		metricParts = append(metricParts, fmt.Sprintf("incoming_traffic=%.1f", result.Metrics.IncomingTraffic))
	}
	if result.Metrics.OutgoingTraffic > 0 {
		metricParts = append(metricParts, fmt.Sprintf("outgoing_traffic=%.1f", result.Metrics.OutgoingTraffic))
	}
	if result.Metrics.NetworkThroughput > 0 {
		metricParts = append(metricParts, fmt.Sprintf("network_throughput=%.1f", result.Metrics.NetworkThroughput))
	}
	if result.Metrics.RequestRate > 0 {
		metricParts = append(metricParts, fmt.Sprintf("request_rate=%.1f", result.Metrics.RequestRate))
	}
	if result.Metrics.AvailabilityPercent > 0 {
		metricParts = append(metricParts, fmt.Sprintf("availability=%.1f%%", result.Metrics.AvailabilityPercent))
	}
	if len(metricParts) > 0 {
		sb.WriteString(fmt.Sprintf("• Telemetry Metrics: %s\n", strings.Join(metricParts, ", ")))
	}

	// Resource Context
	if result.Resource != nil {
		var resParts []string
		if result.Resource.Service != "" {
			resParts = append(resParts, fmt.Sprintf("service=%s", result.Resource.Service))
		}
		if result.Resource.Environment != "" {
			resParts = append(resParts, fmt.Sprintf("env=%s", result.Resource.Environment))
		}
		if result.Resource.Cluster != "" {
			resParts = append(resParts, fmt.Sprintf("cluster=%s", result.Resource.Cluster))
		}
		if result.Resource.Namespace != "" {
			resParts = append(resParts, fmt.Sprintf("namespace=%s", result.Resource.Namespace))
		}
		if result.Resource.Deployment != "" {
			resParts = append(resParts, fmt.Sprintf("deployment=%s", result.Resource.Deployment))
		}
		if len(resParts) > 0 {
			sb.WriteString(fmt.Sprintf("• Resource Context: %s\n", strings.Join(resParts, ", ")))
		}
	}

	// Business Context
	if result.BusinessContext != nil {
		var bizParts []string
		if result.BusinessContext.BusinessService != "" {
			bizParts = append(bizParts, fmt.Sprintf("business_service=%s", result.BusinessContext.BusinessService))
		}
		if result.BusinessContext.ExpectedDataReadyTime != "" {
			bizParts = append(bizParts, fmt.Sprintf("expected_data_ready=%s", result.BusinessContext.ExpectedDataReadyTime))
		}
		if result.BusinessContext.CurrentTime != "" {
			bizParts = append(bizParts, fmt.Sprintf("current_time=%s", result.BusinessContext.CurrentTime))
		}
		if result.BusinessContext.UserQueryWindow != nil {
			bizParts = append(bizParts, fmt.Sprintf("user_query_window=%t", *result.BusinessContext.UserQueryWindow))
		}
		if len(bizParts) > 0 {
			sb.WriteString(fmt.Sprintf("• Business Context: %s\n", strings.Join(bizParts, ", ")))
		}
	}

	// Alert Info Context
	if result.Alert != nil {
		var alertParts []string
		if result.Alert.MonitorName != "" {
			alertParts = append(alertParts, fmt.Sprintf("monitor=%s", result.Alert.MonitorName))
		}
		if result.Alert.Message != "" {
			alertParts = append(alertParts, fmt.Sprintf("message=%s", result.Alert.Message))
		}
		if len(alertParts) > 0 {
			sb.WriteString(fmt.Sprintf("• Alert Info: %s\n", strings.Join(alertParts, ", ")))
		}
	}

	return sb.String(), nil
}

// UnmarshalAlerts decodes a json string into a slice of alert records for display
func UnmarshalAlerts(jsonStr string) ([]types.AlertRecord, error) {
	var alerts []types.AlertRecord
	if err := json.Unmarshal([]byte(jsonStr), &alerts); err != nil {
		return nil, err
	}
	return alerts, nil
}


