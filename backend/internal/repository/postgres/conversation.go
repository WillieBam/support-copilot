package postgres

import (
	"context"
	"time"

	"github.com/WillieBam/support_copilot/backend/internal/interfaces"
	"github.com/WillieBam/support_copilot/backend/types"
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
	var rows []types.ConversationWithMessagesRow
	rawSQL := `SELECT 
		c.id AS conv_id, c.team_id, c.team_incident_id, c.user_id, c.title, c.created_at AS conv_created_at,
		u.email AS user_email, u.display_name AS user_display_name, u.scope AS user_scope,
		m.id AS message_id, m.parent_message_id, m.sender AS message_sender, m.content AS message_content, m.created_at AS message_created_at
	FROM conversations c
	LEFT JOIN users u ON u.id = c.user_id
	LEFT JOIN messages m ON m.conversation_id = c.id
	WHERE c.id = ?
	ORDER BY m.created_at ASC`

	if err := r.db.WithContext(ctx).Raw(rawSQL, id).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	first := rows[0]
	var user *models.User
	if first.UserEmail != nil {
		displayName := ""
		if first.UserDisplayName != nil {
			displayName = *first.UserDisplayName
		}
		scope := ""
		if first.UserScope != nil {
			scope = *first.UserScope
		}
		user = &models.User{
			ID:          first.UserID,
			Email:       *first.UserEmail,
			DisplayName: displayName,
			Scope:       scope,
		}
	}

	conv := &models.Conversation{
		ID:             first.ConvID,
		TeamID:         first.TeamID,
		TeamIncidentID: first.TeamIncidentID,
		UserID:         first.UserID,
		User:           user,
		Title:          first.Title,
		CreatedAt:      first.ConvCreatedAt,
		Messages:       []models.Message{},
	}

	for _, row := range rows {
		if row.MessageID != nil {
			sender := ""
			if row.MessageSender != nil {
				sender = *row.MessageSender
			}
			content := ""
			if row.MessageContent != nil {
				content = *row.MessageContent
			}
			createdAt := time.Time{}
			if row.MessageCreatedAt != nil {
				createdAt = *row.MessageCreatedAt
			}
			conv.Messages = append(conv.Messages, models.Message{
				ID:              *row.MessageID,
				ConversationID:  first.ConvID,
				ParentMessageID: row.ParentMessageID,
				Sender:          sender,
				Content:         content,
				CreatedAt:       createdAt,
			})
		}
	}

	return conv, nil
}

func (r *conversationRepository) ListTeamConversations(ctx context.Context, teamID uuid.UUID, limit int) ([]models.Conversation, error) {
	var rows []types.ConversationWithUserRow
	var rawSQL string
	var args []interface{}

	if limit > 0 {
		rawSQL = `SELECT 
			c.id, c.team_id, c.team_incident_id, c.user_id, c.title, c.created_at,
			u.email AS user_email, u.display_name AS user_display_name, u.scope AS user_scope
		FROM conversations c
		LEFT JOIN users u ON u.id = c.user_id
		WHERE c.team_id = ?
		ORDER BY c.created_at DESC
		LIMIT ?`
		args = append(args, teamID, limit)
	} else {
		rawSQL = `SELECT 
			c.id, c.team_id, c.team_incident_id, c.user_id, c.title, c.created_at,
			u.email AS user_email, u.display_name AS user_display_name, u.scope AS user_scope
		FROM conversations c
		LEFT JOIN users u ON u.id = c.user_id
		WHERE c.team_id = ?
		ORDER BY c.created_at DESC`
		args = append(args, teamID)
	}

	if err := r.db.WithContext(ctx).Raw(rawSQL, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}

	convs := make([]models.Conversation, len(rows))
	for i, row := range rows {
		var user *models.User
		if row.UserEmail != nil {
			displayName := ""
			if row.UserDisplayName != nil {
				displayName = *row.UserDisplayName
			}
			scope := ""
			if row.UserScope != nil {
				scope = *row.UserScope
			}
			user = &models.User{
				ID:          row.UserID,
				Email:       *row.UserEmail,
				DisplayName: displayName,
				Scope:       scope,
			}
		}

		convs[i] = models.Conversation{
			ID:             row.ID,
			TeamID:         row.TeamID,
			TeamIncidentID: row.TeamIncidentID,
			UserID:         row.UserID,
			User:           user,
			Title:          row.Title,
			CreatedAt:      row.CreatedAt,
		}
	}
	return convs, nil
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
