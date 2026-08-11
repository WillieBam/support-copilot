package types

// AlertListItem is json serialisable dto used when listing alerts
type AlertListItem struct {
	ID          string `json:"id"`
	ServiceName string `json:"service_name,omitempty"`
	Severity    string `json:"severity,omitempty"`
	ReceivedAt  string `json:"received_at"`
	IncidentID  string `json:"incident_id,omitempty"`
}

// AlertMetrics contains performance and traffic metric values
type AlertMetrics struct {
	CPUUsage            *float64 `json:"cpu_usage,omitempty"`
	MemoryUsage         *float64 `json:"memory_usage,omitempty"`
	IncomingTraffic     *float64 `json:"incoming_traffic,omitempty"`
	OutgoingTraffic     *float64 `json:"outgoing_traffic,omitempty"`
	ErrorRate           *float64 `json:"error_rate,omitempty"`
	NetworkThroughput   *float64 `json:"network_throughput,omitempty"`
	RequestRate         *float64 `json:"request_rate,omitempty"`
	ResponseLatency     *float64 `json:"response_latency,omitempty"`
	AvailabilityPercent *float64 `json:"availability_percent,omitempty"`
}

// AlertSection holds detailed alert information
type AlertSection struct {
	ID          string `json:"id,omitempty"`
	Source      string `json:"source,omitempty"`
	MonitorName string `json:"monitor_name,omitempty"`
	MonitorType string `json:"monitor_type,omitempty"`
	Severity    string `json:"severity,omitempty"`
	Status      string `json:"status,omitempty"`
	Timestamp   string `json:"timestamp,omitempty"`
	Message     string `json:"message,omitempty"`
}

// ResourceSection holds infrastructure and service environment details
type ResourceSection struct {
	Service     string `json:"service,omitempty"`
	Environment string `json:"environment,omitempty"`
	Cluster     string `json:"cluster,omitempty"`
	Namespace   string `json:"namespace,omitempty"`
	Deployment  string `json:"deployment,omitempty"`
}

// BusinessContextSection holds service schedule and window context
type BusinessContextSection struct {
	BusinessService       string `json:"business_service,omitempty"`
	ExpectedDataReadyTime string `json:"expected_data_ready_time,omitempty"`
	CurrentTime           string `json:"current_time,omitempty"`
	UserQueryWindow       *bool  `json:"user_query_window,omitempty"`
}

// MetadataSection holds webhook payload version and timestamp metadata
type MetadataSection struct {
	ReceivedAt string `json:"received_at,omitempty"`
	Version    string `json:"version,omitempty"`
}

// ParsedAlertRecord aggregates all unmarshaled sections of an alert
type ParsedAlertRecord struct {
	ID              string                  `json:"id"`
	IncidentID      *string                 `json:"incident_id,omitempty"`
	ReceivedAt      string                  `json:"received_at"`
	Alert           *AlertSection           `json:"alert,omitempty"`
	Resource        *ResourceSection        `json:"resource,omitempty"`
	Metrics         *AlertMetrics           `json:"metrics,omitempty"`
	BusinessContext *BusinessContextSection `json:"business_context,omitempty"`
	Metadata        *MetadataSection        `json:"metadata,omitempty"`
}

