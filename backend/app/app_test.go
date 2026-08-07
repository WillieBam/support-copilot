package app_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/WillieBam/support_copilot/backend/app"
	"github.com/WillieBam/support_copilot/backend/internal/mocks"
)

var _ = Describe("App Component Initialization", func() {
	Context("App Container", func() {
		It("should initialize app struct with mock services correctly", func() {
			mockLLM := &mocks.ILLMClient{}
			mockUser := &mocks.IUserRepository{}
			mockAlert := &mocks.IAlertRepository{}
			mockTeam := &mocks.ITeamRepository{}
			mockConv := &mocks.IConversationRepository{}
			mockAppSvc := &mocks.IAppService{}
			mockAuthSvc := &mocks.IAuthService{}
			mockTeamSvc := &mocks.ITeamService{}
			mockUserSvc := &mocks.IUserService{}
			mockDashSvc := &mocks.IDashboardService{}

			appRepo := app.NewAppRepository(mockLLM, mockUser, mockAlert, mockTeam, mockConv)

			application := &app.App{
				Repository:       appRepo,
				Service:          mockAppSvc,
				AuthService:      mockAuthSvc,
				TeamService:      mockTeamSvc,
				UserService:      mockUserSvc,
				DashboardService: mockDashSvc,
			}

			Expect(application).NotTo(BeNil())
			Expect(application.Repository).To(Equal(appRepo))
			Expect(application.Service).To(Equal(mockAppSvc))
			Expect(application.AuthService).To(Equal(mockAuthSvc))
			Expect(application.TeamService).To(Equal(mockTeamSvc))
			Expect(application.UserService).To(Equal(mockUserSvc))
			Expect(application.DashboardService).To(Equal(mockDashSvc))
		})
	})
})
