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
			mock.ExpectQuery(`INSERT INTO "alert_incidents"`).
				WithArgs(alertID, incidentID, "human_ui").
				WillReturnRows(sqlmock.NewRows([]string{"created_at"}).AddRow(time.Now()))
			mock.ExpectCommit()

			mock.ExpectBegin()
			mock.ExpectExec(`UPDATE "alerts" SET "incident_id"=\$1 WHERE id = \$2 AND incident_id IS NULL`).
				WithArgs(incidentID, alertID).
				WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectCommit()

			err := alertRepo.UpdateAlertIncidentID(ctx, alertID, incidentID)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return error when UpdateAlertIncidentID fails on insert", func() {
			alertID := uuid.New()
			incidentID := uuid.New()

			mock.ExpectBegin()
			mock.ExpectQuery(`INSERT INTO "alert_incidents"`).
				WithArgs(alertID, incidentID, "human_ui").
				WillReturnError(errors.New("insert failed"))
			mock.ExpectRollback()

			err := alertRepo.UpdateAlertIncidentID(ctx, alertID, incidentID)
			Expect(err).To(HaveOccurred())
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

		It("should return record not found error for non-UUID string", func() {
			customAlertID := "99999"

			mock.ExpectQuery(`SELECT \* FROM "alerts" WHERE alert_info LIKE \$1 ORDER BY "alerts"\."id" LIMIT \$2`).
				WithArgs("%\"id\":\""+customAlertID+"\"%", 1).
				WillReturnError(gorm.ErrRecordNotFound)

			alert, err := alertRepo.RetrieveAlertbyID(ctx, customAlertID)
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, gorm.ErrRecordNotFound)).To(BeTrue())
			Expect(alert).To(BeNil())
		})

		It("should return generic internal server error for non-UUID string database error", func() {
			customAlertID := "99999"

			mock.ExpectQuery(`SELECT \* FROM "alerts" WHERE alert_info LIKE \$1 ORDER BY "alerts"\."id" LIMIT \$2`).
				WithArgs("%\"id\":\""+customAlertID+"\"%", 1).
				WillReturnError(errors.New("db error"))

			alert, err := alertRepo.RetrieveAlertbyID(ctx, customAlertID)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Internal Server Error"))
			Expect(alert).To(BeNil())
		})
	})

	Context("ListAlerts", func() {
		It("should list alerts ordered by received_at desc", func() {
			alertID := uuid.New()
			rows := sqlmock.NewRows([]string{"id", "alert_info", "resource_info", "received_at"}).
				AddRow(alertID, `{"severity":"high"}`, `{"service":"payment-service"}`, time.Now())

			mock.ExpectQuery(`SELECT \* FROM "alerts" ORDER BY received_at DESC LIMIT \$1`).
				WithArgs(10).
				WillReturnRows(rows)

			alerts, err := alertRepo.ListAlerts(ctx, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(alerts).To(HaveLen(1))
			Expect(alerts[0].ID).To(Equal(alertID))
		})

		It("should return error on ListAlerts query failure", func() {
			mock.ExpectQuery(`SELECT \* FROM "alerts" ORDER BY received_at DESC LIMIT \$1`).
				WithArgs(5).
				WillReturnError(errors.New("db query error"))

			alerts, err := alertRepo.ListAlerts(ctx, 5)
			Expect(err).To(HaveOccurred())
			Expect(alerts).To(BeNil())
		})
	})

	Context("UnlinkAlertFromIncident", func() {
		It("should unlink an alert from an incident successfully", func() {
			alertID := uuid.New()
			incidentID := uuid.New()

			mock.ExpectBegin()
			mock.ExpectExec(`DELETE FROM "alert_incidents" WHERE alert_id = \$1 AND incident_id = \$2`).
				WithArgs(alertID, incidentID).
				WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectCommit()

			mock.ExpectBegin()
			mock.ExpectExec(`UPDATE "alerts" SET "incident_id"=\$1 WHERE id = \$2 AND incident_id = \$3`).
				WithArgs(nil, alertID, incidentID).
				WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectCommit()

			err := alertRepo.UnlinkAlertFromIncident(ctx, alertID, incidentID)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return error when delete from alert_incidents fails", func() {
			alertID := uuid.New()
			incidentID := uuid.New()

			mock.ExpectBegin()
			mock.ExpectExec(`DELETE FROM "alert_incidents" WHERE alert_id = \$1 AND incident_id = \$2`).
				WithArgs(alertID, incidentID).
				WillReturnError(errors.New("delete failed"))
			mock.ExpectRollback()

			err := alertRepo.UnlinkAlertFromIncident(ctx, alertID, incidentID)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("ListAlertsForIncident", func() {
		It("should list alerts tied to incident", func() {
			incidentID := uuid.New()
			alertID := uuid.New()

			rows := sqlmock.NewRows([]string{"id", "incident_id", "received_at"}).
				AddRow(alertID, incidentID, time.Now())

			mock.ExpectQuery(`SELECT DISTINCT a\.\* FROM alerts a LEFT JOIN alert_incidents ai ON ai\.alert_id = a\.id WHERE a\.incident_id = \$1 OR ai\.incident_id = \$2 ORDER BY a\.received_at DESC`).
				WithArgs(incidentID, incidentID).
				WillReturnRows(rows)

			alerts, err := alertRepo.ListAlertsForIncident(ctx, incidentID)
			Expect(err).NotTo(HaveOccurred())
			Expect(alerts).To(HaveLen(1))
			Expect(alerts[0].ID).To(Equal(alertID))
		})

		It("should return error when query fails", func() {
			incidentID := uuid.New()

			mock.ExpectQuery(`SELECT DISTINCT a\.\* FROM alerts a LEFT JOIN alert_incidents ai ON ai\.alert_id = a\.id WHERE a\.incident_id = \$1 OR ai\.incident_id = \$2 ORDER BY a\.received_at DESC`).
				WithArgs(incidentID, incidentID).
				WillReturnError(errors.New("query error"))

			alerts, err := alertRepo.ListAlertsForIncident(ctx, incidentID)
			Expect(err).To(HaveOccurred())
			Expect(alerts).To(BeNil())
		})
	})
})
