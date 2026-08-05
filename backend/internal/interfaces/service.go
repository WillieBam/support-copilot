package interfaces

import (
	"context"

	"github.com/WillieBam/support_copilot/backend/types"
	"github.com/WillieBam/support_copilot/backend/types/models"
	"github.com/google/uuid"
)

type IAppService interface {
	QueryStreamWithTools(ctx context.Context, prompt string, history []types.HistoryMessage, streamChan chan<- types.StreamEvent, opts ...interface{}) error
	CreateConversation(ctx context.Context, teamID, userID uuid.UUID) (*models.Conversation, error)
	GetConversationByID(ctx context.Context, id uuid.UUID) (*models.Conversation, error)
	ListTeamConversations(ctx context.Context, teamID uuid.UUID, limit int) ([]models.Conversation, error)
	SaveMessage(ctx context.Context, convID uuid.UUID, sender, content, reasoning string) (*models.Message, error)
	ListMessagesByConversation(ctx context.Context, convID uuid.UUID) ([]models.Message, error)
	GenerateAndSaveTitle(ctx context.Context, convID uuid.UUID, userPrompt, assistantReply string) (string, error)
	IngestAlert(ctx context.Context, incidentID *uuid.UUID, serviceName, severity, metrics string) error
	Intercept(ctx context.Context, prompt string) (*types.CommandResult, error)
}
