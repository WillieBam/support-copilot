// code generated for IConversationRepository mock

package mocks

import (
	context "context"

	models "github.com/WillieBam/support_copilot/backend/types/models"
	uuid "github.com/google/uuid"
	mock "github.com/stretchr/testify/mock"
)

type IConversationRepository struct {
	mock.Mock
}

func (_m *IConversationRepository) CreateConversation(ctx context.Context, conv *models.Conversation) error {
	ret := _m.Called(ctx, conv)
	return ret.Error(0)
}

func (_m *IConversationRepository) GetConversationByID(ctx context.Context, id uuid.UUID) (*models.Conversation, error) {
	ret := _m.Called(ctx, id)
	if ret.Get(0) == nil {
		return nil, ret.Error(1)
	}
	return ret.Get(0).(*models.Conversation), ret.Error(1)
}

func (_m *IConversationRepository) ListTeamConversations(ctx context.Context, teamID uuid.UUID, limit int) ([]models.Conversation, error) {
	ret := _m.Called(ctx, teamID, limit)
	if ret.Get(0) == nil {
		return nil, ret.Error(1)
	}
	return ret.Get(0).([]models.Conversation), ret.Error(1)
}

func (_m *IConversationRepository) UpdateConversationTitle(ctx context.Context, id uuid.UUID, title string) error {
	ret := _m.Called(ctx, id, title)
	return ret.Error(0)
}

func (_m *IConversationRepository) CreateMessage(ctx context.Context, msg *models.Message) error {
	ret := _m.Called(ctx, msg)
	return ret.Error(0)
}

func (_m *IConversationRepository) ListMessagesByConversation(ctx context.Context, convID uuid.UUID) ([]models.Message, error) {
	ret := _m.Called(ctx, convID)
	if ret.Get(0) == nil {
		return nil, ret.Error(1)
	}
	return ret.Get(0).([]models.Message), ret.Error(1)
}
