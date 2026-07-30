package models

import "github.com/google/uuid"

type RunbookLog struct {
	ID         uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	IncidentID uuid.UUID `gorm:"type:uuid;not null" json:"incident_id"`
	TeamID     uuid.UUID `gorm:"type:uuid;not null" json:"team_id"`
	RunbokID   uuid.UUID `gorm:"type:uuid;not null" json:"runbook_id"`
	UpdatedBy  uuid.UUID `gorm:"type:uuid;not null" json:"user_id"` //user
	Version    int       `gorm:"type:integer" json:"version"`
	Content    string    `gorm:"type:text" json:"content"`
}
