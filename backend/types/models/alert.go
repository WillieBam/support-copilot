package models

import (
	"time"

	"github.com/google/uuid"
)

// Alert is gorm model for storing alert records
type Alert struct {
	ID              uuid.UUID     `gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	IncidentID      *uuid.UUID    `gorm:"type:uuid;default:null"`
	ReceivedAt      time.Time     `gorm:"type:timestamp(0);default:CURRENT_TIMESTAMP"`
	AlertInfo       string        `gorm:"type:text"`
	ResourceInfo    string        `gorm:"type:text"`
	Metrics         string        `gorm:"type:text"`
	BusinessContext string        `gorm:"type:text"`
	Metadata        string        `gorm:"type:text"`
	Incident        *TeamIncident `gorm:"foreignKey:IncidentID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"-"`
}


