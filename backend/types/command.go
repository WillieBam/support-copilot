package types

type CommandResult struct {
	Handled bool   `json:"handled"`
	Message string `json:"message"`
}

type ContextKey string

const (
	TeamIDContextKey           ContextKey = "team_id"
	ActiveIncidentIDContextKey ContextKey = "active_incident_id"
)

type IncidentRecord struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Status    string `json:"status"`
	Summary   string `json:"summary,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
}

type RunbookRecord struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Status    string `json:"status"`
	Content   string `json:"content,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

type AlertRecord struct {
	ID          string `json:"id"`
	ServiceName string `json:"service_name"`
	Severity    string `json:"severity"`
	ReceivedAt  string `json:"received_at"`
	IncidentID  string `json:"incident_id,omitempty"`
}
