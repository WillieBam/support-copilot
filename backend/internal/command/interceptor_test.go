package command_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	"github.com/WillieBam/support_copilot/backend/internal/command"
	"github.com/WillieBam/support_copilot/backend/internal/mocks"
	"github.com/WillieBam/support_copilot/backend/types"
	"github.com/google/uuid"
)

var _ = Describe("CommandInterceptor", func() {
	var (
		mockInterceptor  *mocks.ICommandInterceptor
		mockOrchestrator *mocks.IOrchestratorService
		ctx              context.Context
		teamID           uuid.UUID
		incidentID       uuid.UUID
	)

	BeforeEach(func() {
		ctx = context.Background()
		teamID = uuid.New()
		incidentID = uuid.New()
		mockOrchestrator = &mocks.IOrchestratorService{}
		mockInterceptor = &mocks.ICommandInterceptor{}
	})

	AfterEach(func() {
		mockInterceptor.AssertExpectations(GinkgoT())
		mockOrchestrator.AssertExpectations(GinkgoT())
	})

	Context("Intercept /quit", func() {
		It("should intercept prompt starting with /quit", func() {
			mockInterceptor.On("Intercept", mock.Anything, "/quit").Return(&types.CommandResult{
				Handled: true,
				Message: "stopped by /quit command",
			}, nil)

			res, err := mockInterceptor.Intercept(ctx, "/quit")
			Expect(err).NotTo(HaveOccurred())
			Expect(res).NotTo(BeNil())
			Expect(res.Handled).To(BeTrue())
			Expect(res.Message).To(ContainSubstring("stopped by /quit command"))
		})

		It("should intercept prompt with /quit and additional whitespace or text", func() {
			mockInterceptor.On("Intercept", mock.Anything, "  /QUIT now  ").Return(&types.CommandResult{
				Handled: true,
				Message: "stopped by /quit command",
			}, nil)

			res, err := mockInterceptor.Intercept(ctx, "  /QUIT now  ")
			Expect(err).NotTo(HaveOccurred())
			Expect(res).NotTo(BeNil())
			Expect(res.Handled).To(BeTrue())
		})

		It("should not intercept normal prompt", func() {
			mockInterceptor.On("Intercept", mock.Anything, "What is the system status?").Return(&types.CommandResult{
				Handled: false,
				Message: "",
			}, nil)

			res, err := mockInterceptor.Intercept(ctx, "What is the system status?")
			Expect(err).NotTo(HaveOccurred())
			Expect(res).NotTo(BeNil())
			Expect(res.Handled).To(BeFalse())
			Expect(res.Message).To(BeEmpty())
		})
	})

	Context("Intercept /incident", func() {
		It("should fetch active incident details when no argument provided and active incident exists", func() {
			incCtx := command.WithActiveIncidentID(ctx, incidentID)
			mockInterceptor.On("Intercept", mock.Anything, "/incident").Return(&types.CommandResult{
				Handled: true,
				Message: "Redis latency spike",
			}, nil)

			res, err := mockInterceptor.Intercept(incCtx, "/incident")
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Handled).To(BeTrue())
			Expect(res.Message).To(ContainSubstring("Redis latency spike"))
		})

		It("should return message when no argument provided and no active incident exists", func() {
			mockInterceptor.On("Intercept", mock.Anything, "/incident").Return(&types.CommandResult{
				Handled: true,
				Message: "no active incident found in session context",
			}, nil)

			res, err := mockInterceptor.Intercept(ctx, "/incident")
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Handled).To(BeTrue())
			Expect(res.Message).To(ContainSubstring("no active incident found in session context"))
		})

		It("should search team incidents when argument provided and team context exists", func() {
			teamCtx := command.WithTeamID(ctx, teamID)
			mockInterceptor.On("Intercept", mock.Anything, "/incident redis").Return(&types.CommandResult{
				Handled: true,
				Message: "Redis latency",
			}, nil)

			res, err := mockInterceptor.Intercept(teamCtx, "/incident redis")
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Handled).To(BeTrue())
			Expect(res.Message).To(ContainSubstring("Redis latency"))
		})

		It("should return message when argument provided but team context missing", func() {
			mockInterceptor.On("Intercept", mock.Anything, "/incident redis").Return(&types.CommandResult{
				Handled: true,
				Message: "no active team context",
			}, nil)

			res, err := mockInterceptor.Intercept(ctx, "/incident redis")
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Handled).To(BeTrue())
			Expect(res.Message).To(ContainSubstring("no active team context"))
		})
	})

	Context("Intercept /runbook", func() {
		It("should list active runbooks when no argument provided", func() {
			teamCtx := command.WithTeamID(ctx, teamID)
			mockInterceptor.On("Intercept", mock.Anything, "/runbook").Return(&types.CommandResult{
				Handled: true,
				Message: "DB Failover",
			}, nil)

			res, err := mockInterceptor.Intercept(teamCtx, "/runbook")
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Handled).To(BeTrue())
			Expect(res.Message).To(ContainSubstring("DB Failover"))
		})

		It("should filter runbooks when query argument provided", func() {
			teamCtx := command.WithTeamID(ctx, teamID)
			mockInterceptor.On("Intercept", mock.Anything, "/runbook database").Return(&types.CommandResult{
				Handled: true,
				Message: "Database Connection Guide",
			}, nil)

			res, err := mockInterceptor.Intercept(teamCtx, "/runbook database")
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Handled).To(BeTrue())
			Expect(res.Message).To(ContainSubstring("Database Connection Guide"))
		})

		It("should return no match message when no runbooks match query", func() {
			teamCtx := command.WithTeamID(ctx, teamID)
			mockInterceptor.On("Intercept", mock.Anything, "/runbook database").Return(&types.CommandResult{
				Handled: true,
				Message: "no runbooks matched query",
			}, nil)

			res, err := mockInterceptor.Intercept(teamCtx, "/runbook database")
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Handled).To(BeTrue())
			Expect(res.Message).To(ContainSubstring("no runbooks matched query"))
		})
	})

	Context("Register custom command", func() {
		It("should allow registering custom slash command handlers", func() {
			// use concrete type to access registercommand
			ci := command.NewCommandInterceptor().(*command.CommandInterceptor)
			ci.RegisterCommand("/ping", func(ctx context.Context, prompt string) (*types.CommandResult, error) {
				return &types.CommandResult{
					Handled: true,
					Message: "pong",
				}, nil
			})

			res, err := ci.Intercept(ctx, "/ping")
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Handled).To(BeTrue())
			Expect(res.Message).To(Equal("pong"))
		})
	})
})
