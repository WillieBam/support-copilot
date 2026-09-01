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
		It("should query user with team preloads and memberships", func() {
			userID := uuid.New()
			memID := uuid.New()
			teamID := uuid.New()
			now := time.Now()
			role := "owner"
			teamName := "Platform"

			rows := sqlmock.NewRows([]string{"user_id", "firebase_uid", "username", "email", "display_name", "user_created_at", "deactivated_at", "scope", "membership_id", "team_id", "membership_role", "team_name", "team_created_at"}).
				AddRow(userID, "uid-123", "uname", "user@test.com", "User Name", now, nil, "engineer", &memID, &teamID, &role, &teamName, &now)

			mock.ExpectQuery(`SELECT (.+) FROM users u LEFT JOIN team_members tm ON (.+) LEFT JOIN teams t ON (.+) WHERE u\.id = \$1`).
				WithArgs(userID).
				WillReturnRows(rows)

			user, err := teamRepo.GetUserWithTeamsByID(ctx, userID)
			Expect(err).NotTo(HaveOccurred())
			Expect(user).NotTo(BeNil())
			Expect(user.ID).To(Equal(userID))
			Expect(len(user.Memberships)).To(Equal(1))
			Expect(user.Memberships[0].Role).To(Equal("owner"))
			Expect(user.Memberships[0].Team.TeamName).To(Equal("Platform"))
		})

		It("should return record not found error when user does not exist", func() {
			userID := uuid.New()
			mock.ExpectQuery(`SELECT (.+) FROM users u LEFT JOIN team_members tm ON (.+) LEFT JOIN teams t ON (.+) WHERE u\.id = \$1`).
				WithArgs(userID).
				WillReturnRows(sqlmock.NewRows([]string{"user_id"}))

			user, err := teamRepo.GetUserWithTeamsByID(ctx, userID)
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, gorm.ErrRecordNotFound)).To(BeTrue())
			Expect(user).To(BeNil())
		})

		It("should return query error when database fails", func() {
			userID := uuid.New()
			mock.ExpectQuery(`SELECT (.+) FROM users u LEFT JOIN team_members tm ON (.+) LEFT JOIN teams t ON (.+) WHERE u\.id = \$1`).
				WithArgs(userID).
				WillReturnError(errors.New("connection reset"))

			user, err := teamRepo.GetUserWithTeamsByID(ctx, userID)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("connection reset"))
			Expect(user).To(BeNil())
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

		It("should return error if RemoveTeamMember fails", func() {
			teamID := uuid.New()
			userID := uuid.New()

			mock.ExpectBegin()
			mock.ExpectExec(`DELETE FROM "team_members" WHERE team_id = \$1 AND user_id = \$2`).
				WithArgs(teamID, userID).
				WillReturnError(errors.New("delete error"))
			mock.ExpectRollback()

			err := teamRepo.RemoveTeamMember(ctx, teamID, userID)
			Expect(err).To(HaveOccurred())
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

		It("should return error if DeleteTeam fails", func() {
			teamID := uuid.New()

			mock.ExpectBegin()
			mock.ExpectExec(`DELETE FROM "teams" WHERE id = \$1`).
				WithArgs(teamID).
				WillReturnError(errors.New("foreign key violation"))
			mock.ExpectRollback()

			err := teamRepo.DeleteTeam(ctx, teamID)
			Expect(err).To(HaveOccurred())
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

		It("should return record not found error when member not found in team", func() {
			teamID := uuid.New()
			userID := uuid.New()

			mock.ExpectQuery(`SELECT \* FROM "team_members" WHERE team_id = \$1 AND user_id = \$2 ORDER BY "team_members"\."id" LIMIT \$3`).
				WithArgs(teamID, userID, 1).
				WillReturnError(gorm.ErrRecordNotFound)

			role, err := teamRepo.GetMemberRole(ctx, teamID, userID)
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, gorm.ErrRecordNotFound)).To(BeTrue())
			Expect(role).To(Equal(""))
		})

		It("should return error when GetMemberRole query fails", func() {
			teamID := uuid.New()
			userID := uuid.New()

			mock.ExpectQuery(`SELECT \* FROM "team_members" WHERE team_id = \$1 AND user_id = \$2 ORDER BY "team_members"\."id" LIMIT \$3`).
				WithArgs(teamID, userID, 1).
				WillReturnError(errors.New("db error"))

			role, err := teamRepo.GetMemberRole(ctx, teamID, userID)
			Expect(err).To(HaveOccurred())
			Expect(role).To(Equal(""))
		})

		It("should list team members", func() {
			teamID := uuid.New()
			userID := uuid.New()
			rows := sqlmock.NewRows([]string{"id", "team_id", "user_id", "role", "user_email", "user_display_name", "user_scope"}).
				AddRow(uuid.New(), teamID, userID, "member", "user@test.com", "Test User", "engineer")

			mock.ExpectQuery(`SELECT (.+) FROM team_members tm JOIN users u ON (.+) WHERE tm\.team_id = \$1`).
				WithArgs(teamID).
				WillReturnRows(rows)

			members, err := teamRepo.ListTeamMembers(ctx, teamID)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(members)).To(Equal(1))
		})

		It("should return error when ListTeamMembers query fails", func() {
			teamID := uuid.New()
			mock.ExpectQuery(`SELECT (.+) FROM team_members tm JOIN users u ON (.+) WHERE tm\.team_id = \$1`).
				WithArgs(teamID).
				WillReturnError(errors.New("query error"))

			members, err := teamRepo.ListTeamMembers(ctx, teamID)
			Expect(err).To(HaveOccurred())
			Expect(members).To(BeNil())
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

		It("should return error when GetTeamInstruction query fails", func() {
			teamID := uuid.New()

			mock.ExpectQuery(`SELECT \* FROM "instructions" WHERE team_id = \$1 ORDER BY "instructions"\."id" LIMIT \$2`).
				WithArgs(teamID, 1).
				WillReturnError(errors.New("db error"))

			inst, logs, err := teamRepo.GetTeamInstruction(ctx, teamID)
			Expect(err).To(HaveOccurred())
			Expect(inst).To(BeNil())
			Expect(logs).To(BeNil())
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
		It("should fetch team by ID with members", func() {
			teamID := uuid.New()
			memberID := uuid.New()
			userID := uuid.New()
			now := time.Now()
			role := "member"
			email := "eng@test.com"
			displayName := "Engineer"
			scope := "engineer"

			rows := sqlmock.NewRows([]string{"team_id", "team_name", "team_created_at", "member_id", "member_user_id", "member_role", "user_email", "user_display_name", "user_scope"}).
				AddRow(teamID, "Core Infra", now, &memberID, &userID, &role, &email, &displayName, &scope)

			mock.ExpectQuery(`SELECT (.+) FROM teams t LEFT JOIN team_members tm ON (.+) LEFT JOIN users u ON (.+) WHERE t\.id = \$1`).
				WithArgs(teamID).
				WillReturnRows(rows)

			team, err := teamRepo.GetTeamByID(ctx, teamID)
			Expect(err).NotTo(HaveOccurred())
			Expect(team.ID).To(Equal(teamID))
			Expect(len(team.Members)).To(Equal(1))
			Expect(team.Members[0].User.Email).To(Equal("eng@test.com"))
		})

		It("should return record not found when team ID does not exist", func() {
			teamID := uuid.New()
			mock.ExpectQuery(`SELECT (.+) FROM teams t LEFT JOIN team_members tm ON (.+) LEFT JOIN users u ON (.+) WHERE t\.id = \$1`).
				WithArgs(teamID).
				WillReturnRows(sqlmock.NewRows([]string{"team_id"}))

			team, err := teamRepo.GetTeamByID(ctx, teamID)
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, gorm.ErrRecordNotFound)).To(BeTrue())
			Expect(team).To(BeNil())
		})

		It("should return error when GetTeamByID query fails", func() {
			teamID := uuid.New()
			mock.ExpectQuery(`SELECT (.+) FROM teams t LEFT JOIN team_members tm ON (.+) LEFT JOIN users u ON (.+) WHERE t\.id = \$1`).
				WithArgs(teamID).
				WillReturnError(errors.New("db error"))

			team, err := teamRepo.GetTeamByID(ctx, teamID)
			Expect(err).To(HaveOccurred())
			Expect(team).To(BeNil())
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

		It("should assign team incident and generate incident number when empty", func() {
			incID := uuid.New()
			teamID := uuid.New()
			creatorID := uuid.New()
			inc := &models.TeamIncident{
				ID:        incID,
				TeamID:    teamID,
				CreatedBy: creatorID,
				Title:     "Memory Leak",
				Status:    "OPEN",
			}

			mock.ExpectBegin()
			mock.ExpectQuery(`SELECT \* FROM "team_incidents" WHERE incident_number IS NOT NULL AND incident_number != '' ORDER BY incident_number DESC,"team_incidents"\."id" LIMIT \$1`).
				WithArgs(1).
				WillReturnRows(sqlmock.NewRows([]string{"id", "incident_number"}).AddRow(uuid.New(), "INC-105"))
			mock.ExpectQuery(`INSERT INTO "team_incidents"`).
				WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(incID))
			mock.ExpectQuery(`INSERT INTO "incident_status_histories"`).
				WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New()))
			mock.ExpectCommit()

			err := teamRepo.AssignTeamIncident(ctx, inc)
			Expect(err).NotTo(HaveOccurred())
			Expect(inc.IncidentNumber).To(Equal("INC-106"))
		})

		It("should rollback if AssignTeamIncident fails on incident insert", func() {
			incID := uuid.New()
			teamID := uuid.New()
			creatorID := uuid.New()
			inc := &models.TeamIncident{
				ID:             incID,
				TeamID:         teamID,
				CreatedBy:      creatorID,
				Title:          "Memory Leak",
				IncidentNumber: "INC-101",
			}

			mock.ExpectBegin()
			mock.ExpectQuery(`INSERT INTO "team_incidents"`).
				WillReturnError(errors.New("insert incident error"))
			mock.ExpectRollback()

			err := teamRepo.AssignTeamIncident(ctx, inc)
			Expect(err).To(HaveOccurred())
		})

		It("should rollback if AssignTeamIncident fails on history insert", func() {
			incID := uuid.New()
			teamID := uuid.New()
			creatorID := uuid.New()
			inc := &models.TeamIncident{
				ID:             incID,
				TeamID:         teamID,
				CreatedBy:      creatorID,
				Title:          "Memory Leak",
				IncidentNumber: "INC-101",
			}

			mock.ExpectBegin()
			mock.ExpectQuery(`INSERT INTO "team_incidents"`).
				WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(incID))
			mock.ExpectQuery(`INSERT INTO "incident_status_histories"`).
				WillReturnError(errors.New("insert history error"))
			mock.ExpectRollback()

			err := teamRepo.AssignTeamIncident(ctx, inc)
			Expect(err).To(HaveOccurred())
		})

		It("should list team incidents with histories", func() {
			teamID := uuid.New()
			incID := uuid.New()
			historyID := uuid.New()
			now := time.Now()
			updatedBy := uuid.New()
			hTitle := "Initial status"
			newStatus := "OPEN"
			prevStatus := ""
			hDetails := "Incident opened"

			rows := sqlmock.NewRows([]string{"incident_id", "incident_number", "team_id", "created_by", "title", "status", "details", "created_at", "assigned_at", "resolved_at", "history_id", "history_updated_by", "history_title", "history_new_status", "history_previous_status", "history_details", "history_updated_at"}).
				AddRow(incID, "INC-100", teamID, uuid.New(), "High CPU", "OPEN", "CPU spike", now, now, nil, &historyID, &updatedBy, &hTitle, &newStatus, &prevStatus, &hDetails, &now)

			mock.ExpectQuery(`SELECT (.+) FROM team_incidents i LEFT JOIN incident_status_histories h ON (.+) WHERE i\.team_id = \$1`).
				WithArgs(teamID).
				WillReturnRows(rows)

			incidents, err := teamRepo.ListTeamIncidents(ctx, teamID)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(incidents)).To(Equal(1))
			Expect(len(incidents[0].History)).To(Equal(1))
			Expect(incidents[0].History[0].NewStatus).To(Equal("OPEN"))
		})

		It("should return error when ListTeamIncidents query fails", func() {
			teamID := uuid.New()
			mock.ExpectQuery(`SELECT (.+) FROM team_incidents i LEFT JOIN incident_status_histories h ON (.+) WHERE i\.team_id = \$1`).
				WithArgs(teamID).
				WillReturnError(errors.New("db failure"))

			incidents, err := teamRepo.ListTeamIncidents(ctx, teamID)
			Expect(err).To(HaveOccurred())
			Expect(incidents).To(BeNil())
		})

		It("should get team incident by ID or surrogate key INC-101", func() {
			incID := uuid.New()
			teamID := uuid.New()
			now := time.Now()
			rows := sqlmock.NewRows([]string{"incident_id", "incident_number", "team_id", "created_by", "title", "status", "details", "created_at", "assigned_at", "resolved_at", "history_id", "history_updated_by", "history_title", "history_new_status", "history_previous_status", "history_details", "history_updated_at"}).
				AddRow(incID, "INC-101", teamID, uuid.New(), "Memory Leak", "OPEN", "Details", now, now, nil, nil, nil, nil, nil, nil, nil, nil)

			mock.ExpectQuery(`SELECT (.+) FROM team_incidents i LEFT JOIN incident_status_histories h ON (.+) WHERE LOWER\(i\.incident_number\) = LOWER\(\$1\)`).
				WithArgs("INC-101").
				WillReturnRows(rows)

			inc, err := teamRepo.GetTeamIncidentByIDOrNumber(ctx, "INC-101")
			Expect(err).NotTo(HaveOccurred())
			Expect(inc.IncidentNumber).To(Equal("INC-101"))
		})

		It("should return record not found when incident not found by surrogate key", func() {
			mock.ExpectQuery(`SELECT (.+) FROM team_incidents i LEFT JOIN incident_status_histories h ON (.+) WHERE LOWER\(i\.incident_number\) = LOWER\(\$1\)`).
				WithArgs("INC-404").
				WillReturnRows(sqlmock.NewRows([]string{"incident_id"}))

			inc, err := teamRepo.GetTeamIncidentByIDOrNumber(ctx, "INC-404")
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, gorm.ErrRecordNotFound)).To(BeTrue())
			Expect(inc).To(BeNil())
		})

		It("should get team incident by ID", func() {
			incID := uuid.New()
			teamID := uuid.New()
			now := time.Now()
			rows := sqlmock.NewRows([]string{"incident_id", "incident_number", "team_id", "created_by", "title", "status", "details", "created_at", "assigned_at", "resolved_at", "history_id", "history_updated_by", "history_title", "history_new_status", "history_previous_status", "history_details", "history_updated_at"}).
				AddRow(incID, "INC-102", teamID, uuid.New(), "Incident 1", "OPEN", "Details", now, now, nil, nil, nil, nil, nil, nil, nil, nil)

			mock.ExpectQuery(`SELECT (.+) FROM team_incidents i LEFT JOIN incident_status_histories h ON (.+) WHERE i\.id = \$1`).
				WithArgs(incID).
				WillReturnRows(rows)

			inc, err := teamRepo.GetTeamIncidentByID(ctx, incID)
			Expect(err).NotTo(HaveOccurred())
			Expect(inc.ID).To(Equal(incID))
		})

		It("should return record not found when incident ID does not exist", func() {
			incID := uuid.New()
			mock.ExpectQuery(`SELECT (.+) FROM team_incidents i LEFT JOIN incident_status_histories h ON (.+) WHERE i\.id = \$1`).
				WithArgs(incID).
				WillReturnRows(sqlmock.NewRows([]string{"incident_id"}))

			inc, err := teamRepo.GetTeamIncidentByID(ctx, incID)
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, gorm.ErrRecordNotFound)).To(BeTrue())
			Expect(inc).To(BeNil())
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

		It("should rollback if UpdateTeamIncidentStatus fails", func() {
			incID := uuid.New()
			history := &models.IncidentStatusHistory{
				TeamIncidentID: incID,
				NewStatus:      "RESOLVED",
			}
			inc := &models.TeamIncident{
				ID: incID,
			}

			mock.ExpectBegin()
			mock.ExpectExec(`UPDATE "team_incidents" SET`).
				WillReturnError(errors.New("update error"))
			mock.ExpectRollback()

			err := teamRepo.UpdateTeamIncidentStatus(ctx, history, inc)
			Expect(err).To(HaveOccurred())
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
