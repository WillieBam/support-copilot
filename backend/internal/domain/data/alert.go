package data

import (
	"encoding/json"
	"fmt"
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
	}
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		return "", fmt.Errorf("invalid alert arguments: %w", err)
	}
	return args.AlertID, nil
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
		return "", fmt.Errorf("failed to marshal alerts: %w", err)
	}
	return string(out), nil
}

// MarshalValidationResult serialises a combined validation result to a json string
func MarshalValidationResult(result *responses.CombinedValidationResult) (string, error) {
	b, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal validation result: %w", err)
	}
	return string(b), nil
}

// UnmarshalAlerts decodes a json string into a slice of alert records for display
func UnmarshalAlerts(jsonStr string) ([]types.AlertRecord, error) {
	var alerts []types.AlertRecord
	if err := json.Unmarshal([]byte(jsonStr), &alerts); err != nil {
		return nil, err
	}
	return alerts, nil
}

