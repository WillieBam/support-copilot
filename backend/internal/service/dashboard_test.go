package service_test

import (
	"context"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gorm.io/gorm"

	"github.com/WillieBam/support_copilot/backend/internal/interfaces"
	"github.com/WillieBam/support_copilot/backend/internal/mocks"
	"github.com/WillieBam/support_copilot/backend/internal/service"
	"github.com/WillieBam/support_copilot/backend/types"
	customErrors "github.com/WillieBam/support_copilot/backend/utils/errors"
)

var _ = Describe("DashboardService", func() {
	var (
		dashSvc     interfaces.IDashboardService
		dashRepo    *mocks.IDashboardRepository
		teamRepo    *mocks.ITeamRepository
		ctx         context.Context
		teamID      uuid.UUID
		requesterID uuid.UUID
	)

	BeforeEach(func() {
		ctx = context.Background()
		teamID = uuid.New()
		requesterID = uuid.New()
		dashRepo = &mocks.IDashboardRepository{}
		teamRepo = &mocks.ITeamRepository{}
		dashSvc = service.NewDashboardService(dashRepo, teamRepo)
	})

	AfterEach(func() {
		dashRepo.AssertExpectations(GinkgoT())
		teamRepo.AssertExpectations(GinkgoT())
	})

	Context("GetIncidentTrend", func() {
		It("should return ErrInvalidTimeframe for an unsupported timeframe", func() {
			result, err := dashSvc.GetIncidentTrend(ctx, requesterID, teamID, "engineer", "week")
			Expect(err).To(Equal(customErrors.ErrInvalidTimeframe))
			Expect(result).To(BeNil())
		})

		It("should return ErrDashboardUnauthorized when requester is not a team member", func() {
			teamRepo.On("GetMemberRole", ctx, teamID, requesterID).Return("", gorm.ErrRecordNotFound)

			result, err := dashSvc.GetIncidentTrend(ctx, requesterID, teamID, "engineer", "month")
			Expect(err).To(Equal(customErrors.ErrDashboardUnauthorized))
			Expect(result).To(BeNil())
		})

		It("should bypass membership check when requester is super_admin", func() {
			// teamRepo.GetMemberRole must never be called for super_admin
			expected := []types.IncidentTrendPoint{
				{TimeBucket: "2026-07-01", Status: "OPEN", Count: 3},
			}
			dashRepo.On("GetIncidentTrend", ctx, teamID, "month").Return(expected, nil)

			result, err := dashSvc.GetIncidentTrend(ctx, requesterID, teamID, "super_admin", "month")
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(expected))
		})

		It("should return bucketed trend data for a valid member request", func() {
			teamRepo.On("GetMemberRole", ctx, teamID, requesterID).Return("member", nil)
			expected := []types.IncidentTrendPoint{
				{TimeBucket: "2026-07-01", Status: "OPEN", Count: 2},
				{TimeBucket: "2026-07-01", Status: "RESOLVED", Count: 5},
			}
			dashRepo.On("GetIncidentTrend", ctx, teamID, "day").Return(expected, nil)

			result, err := dashSvc.GetIncidentTrend(ctx, requesterID, teamID, "engineer", "day")
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(expected))
		})
	})

	Context("GetMTTR", func() {
		It("should return ErrInvalidSLATarget when sla_target_minutes is zero", func() {
			result, err := dashSvc.GetMTTR(ctx, requesterID, teamID, "engineer", 0)
			Expect(err).To(Equal(customErrors.ErrInvalidSLATarget))
			Expect(result).To(BeNil())
		})

		It("should return ErrDashboardUnauthorized when requester is not a team member", func() {
			teamRepo.On("GetMemberRole", ctx, teamID, requesterID).Return("", gorm.ErrRecordNotFound)

			result, err := dashSvc.GetMTTR(ctx, requesterID, teamID, "engineer", 30)
			Expect(err).To(Equal(customErrors.ErrDashboardUnauthorized))
			Expect(result).To(BeNil())
		})

		It("should return zero compliance_rate when there are no resolved incidents", func() {
			teamRepo.On("GetMemberRole", ctx, teamID, requesterID).Return("owner", nil)
			dashRepo.On("GetMTTRStats", ctx, teamID).Return(0.0, 0, nil)
			dashRepo.On("CountBreachedIncidents", ctx, teamID, 30).Return(0, nil)

			result, err := dashSvc.GetMTTR(ctx, requesterID, teamID, "engineer", 30)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.TotalResolved).To(Equal(0))
			Expect(result.ComplianceRate).To(Equal(0.0))
		})

		It("should correctly compute mttr, breach count, and compliance_rate", func() {
			teamRepo.On("GetMemberRole", ctx, teamID, requesterID).Return("member", nil)
			dashRepo.On("GetMTTRStats", ctx, teamID).Return(45.0, 10, nil)
			dashRepo.On("CountBreachedIncidents", ctx, teamID, 30).Return(2, nil)

			result, err := dashSvc.GetMTTR(ctx, requesterID, teamID, "engineer", 30)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.MTTRMinutes).To(Equal(45.0))
			Expect(result.TotalResolved).To(Equal(10))
			Expect(result.SLABreaches).To(Equal(2))
			// compliance_rate = (10 - 2) / 10 * 100 = 80.0
			Expect(result.ComplianceRate).To(BeNumerically("~", 80.0, 0.01))
		})
	})

	Context("GetBreachedIncidents", func() {
		It("should return ErrInvalidSLATarget for a negative sla target", func() {
			result, err := dashSvc.GetBreachedIncidents(ctx, requesterID, teamID, "engineer", -1, 50, 0)
			Expect(err).To(Equal(customErrors.ErrInvalidSLATarget))
			Expect(result).To(BeNil())
		})

		It("should allow sla_target_minutes=0 to return all resolved incidents", func() {
			teamRepo.On("GetMemberRole", ctx, teamID, requesterID).Return("member", nil)
			dashRepo.On("GetBreachedIncidents", ctx, teamID, 0, 50, 0).Return([]types.BreachedIncident{}, nil)

			result, err := dashSvc.GetBreachedIncidents(ctx, requesterID, teamID, "engineer", 0, 50, 0)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(BeEmpty())
		})

		It("should normalise a zero limit to the default of 50", func() {
			teamRepo.On("GetMemberRole", ctx, teamID, requesterID).Return("member", nil)
			// service must normalise limit 0 → 50 before calling repo
			dashRepo.On("GetBreachedIncidents", ctx, teamID, 30, 50, 0).Return([]types.BreachedIncident{}, nil)

			_, err := dashSvc.GetBreachedIncidents(ctx, requesterID, teamID, "engineer", 30, 0, 0)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return ErrDashboardUnauthorized when requester is not a team member", func() {
			teamRepo.On("GetMemberRole", ctx, teamID, requesterID).Return("", gorm.ErrRecordNotFound)

			result, err := dashSvc.GetBreachedIncidents(ctx, requesterID, teamID, "engineer", 30, 50, 0)
			Expect(err).To(Equal(customErrors.ErrDashboardUnauthorized))
			Expect(result).To(BeNil())
		})

		It("should return a paginated list of breached incidents", func() {
			teamRepo.On("GetMemberRole", ctx, teamID, requesterID).Return("member", nil)
			expected := []types.BreachedIncident{
				{ID: uuid.New().String(), Title: "DB down", DurationMinutes: 65.5},
			}
			dashRepo.On("GetBreachedIncidents", ctx, teamID, 60, 10, 0).Return(expected, nil)

			result, err := dashSvc.GetBreachedIncidents(ctx, requesterID, teamID, "engineer", 60, 10, 0)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(expected))
		})
	})

	Context("GetAllTeamsIncidentTrend", func() {
		It("should return ErrSuperAdminRequired for non super_admin scope", func() {
			result, err := dashSvc.GetAllTeamsIncidentTrend(ctx, "engineer", "month")
			Expect(err).To(Equal(customErrors.ErrSuperAdminRequired))
			Expect(result).To(BeNil())
		})

		It("should return ErrInvalidTimeframe for invalid timeframe", func() {
			result, err := dashSvc.GetAllTeamsIncidentTrend(ctx, "super_admin", "invalid")
			Expect(err).To(Equal(customErrors.ErrInvalidTimeframe))
			Expect(result).To(BeNil())
		})

		It("should return trend data for super_admin", func() {
			expected := []types.IncidentTrendPoint{
				{TimeBucket: "2026-08-01", Status: "OPEN", Count: 10},
			}
			dashRepo.On("GetAllTeamsIncidentTrend", ctx, "month").Return(expected, nil)

			result, err := dashSvc.GetAllTeamsIncidentTrend(ctx, "super_admin", "month")
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(expected))
		})
	})

	Context("GetAllTeamsMTTR", func() {
		It("should return ErrSuperAdminRequired for non super_admin scope", func() {
			result, err := dashSvc.GetAllTeamsMTTR(ctx, "engineer", 30)
			Expect(err).To(Equal(customErrors.ErrSuperAdminRequired))
			Expect(result).To(BeNil())
		})

		It("should return ErrInvalidSLATarget when target is non-positive", func() {
			result, err := dashSvc.GetAllTeamsMTTR(ctx, "super_admin", 0)
			Expect(err).To(Equal(customErrors.ErrInvalidSLATarget))
			Expect(result).To(BeNil())
		})

		It("should return all teams mttr stats for super_admin", func() {
			dashRepo.On("GetAllTeamsMTTRStats", ctx).Return(20.0, 5, nil)
			dashRepo.On("CountAllTeamsBreachedIncidents", ctx, 30).Return(1, nil)

			result, err := dashSvc.GetAllTeamsMTTR(ctx, "super_admin", 30)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.MTTRMinutes).To(Equal(20.0))
			Expect(result.TotalResolved).To(Equal(5))
			Expect(result.SLABreaches).To(Equal(1))
			Expect(result.ComplianceRate).To(BeNumerically("~", 80.0, 0.01))
		})
	})

	Context("GetAllTeamsBreachedIncidents", func() {
		It("should return ErrSuperAdminRequired for non super_admin scope", func() {
			result, err := dashSvc.GetAllTeamsBreachedIncidents(ctx, "engineer", 30, 50, 0)
			Expect(err).To(Equal(customErrors.ErrSuperAdminRequired))
			Expect(result).To(BeNil())
		})

		It("should return all teams breached incidents for super_admin", func() {
			expected := []types.BreachedIncident{
				{ID: uuid.New().String(), Title: "Global Outage", DurationMinutes: 120.0},
			}
			dashRepo.On("GetAllTeamsBreachedIncidents", ctx, 30, 50, 0).Return(expected, nil)

			result, err := dashSvc.GetAllTeamsBreachedIncidents(ctx, "super_admin", 30, 50, 0)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(expected))
		})
	})
})
