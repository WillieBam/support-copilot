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


