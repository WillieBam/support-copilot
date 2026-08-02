package postgres_test

import (
	"context"
	"errors"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/WillieBam/support_copilot/backend/internal/interfaces"
	postgresRepo "github.com/WillieBam/support_copilot/backend/internal/repository/postgres"
	"github.com/WillieBam/support_copilot/backend/types/models"
)

var _ = Describe("AlertRepository", func() {
	var (
		gormDB    *gorm.DB
		mock      sqlmock.Sqlmock
		alertRepo interfaces.IAlertRepository
		ctx       context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		db, sqlMock, err := sqlmock.New()
		Expect(err).NotTo(HaveOccurred())

		mock = sqlMock
		dialector := postgres.New(postgres.Config{
			Conn: db,
		})

		gormDB, err = gorm.Open(dialector, &gorm.Config{})
		Expect(err).NotTo(HaveOccurred())

		alertRepo = postgresRepo.NewAlertRepository(gormDB)
		Expect(alertRepo).NotTo(BeNil())
	})

	AfterEach(func() {
		Expect(mock.ExpectationsWereMet()).To(Succeed())
	})

	Context("StoreAlert", func() {
		It("should successfully insert an alert", func() {
			incID := uuid.New()
			alert := &models.Alert{
				IncidentID:  &incID,
				ServiceName: "payment-service",
				Severity:    "high",
				Metrics:     `{"cpu": 98}`,
				ReceivedAt:  time.Now(),
			}

			mock.ExpectBegin()
			mock.ExpectQuery(`INSERT INTO "alerts"`).
				WithArgs(alert.ServiceName, alert.Severity, alert.Metrics, *alert.IncidentID, sqlmock.AnyArg()).
				WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New()))
			mock.ExpectCommit()

			err := alertRepo.StoreAlert(ctx, alert)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return an error if insert fails", func() {
			incID := uuid.New()
			alert := &models.Alert{
				IncidentID:  &incID,
				ServiceName: "payment-service",
				Severity:    "high",
				Metrics:     `{"cpu": 98}`,
			}

			mock.ExpectBegin()
			mock.ExpectQuery(`INSERT INTO "alerts"`).
				WillReturnError(errors.New("db write error"))
			mock.ExpectRollback()

			err := alertRepo.StoreAlert(ctx, alert)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("db write error"))
		})
	})

	Context("UpdateAlertIncidentID", func() {
		It("should update incident_id for an alert", func() {
			alertID := uuid.New()
			incidentID := uuid.New()

			mock.ExpectBegin()
			mock.ExpectExec(`UPDATE "alerts" SET "incident_id"=\$1 WHERE id = \$2`).
				WithArgs(incidentID, alertID).
				WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectCommit()

			err := alertRepo.UpdateAlertIncidentID(ctx, alertID, incidentID)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("RetrieveAlert", func() {
		It("should retrieve an alert by ID", func() {
			alertID := uuid.New()
			incID := uuid.New()

			rows := sqlmock.NewRows([]string{"id", "incident_id", "service_name", "severity", "metrics"}).
				AddRow(alertID, incID, "auth-service", "critical", `{"memory": 90}`)

			mock.ExpectQuery(`SELECT \* FROM "alerts" WHERE id = \$1 ORDER BY "alerts"\."id" LIMIT \$2`).
				WithArgs(alertID, 1).
				WillReturnRows(rows)

			alert, err := alertRepo.RetrieveAlertbyID(ctx, alertID)
			Expect(err).NotTo(HaveOccurred())
			Expect(alert).NotTo(BeNil())
			Expect(alert.ID).To(Equal(alertID))
			Expect(alert.ServiceName).To(Equal("auth-service"))
		})

		It("should return record not found error", func() {
			alertID := uuid.New()

			mock.ExpectQuery(`SELECT \* FROM "alerts" WHERE id = \$1 ORDER BY "alerts"\."id" LIMIT \$2`).
				WithArgs(alertID, 1).
				WillReturnError(gorm.ErrRecordNotFound)

			alert, err := alertRepo.RetrieveAlertbyID(ctx, alertID)
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, gorm.ErrRecordNotFound)).To(BeTrue())
			Expect(alert).To(BeNil())
		})

		It("should return generic internal server error for database error", func() {
			alertID := uuid.New()

			mock.ExpectQuery(`SELECT \* FROM "alerts" WHERE id = \$1 ORDER BY "alerts"\."id" LIMIT \$2`).
				WithArgs(alertID, 1).
				WillReturnError(errors.New("connection failed"))

			alert, err := alertRepo.RetrieveAlertbyID(ctx, alertID)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Internal Server Error"))
			Expect(alert).To(BeNil())
		})
	})
})
