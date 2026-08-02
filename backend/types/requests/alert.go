package requests

import (
	"encoding/json"

	"github.com/google/uuid"
)

type AlertIngestRequest struct {
	IncidentID  *uuid.UUID      `json:"incident_id,omitempty" query:"incident_id"`
	ServiceName string          `json:"service_name" query:"service_name"`
	Severity    string          `json:"severity" query:"severity"`
	Metrics     json.RawMessage `json:"metrics" query:"metrics"`
}
