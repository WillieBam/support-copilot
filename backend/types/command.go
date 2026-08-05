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
