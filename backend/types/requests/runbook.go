package requests

import "github.com/google/uuid"

type CreateRunbookRequest struct {
	IncidentID uuid.UUID `json:"incident_id"`
	Title      string    `json:"title"`
	Content    string    `json:"content"`
}

type UpdateRunbookRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}
