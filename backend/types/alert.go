package types

// AlertListItem is a JSON-serialisable DTO used when listing alerts.
// models.Alert relies on GORM struct tags and has no json tags, so mapping
// to this DTO guarantees the snake_case field names that AlertRecord expects.
type AlertListItem struct {
	ID          string `json:"id"`
	ServiceName string `json:"service_name"`
	Severity    string `json:"severity"`
	ReceivedAt  string `json:"received_at"`
	IncidentID  string `json:"incident_id,omitempty"`
}

type AlertMetrics struct {
	CPUUsage            float64 `json:"cpu_usage"`
	MemoryUsage         float64 `json:"memory_usage"`
	IncomingTraffic     float64 `json:"incoming_traffic"`
	OutgoingTraffic     float64 `json:"outgoing_traffic"`
	ErrorRate           float64 `json:"error_rate"`
	NetworkThroughput   float64 `json:"network_throughput"`
	RequestRate         float64 `json:"request_rate"`
	ResponseLatency     float64 `json:"response_latency"`
	AvailabilityPercent float64 `json:"availability_percent"`
}
