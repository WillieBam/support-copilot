package responses

// IncidentContextResponse is the cleansed, llm-optimised incident context payload
// designed to stay <=400 tokens for local llm compatibility
type IncidentContextResponse struct {
	IncidentID       string           `json:"incident_id"`
	Title            string           `json:"title"`
	Status           string           `json:"status"`
	Age              string           `json:"age"`
	Details          string           `json:"details"`
	Alerts           []CleansedAlert  `json:"alerts"`
	Timeline         []TimelineEntry  `json:"timeline"`
	ExistingRunbooks []RunbookSummary `json:"existing_runbooks"`
}

type CleansedAlert struct {
	Service    string         `json:"service"`
	Severity   string         `json:"severity"`
	Received   string         `json:"received"`
	KeyMetrics map[string]any `json:"key_metrics"`
}

type TimelineEntry struct {
	At   string `json:"at,omitempty"`
	From string `json:"from,omitempty"`
	To   string `json:"to,omitempty"`
	Note string `json:"note"`
}

type RunbookSummary struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
}
