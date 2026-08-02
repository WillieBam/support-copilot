package models

import (
	"time"

	"github.com/google/uuid"
)

type Alert struct {
	ID          uuid.UUID  `gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	IncidentID  *uuid.UUID `gorm:"type:uuid;default:null"`
	ReceivedAt  time.Time  `gorm:"type:timestamp(0);default:CURRENT_TIMESTAMP"`
	ServiceName string     `gorm:"type:varchar(255);not null"`
	Severity    string     `gorm:"type:varchar(20)"`
	Metrics     string     `gorm:"type:text"`
}
