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
	"github.com/WillieBam/support_copilot/backend/types/models"
)

var _ = Describe("RunbookRepository", func() {
	var (
		gormDB   *gorm.DB
		mock     sqlmock.Sqlmock
		teamRepo interfaces.ITeamRepository
		ctx      context.Context
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

		teamRepo = postgresRepo.NewTeamRepository(gormDB)
		Expect(teamRepo).NotTo(BeNil())
	})

	AfterEach(func() {
		Expect(mock.ExpectationsWereMet()).To(Succeed())
	})

	Context("CreateRunbook", func() {
		It("should insert runbook successfully", func() {
			incID := uuid.New()
			rb := &models.Runbook{
				ID:         uuid.New(),
				TeamID:     uuid.New(),
				IncidentID: &incID,
				Title:      "Restart Guide",
				Content:    "Run restart command",
				Status:     "active",
			}

			mock.ExpectBegin()
			mock.ExpectQuery(`INSERT INTO "runbooks"`).
				WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
				WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(rb.ID))
			mock.ExpectCommit()

			err := teamRepo.CreateRunbook(ctx, rb)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("GetRunbookByID", func() {
		It("should return runbook by ID", func() {
			rbID := uuid.New()
			rows := sqlmock.NewRows([]string{"id", "title", "content", "status"}).
				AddRow(rbID, "Restart Guide", "Run restart command", "active")

			mock.ExpectQuery(`SELECT \* FROM "runbooks" WHERE id = \$1 ORDER BY "runbooks"\."id" LIMIT \$2`).
				WithArgs(rbID, 1).
				WillReturnRows(rows)

			rb, err := teamRepo.GetRunbookByID(ctx, rbID)
			Expect(err).NotTo(HaveOccurred())
			Expect(rb).NotTo(BeNil())
			Expect(rb.Title).To(Equal("Restart Guide"))
		})
	})

	Context("ListRunbooks", func() {
		It("should list runbooks by team and status", func() {
			teamID := uuid.New()
			rows := sqlmock.NewRows([]string{"id", "team_id", "title", "status"}).
				AddRow(uuid.New(), teamID, "Active Guide", "active")

			mock.ExpectQuery(`SELECT \* FROM "runbooks" WHERE team_id = \$1 AND status = \$2 ORDER BY created_at DESC`).
				WithArgs(teamID, "active").
				WillReturnRows(rows)

			runbooks, err := teamRepo.ListRunbooks(ctx, teamID, "active")
			Expect(err).NotTo(HaveOccurred())
			Expect(len(runbooks)).To(Equal(1))
		})
	})

	Context("GetRunbookLogs", func() {
		It("should list runbook version history logs", func() {
			rbID := uuid.New()
			rows := sqlmock.NewRows([]string{"id", "runbook_id", "version", "older_title"}).
				AddRow(uuid.New(), rbID, 1, "Old Title")

			mock.ExpectQuery(`SELECT \* FROM "runbook_logs" WHERE runbook_id = \$1 ORDER BY version DESC`).
				WithArgs(rbID).
				WillReturnRows(rows)

			logs, err := teamRepo.GetRunbookLogs(ctx, rbID)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(logs)).To(Equal(1))
		})
	})

	Context("UpdateRunbook & DeprecateRunbook", func() {
		It("should update runbook and create version log in transaction", func() {
			rbID := uuid.New()
			teamID := uuid.New()
			incID := uuid.New()

			mock.ExpectBegin()
			// query first runbook
			mock.ExpectQuery(`SELECT \* FROM "runbooks" WHERE id = \$1 ORDER BY "runbooks"\."id" LIMIT \$2`).
				WithArgs(rbID, 1).
				WillReturnRows(sqlmock.NewRows([]string{"id", "team_id", "incident_id", "title", "content"}).AddRow(rbID, teamID, incID, "Old Title", "Old Content"))

			// create log
			mock.ExpectQuery(`INSERT INTO "runbook_logs"`).
				WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
				WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New()))

			// update runbook
			mock.ExpectExec(`UPDATE "runbooks" SET`).
				WillReturnResult(sqlmock.NewResult(1, 1))

			mock.ExpectCommit()

			log := &models.RunbookLog{
				OlderTitle:   "Old Title",
				OlderContent: "Old Content",
				Version:      1,
			}
			updated, err := teamRepo.UpdateRunbook(ctx, rbID, "New Title", "New Content", log)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.Title).To(Equal("New Title"))
			Expect(updated.Content).To(Equal("New Content"))
		})

		It("should deprecate a runbook successfully", func() {
			rbID := uuid.New()
			mock.ExpectQuery(`SELECT \* FROM "runbooks" WHERE id = \$1 ORDER BY "runbooks"\."id" LIMIT \$2`).
				WithArgs(rbID, 1).
				WillReturnRows(sqlmock.NewRows([]string{"id", "status"}).AddRow(rbID, "active"))

			mock.ExpectBegin()
			mock.ExpectExec(`UPDATE "runbooks" SET`).
				WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectCommit()

			deprecated, err := teamRepo.DeprecateRunbook(ctx, rbID)
			Expect(err).NotTo(HaveOccurred())
			Expect(deprecated.Status).To(Equal("deprecated"))
		})
	})

	Context("GetRunbooksByIncidentID & GetIncidentContextByIDOrNumber", func() {
		It("should get active runbooks by incident ID", func() {
			incID := uuid.New()
			mock.ExpectQuery(`SELECT \* FROM "runbooks" WHERE incident_id = \$1 AND status = 'active' ORDER BY created_at DESC`).
				WithArgs(incID).
				WillReturnRows(sqlmock.NewRows([]string{"id", "title"}).AddRow(uuid.New(), "Inc Runbook"))

			rbs, err := teamRepo.GetRunbooksByIncidentID(ctx, incID)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(rbs)).To(Equal(1))
		})

		It("should get incident context by surrogate key INC-101", func() {
			incID := uuid.New()
			mock.ExpectQuery(`SELECT \* FROM "team_incidents" WHERE LOWER\(incident_number\) = LOWER\(\$1\) ORDER BY "team_incidents"\."id" LIMIT \$2`).
				WithArgs("INC-101", 1).
				WillReturnRows(sqlmock.NewRows([]string{"id", "incident_number", "title"}).AddRow(incID, "INC-101", "Database Latency"))

			mock.ExpectQuery(`SELECT \* FROM "incident_status_histories" WHERE "incident_status_histories"\."team_incident_id" = \$1 ORDER BY updated_at ASC`).
				WithArgs(incID).
				WillReturnRows(sqlmock.NewRows([]string{"id", "team_incident_id"}))

			mock.ExpectQuery(`SELECT \* FROM "alerts" WHERE incident_id = \$1 OR id IN \(SELECT alert_id FROM alert_incidents WHERE incident_id = \$2\) ORDER BY received_at DESC`).
				WithArgs(incID, incID).
				WillReturnRows(sqlmock.NewRows([]string{"id", "alert_info"}))

			inc, alerts, err := teamRepo.GetIncidentContextByIDOrNumber(ctx, "INC-101")
			Expect(err).NotTo(HaveOccurred())
			Expect(inc).NotTo(BeNil())
			Expect(inc.IncidentNumber).To(Equal("INC-101"))
			Expect(alerts).NotTo(BeNil())
		})

		It("should get incident context by UUID", func() {
			incID := uuid.New()
			mock.ExpectQuery(`SELECT \* FROM "team_incidents" WHERE id = \$1 ORDER BY "team_incidents"\."id" LIMIT \$2`).
				WithArgs(incID, 1).
				WillReturnRows(sqlmock.NewRows([]string{"id", "title"}).AddRow(incID, "Context Test"))

			mock.ExpectQuery(`SELECT \* FROM "incident_status_histories" WHERE "incident_status_histories"\."team_incident_id" = \$1 ORDER BY updated_at ASC`).
				WithArgs(incID).
				WillReturnRows(sqlmock.NewRows([]string{"id"}))

			mock.ExpectQuery(`SELECT \* FROM "alerts" WHERE incident_id = \$1 OR id IN \(SELECT alert_id FROM alert_incidents WHERE incident_id = \$2\) ORDER BY received_at DESC`).
				WithArgs(incID, incID).
				WillReturnRows(sqlmock.NewRows([]string{"id"}))

			inc, alerts, err := teamRepo.GetIncidentContext(ctx, incID)
			Expect(err).NotTo(HaveOccurred())
			Expect(inc).NotTo(BeNil())
			Expect(alerts).NotTo(BeNil())
		})

		It("should return error when GetIncidentContextByIDOrNumber received empty string", func() {
			_, _, err := teamRepo.GetIncidentContextByIDOrNumber(ctx, "   ")
			Expect(err).To(HaveOccurred())
		})

		It("should return record not found error when deprecating non-existent runbook", func() {
			rbID := uuid.New()
			mock.ExpectQuery(`SELECT \* FROM "runbooks" WHERE id = \$1 ORDER BY "runbooks"\."id" LIMIT \$2`).
				WithArgs(rbID, 1).
				WillReturnError(gorm.ErrRecordNotFound)

			deprecated, err := teamRepo.DeprecateRunbook(ctx, rbID)
			Expect(err).To(Equal(gorm.ErrRecordNotFound))
			Expect(deprecated).To(BeNil())
		})

		It("should list all runbooks when status filter is empty or 'all'", func() {
			teamID := uuid.New()
			rows := sqlmock.NewRows([]string{"id", "team_id", "title"}).
				AddRow(uuid.New(), teamID, "All Status Guide")

			mock.ExpectQuery(`SELECT \* FROM "runbooks" WHERE team_id = \$1 ORDER BY created_at DESC`).
				WithArgs(teamID).
				WillReturnRows(rows)

			rbs, err := teamRepo.ListRunbooks(ctx, teamID, "all")
			Expect(err).NotTo(HaveOccurred())
			Expect(len(rbs)).To(Equal(1))
		})
	})
})
