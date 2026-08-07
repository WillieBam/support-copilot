package endpoint_test

import (
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

		testUser = &models.User{
			ID:          userID,
			FirebaseUID: "fb-uid-123",
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
})
