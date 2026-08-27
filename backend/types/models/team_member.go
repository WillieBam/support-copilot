package models

import (
	"github.com/google/uuid"
)

type TeamMember struct {
	ID     uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	TeamID uuid.UUID `gorm:"type:uuid;not null;index;uniqueIndex:idx_user_team" json:"team_id"`
	UserID uuid.UUID `gorm:"type:uuid;not null;index;uniqueIndex:idx_user_team" json:"user_id"`
	Role   string    `gorm:"type:varchar(20);not null;default:'member'" json:"role"`

	Team *Team `gorm:"foreignKey:TeamID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"team,omitempty"`
	User *User `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"user,omitempty"`
}
