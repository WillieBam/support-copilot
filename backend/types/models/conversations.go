package models

import (
	"time"

	"github.com/google/uuid"
)

type Conversation struct {
	ID             uuid.UUID  `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	TeamID         uuid.UUID  `gorm:"type:uuid;not null" json:"team_id"`
	TeamIncidentID *uuid.UUID `gorm:"type:uuid" json:"team_incident_id,omitempty"`
	UserID         uuid.UUID  `gorm:"type:uuid;not null" json:"user_id"`
	User           *User         `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"user,omitempty"`
	Title          string        `gorm:"type:varchar(255)" json:"title"`
	CreatedAt      time.Time     `gorm:"type:timestamp(0);default:CURRENT_TIMESTAMP" json:"created_at"`
	Messages       []Message     `gorm:"foreignKey:ConversationID;constraint:OnDelete:CASCADE" json:"messages,omitempty"`
	Team           Team          `gorm:"foreignKey:TeamID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
	TeamIncident   *TeamIncident `gorm:"foreignKey:TeamIncidentID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"-"`
}

