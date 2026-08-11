package models

import (
	"time"

	"github.com/google/uuid"
)

type Runbook struct {
	ID         uuid.UUID     `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	IncidentID *uuid.UUID    `gorm:"type:uuid;default:null" json:"incident_id"`
	TeamID     uuid.UUID     `gorm:"type:uuid;not null" json:"team_id"`
	CreatedBy  uuid.UUID     `gorm:"type:uuid;not null" json:"created_by"`
	Title      string        `gorm:"type:varchar(255);not null" json:"title"`
	Status     string        `gorm:"type:varchar(20);default:'active'" json:"status"` // active | deprecated
	CreatedAt  time.Time     `gorm:"type:timestamp(0);default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt  time.Time     `gorm:"type:timestamp(0);autoUpdateTime" json:"updated_at"`
	Content    string        `gorm:"type:text" json:"content"`
	Incident   *TeamIncident `gorm:"foreignKey:IncidentID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
	Team       Team          `gorm:"foreignKey:TeamID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
	Creator    User          `gorm:"foreignKey:CreatedBy;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
}