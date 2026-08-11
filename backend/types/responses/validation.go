package responses

import (
	"time"

	"github.com/WillieBam/support_copilot/backend/types"
	"github.com/WillieBam/support_copilot/backend/types/requests"
)

// CombinedValidationResult aggregates mcp ml predictions with complete alert context for llm analysis
type CombinedValidationResult struct {
	AlertID         string                            `json:"alert_id"`
	ServiceName     string                            `json:"service_name"`
	Severity        string                            `json:"severity"`
	ReceivedAt      time.Time                         `json:"received_at"`
	Alert           *types.AlertSection               `json:"alert,omitempty"`
	Resource        *types.ResourceSection            `json:"resource,omitempty"`
	BusinessContext *types.BusinessContextSection     `json:"business_context,omitempty"`
	Metadata        *types.MetadataSection            `json:"metadata,omitempty"`
	Metrics         requests.AnomalyDetectionRequest  `json:"metrics"`
	MLPrediction    requests.AnomalyDetectionResponse `json:"ml_prediction"`
}


