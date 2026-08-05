package app

import (
	"github.com/WillieBam/support_copilot/backend/internal/interfaces"
)

type AppRepository struct {
	User         interfaces.IUserRepository
	Alert        interfaces.IAlertRepository
	Team         interfaces.ITeamRepository
	LLM          interfaces.ILLMClient
	Conversation interfaces.IConversationRepository
}

func NewAppRepository(llmClient interfaces.ILLMClient, user interfaces.IUserRepository, alert interfaces.IAlertRepository, team interfaces.ITeamRepository, conv interfaces.IConversationRepository) *AppRepository {
	return &AppRepository{
		User:         user,
		Alert:        alert,
		Team:         team,
		LLM:          llmClient,
		Conversation: conv,
	}
}
