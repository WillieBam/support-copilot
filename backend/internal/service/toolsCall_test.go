package service_test

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	"github.com/WillieBam/support_copilot/backend/internal/interfaces"
	"github.com/WillieBam/support_copilot/backend/internal/mocks"
	"github.com/WillieBam/support_copilot/backend/internal/service"
	"github.com/WillieBam/support_copilot/backend/types/models"
	"github.com/WillieBam/support_copilot/backend/types/requests"
)

var _ = Describe("OrchestratorService (Tools Calling Gateway)", func() {
	var (
		orchestratorSvc  interfaces.IOrchestratorService
		mockAlertRepo    *mocks.IAlertRepository
		mockMcpOne       *mocks.IMCPClient
		mockMcpTwo       *mocks.IMCP2Client
		mockTeamRepo     *mocks.ITeamRepository
		ctx              context.Context
		testAlertID      uuid.UUID
		testAlert        *models.Alert
		validMetricsJSON string
	)

	BeforeEach(func() {
		ctx = context.Background()
		mockAlertRepo = &mocks.IAlertRepository{}
		mockMcpOne = &mocks.IMCPClient{}
		mockMcpTwo = &mocks.IMCP2Client{}
		mockTeamRepo = &mocks.ITeamRepository{}
		testAlertID = uuid.New()

		validMetricsJSON = `{
			"cpu_usage": 92.5,
			"memory_usage": 88.0,
			"incoming_traffic": 1200.0,
			"outgoing_traffic": 1100.0,
			"error_rate": 0.05,
			"network_throughput": 450.0,
			"request_rate": 300.0,
			"response_latency": 350.0,
			"availability_percent": 99.1
		}`

		incID := uuid.New()
		testAlert = &models.Alert{
			ID:          testAlertID,
			IncidentID:  &incID,
			ReceivedAt:  time.Now(),
			ServiceName: "payment-service",
			Severity:    "CRITICAL",
			Metrics:     validMetricsJSON,
		}

		orchestratorSvc = service.NewOrchestratorService(mockAlertRepo, mockMcpOne, mockMcpTwo, mockTeamRepo)
	})

	Context("ExecuteValidateAlert", func() {
		It("should successfully fetch alert from DB and predict anomaly via MCP", func() {
			mockAlertRepo.On("RetrieveAlertbyID", mock.Anything, testAlertID).Return(testAlert, nil)

			expectedMcpResp := &requests.AnomalyDetectionResponse{
				Status: 0,
				Label:  "Anomaly",
				Engine: "IsolationForest",
			}
			mockMcpOne.On("DetectAnomalies", mock.Anything, mock.MatchedBy(func(req requests.AnomalyDetectionRequest) bool {
				return req.CpuUsage == 92.5 && req.ResponseLatency == 350.0
			})).Return(expectedMcpResp, nil)

			result, err := orchestratorSvc.ExecuteValidateAlert(ctx, testAlertID)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())
			Expect(result.AlertID).To(Equal(testAlertID.String()))
			Expect(result.ServiceName).To(Equal("payment-service"))
			Expect(result.Severity).To(Equal("CRITICAL"))
			Expect(result.MLPrediction.Status).To(Equal(0))
			Expect(result.MLPrediction.Label).To(Equal("Anomaly"))

			mockAlertRepo.AssertExpectations(GinkgoT())
			mockMcpOne.AssertExpectations(GinkgoT())
		})

		It("should return error if alert ID is not found in Postgres", func() {
			mockAlertRepo.On("RetrieveAlertbyID", mock.Anything, testAlertID).Return(nil, errors.New("record not found"))

			result, err := orchestratorSvc.ExecuteValidateAlert(ctx, testAlertID)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to fetch alert"))
			Expect(result).To(BeNil())
		})

		It("should return error if alert metrics JSON in Postgres is corrupted", func() {
			corruptedAlert := &models.Alert{
				ID:          testAlertID,
				ServiceName: "cart-service",
				Severity:    "WARNING",
				Metrics:     "{invalid_json",
			}
			mockAlertRepo.On("RetrieveAlertbyID", mock.Anything, testAlertID).Return(corruptedAlert, nil)

			result, err := orchestratorSvc.ExecuteValidateAlert(ctx, testAlertID)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to parse alert metrics JSON"))
			Expect(result).To(BeNil())
		})

		It("should return error if MCP client returns error", func() {
			mockAlertRepo.On("RetrieveAlertbyID", mock.Anything, testAlertID).Return(testAlert, nil)
			mockMcpOne.On("DetectAnomalies", mock.Anything, mock.Anything).Return(nil, errors.New("mcp connection refused"))

			result, err := orchestratorSvc.ExecuteValidateAlert(ctx, testAlertID)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to analyze metrics via MCP server"))
			Expect(result).To(BeNil())
		})
	})

	Context("ExecuteValidateAlertRaw", func() {
		It("should parse JSON raw args and return serialized combined payload", func() {
			mockAlertRepo.On("RetrieveAlertbyID", mock.Anything, testAlertID).Return(testAlert, nil)
			expectedMcpResp := &requests.AnomalyDetectionResponse{
				Status: 1,
				Label:  "Normal",
				Engine: "IsolationForest",
			}
			mockMcpOne.On("DetectAnomalies", mock.Anything, mock.Anything).Return(expectedMcpResp, nil)

			rawArgs := `{"alert_id": "` + testAlertID.String() + `"}`
			jsonResult, err := orchestratorSvc.ExecuteValidateAlertRaw(ctx, rawArgs)
			Expect(err).NotTo(HaveOccurred())
			Expect(jsonResult).To(ContainSubstring("payment-service"))
			Expect(jsonResult).To(ContainSubstring("Normal"))
		})

		It("should fail on invalid UUID in raw args", func() {
			rawArgs := `{"alert_id": "invalid-uuid"}`
			_, err := orchestratorSvc.ExecuteValidateAlertRaw(ctx, rawArgs)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid alert id"))
		})
	})

	Context("MCP2 Tools Execution", func() {
		It("should delegate ExecuteGetIncidentRaw to mcpClient2", func() {
			mockMcpTwo.On("GetIncident", mock.Anything, requests.MCP2GetIncidentArgs{
				IncidentID: "inc-100",
			}).Return(`{"incident_id":"inc-100"}`, nil)

			res, err := orchestratorSvc.ExecuteGetIncidentRaw(ctx, `{"incident_id":"inc-100"}`)
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(ContainSubstring("inc-100"))
		})

		It("should delegate ExecuteListIncidentsRaw to mcpClient2", func() {
			mockMcpTwo.On("ListIncidents", mock.Anything, requests.MCP2ListIncidentsArgs{
				TeamID: "team-1",
			}).Return(`[{"id":"inc-1"}]`, nil)

			res, err := orchestratorSvc.ExecuteListIncidentsRaw(ctx, `{"team_id":"team-1"}`)
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(ContainSubstring("inc-1"))
		})

		It("should delegate ExecuteCreateRunbookRaw to mcpClient2", func() {
			mockMcpTwo.On("CreateRunbook", mock.Anything, requests.MCP2CreateRunbookArgs{
				TeamID:     "team-1",
				IncidentID: "inc-1",
				Title:      "Runbook Title",
				Content:    "Runbook Content",
			}).Return(`{"id":"rb-1"}`, nil)

			res, err := orchestratorSvc.ExecuteCreateRunbookRaw(ctx, `{"team_id":"team-1","incident_id":"inc-1","title":"Runbook Title","content":"Runbook Content"}`)
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(ContainSubstring("rb-1"))
		})

		It("should delegate ExecuteUpdateRunbookRaw to mcpClient2", func() {
			mockMcpTwo.On("UpdateRunbook", mock.Anything, requests.MCP2UpdateRunbookArgs{
				RunbookID: "rb-1",
				Title:     "New Title",
				Content:   "New Content",
			}).Return(`{"updated":true}`, nil)

			res, err := orchestratorSvc.ExecuteUpdateRunbookRaw(ctx, `{"runbook_id":"rb-1","title":"New Title","content":"New Content"}`)
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(ContainSubstring("updated"))
		})

		It("should delegate ExecuteDeprecateRunbookRaw to mcpClient2", func() {
			mockMcpTwo.On("DeprecateRunbook", mock.Anything, requests.MCP2DeprecateRunbookArgs{
				RunbookID: "rb-1",
			}).Return(`{"status":"deprecated"}`, nil)

			res, err := orchestratorSvc.ExecuteDeprecateRunbookRaw(ctx, `{"runbook_id":"rb-1"}`)
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(ContainSubstring("deprecated"))
		})

		It("should delegate ExecuteGetRunbookRaw to mcpClient2", func() {
			mockMcpTwo.On("GetRunbook", mock.Anything, requests.MCP2GetRunbookArgs{
				RunbookID: "rb-1",
			}).Return(`{"id":"rb-1"}`, nil)

			res, err := orchestratorSvc.ExecuteGetRunbookRaw(ctx, `{"runbook_id":"rb-1"}`)
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(ContainSubstring("rb-1"))
		})

		It("should delegate ExecuteListRunbooksRaw to mcpClient2", func() {
			mockMcpTwo.On("ListRunbooks", mock.Anything, requests.MCP2ListRunbooksArgs{
				TeamID: "team-1",
				Status: "active",
			}).Return(`[{"id":"rb-1"}]`, nil)

			res, err := orchestratorSvc.ExecuteListRunbooksRaw(ctx, `{"team_id":"team-1","status":"active"}`)
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(ContainSubstring("rb-1"))
		})

		It("should execute ExecuteLinkAlertToIncidentRaw successfully", func() {
			alertID := uuid.New()
			incidentID := uuid.New()
			mockAlertRepo.On("UpdateAlertIncidentID", mock.Anything, alertID, incidentID).Return(nil)

			rawArgs := `{"alert_id":"` + alertID.String() + `","incident_id":"` + incidentID.String() + `"}`
			res, err := orchestratorSvc.ExecuteLinkAlertToIncidentRaw(ctx, rawArgs)
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(ContainSubstring("success"))
			Expect(res).To(ContainSubstring(alertID.String()))
			Expect(res).To(ContainSubstring(incidentID.String()))
		})

		It("should recover when alert_id is missing but the UUID is placed in incident_id", func() {
			alertID := uuid.New()
			incidentID := uuid.New()
			resolvedIncident := models.TeamIncident{ID: incidentID, Title: "report-download-service CPU Spike"}

			mockTeamRepo.On("ListTeamIncidents", mock.Anything, uuid.Nil).Return([]models.TeamIncident{resolvedIncident}, nil)
			mockAlertRepo.On("UpdateAlertIncidentID", mock.Anything, alertID, incidentID).Return(nil)

			rawArgs := `{"incident_id":"` + alertID.String() + `","incident_title":"report-download-service CPU Spike"}`
			res, err := orchestratorSvc.ExecuteLinkAlertToIncidentRaw(ctx, rawArgs)
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(ContainSubstring("success"))
			Expect(res).To(ContainSubstring(alertID.String()))
			Expect(res).To(ContainSubstring(incidentID.String()))
		})
	})

	Context("ExecuteListAlertsRaw", func() {
		It("should return json formatted list of alerts", func() {
			alerts := []*models.Alert{
				testAlert,
			}
			mockAlertRepo.On("ListAlerts", mock.Anything, 20).Return(alerts, nil)

			res, err := orchestratorSvc.ExecuteListAlertsRaw(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(ContainSubstring("payment-service"))
		})

		It("should return error when alertRepo ListAlerts fails", func() {
			mockAlertRepo.On("ListAlerts", mock.Anything, 20).Return(nil, errors.New("db error"))

			_, err := orchestratorSvc.ExecuteListAlertsRaw(ctx)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to list alerts"))
		})
	})

	Context("Raw Tools Argument Validation and Errors", func() {
		It("should return error on empty or dummy alert_id in ExecuteValidateAlertRaw", func() {
			_, err := orchestratorSvc.ExecuteValidateAlertRaw(ctx, `{"alert_id": "null"}`)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no valid alert_id provided"))
		})

		It("should return error on missing incident_id in ExecuteGetIncidentRaw", func() {
			_, err := orchestratorSvc.ExecuteGetIncidentRaw(ctx, `{"incident_id": ""}`)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("incident_id is required"))
		})

		It("should return error on invalid json in ExecuteGetIncidentRaw", func() {
			_, err := orchestratorSvc.ExecuteGetIncidentRaw(ctx, `invalid json`)
			Expect(err).To(HaveOccurred())
		})

		It("should return error on missing team_id in ExecuteListIncidentsRaw", func() {
			_, err := orchestratorSvc.ExecuteListIncidentsRaw(ctx, `{"team_id": ""}`)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("team_id is required"))
		})

		It("should return error on missing required fields in ExecuteCreateRunbookRaw", func() {
			_, err := orchestratorSvc.ExecuteCreateRunbookRaw(ctx, `{"title": "Title"}`)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("team_id, incident_id, title, and content are required"))
		})

		It("should return error on missing runbook_id in ExecuteUpdateRunbookRaw", func() {
			_, err := orchestratorSvc.ExecuteUpdateRunbookRaw(ctx, `{"title": "Title"}`)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("runbook_id is required"))
		})

		It("should return error on missing runbook_id in ExecuteDeprecateRunbookRaw", func() {
			_, err := orchestratorSvc.ExecuteDeprecateRunbookRaw(ctx, `{}`)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("runbook_id is required"))
		})

		It("should return error on missing runbook_id in ExecuteGetRunbookRaw", func() {
			_, err := orchestratorSvc.ExecuteGetRunbookRaw(ctx, `{}`)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("runbook_id is required"))
		})

		It("should return error on missing team_id in ExecuteListRunbooksRaw", func() {
			_, err := orchestratorSvc.ExecuteListRunbooksRaw(ctx, `{}`)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("team_id is required"))
		})

		It("should return error on invalid alert_id in ExecuteLinkAlertToIncidentRaw", func() {
			_, err := orchestratorSvc.ExecuteLinkAlertToIncidentRaw(ctx, `{"alert_id": "invalid"}`)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid alert_id"))
		})

		It("should return error when incident title cannot be resolved in ExecuteLinkAlertToIncidentRaw", func() {
			alertID := uuid.New()
			mockTeamRepo.On("ListTeamIncidents", mock.Anything, uuid.Nil).Return([]models.TeamIncident{}, nil)

			_, err := orchestratorSvc.ExecuteLinkAlertToIncidentRaw(ctx, `{"alert_id": "`+alertID.String()+`", "incident_title": "NonExistent"}`)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("could not resolve valid incident UUID"))
		})

		It("should return error when UpdateAlertIncidentID fails in ExecuteLinkAlertToIncidentRaw", func() {
			alertID := uuid.New()
			incidentID := uuid.New()
			mockAlertRepo.On("UpdateAlertIncidentID", mock.Anything, alertID, incidentID).Return(errors.New("db error"))

			_, err := orchestratorSvc.ExecuteLinkAlertToIncidentRaw(ctx, `{"alert_id": "`+alertID.String()+`", "incident_id": "`+incidentID.String()+`"}`)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to link alert to incident"))
		})
	})
})

