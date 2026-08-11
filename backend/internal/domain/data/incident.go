package data

import (
	"encoding/json"
	"fmt"

	"github.com/WillieBam/support_copilot/backend/types"
	"github.com/WillieBam/support_copilot/backend/types/requests"
)

// ParseGetIncidentArgs decodes incident id from raw arguments
func ParseGetIncidentArgs(rawArgs string) (requests.MCP2GetIncidentArgs, error) {
	var args requests.MCP2GetIncidentArgs
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		return args, fmt.Errorf("invalid get_incident arguments: %w", err)
	}
	return args, nil
}

// ParseListIncidentsArgs decodes team id from raw arguments
func ParseListIncidentsArgs(rawArgs string) (requests.MCP2ListIncidentsArgs, error) {
	var args requests.MCP2ListIncidentsArgs
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		return args, fmt.Errorf("invalid list_incidents arguments: %w", err)
	}
	return args, nil
}

// UnmarshalIncidents decodes a json string into a slice of incident records for display
func UnmarshalIncidents(jsonStr string) ([]types.IncidentRecord, error) {
	var incidents []types.IncidentRecord
	if err := json.Unmarshal([]byte(jsonStr), &incidents); err != nil {
		return nil, err
	}
	return incidents, nil
}
