package data

import (
	"encoding/json"
	"fmt"

	"github.com/WillieBam/support_copilot/backend/types"
	"github.com/WillieBam/support_copilot/backend/types/requests"
)

// ParseCreateRunbookArgs decodes create runbook arguments from raw json
func ParseCreateRunbookArgs(rawArgs string) (requests.MCP2CreateRunbookArgs, error) {
	var args requests.MCP2CreateRunbookArgs
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		return args, fmt.Errorf("invalid create_runbook arguments: %w", err)
	}
	return args, nil
}

// ParseUpdateRunbookArgs decodes update runbook arguments from raw json
func ParseUpdateRunbookArgs(rawArgs string) (requests.MCP2UpdateRunbookArgs, error) {
	var args requests.MCP2UpdateRunbookArgs
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		return args, fmt.Errorf("invalid update_runbook arguments: %w", err)
	}
	return args, nil
}

// ParseDeprecateRunbookArgs decodes deprecate runbook arguments from raw json
func ParseDeprecateRunbookArgs(rawArgs string) (requests.MCP2DeprecateRunbookArgs, error) {
	var args requests.MCP2DeprecateRunbookArgs
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		return args, fmt.Errorf("invalid deprecate_runbook arguments: %w", err)
	}
	return args, nil
}

// ParseGetRunbookArgs decodes get runbook arguments from raw json
func ParseGetRunbookArgs(rawArgs string) (requests.MCP2GetRunbookArgs, error) {
	var args requests.MCP2GetRunbookArgs
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		return args, fmt.Errorf("invalid get_runbook arguments: %w", err)
	}
	return args, nil
}

// ParseListRunbooksArgs decodes list runbooks arguments from raw json
func ParseListRunbooksArgs(rawArgs string) (requests.MCP2ListRunbooksArgs, error) {
	var args requests.MCP2ListRunbooksArgs
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		return args, fmt.Errorf("invalid list_runbooks arguments: %w", err)
	}
	return args, nil
}

// ParseLinkAlertArgs decodes link alert to incident arguments from raw json
func ParseLinkAlertArgs(rawArgs string) (requests.MCP2LinkAlertIncidentArgs, error) {
	var args requests.MCP2LinkAlertIncidentArgs
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		return args, fmt.Errorf("invalid link_alert_to_incident arguments: %w", err)
	}
	return args, nil
}

// MarshalLinkResult serializes alert and incident link result to json
func MarshalLinkResult(alertID, incidentID string) (string, error) {
	result := map[string]string{
		"status":      "success",
		"alert_id":    alertID,
		"incident_id": incidentID,
	}
	b, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal link result: %w", err)
	}
	return string(b), nil
}

// UnmarshalRunbooks decodes json string to runbook records
func UnmarshalRunbooks(jsonStr string) ([]types.RunbookRecord, error) {
	var runbooks []types.RunbookRecord
	if err := json.Unmarshal([]byte(jsonStr), &runbooks); err != nil {
		return nil, err
	}
	return runbooks, nil
}
