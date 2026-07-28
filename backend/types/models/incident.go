package models

import (
	"time"

	"github.com/google/uuid"
)

type Incident struct {
	ID        uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	AlertID   uuid.UUID `gorm:"type:uuid;not null" json:"alert_id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null" json:"user_id"`
	CreatedAt time.Time `gorm:"type:timestamp(3);default:CURRENT_TIMESTAMP" json:"created_at"`
	Details   string    `gorm:"type:text" json:"details"`
	Status    string    `gorm:"type:varchar(20)" json:"status"`
}