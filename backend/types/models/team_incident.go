package models

import (
	"time"

	"github.com/google/uuid"
)

type TeamIncident struct {
	ID         uuid.UUID               `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	TeamID     uuid.UUID               `gorm:"type:uuid;not null" json:"team_id"`
	AssignedBy uuid.UUID               `gorm:"type:uuid;not null" json:"assigned_by"`
	Title      string                  `gorm:"type:varchar(255);not null" json:"title"`
	Status     string                  `gorm:"type:varchar(20)" json:"status"`
	Details    string                  `gorm:"type:text" json:"details"`
	CreatedAt  time.Time               `gorm:"type:timestamp(3);default:CURRENT_TIMESTAMP" json:"created_at"`
	AssignedAt time.Time               `gorm:"type:timestamp(3);default:CURRENT_TIMESTAMP" json:"assigned_at"`
	// resolved_at is only set when an incident transitions to RESOLVED status
	ResolvedAt *time.Time              `gorm:"type:timestamp(3)" json:"resolved_at,omitempty"`
	History    []IncidentStatusHistory `gorm:"foreignKey:TeamIncidentID" json:"history,omitempty"`
}
