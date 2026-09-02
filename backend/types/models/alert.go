package models

import (
	"time"

	"github.com/google/uuid"
)

// Alert is gorm model for storing alert records
type Alert struct {
	ID              uuid.UUID      `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	IncidentID      *uuid.UUID     `gorm:"type:uuid;default:null" json:"incident_id,omitempty"`
	ReceivedAt      time.Time      `gorm:"type:timestamp(0);default:CURRENT_TIMESTAMP" json:"received_at"`
	AlertInfo       string         `gorm:"type:text" json:"alert_info"`
	ResourceInfo    string         `gorm:"type:text" json:"resource_info"`
	Metrics         string         `gorm:"type:text" json:"metrics,omitempty"`
	BusinessContext string         `gorm:"type:text" json:"business_context,omitempty"`
	Metadata        string         `gorm:"type:text" json:"metadata,omitempty"`
	Incident        *TeamIncident  `gorm:"foreignKey:IncidentID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"-"`
	Incidents       []TeamIncident `gorm:"many2many:alert_incidents;foreignKey:ID;joinForeignKey:AlertID;references:ID;joinReferences:IncidentID" json:"incidents,omitempty"`
}


