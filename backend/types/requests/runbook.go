package requests

type CreateRunbookRequest struct {
	IncidentID string `json:"incident_id"`
	Title      string `json:"title"`
	Content    string `json:"content"`
}

type UpdateRunbookRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}
