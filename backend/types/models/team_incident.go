package models

import (
	"time"

	"github.com/google/uuid"
)

type TeamIncident struct {
	ID         uuid.UUID               `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	IncidentID uuid.UUID               `gorm:"type:uuid;not null" json:"incident_id"`
	TeamID     uuid.UUID               `gorm:"type:uuid;not null" json:"team_id"`
	AssignedBy uuid.UUID               `gorm:"type:uuid;not null" json:"assigned_by"`
	Title      string                  `gorm:"type:varchar(255);not null" json:"title"`
	Status     string                  `gorm:"type:varchar(20)" json:"status"`
	Details    string                  `gorm:"type:text" json:"details"`
	AssignedAt time.Time               `gorm:"type:timestamp(3);default:CURRENT_TIMESTAMP" json:"assigned_at"`
	History    []IncidentStatusHistory `gorm:"foreignKey:TeamIncidentID" json:"history,omitempty"`
}
