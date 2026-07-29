package models

import (
	"time"

	"github.com/google/uuid"
)

type InstructionLog struct {
	ID               uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	InstructionID    uuid.UUID `gorm:"type:uuid;not null" json:"instruction_id"`
	UpdatedBy        uuid.UUID `gorm:"type:uuid;not null" json:"updated_by"`
	OlderInstruction string    `gorm:"type:text" json:"older_instruction"`
	Version          int       `gorm:"type:integer" json:"version"`
	UpdatedAt        time.Time `gorm:"type:timestamp(0);default:CURRENT_TIMESTAMP" json:"updated_at"`
}
