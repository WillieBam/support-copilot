package app

import (
	"github.com/WillieBam/support_copilot/backend/internal/interfaces"
)

type AppRepository struct {
	User         interfaces.IUserRepository
	Alert        interfaces.IAlertRepository
	Team         interfaces.ITeamRepository
	LLM          interfaces.IOllamaClient
	Conversation interfaces.IConversationRepository
}

func NewAppRepository(ollama interfaces.IOllamaClient, user interfaces.IUserRepository, alert interfaces.IAlertRepository, team interfaces.ITeamRepository, conv interfaces.IConversationRepository) *AppRepository {
	return &AppRepository{
		User:         user,
		Alert:        alert,
		Team:         team,
		LLM:          ollama,
		Conversation: conv,
	}
}
