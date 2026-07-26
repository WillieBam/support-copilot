package postgres

import (
	"context"

	"github.com/WillieBam/support_copilot/backend/internal/interfaces"
	"github.com/WillieBam/support_copilot/backend/types/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type conversationRepository struct {
	db *gorm.DB
}

// newconversationrepository creates a new postgres conversation repository
func NewConversationRepository(db *gorm.DB) interfaces.IConversationRepository {
	return &conversationRepository{db: db}
}

func (r *conversationRepository) CreateConversation(ctx context.Context, conv *models.Conversation) error {
	return r.db.WithContext(ctx).Create(conv).Error
}

func (r *conversationRepository) GetConversationByID(ctx context.Context, id uuid.UUID) (*models.Conversation, error) {
	var conv models.Conversation
	err := r.db.WithContext(ctx).Preload("User").Preload("Messages", func(db *gorm.DB) *gorm.DB {
		return db.Order("created_at ASC")
	}).Where("id = ?", id).First(&conv).Error
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

func (r *conversationRepository) ListTeamConversations(ctx context.Context, teamID uuid.UUID, limit int) ([]models.Conversation, error) {
	var convs []models.Conversation
	query := r.db.WithContext(ctx).Preload("User").Where("team_id = ?", teamID).Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&convs).Error
	return convs, err
}

func (r *conversationRepository) UpdateConversationTitle(ctx context.Context, id uuid.UUID, title string) error {
	return r.db.WithContext(ctx).Model(&models.Conversation{}).Where("id = ?", id).Update("title", title).Error
}

func (r *conversationRepository) CreateMessage(ctx context.Context, msg *models.Message) error {
	return r.db.WithContext(ctx).Create(msg).Error
}

func (r *conversationRepository) ListMessagesByConversation(ctx context.Context, convID uuid.UUID) ([]models.Message, error) {
	var msgs []models.Message
	err := r.db.WithContext(ctx).Where("conversation_id = ?", convID).Order("created_at ASC").Find(&msgs).Error
	return msgs, err
}
