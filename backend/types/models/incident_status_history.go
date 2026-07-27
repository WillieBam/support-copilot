package models

import (
	"time"

	"github.com/google/uuid"
)

type IncidentStatusHistory struct {
	ID             uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	TeamIncidentID uuid.UUID `gorm:"type:uuid;not null" json:"team_incident_id"`
	UpdatedBy      uuid.UUID `gorm:"type:uuid;not null" json:"updated_by"`
	Title          string    `gorm:"type:varchar(255);not null" json:"title"`
	NewStatus      string    `gorm:"type:varchar(20)" json:"new_status"`
	PreviousStatus string    `gorm:"type:varchar(20)" json:"previous_status"`
	Details        string    `gorm:"type:text" json:"details"`
	UpdatedAt      time.Time `gorm:"type:timestamp(3);default:CURRENT_TIMESTAMP" json:"updated_at"`
}
