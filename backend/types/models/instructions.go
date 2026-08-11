package models

import (
	"time"

	"github.com/google/uuid"
)

type Instruction struct {
	ID                 uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	CreatedBy          uuid.UUID `gorm:"type:uuid;not null" json:"created_by"`
	TeamID             uuid.UUID `gorm:"type:uuid;not null" json:"team_id"`
	InstructionDetails string    `gorm:"type:text" json:"instruction_details"`
	CreatedAt          time.Time `gorm:"type:timestamp(0);default:CURRENT_TIMESTAMP" json:"created_at"`
	Team    Team `gorm:"foreignKey:TeamID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
	Creator User `gorm:"foreignKey:CreatedBy;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
}
