package models

import (
	"time"

	"github.com/google/uuid"
)

// AlertIncident represents the join table supporting 1-to-N and N-to-M alert-incident relationships
type AlertIncident struct {
	AlertID    uuid.UUID    `gorm:"type:uuid;primaryKey" json:"alert_id"`
	IncidentID uuid.UUID    `gorm:"type:uuid;primaryKey" json:"incident_id"`
	CreatedAt  time.Time    `gorm:"type:timestamp(3);default:CURRENT_TIMESTAMP" json:"created_at"`
	LinkedBy   string       `gorm:"type:varchar(50);default:'human_ui'" json:"linked_by"`
	Alert      Alert        `gorm:"foreignKey:AlertID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
	Incident   TeamIncident `gorm:"foreignKey:IncidentID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
}
