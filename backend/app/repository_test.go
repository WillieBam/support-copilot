package app_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/WillieBam/support_copilot/backend/app"
	"github.com/WillieBam/support_copilot/backend/internal/mocks"
)

var _ = Describe("AppRepository", func() {
	It("should construct AppRepository correctly", func() {
		mockUser := &mocks.IUserRepository{}
		mockAlert := &mocks.IAlertRepository{}
		mockTeam := &mocks.ITeamRepository{}
		mockLLM := &mocks.ILLMClient{}
		mockConv := &mocks.IConversationRepository{}

		appRepo := app.NewAppRepository(mockLLM, mockUser, mockAlert, mockTeam, mockConv)
		Expect(appRepo).NotTo(BeNil())
		Expect(appRepo.User).To(Equal(mockUser))
		Expect(appRepo.Alert).To(Equal(mockAlert))
		Expect(appRepo.Team).To(Equal(mockTeam))
		Expect(appRepo.LLM).To(Equal(mockLLM))
		Expect(appRepo.Conversation).To(Equal(mockConv))
	})
})
