package interfaces

import (
	"context"

	"github.com/WillieBam/support_copilot/backend/types/models"
	"github.com/google/uuid"
)

// iconversationrepository defines database operations for conversations and messages
type IConversationRepository interface {
	CreateConversation(ctx context.Context, conv *models.Conversation) error
	GetConversationByID(ctx context.Context, id uuid.UUID) (*models.Conversation, error)
	ListTeamConversations(ctx context.Context, teamID uuid.UUID, limit int) ([]models.Conversation, error)
	UpdateConversationTitle(ctx context.Context, id uuid.UUID, title string) error
	CreateMessage(ctx context.Context, msg *models.Message) error
	ListMessagesByConversation(ctx context.Context, convID uuid.UUID) ([]models.Message, error)
}
