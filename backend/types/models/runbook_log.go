package models

import (
	"time"

	"github.com/google/uuid"
)

// RunbookLog records a versioned snapshot of runbook edits
type RunbookLog struct {
	ID           uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	RunbookID    uuid.UUID `gorm:"type:uuid;not null;index" json:"runbook_id"`
	IncidentID   uuid.UUID `gorm:"type:uuid;not null" json:"incident_id"`
	TeamID       uuid.UUID `gorm:"type:uuid;not null" json:"team_id"`
	UpdatedBy    uuid.UUID `gorm:"type:uuid;not null" json:"updated_by"`
	OlderTitle   string    `gorm:"type:varchar(255)" json:"older_title"`
	OlderContent string    `gorm:"type:text" json:"older_content"`
	Version      int       `gorm:"type:integer;not null" json:"version"`
	UpdatedAt    time.Time `gorm:"type:timestamp(0);default:CURRENT_TIMESTAMP" json:"updated_at"`
	Runbook  Runbook      `gorm:"foreignKey:RunbookID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
	Incident TeamIncident `gorm:"foreignKey:IncidentID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
	Team     Team         `gorm:"foreignKey:TeamID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
	User     User         `gorm:"foreignKey:UpdatedBy;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
}
