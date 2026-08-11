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
				IncidentID:   &incID,
				ResourceInfo: `{"service":"payment-service"}`,
				AlertInfo:    `{"severity":"high"}`,
				Metrics:      `{"cpu": 98}`,
				ReceivedAt:   time.Now(),
			}

			mock.ExpectBegin()
			mock.ExpectQuery(`INSERT INTO "alerts"`).
				WithArgs(alert.AlertInfo, alert.ResourceInfo, alert.Metrics, alert.BusinessContext, alert.Metadata, *alert.IncidentID, sqlmock.AnyArg()).
				WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New()))
			mock.ExpectCommit()

			err := alertRepo.StoreAlert(ctx, alert)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return an error if insert fails", func() {
			incID := uuid.New()
			alert := &models.Alert{
				IncidentID:   &incID,
				ResourceInfo: `{"service":"payment-service"}`,
				AlertInfo:    `{"severity":"high"}`,
				Metrics:      `{"cpu": 98}`,
			}

			mock.ExpectBegin()
			mock.ExpectQuery(`INSERT INTO "alerts"`).
				WithArgs(alert.AlertInfo, alert.ResourceInfo, alert.Metrics, alert.BusinessContext, alert.Metadata, *alert.IncidentID).
				WillReturnError(errors.New("db error"))
			mock.ExpectRollback()

			err := alertRepo.StoreAlert(ctx, alert)
			Expect(err).To(HaveOccurred())
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

			rows := sqlmock.NewRows([]string{"id", "incident_id", "alert_info", "resource_info", "metrics", "business_context", "metadata"}).
				AddRow(alertID, incID, `{"severity":"critical"}`, `{"service":"auth-service"}`, `{"memory": 90}`, `{}`, `{}`)

			mock.ExpectQuery(`SELECT \* FROM "alerts" WHERE id = \$1 OR alert_info LIKE \$2 ORDER BY "alerts"\."id" LIMIT \$3`).
				WithArgs(alertID, "%\"id\":\""+alertID.String()+"\"%", 1).
				WillReturnRows(rows)

			alert, err := alertRepo.RetrieveAlertbyID(ctx, alertID.String())
			Expect(err).NotTo(HaveOccurred())
			Expect(alert).NotTo(BeNil())
			Expect(alert.ID).To(Equal(alertID))
		})

		It("should retrieve an alert by non-UUID string alertinfo ID", func() {
			alertID := uuid.New()
			incID := uuid.New()
			customAlertID := "165028917"

			rows := sqlmock.NewRows([]string{"id", "incident_id", "alert_info", "resource_info", "metrics", "business_context", "metadata"}).
				AddRow(alertID, incID, `{"id":"165028917","severity":"critical"}`, `{"service":"auth-service"}`, `{"memory": 90}`, `{}`, `{}`)

			mock.ExpectQuery(`SELECT \* FROM "alerts" WHERE alert_info LIKE \$1 ORDER BY "alerts"\."id" LIMIT \$2`).
				WithArgs("%\"id\":\""+customAlertID+"\"%", 1).
				WillReturnRows(rows)

			alert, err := alertRepo.RetrieveAlertbyID(ctx, customAlertID)
			Expect(err).NotTo(HaveOccurred())
			Expect(alert).NotTo(BeNil())
			Expect(alert.ID).To(Equal(alertID))
		})

		It("should return record not found error", func() {
			alertID := uuid.New()

			mock.ExpectQuery(`SELECT \* FROM "alerts" WHERE id = \$1 OR alert_info LIKE \$2 ORDER BY "alerts"\."id" LIMIT \$3`).
				WithArgs(alertID, "%\"id\":\""+alertID.String()+"\"%", 1).
				WillReturnError(gorm.ErrRecordNotFound)

			alert, err := alertRepo.RetrieveAlertbyID(ctx, alertID.String())
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, gorm.ErrRecordNotFound)).To(BeTrue())
			Expect(alert).To(BeNil())
		})

		It("should return generic internal server error for database error", func() {
			alertID := uuid.New()

			mock.ExpectQuery(`SELECT \* FROM "alerts" WHERE id = \$1 OR alert_info LIKE \$2 ORDER BY "alerts"\."id" LIMIT \$3`).
				WithArgs(alertID, "%\"id\":\""+alertID.String()+"\"%", 1).
				WillReturnError(errors.New("connection failed"))

			alert, err := alertRepo.RetrieveAlertbyID(ctx, alertID.String())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Internal Server Error"))
			Expect(alert).To(BeNil())
		})
	})
})
