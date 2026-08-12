package postgres_test

import (
	"context"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/WillieBam/support_copilot/backend/internal/interfaces"
	postgresRepo "github.com/WillieBam/support_copilot/backend/internal/repository/postgres"
)

var _ = Describe("DashboardRepository", func() {
	var (
		gormDB   *gorm.DB
		mock     sqlmock.Sqlmock
		dashRepo interfaces.IDashboardRepository
		ctx      context.Context
		teamID   uuid.UUID
	)

	BeforeEach(func() {
		ctx = context.Background()
		teamID = uuid.New()

		db, sqlMock, err := sqlmock.New()
		Expect(err).NotTo(HaveOccurred())

		mock = sqlMock
		dialector := postgres.New(postgres.Config{
			Conn: db,
		})

		gormDB, err = gorm.Open(dialector, &gorm.Config{})
		Expect(err).NotTo(HaveOccurred())

		dashRepo = postgresRepo.NewDashboardRepository(gormDB)
		Expect(dashRepo).NotTo(BeNil())
	})

	AfterEach(func() {
		Expect(mock.ExpectationsWereMet()).To(Succeed())
	})

	Context("GetIncidentTrend", func() {
		It("should execute trend aggregate query", func() {
			rows := sqlmock.NewRows([]string{"time_bucket", "status", "count"}).
				AddRow("2026-08", "RESOLVED", 5)

			mock.ExpectQuery(`SELECT DATE_TRUNC`).
				WithArgs("month", teamID).
				WillReturnRows(rows)

			points, err := dashRepo.GetIncidentTrend(ctx, teamID, "month")
			Expect(err).NotTo(HaveOccurred())
			Expect(len(points)).To(Equal(1))
			Expect(points[0].Count).To(Equal(5))
		})
	})

	Context("GetMTTRStats", func() {
		It("should return average resolution time and total resolved count", func() {
			rows := sqlmock.NewRows([]string{"avg_minutes", "total"}).
				AddRow(12.5, 10)

			mock.ExpectQuery(`SELECT COALESCE`).
				WithArgs(teamID).
				WillReturnRows(rows)

			avg, total, err := dashRepo.GetMTTRStats(ctx, teamID)
			Expect(err).NotTo(HaveOccurred())
			Expect(avg).To(Equal(12.5))
			Expect(total).To(Equal(10))
		})
	})

	Context("GetBreachedIncidents & CountBreachedIncidents", func() {
		It("should list breached incidents", func() {
			rows := sqlmock.NewRows([]string{"id", "title", "duration_minutes"}).
				AddRow(uuid.New().String(), "High CPU", 45.0)

			mock.ExpectQuery(`SELECT id::text`).
				WithArgs(teamID, 30, 10, 0).
				WillReturnRows(rows)

			breached, err := dashRepo.GetBreachedIncidents(ctx, teamID, 30, 10, 0)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(breached)).To(Equal(1))
		})

		It("should count breached incidents", func() {
			rows := sqlmock.NewRows([]string{"count"}).AddRow(2)

			mock.ExpectQuery(`SELECT COUNT\(\*\)`).
				WithArgs(teamID, 30).
				WillReturnRows(rows)

			count, err := dashRepo.CountBreachedIncidents(ctx, teamID, 30)
			Expect(err).NotTo(HaveOccurred())
			Expect(count).To(Equal(2))
		})
	})

	Context("Admin All Teams Dashboard Functions", func() {
		It("should fetch all teams incident trend", func() {
			rows := sqlmock.NewRows([]string{"time_bucket", "status", "count"}).
				AddRow("2026-08", "OPEN", 3)

			mock.ExpectQuery(`SELECT`).
				WithArgs("day").
				WillReturnRows(rows)

			points, err := dashRepo.GetAllTeamsIncidentTrend(ctx, "day")
			Expect(err).NotTo(HaveOccurred())
			Expect(len(points)).To(Equal(1))
		})

		It("should fetch all teams MTTR stats", func() {
			rows := sqlmock.NewRows([]string{"avg_minutes", "total"}).
				AddRow(20.0, 15)

			mock.ExpectQuery(`SELECT`).
				WillReturnRows(rows)

			avg, total, err := dashRepo.GetAllTeamsMTTRStats(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(avg).To(Equal(20.0))
			Expect(total).To(Equal(15))
		})

		It("should fetch all teams breached incidents and count", func() {
			rows := sqlmock.NewRows([]string{"id", "title", "duration_minutes"}).
				AddRow(uuid.New().String(), "Outage", 60.0)

			mock.ExpectQuery(`SELECT`).
				WithArgs(30, 10, 0).
				WillReturnRows(rows)

			incidents, err := dashRepo.GetAllTeamsBreachedIncidents(ctx, 30, 10, 0)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(incidents)).To(Equal(1))

			countRows := sqlmock.NewRows([]string{"count"}).AddRow(5)
			mock.ExpectQuery(`SELECT COUNT\(\*\)`).
				WithArgs(30).
				WillReturnRows(countRows)

			count, err := dashRepo.CountAllTeamsBreachedIncidents(ctx, 30)
			Expect(err).NotTo(HaveOccurred())
			Expect(count).To(Equal(5))
		})
	})
})
