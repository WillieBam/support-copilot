package requests

import (
	"time"

	"github.com/google/uuid"
)

type AlertIngestRequest struct {
	IncidentID      *uuid.UUID           `json:"incident_id,omitempty"`
	Alert           AlertInfo            `json:"alert"`
	Resource        ResourceInfo         `json:"resource"`
	Metrics         MetricsInfo          `json:"metrics"`
	BusinessContext *BusinessContextInfo `json:"business_context,omitempty"`
	Metadata        Metadata             `json:"metadata"`
}

// AlertQueryRequest is used when user queries to find an alert
type AlertQueryRequest struct {
	Alert AlertInfo `json:"alert"`
}



type AlertInfo struct {
	ID          string    `json:"id"`
	Source      string    `json:"source"`
	MonitorName string    `json:"monitor_name"`
	MonitorType string    `json:"monitor_type"`
	Severity    string    `json:"severity"`
	Status      string    `json:"status"`
	Timestamp   time.Time `json:"timestamp"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
}

type ResourceInfo struct {
	Service     string `json:"service"`
	Environment string `json:"environment"`
	Cluster     string `json:"cluster"`
	Namespace   string `json:"namespace"`
	Deployment  string `json:"deployment"`
}

type MetricsInfo struct {
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

type Metadata struct {
	ReceivedAt string `json:"received_at"`
	Version    string `json:"version"`
}

type BusinessContextInfo struct {
	BusinessService       string `json:"business_service"`
	ExpectedDataReadyTime string `json:"expected_data_ready_time"`
	UserQueryWindow       bool   `json:"user_query_window"`
}
