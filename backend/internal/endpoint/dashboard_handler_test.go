package endpoint_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/WillieBam/support_copilot/backend/internal/endpoint"
	"github.com/WillieBam/support_copilot/backend/internal/mocks"
	"github.com/WillieBam/support_copilot/backend/types"
	"github.com/WillieBam/support_copilot/backend/types/models"
	customErrors "github.com/WillieBam/support_copilot/backend/utils/errors"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

var _ = Describe("DashboardHandler", func() {
	var (
		e           *echo.Echo
		mockAppSvc  *mocks.IAppService
		mockAuthSvc *mocks.IAuthService
		mockUserSvc *mocks.IUserService
		mockDashSvc *mocks.IDashboardService
		h           *endpoint.Handler
		testUser    *models.User
		userID      uuid.UUID
		teamID      uuid.UUID
	)

	BeforeEach(func() {
		e = echo.New()
		mockAppSvc = &mocks.IAppService{}
		mockAuthSvc = &mocks.IAuthService{}
		mockUserSvc = &mocks.IUserService{}
		mockDashSvc = &mocks.IDashboardService{}

		h = endpoint.NewHandler(
			mockAppSvc,
			mockAuthSvc,
			mockUserSvc,
			mockDashSvc,
		)

		userID = uuid.New()
		teamID = uuid.New()

		fbUID := "fb-uid-123"
		testUser = &models.User{
			ID:          userID,
			FirebaseUID: &fbUID,
			Email:       "engineer@test.com",
			Scope:       "engineer",
		}
	})

	Context("GetIncidentTrend", func() {
		It("should return 400 when team_id is missing or invalid", func() {
			req := httptest.NewRequest(http.MethodGet, "/api/dashboard/incidents/trend?team_id=invalid", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.Set("user_uid", "fb-uid-123")

			mockUserSvc.On("GetUserByFirebaseUID", mock.Anything, "fb-uid-123").Return(testUser, nil)

			err := h.GetIncidentTrend(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusBadRequest))
		})

		It("should return 400 when timeframe is invalid", func() {
			req := httptest.NewRequest(http.MethodGet, "/api/dashboard/incidents/trend?team_id="+teamID.String()+"&timeframe=invalid", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.Set("user_uid", "fb-uid-123")

			mockUserSvc.On("GetUserByFirebaseUID", mock.Anything, "fb-uid-123").Return(testUser, nil)
			mockDashSvc.On("GetIncidentTrend", mock.Anything, userID, teamID, "engineer", "invalid").
				Return(nil, customErrors.ErrInvalidTimeframe)

			err := h.GetIncidentTrend(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusBadRequest))
		})

		It("should return 200 on successful GetIncidentTrend", func() {
			req := httptest.NewRequest(http.MethodGet, "/api/dashboard/incidents/trend?team_id="+teamID.String()+"&timeframe=month", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.Set("user_uid", "fb-uid-123")

			trendData := []types.IncidentTrendPoint{
				{TimeBucket: "2026-08", Status: "OPEN", Count: 5},
			}
			mockUserSvc.On("GetUserByFirebaseUID", mock.Anything, "fb-uid-123").Return(testUser, nil)
			mockDashSvc.On("GetIncidentTrend", mock.Anything, userID, teamID, "engineer", "month").
				Return(trendData, nil)

			err := h.GetIncidentTrend(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusOK))
		})
	})

	Context("GetMTTR", func() {
		It("should return 400 when sla_target_minutes is invalid integer", func() {
			req := httptest.NewRequest(http.MethodGet, "/api/dashboard/mttr?team_id="+teamID.String()+"&sla_target_minutes=-5", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.Set("user_uid", "fb-uid-123")

			mockUserSvc.On("GetUserByFirebaseUID", mock.Anything, "fb-uid-123").Return(testUser, nil)

			err := h.GetMTTR(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusBadRequest))
		})

		It("should return 200 on successful GetMTTR", func() {
			req := httptest.NewRequest(http.MethodGet, "/api/dashboard/mttr?team_id="+teamID.String()+"&sla_target_minutes=30", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.Set("user_uid", "fb-uid-123")

			mttrRes := &types.MTTRResult{
				MTTRMinutes:    15.5,
				TotalResolved:  20,
				SLABreaches:    2,
				ComplianceRate: 90.0,
			}
			mockUserSvc.On("GetUserByFirebaseUID", mock.Anything, "fb-uid-123").Return(testUser, nil)
			mockDashSvc.On("GetMTTR", mock.Anything, userID, teamID, "engineer", 30).
				Return(mttrRes, nil)

			err := h.GetMTTR(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusOK))
		})
	})

	Context("GetBreachedIncidents", func() {
		It("should return 200 on successful GetBreachedIncidents with limit and offset", func() {
			req := httptest.NewRequest(http.MethodGet, "/api/dashboard/incidents/breached?team_id="+teamID.String()+"&sla_target_minutes=30&limit=10&offset=0", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.Set("user_uid", "fb-uid-123")

			now := time.Now()
			breached := []types.BreachedIncident{
				{ID: uuid.New().String(), Title: "Slow Queries", CreatedAt: now, DurationMinutes: 45.0},
			}
			mockUserSvc.On("GetUserByFirebaseUID", mock.Anything, "fb-uid-123").Return(testUser, nil)
			mockDashSvc.On("GetBreachedIncidents", mock.Anything, userID, teamID, "engineer", 30, 10, 0).
				Return(breached, nil)

			err := h.GetBreachedIncidents(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusOK))
		})
	})

	Context("GetAllTeamsIncidentTrend", func() {
		It("should return 403 when user is not super_admin", func() {
			req := httptest.NewRequest(http.MethodGet, "/api/admin/dashboard/incidents/trend?timeframe=month", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.Set("user_uid", "fb-uid-123")

			mockUserSvc.On("GetUserByFirebaseUID", mock.Anything, "fb-uid-123").Return(testUser, nil)
			mockDashSvc.On("GetAllTeamsIncidentTrend", mock.Anything, "engineer", "month").
				Return(nil, customErrors.ErrSuperAdminRequired)

			err := h.GetAllTeamsIncidentTrend(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusForbidden))
		})

		It("should return 200 on successful GetAllTeamsIncidentTrend", func() {
			fbUID := "fb-uid-123"
			adminUser := &models.User{ID: userID, FirebaseUID: &fbUID, Scope: "super_admin"}
			req := httptest.NewRequest(http.MethodGet, "/api/admin/dashboard/incidents/trend?timeframe=month", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.Set("user_uid", "fb-uid-123")

			trendData := []types.IncidentTrendPoint{
				{TimeBucket: "2026-08", Status: "OPEN", Count: 10},
			}
			mockUserSvc.On("GetUserByFirebaseUID", mock.Anything, "fb-uid-123").Return(adminUser, nil)
			mockDashSvc.On("GetAllTeamsIncidentTrend", mock.Anything, "super_admin", "month").
				Return(trendData, nil)

			err := h.GetAllTeamsIncidentTrend(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusOK))
		})
	})

	Context("GetAllTeamsMTTR", func() {
		It("should return 200 on successful GetAllTeamsMTTR", func() {
			fbUID := "fb-uid-123"
			adminUser := &models.User{ID: userID, FirebaseUID: &fbUID, Scope: "super_admin"}
			req := httptest.NewRequest(http.MethodGet, "/api/admin/dashboard/mttr?sla_target_minutes=30", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.Set("user_uid", "fb-uid-123")

			mttrRes := &types.MTTRResult{
				MTTRMinutes:    20.0,
				TotalResolved:  10,
				SLABreaches:    1,
				ComplianceRate: 90.0,
			}
			mockUserSvc.On("GetUserByFirebaseUID", mock.Anything, "fb-uid-123").Return(adminUser, nil)
			mockDashSvc.On("GetAllTeamsMTTR", mock.Anything, "super_admin", 30).
				Return(mttrRes, nil)

			err := h.GetAllTeamsMTTR(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusOK))
		})
	})

	Context("GetAllTeamsBreachedIncidents", func() {
		It("should return 200 on successful GetAllTeamsBreachedIncidents", func() {
			fbUID := "fb-uid-123"
			adminUser := &models.User{ID: userID, FirebaseUID: &fbUID, Scope: "super_admin"}
			req := httptest.NewRequest(http.MethodGet, "/api/admin/dashboard/incidents/breached?sla_target_minutes=30&limit=10&offset=0", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.Set("user_uid", "fb-uid-123")

			now := time.Now()
			breached := []types.BreachedIncident{
				{ID: uuid.New().String(), Title: "Major Incident", CreatedAt: now, DurationMinutes: 90.0},
			}
			mockUserSvc.On("GetUserByFirebaseUID", mock.Anything, "fb-uid-123").Return(adminUser, nil)
			mockDashSvc.On("GetAllTeamsBreachedIncidents", mock.Anything, "super_admin", 30, 10, 0).
				Return(breached, nil)

			err := h.GetAllTeamsBreachedIncidents(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusOK))
		})

		It("should return 400 when query params are invalid across dashboard handlers", func() {
			mockUserSvc.On("GetUserByFirebaseUID", mock.Anything, "fb-uid-123").Return(testUser, nil)

			// GetBreachedIncidents invalid team_id
			c1 := e.NewContext(httptest.NewRequest(http.MethodGet, "/api/dashboard/incidents/breached?team_id=invalid", nil), httptest.NewRecorder())
			c1.Set("user_uid", "fb-uid-123")
			err := h.GetBreachedIncidents(c1)
			Expect(err).NotTo(HaveOccurred())

			// GetAllTeamsMTTR invalid sla
			c2 := e.NewContext(httptest.NewRequest(http.MethodGet, "/api/admin/dashboard/mttr?sla_target_minutes=-5", nil), httptest.NewRecorder())
			c2.Set("user_uid", "fb-uid-123")
			err = h.GetAllTeamsMTTR(c2)
			Expect(err).NotTo(HaveOccurred())

			// GetAllTeamsBreachedIncidents invalid sla
			c3 := e.NewContext(httptest.NewRequest(http.MethodGet, "/api/admin/dashboard/incidents/breached?sla_target_minutes=-5", nil), httptest.NewRecorder())
			c3.Set("user_uid", "fb-uid-123")
			err = h.GetAllTeamsBreachedIncidents(c3)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return 500 when dashboard service returns error", func() {
			adminUID := "admin-uid"
			adminUser := &models.User{ID: userID, FirebaseUID: &adminUID, Scope: "super_admin"}
			mockUserSvc.On("GetUserByFirebaseUID", mock.Anything, "admin-uid").Return(adminUser, nil)
			mockUserSvc.On("GetUserByFirebaseUID", mock.Anything, "fb-uid-123").Return(testUser, nil)

			// GetIncidentTrend 500
			cTrend1 := e.NewContext(httptest.NewRequest(http.MethodGet, "/api/dashboard/incidents/trend?team_id="+teamID.String()+"&timeframe=month", nil), httptest.NewRecorder())
			cTrend1.Set("user_uid", "fb-uid-123")
			mockDashSvc.On("GetIncidentTrend", mock.Anything, userID, teamID, "engineer", "month").Return(nil, errors.New("db error")).Once()
			err := h.GetIncidentTrend(cTrend1)
			Expect(err).NotTo(HaveOccurred())

			// GetMTTR 500
			cMTTR1 := e.NewContext(httptest.NewRequest(http.MethodGet, "/api/dashboard/mttr?team_id="+teamID.String()+"&sla_target_minutes=30", nil), httptest.NewRecorder())
			cMTTR1.Set("user_uid", "fb-uid-123")
			mockDashSvc.On("GetMTTR", mock.Anything, userID, teamID, "engineer", 30).Return(nil, errors.New("db error")).Once()
			err = h.GetMTTR(cMTTR1)
			Expect(err).NotTo(HaveOccurred())

			// GetBreachedIncidents 500
			cBreach1 := e.NewContext(httptest.NewRequest(http.MethodGet, "/api/dashboard/incidents/breached?team_id="+teamID.String()+"&sla_target_minutes=30", nil), httptest.NewRecorder())
			cBreach1.Set("user_uid", "fb-uid-123")
			mockDashSvc.On("GetBreachedIncidents", mock.Anything, userID, teamID, "engineer", 30, 50, 0).Return(nil, errors.New("db error")).Once()
			err = h.GetBreachedIncidents(cBreach1)
			Expect(err).NotTo(HaveOccurred())

			cTrend := e.NewContext(httptest.NewRequest(http.MethodGet, "/api/admin/dashboard/incidents/trend?timeframe=month", nil), httptest.NewRecorder())
			cTrend.Set("user_uid", "admin-uid")
			mockDashSvc.On("GetAllTeamsIncidentTrend", mock.Anything, "super_admin", "month").Return(nil, errors.New("db error")).Once()
			err = h.GetAllTeamsIncidentTrend(cTrend)
			Expect(err).NotTo(HaveOccurred())

			cMTTR := e.NewContext(httptest.NewRequest(http.MethodGet, "/api/admin/dashboard/mttr?sla_target_minutes=30", nil), httptest.NewRecorder())
			cMTTR.Set("user_uid", "admin-uid")
			mockDashSvc.On("GetAllTeamsMTTR", mock.Anything, "super_admin", 30).Return(nil, errors.New("db error")).Once()
			err = h.GetAllTeamsMTTR(cMTTR)
			Expect(err).NotTo(HaveOccurred())

			cBreach := e.NewContext(httptest.NewRequest(http.MethodGet, "/api/admin/dashboard/incidents/breached?sla_target_minutes=30", nil), httptest.NewRecorder())
			cBreach.Set("user_uid", "admin-uid")
			mockDashSvc.On("GetAllTeamsBreachedIncidents", mock.Anything, "super_admin", 30, 50, 0).Return(nil, errors.New("db error")).Once()
			err = h.GetAllTeamsBreachedIncidents(cBreach)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
