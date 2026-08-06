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
			rb := &models.Runbook{
				ID:      uuid.New(),
				TeamID:  uuid.New(),
				Title:   "Restart Guide",
				Content: "Run restart command",
				Status:  "active",
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
})
