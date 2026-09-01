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

		It("should rollback if UpdateRunbook fails to find runbook", func() {
			rbID := uuid.New()

			mock.ExpectBegin()
			mock.ExpectQuery(`SELECT \* FROM "runbooks" WHERE id = \$1 ORDER BY "runbooks"\."id" LIMIT \$2`).
				WithArgs(rbID, 1).
				WillReturnError(errors.New("not found"))
			mock.ExpectRollback()

			updated, err := teamRepo.UpdateRunbook(ctx, rbID, "New Title", "New Content", nil)
			Expect(err).To(HaveOccurred())
			Expect(updated).To(BeNil())
		})

		It("should rollback if UpdateRunbook log creation fails", func() {
			rbID := uuid.New()
			teamID := uuid.New()
			incID := uuid.New()

			mock.ExpectBegin()
			mock.ExpectQuery(`SELECT \* FROM "runbooks" WHERE id = \$1 ORDER BY "runbooks"\."id" LIMIT \$2`).
				WithArgs(rbID, 1).
				WillReturnRows(sqlmock.NewRows([]string{"id", "team_id", "incident_id"}).AddRow(rbID, teamID, incID))

			mock.ExpectQuery(`INSERT INTO "runbook_logs"`).
				WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
				WillReturnError(errors.New("log insert failed"))
			mock.ExpectRollback()

			log := &models.RunbookLog{
				OlderTitle: "Title",
				Version:    1,
			}
			updated, err := teamRepo.UpdateRunbook(ctx, rbID, "New Title", "New Content", log)
			Expect(err).To(HaveOccurred())
			Expect(updated).To(BeNil())
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

		It("should return error if DeprecateRunbook query fails", func() {
			rbID := uuid.New()
			mock.ExpectQuery(`SELECT \* FROM "runbooks" WHERE id = \$1 ORDER BY "runbooks"\."id" LIMIT \$2`).
				WithArgs(rbID, 1).
				WillReturnError(errors.New("db error"))

			deprecated, err := teamRepo.DeprecateRunbook(ctx, rbID)
			Expect(err).To(HaveOccurred())
			Expect(deprecated).To(BeNil())
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

		It("should return error when GetRunbooksByIncidentID query fails", func() {
			incID := uuid.New()
			mock.ExpectQuery(`SELECT \* FROM "runbooks" WHERE incident_id = \$1 AND status = 'active' ORDER BY created_at DESC`).
				WithArgs(incID).
				WillReturnError(errors.New("db failure"))

			rbs, err := teamRepo.GetRunbooksByIncidentID(ctx, incID)
			Expect(err).To(HaveOccurred())
			Expect(rbs).To(BeNil())
		})

		It("should get incident context by surrogate key INC-101", func() {
			incID := uuid.New()
			teamID := uuid.New()
			now := time.Now()
			rows := sqlmock.NewRows([]string{"incident_id", "incident_number", "team_id", "created_by", "title", "status", "details", "created_at", "assigned_at", "resolved_at", "history_id", "history_updated_by", "history_title", "history_new_status", "history_previous_status", "history_details", "history_updated_at"}).
				AddRow(incID, "INC-101", teamID, uuid.New(), "Database Latency", "OPEN", "Details", now, now, nil, nil, nil, nil, nil, nil, nil, nil)

			mock.ExpectQuery(`SELECT (.+) FROM team_incidents i LEFT JOIN incident_status_histories h ON (.+) WHERE LOWER\(i\.incident_number\) = LOWER\(\$1\) ORDER BY h\.updated_at ASC`).
				WithArgs("INC-101").
				WillReturnRows(rows)

			mock.ExpectQuery(`SELECT DISTINCT (.+) FROM alerts a LEFT JOIN alert_incidents ai ON (.+) WHERE a\.incident_id = \$1 OR ai\.incident_id = \$2 ORDER BY a\.received_at DESC`).
				WithArgs(incID, incID).
				WillReturnRows(sqlmock.NewRows([]string{"id", "alert_info"}))

			inc, alerts, err := teamRepo.GetIncidentContextByIDOrNumber(ctx, "INC-101")
			Expect(err).NotTo(HaveOccurred())
			Expect(inc).NotTo(BeNil())
			Expect(inc.IncidentNumber).To(Equal("INC-101"))
			Expect(alerts).NotTo(BeNil())
		})

		It("should return error when GetIncidentContextByIDOrNumber incident query fails", func() {
			mock.ExpectQuery(`SELECT (.+) FROM team_incidents i LEFT JOIN incident_status_histories h ON (.+) WHERE LOWER\(i\.incident_number\) = LOWER\(\$1\) ORDER BY h\.updated_at ASC`).
				WithArgs("INC-500").
				WillReturnError(errors.New("db error"))

			inc, alerts, err := teamRepo.GetIncidentContextByIDOrNumber(ctx, "INC-500")
			Expect(err).To(HaveOccurred())
			Expect(inc).To(BeNil())
			Expect(alerts).To(BeNil())
		})

		It("should get incident context by UUID", func() {
			incID := uuid.New()
			teamID := uuid.New()
			now := time.Now()
			rows := sqlmock.NewRows([]string{"incident_id", "incident_number", "team_id", "created_by", "title", "status", "details", "created_at", "assigned_at", "resolved_at", "history_id", "history_updated_by", "history_title", "history_new_status", "history_previous_status", "history_details", "history_updated_at"}).
				AddRow(incID, "INC-102", teamID, uuid.New(), "Context Test", "OPEN", "Details", now, now, nil, nil, nil, nil, nil, nil, nil, nil)

			mock.ExpectQuery(`SELECT (.+) FROM team_incidents i LEFT JOIN incident_status_histories h ON (.+) WHERE i\.id = \$1 ORDER BY h\.updated_at ASC`).
				WithArgs(incID).
				WillReturnRows(rows)

			mock.ExpectQuery(`SELECT DISTINCT (.+) FROM alerts a LEFT JOIN alert_incidents ai ON (.+) WHERE a\.incident_id = \$1 OR ai\.incident_id = \$2 ORDER BY a\.received_at DESC`).
				WithArgs(incID, incID).
				WillReturnRows(sqlmock.NewRows([]string{"id"}))

			inc, alerts, err := teamRepo.GetIncidentContext(ctx, incID)
			Expect(err).NotTo(HaveOccurred())
			Expect(inc).NotTo(BeNil())
			Expect(alerts).NotTo(BeNil())
		})

		It("should return error when GetIncidentContext incident query fails", func() {
			incID := uuid.New()
			mock.ExpectQuery(`SELECT (.+) FROM team_incidents i LEFT JOIN incident_status_histories h ON (.+) WHERE i\.id = \$1 ORDER BY h\.updated_at ASC`).
				WithArgs(incID).
				WillReturnError(errors.New("db error"))

			inc, alerts, err := teamRepo.GetIncidentContext(ctx, incID)
			Expect(err).To(HaveOccurred())
			Expect(inc).To(BeNil())
			Expect(alerts).To(BeNil())
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

		It("should list active runbooks when status filter is 'active'", func() {
			teamID := uuid.New()
			rows := sqlmock.NewRows([]string{"id", "team_id", "title", "status"}).
				AddRow(uuid.New(), teamID, "Active Guide", "active")

			mock.ExpectQuery(`SELECT \* FROM "runbooks" WHERE team_id = \$1 AND status = \$2 ORDER BY created_at DESC`).
				WithArgs(teamID, "active").
				WillReturnRows(rows)

			rbs, err := teamRepo.ListRunbooks(ctx, teamID, "active")
			Expect(err).NotTo(HaveOccurred())
			Expect(len(rbs)).To(Equal(1))
		})

		It("should return error when ListRunbooks fails", func() {
			teamID := uuid.New()
			mock.ExpectQuery(`SELECT \* FROM "runbooks" WHERE team_id = \$1 ORDER BY created_at DESC`).
				WithArgs(teamID).
				WillReturnError(errors.New("db query error"))

			rbs, err := teamRepo.ListRunbooks(ctx, teamID, "all")
			Expect(err).To(HaveOccurred())
			Expect(rbs).To(BeNil())
		})

		It("should return error when GetRunbookLogs fails", func() {
			rbID := uuid.New()
			mock.ExpectQuery(`SELECT \* FROM "runbook_logs" WHERE runbook_id = \$1 ORDER BY version DESC`).
				WithArgs(rbID).
				WillReturnError(errors.New("db error"))

			logs, err := teamRepo.GetRunbookLogs(ctx, rbID)
			Expect(err).To(HaveOccurred())
			Expect(logs).To(BeNil())
		})

		It("should return error when GetRunbookByID fails", func() {
			rbID := uuid.New()
			mock.ExpectQuery(`SELECT \* FROM "runbooks" WHERE id = \$1 ORDER BY "runbooks"\."id" LIMIT \$2`).
				WithArgs(rbID, 1).
				WillReturnError(errors.New("db error"))

			rb, err := teamRepo.GetRunbookByID(ctx, rbID)
			Expect(err).To(HaveOccurred())
			Expect(rb).To(BeNil())
		})
	})
})
