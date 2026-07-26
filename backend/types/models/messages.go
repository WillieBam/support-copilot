package models

import (
	"time"

	"github.com/google/uuid"
)

type Message struct {
	ID              uuid.UUID  `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	ConversationID  uuid.UUID  `gorm:"type:uuid;not null" json:"conversation_id"`
	ParentMessageID *uuid.UUID `gorm:"type:uuid" json:"parent_message_id,omitempty"`
	Sender          string     `gorm:"type:varchar(20)" json:"sender"`
	Content         string     `gorm:"type:text" json:"content"`
	ReasoningSteps  string     `gorm:"type:text" json:"reasoning_steps,omitempty"`
	CreatedAt       time.Time  `gorm:"type:timestamp(0);default:CURRENT_TIMESTAMP" json:"created_at"`
}

