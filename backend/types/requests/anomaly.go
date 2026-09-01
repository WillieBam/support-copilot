package requests

type AnomalyDetectionRequest struct {
	CpuUsage            float64 `json:"cpu_usage"`
	MemoryUsage         float64 `json:"memory_usage"`
	IncomingTraffic     float64 `json:"incoming_traffic"`
	OutgoingTraffic     float64 `json:"outgoing_traffic"`
	ErrorRate           float64 `json:"error_rate"`
	NetworkThroughput   float64 `json:"network_throughput"`
	RequestRate         float64 `json:"request_rate"`
	ResponseLatency     float64 `json:"response_latency"`
	AvailabilityPercent float64 `json:"availability_percent"`
}

type AnomalyDetectionResponse struct {
	Status       int     `json:"status"` // 0 for Anomaly, 1 for Normal
	Label        string  `json:"label"`  // "Real Alert (Anomaly)" or "False Alarm (Normal)"
	Engine       string  `json:"engine"` // "IsolationForest_v3"
	Prediction   string  `json:"prediction,omitempty"`
	Confidence   float64 `json:"confidence,omitempty"`
	RiskLevel    string  `json:"risk_level,omitempty"`
	AnomalyScore float64 `json:"anomaly_score,omitempty"`
	Threshold    float64 `json:"threshold,omitempty"`
	Summary      string  `json:"summary,omitempty"`
	Error        string  `json:"error,omitempty"`
}
