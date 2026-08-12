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

	Context("GetTeamInstruction & SaveTeamInstruction", func() {
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

		It("should save team instruction in transaction", func() {
			teamID := uuid.New()
			userID := uuid.New()
			inst := &models.Instruction{
				ID:                 uuid.New(),
				TeamID:             teamID,
				CreatedBy:          userID,
				InstructionDetails: "Always check Redis latency",
			}

			mock.ExpectBegin()
			mock.ExpectQuery(`SELECT \* FROM "instructions" WHERE team_id = \$1 ORDER BY "instructions"\."id" LIMIT \$2`).
				WithArgs(teamID, 1).
				WillReturnError(gorm.ErrRecordNotFound)

			mock.ExpectQuery(`INSERT INTO "instructions"`).
				WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(inst.ID))
			mock.ExpectCommit()

			err := teamRepo.SaveTeamInstruction(ctx, inst, nil)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return team instruction and logs when found", func() {
			teamID := uuid.New()
			instID := uuid.New()

			mock.ExpectQuery(`SELECT \* FROM "instructions" WHERE team_id = \$1 ORDER BY "instructions"\."id" LIMIT \$2`).
				WithArgs(teamID, 1).
				WillReturnRows(sqlmock.NewRows([]string{"id", "team_id", "instruction_details"}).AddRow(instID, teamID, "Check logs"))

			mock.ExpectQuery(`SELECT \* FROM "instruction_logs" WHERE instruction_id = \$1 ORDER BY version DESC`).
				WithArgs(instID).
				WillReturnRows(sqlmock.NewRows([]string{"id", "instruction_id", "version"}).AddRow(uuid.New(), instID, 1))

			inst, logs, err := teamRepo.GetTeamInstruction(ctx, teamID)
			Expect(err).NotTo(HaveOccurred())
			Expect(inst).NotTo(BeNil())
			Expect(len(logs)).To(Equal(1))
		})

		It("should update existing instruction in transaction", func() {
			teamID := uuid.New()
			userID := uuid.New()
			instID := uuid.New()
			inst := &models.Instruction{
				TeamID:             teamID,
				CreatedBy:          userID,
				InstructionDetails: "Updated instruction",
			}
			log := &models.InstructionLog{
				OlderInstruction: "Old instruction",
				Version:          1,
			}

			mock.ExpectBegin()
			mock.ExpectQuery(`SELECT \* FROM "instructions" WHERE team_id = \$1 ORDER BY "instructions"\."id" LIMIT \$2`).
				WithArgs(teamID, 1).
				WillReturnRows(sqlmock.NewRows([]string{"id", "team_id"}).AddRow(instID, teamID))

			mock.ExpectQuery(`INSERT INTO "instruction_logs"`).
				WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New()))

			mock.ExpectExec(`UPDATE "instructions" SET`).
				WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectCommit()

			err := teamRepo.SaveTeamInstruction(ctx, inst, log)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("Incident & Team Queries", func() {
		It("should fetch team by ID", func() {
			teamID := uuid.New()
			rows := sqlmock.NewRows([]string{"id", "team_name"}).AddRow(teamID, "Core Infra")

			mock.ExpectQuery(`SELECT \* FROM "teams" WHERE id = \$1 ORDER BY "teams"\."id" LIMIT \$2`).
				WithArgs(teamID, 1).
				WillReturnRows(rows)

			mock.ExpectQuery(`SELECT \* FROM "team_members" WHERE "team_members"\."team_id" = \$1`).
				WithArgs(teamID).
				WillReturnRows(sqlmock.NewRows([]string{"id", "team_id"}))

			team, err := teamRepo.GetTeamByID(ctx, teamID)
			Expect(err).NotTo(HaveOccurred())
			Expect(team.ID).To(Equal(teamID))
		})

		It("should assign team incident in transaction", func() {
			incID := uuid.New()
			teamID := uuid.New()
			creatorID := uuid.New()
			inc := &models.TeamIncident{
				ID:             incID,
				TeamID:         teamID,
				CreatedBy:      creatorID,
				Title:          "Memory Leak",
				Status:         "OPEN",
				IncidentNumber: "INC-101",
			}

			mock.ExpectBegin()
			mock.ExpectQuery(`INSERT INTO "team_incidents"`).
				WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(incID))
			mock.ExpectQuery(`INSERT INTO "incident_status_histories"`).
				WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New()))
			mock.ExpectCommit()

			err := teamRepo.AssignTeamIncident(ctx, inc)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should list team incidents", func() {
			teamID := uuid.New()
			mock.ExpectQuery(`SELECT \* FROM "team_incidents" WHERE team_id = \$1 ORDER BY assigned_at DESC`).
				WithArgs(teamID).
				WillReturnRows(sqlmock.NewRows([]string{"id", "title"}).AddRow(uuid.New(), "High CPU"))

			mock.ExpectQuery(`SELECT \* FROM "incident_status_histories" WHERE "incident_status_histories"\."team_incident_id" = \$1 ORDER BY updated_at DESC`).
				WillReturnRows(sqlmock.NewRows([]string{"id"}))

			incidents, err := teamRepo.ListTeamIncidents(ctx, teamID)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(incidents)).To(Equal(1))
		})

		It("should get team incident by ID or surrogate key INC-101", func() {
			incID := uuid.New()
			mock.ExpectQuery(`SELECT \* FROM "team_incidents" WHERE LOWER\(incident_number\) = LOWER\(\$1\) ORDER BY "team_incidents"\."id" LIMIT \$2`).
				WithArgs("INC-101", 1).
				WillReturnRows(sqlmock.NewRows([]string{"id", "incident_number"}).AddRow(incID, "INC-101"))

			mock.ExpectQuery(`SELECT \* FROM "incident_status_histories" WHERE "incident_status_histories"\."team_incident_id" = \$1 ORDER BY updated_at DESC`).
				WithArgs(incID).
				WillReturnRows(sqlmock.NewRows([]string{"id"}))

			inc, err := teamRepo.GetTeamIncidentByIDOrNumber(ctx, "INC-101")
			Expect(err).NotTo(HaveOccurred())
			Expect(inc.IncidentNumber).To(Equal("INC-101"))
		})

		It("should get team incident by ID", func() {
			incID := uuid.New()
			mock.ExpectQuery(`SELECT \* FROM "team_incidents" WHERE id = \$1 ORDER BY "team_incidents"\."id" LIMIT \$2`).
				WithArgs(incID, 1).
				WillReturnRows(sqlmock.NewRows([]string{"id", "title"}).AddRow(incID, "Incident 1"))

			mock.ExpectQuery(`SELECT \* FROM "incident_status_histories" WHERE "incident_status_histories"\."team_incident_id" = \$1 ORDER BY updated_at DESC`).
				WithArgs(incID).
				WillReturnRows(sqlmock.NewRows([]string{"id"}))

			inc, err := teamRepo.GetTeamIncidentByID(ctx, incID)
			Expect(err).NotTo(HaveOccurred())
			Expect(inc.ID).To(Equal(incID))
		})

		It("should update team incident status in transaction", func() {
			incID := uuid.New()
			history := &models.IncidentStatusHistory{
				ID:             uuid.New(),
				TeamIncidentID: incID,
				NewStatus:      "RESOLVED",
			}
			inc := &models.TeamIncident{
				ID:     incID,
				Status: "RESOLVED",
			}

			mock.ExpectBegin()
			mock.ExpectExec(`UPDATE "team_incidents" SET`).
				WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectQuery(`INSERT INTO "incident_status_histories"`).
				WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(history.ID))
			mock.ExpectCommit()

			err := teamRepo.UpdateTeamIncidentStatus(ctx, history, inc)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should list all teams", func() {
			mock.ExpectQuery(`SELECT \* FROM "teams" ORDER BY team_name ASC`).
				WillReturnRows(sqlmock.NewRows([]string{"id", "team_name"}).AddRow(uuid.New(), "Core Infra"))

			teams, err := teamRepo.ListAllTeams(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(teams)).To(Equal(1))
		})
	})
})
