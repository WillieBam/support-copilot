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

var _ = Describe("TeamRepository", func() {
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

	Context("CreateTeamWithOwner", func() {
		It("should successfully create a team and assign owner in a transaction", func() {
			ownerID := uuid.New()
			team := &models.Team{
				TeamName:  "Core Infra",
				CreatedAt: time.Now(),
			}

			mock.ExpectBegin()
			mock.ExpectQuery(`INSERT INTO "teams"`).
				WithArgs(team.TeamName, sqlmock.AnyArg()).
				WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New()))
			mock.ExpectQuery(`INSERT INTO "team_members"`).
				WithArgs(sqlmock.AnyArg(), ownerID, "owner", sqlmock.AnyArg()).
				WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New()))
			mock.ExpectCommit()

			err := teamRepo.CreateTeamWithOwner(ctx, team, ownerID)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should rollback if team insertion fails", func() {
			ownerID := uuid.New()
			team := &models.Team{
				TeamName: "Core Infra",
			}

			mock.ExpectBegin()
			mock.ExpectQuery(`INSERT INTO "teams"`).
				WillReturnError(errors.New("duplicate team_name"))
			mock.ExpectRollback()

			err := teamRepo.CreateTeamWithOwner(ctx, team, ownerID)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("AddTeamMember", func() {
		It("should insert team member", func() {
			member := &models.TeamMember{
				ID:     uuid.New(),
				TeamID: uuid.New(),
				UserID: uuid.New(),
				Role:   "member",
			}

			mock.ExpectBegin()
			mock.ExpectQuery(`INSERT INTO "team_members"`).
				WithArgs(member.TeamID, member.UserID, member.Role, member.ID).
				WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(member.ID))
			mock.ExpectCommit()

			err := teamRepo.AddTeamMember(ctx, member)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("GetUserWithTeamsByID", func() {
		It("should query user with team preloads", func() {
			userID := uuid.New()
			rows := sqlmock.NewRows([]string{"id", "firebase_uid", "email", "scope"}).
				AddRow(userID, "uid-123", "user@test.com", "engineer")

			mock.ExpectQuery(`SELECT \* FROM "users" WHERE id = \$1 ORDER BY "users"\."id" LIMIT \$2`).
				WithArgs(userID, 1).
				WillReturnRows(rows)

			mock.ExpectQuery(`SELECT \* FROM "team_members" WHERE "team_members"\."user_id" = \$1`).
				WithArgs(userID).
				WillReturnRows(sqlmock.NewRows([]string{"id", "team_id", "user_id", "role"}))

			user, err := teamRepo.GetUserWithTeamsByID(ctx, userID)
			Expect(err).NotTo(HaveOccurred())
			Expect(user).NotTo(BeNil())
			Expect(user.ID).To(Equal(userID))
		})
	})

	Context("RemoveTeamMember & DeleteTeam", func() {
		It("should remove team member", func() {
			teamID := uuid.New()
			userID := uuid.New()

			mock.ExpectBegin()
			mock.ExpectExec(`DELETE FROM "team_members" WHERE team_id = \$1 AND user_id = \$2`).
				WithArgs(teamID, userID).
				WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectCommit()

			err := teamRepo.RemoveTeamMember(ctx, teamID, userID)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should delete team", func() {
			teamID := uuid.New()

			mock.ExpectBegin()
			mock.ExpectExec(`DELETE FROM "teams" WHERE id = \$1`).
				WithArgs(teamID).
				WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectCommit()

			err := teamRepo.DeleteTeam(ctx, teamID)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("GetMemberRole & ListTeamMembers", func() {
		It("should get member role", func() {
			teamID := uuid.New()
			userID := uuid.New()

			rows := sqlmock.NewRows([]string{"id", "team_id", "user_id", "role"}).
				AddRow(uuid.New(), teamID, userID, "owner")

			mock.ExpectQuery(`SELECT \* FROM "team_members" WHERE team_id = \$1 AND user_id = \$2 ORDER BY "team_members"\."id" LIMIT \$3`).
				WithArgs(teamID, userID, 1).
				WillReturnRows(rows)

			role, err := teamRepo.GetMemberRole(ctx, teamID, userID)
			Expect(err).NotTo(HaveOccurred())
			Expect(role).To(Equal("owner"))
		})

		It("should list team members", func() {
			teamID := uuid.New()
			userID := uuid.New()
			rows := sqlmock.NewRows([]string{"id", "team_id", "user_id", "role"}).
				AddRow(uuid.New(), teamID, userID, "member")

			mock.ExpectQuery(`SELECT \* FROM "team_members" WHERE team_id = \$1`).
				WithArgs(teamID).
				WillReturnRows(rows)

			userRows := sqlmock.NewRows([]string{"id", "email"}).
				AddRow(userID, "user@test.com")

			mock.ExpectQuery(`SELECT \* FROM "users" WHERE "users"\."id" = \$1`).
				WithArgs(userID).
				WillReturnRows(userRows)

			members, err := teamRepo.ListTeamMembers(ctx, teamID)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(members)).To(Equal(1))
		})
	})

	Context("GetTeamInstruction", func() {
		It("should return empty instruction list when record not found", func() {
			teamID := uuid.New()

			mock.ExpectQuery(`SELECT \* FROM "instructions" WHERE team_id = \$1 ORDER BY "instructions"\."id" LIMIT \$2`).
				WithArgs(teamID, 1).
				WillReturnError(gorm.ErrRecordNotFound)

			inst, logs, err := teamRepo.GetTeamInstruction(ctx, teamID)
			Expect(err).NotTo(HaveOccurred())
			Expect(inst).To(BeNil())
			Expect(len(logs)).To(Equal(0))
		})
	})
})
