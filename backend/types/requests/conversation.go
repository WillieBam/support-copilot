package requests

import (
	"github.com/WillieBam/support_copilot/backend/types"
	"github.com/google/uuid"
)

// createconversationrequest holds request payload to initialize a chat conversation
type CreateConversationRequest struct {
	TeamID uuid.UUID `json:"team_id"`
}

// chatqueryrequest holds request payload for query stream endpoint
type ChatQueryRequest struct {
	Input          string                 `json:"input"`
	History        []types.HistoryMessage `json:"history"`
	ConversationID *uuid.UUID             `json:"conversation_id,omitempty"`
	TeamID         *uuid.UUID             `json:"team_id,omitempty"`
}
