package command_test

import (
	"context"
	"errors"
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
		mockOrchestrator *mocks.IOrchestratorService
		ci               interfaces.ICommandInterceptor
		ctx              context.Context
		teamID           uuid.UUID
		incidentID       uuid.UUID
	)

	BeforeEach(func() {
		ctx = context.Background()
		teamID = uuid.New()
		incidentID = uuid.New()
		mockOrchestrator = &mocks.IOrchestratorService{}
		ci = command.NewCommandInterceptor(mockOrchestrator)
	})

	AfterEach(func() {
		mockOrchestrator.AssertExpectations(GinkgoT())
	})

	Context("Intercept /quit", func() {
		It("should intercept prompt starting with /quit", func() {
			res, err := ci.Intercept(ctx, "/quit")
			Expect(err).NotTo(HaveOccurred())
			Expect(res).NotTo(BeNil())
			Expect(res.Handled).To(BeTrue())
			Expect(res.Message).To(ContainSubstring("stopped by /quit command"))
		})

		It("should intercept prompt with /quit and additional whitespace or text", func() {
			res, err := ci.Intercept(ctx, "  /QUIT now  ")
			Expect(err).NotTo(HaveOccurred())
			Expect(res).NotTo(BeNil())
			Expect(res.Handled).To(BeTrue())
			Expect(res.Message).To(ContainSubstring("stopped by /quit command"))
		})

		It("should not intercept normal prompt", func() {
			res, err := ci.Intercept(ctx, "What is the system status?")
			Expect(err).NotTo(HaveOccurred())
			Expect(res).NotTo(BeNil())
			Expect(res.Handled).To(BeFalse())
			Expect(res.Message).To(BeEmpty())
		})
	})

	Context("Intercept /incident", func() {
		It("should return error message when orchestrator is nil", func() {
			nilCi := command.NewCommandInterceptor()
			res, err := nilCi.Intercept(ctx, "/incident")
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Handled).To(BeTrue())
			Expect(res.Message).To(Equal("orchestrator service is unavailable"))
		})

		It("should fetch active incident details when active incident exists in context", func() {
			incCtx := command.WithActiveIncidentID(ctx, incidentID)
			rawArgs := fmt.Sprintf(`{"incident_id": "%s"}`, incidentID.String())
			mockOrchestrator.On("ExecuteGetIncidentRaw", mock.Anything, rawArgs).Return("Incident: Redis Memory Spike", nil)

			res, err := ci.Intercept(incCtx, "/incident")
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Handled).To(BeTrue())
			Expect(res.Message).To(Equal("Incident: Redis Memory Spike"))
		})

		It("should return message when no argument provided and no active incident in context", func() {
			res, err := ci.Intercept(ctx, "/incident")
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Handled).To(BeTrue())
			Expect(res.Message).To(ContainSubstring("no active incident found in session context"))
		})

		It("should return error message when ExecuteGetIncidentRaw fails", func() {
			incCtx := command.WithActiveIncidentID(ctx, incidentID)
			rawArgs := fmt.Sprintf(`{"incident_id": "%s"}`, incidentID.String())
			mockOrchestrator.On("ExecuteGetIncidentRaw", mock.Anything, rawArgs).Return("", errors.New("db error"))

			res, err := ci.Intercept(incCtx, "/incident")
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Handled).To(BeTrue())
			Expect(res.Message).To(ContainSubstring("failed to fetch incident"))
		})

		It("should return message when search query provided but team context is missing", func() {
			res, err := ci.Intercept(ctx, "/incident redis")
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Handled).To(BeTrue())
			Expect(res.Message).To(Equal("no active team context associated with session"))
		})

		It("should search team incidents when query provided and team context exists", func() {
			teamCtx := command.WithTeamID(ctx, teamID)
			rawArgs := fmt.Sprintf(`{"team_id": "%s"}`, teamID.String())
			incidentsJSON := fmt.Sprintf(`[{"id":"%s","title":"Redis Latency","summary":"High latency on redis","status":"OPEN"}]`, incidentID.String())
			mockOrchestrator.On("ExecuteListIncidentsRaw", mock.Anything, rawArgs).Return(incidentsJSON, nil)

			res, err := ci.Intercept(teamCtx, "/incident redis")
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Handled).To(BeTrue())
			Expect(res.Message).To(ContainSubstring("found 1 matching incident(s)"))
		})

		It("should return no match message when query does not match any team incidents", func() {
			teamCtx := command.WithTeamID(ctx, teamID)
			rawArgs := fmt.Sprintf(`{"team_id": "%s"}`, teamID.String())
			incidentsJSON := fmt.Sprintf(`[{"id":"%s","title":"Postgres Latency","summary":"High latency on postgres","status":"OPEN"}]`, incidentID.String())
			mockOrchestrator.On("ExecuteListIncidentsRaw", mock.Anything, rawArgs).Return(incidentsJSON, nil)

			res, err := ci.Intercept(teamCtx, "/incident redis")
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Handled).To(BeTrue())
			Expect(res.Message).To(Equal("no incidents matched query: 'redis'"))
		})

		It("should return error message when ExecuteListIncidentsRaw fails", func() {
			teamCtx := command.WithTeamID(ctx, teamID)
			rawArgs := fmt.Sprintf(`{"team_id": "%s"}`, teamID.String())
			mockOrchestrator.On("ExecuteListIncidentsRaw", mock.Anything, rawArgs).Return("", errors.New("db list error"))

			res, err := ci.Intercept(teamCtx, "/incident redis")
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Handled).To(BeTrue())
			Expect(res.Message).To(ContainSubstring("failed to list incidents"))
		})

		It("should fallback to raw string when /incident response is invalid JSON", func() {
			teamCtx := command.WithTeamID(ctx, teamID)
			rawArgs := fmt.Sprintf(`{"team_id": "%s"}`, teamID.String())
			mockOrchestrator.On("ExecuteListIncidentsRaw", mock.Anything, rawArgs).Return("raw text output", nil)

			res, err := ci.Intercept(teamCtx, "/incident redis")
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Handled).To(BeTrue())
			Expect(res.Message).To(Equal("raw text output"))
		})
	})

	Context("Intercept /runbook", func() {
		It("should return error message when orchestrator is nil", func() {
			nilCi := command.NewCommandInterceptor()
			res, err := nilCi.Intercept(ctx, "/runbook")
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Handled).To(BeTrue())
			Expect(res.Message).To(Equal("Orchestrator service is unavailable."))
		})

		It("should return message when team context is missing", func() {
			res, err := ci.Intercept(ctx, "/runbook")
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Handled).To(BeTrue())
			Expect(res.Message).To(Equal("No active team context associated with this session."))
		})

		It("should format runbook list when no argument provided and team context exists", func() {
			teamCtx := command.WithTeamID(ctx, teamID)
			rawArgs := fmt.Sprintf(`{"team_id": "%s", "status": "active"}`, teamID.String())
			runbooksJSON := `[{"title":"Database Recovery","status":"active"}]`
			mockOrchestrator.On("ExecuteListRunbooksRaw", mock.Anything, rawArgs).Return(runbooksJSON, nil)

			res, err := ci.Intercept(teamCtx, "/runbook")
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Handled).To(BeTrue())
			Expect(res.Message).To(ContainSubstring("found 1 active runbook(s)"))
		})

		It("should return message when active runbook list is empty", func() {
			teamCtx := command.WithTeamID(ctx, teamID)
			rawArgs := fmt.Sprintf(`{"team_id": "%s", "status": "active"}`, teamID.String())
			mockOrchestrator.On("ExecuteListRunbooksRaw", mock.Anything, rawArgs).Return("[]", nil)

			res, err := ci.Intercept(teamCtx, "/runbook")
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Handled).To(BeTrue())
			Expect(res.Message).To(Equal("no active runbooks found for team"))
		})

		It("should return error message when ExecuteListRunbooksRaw fails", func() {
			teamCtx := command.WithTeamID(ctx, teamID)
			rawArgs := fmt.Sprintf(`{"team_id": "%s", "status": "active"}`, teamID.String())
			mockOrchestrator.On("ExecuteListRunbooksRaw", mock.Anything, rawArgs).Return("", errors.New("mcp error"))

			res, err := ci.Intercept(teamCtx, "/runbook")
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Handled).To(BeTrue())
			Expect(res.Message).To(ContainSubstring("Failed to list runbooks"))
		})

		It("should fallback to raw string when /runbook response is invalid JSON", func() {
			teamCtx := command.WithTeamID(ctx, teamID)
			rawArgs := fmt.Sprintf(`{"team_id": "%s", "status": "active"}`, teamID.String())
			mockOrchestrator.On("ExecuteListRunbooksRaw", mock.Anything, rawArgs).Return("raw runbook string", nil)

			res, err := ci.Intercept(teamCtx, "/runbook")
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Handled).To(BeTrue())
			Expect(res.Message).To(Equal("raw runbook string"))
		})

		It("should search runbooks with keyword query matching title or content", func() {
			teamCtx := command.WithTeamID(ctx, teamID)
			rawArgs := fmt.Sprintf(`{"team_id": "%s", "status": "active"}`, teamID.String())
			runbooksJSON := `[{"title":"Database Failover Guide","content":"Steps for postgres failover","status":"active"}]`
			mockOrchestrator.On("ExecuteListRunbooksRaw", mock.Anything, rawArgs).Return(runbooksJSON, nil)

			res, err := ci.Intercept(teamCtx, "/runbook postgres failover")
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Handled).To(BeTrue())
			Expect(res.Message).To(ContainSubstring("found 1 matching runbook(s)"))
		})

		It("should return no match message when query does not match any runbooks", func() {
			teamCtx := command.WithTeamID(ctx, teamID)
			rawArgs := fmt.Sprintf(`{"team_id": "%s", "status": "active"}`, teamID.String())
			runbooksJSON := `[{"title":"Database Failover Guide","content":"Steps for postgres failover","status":"active"}]`
			mockOrchestrator.On("ExecuteListRunbooksRaw", mock.Anything, rawArgs).Return(runbooksJSON, nil)

			res, err := ci.Intercept(teamCtx, "/runbook redis")
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Handled).To(BeTrue())
			Expect(res.Message).To(Equal("no runbooks matched query: 'redis'"))
		})
	})

	Context("Intercept /alert", func() {
		It("should return error message when orchestrator is nil", func() {
			nilCi := command.NewCommandInterceptor()
			res, err := nilCi.Intercept(ctx, "/alert")
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Handled).To(BeTrue())
			Expect(res.Message).To(Equal("Orchestrator service is unavailable."))
		})

		It("should format alert list when alerts exist", func() {
			alertID := uuid.New()
			alertsJSON := `[{"id":"` + alertID.String() + `","service_name":"auth-service","severity":"CRITICAL","received_at":"2026-08-07T10:00:00Z"}]`
			mockOrchestrator.On("ExecuteListAlertsRaw", mock.Anything).Return(alertsJSON, nil)

			res, err := ci.Intercept(ctx, "/alert")
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Handled).To(BeTrue())
			Expect(res.Message).To(ContainSubstring("found 1 alert(s)"))
			Expect(res.Message).To(ContainSubstring("auth-service"))
			Expect(res.Message).To(ContainSubstring("CRITICAL"))
		})

		It("should return no alerts found message when alert list is empty", func() {
			mockOrchestrator.On("ExecuteListAlertsRaw", mock.Anything).Return("[]", nil)

			res, err := ci.Intercept(ctx, "/alert")
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Handled).To(BeTrue())
			Expect(res.Message).To(Equal("no alerts found"))
		})

		It("should return error message when ExecuteListAlertsRaw fails", func() {
			mockOrchestrator.On("ExecuteListAlertsRaw", mock.Anything).Return("", errors.New("db query error"))

			res, err := ci.Intercept(ctx, "/alert")
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Handled).To(BeTrue())
			Expect(res.Message).To(ContainSubstring("Failed to list alerts"))
		})

		It("should fallback to raw string when /alert response is invalid JSON", func() {
			mockOrchestrator.On("ExecuteListAlertsRaw", mock.Anything).Return("raw alert string", nil)

			res, err := ci.Intercept(ctx, "/alert")
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Handled).To(BeTrue())
			Expect(res.Message).To(Equal("raw alert string"))
		})
	})

	Context("Intercept /help", func() {
		It("should list all available registered slash commands", func() {
			res, err := ci.Intercept(ctx, "/help")
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Handled).To(BeTrue())
			Expect(res.Message).To(ContainSubstring("Available slash commands:"))
			Expect(res.Message).To(ContainSubstring("/quit"))
			Expect(res.Message).To(ContainSubstring("/incident"))
			Expect(res.Message).To(ContainSubstring("/runbook"))
			Expect(res.Message).To(ContainSubstring("/alert"))
			Expect(res.Message).To(ContainSubstring("/help"))
		})
	})

	Context("Register custom command", func() {
		It("should allow registering custom slash command handlers via RegisterCommand", func() {
			realCi := command.NewCommandInterceptor().(*command.CommandInterceptor)
			realCi.RegisterCommand("/ping", func(ctx context.Context, prompt string) (*types.CommandResult, error) {
				return &types.CommandResult{
					Handled: true,
					Message: "pong",
				}, nil
			})

			res, err := realCi.Intercept(ctx, "/ping")
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Handled).To(BeTrue())
			Expect(res.Message).To(Equal("pong"))
		})

		It("should allow registering custom slash command handlers via RegisterHandler", func() {
			realCi := command.NewCommandInterceptor().(*command.CommandInterceptor)
			customHandler := command.NewFuncCommandHandler("/echo", "Echoes the input", func(ctx context.Context, prompt string) (*types.CommandResult, error) {
				return &types.CommandResult{
					Handled: true,
					Message: prompt,
				}, nil
			})
			realCi.RegisterHandler(customHandler)

			res, err := realCi.Intercept(ctx, "/echo hello world")
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Handled).To(BeTrue())
			Expect(res.Message).To(Equal("/echo hello world"))
		})
	})

	Context("CommandContextHelpers", func() {
		It("should inject and retrieve TeamID from context", func() {
			teamCtx := command.WithTeamID(ctx, teamID)
			retrieved, ok := command.GetTeamID(teamCtx)
			Expect(ok).To(BeTrue())
			Expect(retrieved).To(Equal(teamID))

			emptyID, ok := command.GetTeamID(ctx)
			Expect(ok).To(BeFalse())
			Expect(emptyID).To(Equal(uuid.Nil))
		})

		It("should inject and retrieve ActiveIncidentID from context", func() {
			incCtx := command.WithActiveIncidentID(ctx, incidentID)
			retrieved, ok := command.GetActiveIncidentID(incCtx)
			Expect(ok).To(BeTrue())
			Expect(retrieved).To(Equal(incidentID))

			emptyID, ok := command.GetActiveIncidentID(ctx)
			Expect(ok).To(BeFalse())
			Expect(emptyID).To(Equal(uuid.Nil))
		})
	})
})
