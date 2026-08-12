package endpoint_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/WillieBam/support_copilot/backend/internal/endpoint"
	"github.com/WillieBam/support_copilot/backend/internal/mocks"
	"github.com/WillieBam/support_copilot/backend/types/models"
	"github.com/WillieBam/support_copilot/backend/types/requests"
	customErrors "github.com/WillieBam/support_copilot/backend/utils/errors"
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
				IncidentID: &incidentID,
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

		It("should handle CreateRunbook validation and service error paths", func() {
			// empty content
			bodyEmptyContent, _ := json.Marshal(requests.CreateRunbookRequest{Title: "Title", Content: ""})
			c1 := e.NewContext(httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(bodyEmptyContent)), httptest.NewRecorder())
			c1.Request().Header.Set("Content-Type", "application/json")
			c1.SetPathValues(echo.PathValues{{Name: "team_id", Value: teamID.String()}})
			err := h.CreateRunbook(c1)
			Expect(err).NotTo(HaveOccurred())

			// ErrTeamNotFound (400)
			bodyValid, _ := json.Marshal(requests.CreateRunbookRequest{Title: "Title", Content: "Content"})
			c2 := e.NewContext(httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(bodyValid)), httptest.NewRecorder())
			c2.Request().Header.Set("Content-Type", "application/json")
			c2.SetPathValues(echo.PathValues{{Name: "team_id", Value: teamID.String()}})
			mockTeamSvc.On("CreateRunbook", mock.Anything, uuid.Nil, teamID, uuid.Nil, "Title", "Content").Return(nil, customErrors.ErrTeamNotFound).Once()
			err = h.CreateRunbook(c2)
			Expect(err).NotTo(HaveOccurred())

			// generic error (500)
			c3 := e.NewContext(httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(bodyValid)), httptest.NewRecorder())
			c3.Request().Header.Set("Content-Type", "application/json")
			c3.SetPathValues(echo.PathValues{{Name: "team_id", Value: teamID.String()}})
			mockTeamSvc.On("CreateRunbook", mock.Anything, uuid.Nil, teamID, uuid.Nil, "Title", "Content").Return(nil, errors.New("db error")).Once()
			err = h.CreateRunbook(c3)
			Expect(err).NotTo(HaveOccurred())
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
				{ResourceInfo: `{"service":"auth-service"}`, AlertInfo: `{"severity":"critical"}`, ReceivedAt: time.Now(), Metrics: `{"container.cpu.usage": 98.5}`},
			}


			mockTeamSvc.On("GetIncidentContextByIDOrNumber", mock.Anything, incidentID.String()).Return(inc, alerts, nil)
			mockTeamSvc.On("ListRunbooks", mock.Anything, teamID, "active").Return([]models.Runbook{}, nil)

			err := h.GetIncidentContext(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusOK))
		})

		It("should return 400 on GetIncidentContext when id param is empty", func() {
			req := httptest.NewRequest(http.MethodGet, "/internal/incidents//context", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPathValues(echo.PathValues{{Name: "id", Value: ""}})

			err := h.GetIncidentContext(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusBadRequest))
		})

		It("should ListIncidentsInternal successfully and handle invalid team_id", func() {
			cBad := e.NewContext(httptest.NewRequest(http.MethodGet, "/internal/incidents/invalid", nil), httptest.NewRecorder())
			cBad.SetPathValues(echo.PathValues{{Name: "team_id", Value: "invalid"}})
			err := h.ListIncidentsInternal(cBad)
			Expect(err).NotTo(HaveOccurred())

			recOk := httptest.NewRecorder()
			cOk := e.NewContext(httptest.NewRequest(http.MethodGet, "/internal/incidents/"+teamID.String(), nil), recOk)
			cOk.SetPathValues(echo.PathValues{{Name: "team_id", Value: teamID.String()}})
			mockTeamSvc.On("ListTeamIncidents", mock.Anything, teamID).Return([]models.TeamIncident{{ID: incidentID}}, nil)
			err = h.ListIncidentsInternal(cOk)
			Expect(err).NotTo(HaveOccurred())
			Expect(recOk.Code).To(Equal(http.StatusOK))
		})

		It("should handle error paths in DeprecateRunbook, UpdateRunbook, GetRunbookLogs, and ListRunbooks", func() {
			// DeprecateRunbook invalid id & service error
			cBadDep := e.NewContext(httptest.NewRequest(http.MethodPatch, "/", nil), httptest.NewRecorder())
			cBadDep.SetPathValues(echo.PathValues{{Name: "id", Value: "invalid"}})
			err := h.DeprecateRunbook(cBadDep)
			Expect(err).NotTo(HaveOccurred())

			cErrDep := e.NewContext(httptest.NewRequest(http.MethodPatch, "/", nil), httptest.NewRecorder())
			cErrDep.SetPathValues(echo.PathValues{{Name: "id", Value: runbookID.String()}})
			mockTeamSvc.On("DeprecateRunbook", mock.Anything, runbookID).Return(nil, errors.New("db error")).Once()
			err = h.DeprecateRunbook(cErrDep)
			Expect(err).NotTo(HaveOccurred())

			// UpdateRunbook invalid id & bind error & service error
			cBadUpd := e.NewContext(httptest.NewRequest(http.MethodPatch, "/", nil), httptest.NewRecorder())
			cBadUpd.SetPathValues(echo.PathValues{{Name: "id", Value: "invalid"}})
			err = h.UpdateRunbook(cBadUpd)
			Expect(err).NotTo(HaveOccurred())

			cErrUpd := e.NewContext(httptest.NewRequest(http.MethodPatch, "/", strings.NewReader(`{"title":"T","content":"C"}`)), httptest.NewRecorder())
			cErrUpd.Request().Header.Set("Content-Type", "application/json")
			cErrUpd.SetPathValues(echo.PathValues{{Name: "id", Value: runbookID.String()}})
			mockTeamSvc.On("UpdateRunbook", mock.Anything, mock.Anything, runbookID, "T", "C").Return(nil, errors.New("db error")).Once()
			err = h.UpdateRunbook(cErrUpd)
			Expect(err).NotTo(HaveOccurred())

			// GetRunbookLogs invalid id & service error
			cBadLog := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder())
			cBadLog.SetPathValues(echo.PathValues{{Name: "id", Value: "invalid"}})
			err = h.GetRunbookLogs(cBadLog)
			Expect(err).NotTo(HaveOccurred())

			cErrLog := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder())
			cErrLog.SetPathValues(echo.PathValues{{Name: "id", Value: runbookID.String()}})
			mockTeamSvc.On("GetRunbookLogs", mock.Anything, runbookID).Return(nil, errors.New("db error")).Once()
			err = h.GetRunbookLogs(cErrLog)
			Expect(err).NotTo(HaveOccurred())

			// ListRunbooks invalid team_id & service error
			cBadList := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder())
			cBadList.SetPathValues(echo.PathValues{{Name: "team_id", Value: "invalid"}})
			err = h.ListRunbooks(cBadList)
			Expect(err).NotTo(HaveOccurred())

			cErrList := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder())
			cErrList.SetPathValues(echo.PathValues{{Name: "team_id", Value: teamID.String()}})
			mockTeamSvc.On("ListRunbooks", mock.Anything, teamID, mock.Anything).Return(nil, errors.New("db error")).Once()
			err = h.ListRunbooks(cErrList)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should test relativeTime, parseAndCleanseMetrics, and normalizeUUID in GetIncidentContext", func() {
			cCtx := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder())
			cCtx.SetPathValues(echo.PathValues{{Name: "id", Value: "12345678-1234-1234-1234-1234567890ab"}})

			now := time.Now()
			inc := &models.TeamIncident{
				ID:        incidentID,
				TeamID:    teamID,
				Title:     "Test Inc",
				Status:    "OPEN",
				CreatedAt: now.Add(-50 * 24 * time.Hour),
			}
			alerts := []models.Alert{
				{ReceivedAt: now.Add(10 * time.Second), Metrics: ""},
				{ReceivedAt: now.Add(-10 * time.Minute), Metrics: "invalid json"},
				{ReceivedAt: now.Add(-2 * time.Hour), Metrics: `{"container.cpu.usage": 0.001, "runtime.go.mem_stats.total_alloc": 50000000.0, "custom.hits": 100.0}`},
				{ReceivedAt: now.Add(-48 * time.Hour), Metrics: `{"error_rate": 0.05}`},
			}

			mockTeamSvc.On("GetIncidentContextByIDOrNumber", mock.Anything, "12345678-1234-1234-1234-1234567890ab").Return(inc, alerts, nil).Once()
			mockTeamSvc.On("ListRunbooks", mock.Anything, teamID, "active").Return([]models.Runbook{}, nil).Once()

			err := h.GetIncidentContext(cCtx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should handle 500 errors in ListIncidentsInternal and GetIncidentContext", func() {
			// ListIncidentsInternal 500
			cListIncErr := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder())
			cListIncErr.SetPathValues(echo.PathValues{{Name: "team_id", Value: teamID.String()}})
			mockTeamSvc.On("ListTeamIncidents", mock.Anything, teamID).Return(nil, errors.New("db error")).Once()
			err := h.ListIncidentsInternal(cListIncErr)
			Expect(err).NotTo(HaveOccurred())

			// GetIncidentContext 404
			cCtxErr := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder())
			cCtxErr.SetPathValues(echo.PathValues{{Name: "id", Value: incidentID.String()}})
			mockTeamSvc.On("GetIncidentContextByIDOrNumber", mock.Anything, incidentID.String()).Return(nil, nil, errors.New("not found")).Once()
			err = h.GetIncidentContext(cCtxErr)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
