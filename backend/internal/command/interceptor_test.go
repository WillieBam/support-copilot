package command_test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	"github.com/WillieBam/support_copilot/backend/internal/command"
	"github.com/WillieBam/support_copilot/backend/internal/interfaces"
	"github.com/WillieBam/support_copilot/backend/internal/mocks"
	"github.com/WillieBam/support_copilot/backend/types"
	"github.com/google/uuid"
)

var _ = Describe("CommandInterceptor", func() {
	var (
		interceptor      interfaces.ICommandInterceptor
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
		interceptor = command.NewCommandInterceptor(mockOrchestrator)
	})

	AfterEach(func() {
		mockOrchestrator.AssertExpectations(GinkgoT())
	})

	Context("Intercept /quit", func() {
		It("should intercept prompt starting with /quit", func() {
			res, err := interceptor.Intercept(ctx, "/quit")
			Expect(err).NotTo(HaveOccurred())
			Expect(res).NotTo(BeNil())
			Expect(res.Handled).To(BeTrue())
			Expect(res.Message).To(ContainSubstring("stopped by /quit command"))
		})

		It("should intercept prompt with /quit and additional whitespace or text", func() {
			res, err := interceptor.Intercept(ctx, "  /QUIT now  ")
			Expect(err).NotTo(HaveOccurred())
			Expect(res).NotTo(BeNil())
			Expect(res.Handled).To(BeTrue())
		})

		It("should not intercept normal prompt", func() {
			res, err := interceptor.Intercept(ctx, "What is the system status?")
			Expect(err).NotTo(HaveOccurred())
			Expect(res).NotTo(BeNil())
			Expect(res.Handled).To(BeFalse())
			Expect(res.Message).To(BeEmpty())
		})
	})

	Context("Intercept /incident", func() {
		It("should fetch active incident details when no argument provided and active incident exists", func() {
			incCtx := command.WithActiveIncidentID(ctx, incidentID)
			rawArgs := fmt.Sprintf(`{"incident_id": "%s"}`, incidentID.String())
			mockOrchestrator.On("ExecuteGetIncidentRaw", mock.Anything, rawArgs).Return(`{"id":"`+incidentID.String()+`","title":"Redis latency spike"}`, nil)

			res, err := interceptor.Intercept(incCtx, "/incident")
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Handled).To(BeTrue())
			Expect(res.Message).To(ContainSubstring("Redis latency spike"))
		})

		It("should return message when no argument provided and no active incident exists", func() {
			res, err := interceptor.Intercept(ctx, "/incident")
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Handled).To(BeTrue())
			Expect(res.Message).To(ContainSubstring("no active incident found in session context"))
		})

		It("should search team incidents when argument provided and team context exists", func() {
			teamCtx := command.WithTeamID(ctx, teamID)
			rawArgs := fmt.Sprintf(`{"team_id": "%s"}`, teamID.String())
			mockOrchestrator.On("ExecuteListIncidentsRaw", mock.Anything, rawArgs).Return(`[{"id":"inc-1","title":"Redis latency","status":"OPEN"}]`, nil)

			res, err := interceptor.Intercept(teamCtx, "/incident redis")
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Handled).To(BeTrue())
			Expect(res.Message).To(ContainSubstring("Redis latency"))
		})

		It("should return message when argument provided but team context missing", func() {
			res, err := interceptor.Intercept(ctx, "/incident redis")
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Handled).To(BeTrue())
			Expect(res.Message).To(ContainSubstring("no active team context"))
		})
	})

	Context("Intercept /runbook", func() {
		It("should list active runbooks when no argument provided", func() {
			teamCtx := command.WithTeamID(ctx, teamID)
			rawArgs := fmt.Sprintf(`{"team_id": "%s", "status": "active"}`, teamID.String())
			mockOrchestrator.On("ExecuteListRunbooksRaw", mock.Anything, rawArgs).Return(`[{"id":"rb-1","title":"DB Failover","status":"active"}]`, nil)

			res, err := interceptor.Intercept(teamCtx, "/runbook")
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Handled).To(BeTrue())
			Expect(res.Message).To(ContainSubstring("DB Failover"))
		})

		It("should filter runbooks when query argument provided", func() {
			teamCtx := command.WithTeamID(ctx, teamID)
			rawArgs := fmt.Sprintf(`{"team_id": "%s", "status": "active"}`, teamID.String())
			mockOrchestrator.On("ExecuteListRunbooksRaw", mock.Anything, rawArgs).Return(`[{"id":"rb-1","title":"Database Connection Guide","status":"active","content":"restart postgres"}]`, nil)

			res, err := interceptor.Intercept(teamCtx, "/runbook database")
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Handled).To(BeTrue())
			Expect(res.Message).To(ContainSubstring("Database Connection Guide"))
		})

		It("should return no match message when no runbooks match query", func() {
			teamCtx := command.WithTeamID(ctx, teamID)
			rawArgs := fmt.Sprintf(`{"team_id": "%s", "status": "active"}`, teamID.String())
			mockOrchestrator.On("ExecuteListRunbooksRaw", mock.Anything, rawArgs).Return(`[{"id":"rb-1","title":"Network guide","status":"active","content":"check switch"}]`, nil)

			res, err := interceptor.Intercept(teamCtx, "/runbook database")
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
