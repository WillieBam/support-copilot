package endpoint_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/WillieBam/support_copilot/backend/internal/endpoint"
	"github.com/WillieBam/support_copilot/backend/internal/mocks"
	"github.com/WillieBam/support_copilot/backend/types/models"
	"github.com/WillieBam/support_copilot/backend/types/requests"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

var _ = Describe("RunbookHandler", func() {
	var (
		e           *echo.Echo
		mockAppSvc  *mocks.IAppService
		mockAuthSvc *mocks.IAuthService
		mockTeamSvc *mocks.ITeamService
		h           *endpoint.Handler
		teamID      uuid.UUID
		runbookID   uuid.UUID
		incidentID  uuid.UUID
	)

	BeforeEach(func() {
		e = echo.New()
		mockAppSvc = &mocks.IAppService{}
		mockAuthSvc = &mocks.IAuthService{}
		mockTeamSvc = &mocks.ITeamService{}

		h = endpoint.NewHandler(
			mockAppSvc,
			mockAuthSvc,
			mockTeamSvc,
		)

		teamID = uuid.New()
		runbookID = uuid.New()
		incidentID = uuid.New()
	})

	Context("CreateRunbook", func() {
		It("should return 400 when team_id is invalid", func() {
			req := httptest.NewRequest(http.MethodPost, "/internal/teams/invalid/runbooks", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPathValues(echo.PathValues{{Name: "team_id", Value: "invalid-uuid"}})

			err := h.CreateRunbook(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusBadRequest))
		})

		It("should return 400 when title is empty", func() {
			body, _ := json.Marshal(requests.CreateRunbookRequest{Title: "", Content: "some content"})
			req := httptest.NewRequest(http.MethodPost, "/internal/teams/"+teamID.String()+"/runbooks", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPathValues(echo.PathValues{{Name: "team_id", Value: teamID.String()}})

			err := h.CreateRunbook(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusBadRequest))
		})

		It("should return 201 on successful runbook creation", func() {
			body, _ := json.Marshal(requests.CreateRunbookRequest{
				IncidentID: incidentID,
				Title:      "Pod Eviction Recovery",
				Content:    "Execute kubectl rollout restart deployment",
			})
			req := httptest.NewRequest(http.MethodPost, "/internal/teams/"+teamID.String()+"/runbooks", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPathValues(echo.PathValues{{Name: "team_id", Value: teamID.String()}})

			createdRb := &models.Runbook{
				ID:         runbookID,
				TeamID:     teamID,
				IncidentID: incidentID,
				Title:      "Pod Eviction Recovery",
				Content:    "Execute kubectl rollout restart deployment",
				Status:     "active",
			}
			mockTeamSvc.On("CreateRunbook", mock.Anything, uuid.Nil, teamID, incidentID, "Pod Eviction Recovery", "Execute kubectl rollout restart deployment").
				Return(createdRb, nil)

			err := h.CreateRunbook(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusCreated))
		})
	})

	Context("UpdateRunbook & DeprecateRunbook", func() {
		It("should UpdateRunbook successfully", func() {
			body, _ := json.Marshal(requests.UpdateRunbookRequest{
				Title:   "Pod Eviction Recovery v2",
				Content: "Updated content",
			})
			req := httptest.NewRequest(http.MethodPatch, "/internal/runbooks/"+runbookID.String(), bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPathValues(echo.PathValues{{Name: "id", Value: runbookID.String()}})

			updatedRb := &models.Runbook{
				ID:      runbookID,
				Title:   "Pod Eviction Recovery v2",
				Content: "Updated content",
			}
			mockTeamSvc.On("UpdateRunbook", mock.Anything, uuid.Nil, runbookID, "Pod Eviction Recovery v2", "Updated content").
				Return(updatedRb, nil)

			err := h.UpdateRunbook(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusOK))
		})

		It("should DeprecateRunbook successfully", func() {
			req := httptest.NewRequest(http.MethodPatch, "/internal/runbooks/"+runbookID.String()+"/deprecate", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPathValues(echo.PathValues{{Name: "id", Value: runbookID.String()}})

			deprecatedRb := &models.Runbook{ID: runbookID, Status: "deprecated"}
			mockTeamSvc.On("DeprecateRunbook", mock.Anything, runbookID).Return(deprecatedRb, nil)

			err := h.DeprecateRunbook(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusOK))
		})
	})

	Context("GetRunbook & GetRunbookLogs", func() {
		It("should return 404 when GetRunbook fails", func() {
			req := httptest.NewRequest(http.MethodGet, "/internal/runbooks/"+runbookID.String(), nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPathValues(echo.PathValues{{Name: "id", Value: runbookID.String()}})

			mockTeamSvc.On("GetRunbook", mock.Anything, runbookID).Return(nil, errors.New("not found"))

			err := h.GetRunbook(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusNotFound))
		})

		It("should return GetRunbook successfully", func() {
			req := httptest.NewRequest(http.MethodGet, "/internal/runbooks/"+runbookID.String(), nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPathValues(echo.PathValues{{Name: "id", Value: runbookID.String()}})

			rb := &models.Runbook{ID: runbookID, Title: "Redis Guide"}
			mockTeamSvc.On("GetRunbook", mock.Anything, runbookID).Return(rb, nil)

			err := h.GetRunbook(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusOK))
		})

		It("should return GetRunbookLogs successfully", func() {
			req := httptest.NewRequest(http.MethodGet, "/internal/runbooks/"+runbookID.String()+"/logs", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPathValues(echo.PathValues{{Name: "id", Value: runbookID.String()}})

			logs := []models.RunbookLog{{ID: uuid.New(), RunbookID: runbookID, Version: 1}}
			mockTeamSvc.On("GetRunbookLogs", mock.Anything, runbookID).Return(logs, nil)

			err := h.GetRunbookLogs(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusOK))
		})
	})

	Context("ListRunbooks & GetIncidentContext", func() {
		It("should ListRunbooks successfully", func() {
			req := httptest.NewRequest(http.MethodGet, "/internal/teams/"+teamID.String()+"/runbooks?status=active", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPathValues(echo.PathValues{{Name: "team_id", Value: teamID.String()}})

			runbooks := []models.Runbook{{ID: runbookID, Title: "Active Runbook"}}
			mockTeamSvc.On("ListRunbooks", mock.Anything, teamID, "active").Return(runbooks, nil)

			err := h.ListRunbooks(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusOK))
		})

		It("should GetIncidentContext successfully", func() {
			req := httptest.NewRequest(http.MethodGet, "/internal/incidents/"+incidentID.String()+"/context", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPathValues(echo.PathValues{{Name: "id", Value: incidentID.String()}})

			inc := &models.TeamIncident{
				ID:         incidentID,
				TeamID:     teamID,
				Title:      "Memory Spike",
				Status:     "OPEN",
				AssignedAt: time.Now().Add(-1 * time.Hour),
				History: []models.IncidentStatusHistory{
					{PreviousStatus: "", NewStatus: "OPEN", Details: "Alert triggered", UpdatedAt: time.Now()},
				},
			}
			alerts := []models.Alert{
				{ServiceName: "auth-service", Severity: "critical", ReceivedAt: time.Now(), Metrics: `{"container.cpu.usage": 98.5}`},
			}

			mockTeamSvc.On("GetIncidentContext", mock.Anything, incidentID).Return(inc, alerts, nil)
			mockTeamSvc.On("ListRunbooks", mock.Anything, teamID, "active").Return([]models.Runbook{}, nil)

			err := h.GetIncidentContext(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusOK))
		})
	})
})
