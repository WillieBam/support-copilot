package service_test

import (
	"context"
	"errors"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"

	"github.com/WillieBam/support_copilot/backend/internal/interfaces"
	"github.com/WillieBam/support_copilot/backend/internal/mocks"
	"github.com/WillieBam/support_copilot/backend/internal/service"
	"github.com/WillieBam/support_copilot/backend/types/models"
	customErrors "github.com/WillieBam/support_copilot/backend/utils/errors"
)

var _ = Describe("TeamService", func() {
	var (
		teamSvc  interfaces.ITeamService
		teamRepo *mocks.ITeamRepository
		ctx      context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		teamRepo = &mocks.ITeamRepository{}
		teamSvc = service.NewTeamService(teamRepo)
	})

	AfterEach(func() {
		teamRepo.AssertExpectations(GinkgoT())
	})

	Context("CreateTeam", func() {
		It("should fail if team name is empty", func() {
			team, err := teamSvc.CreateTeam(ctx, "   ", uuid.New())
			Expect(err).To(Equal(customErrors.ErrTeamNameRequired))
			Expect(team).To(BeNil())
		})

		It("should fail if team name is longer than 20 characters", func() {
			longName := "ThisTeamNameIsWayTooLongForConstraint"
			team, err := teamSvc.CreateTeam(ctx, longName, uuid.New())
			Expect(err).To(Equal(customErrors.ErrTeamNameTooLong))
			Expect(team).To(BeNil())
		})

		It("should succeed when team name is valid", func() {
			creatorID := uuid.New()
			teamRepo.On("CreateTeamWithOwner", ctx, mock.AnythingOfType("*models.Team"), creatorID).Return(nil)

			team, err := teamSvc.CreateTeam(ctx, "DevOps", creatorID)
			Expect(err).NotTo(HaveOccurred())
			Expect(team).NotTo(BeNil())
			Expect(team.TeamName).To(Equal("DevOps"))
		})

		It("should return error if repo fails to create team", func() {
			creatorID := uuid.New()
			teamRepo.On("CreateTeamWithOwner", ctx, mock.AnythingOfType("*models.Team"), creatorID).Return(errors.New("db error"))

			team, err := teamSvc.CreateTeam(ctx, "DevOps", creatorID)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("db error"))
			Expect(team).To(BeNil())
		})
	})

	Context("GetTeam", func() {
		It("should fetch team by ID", func() {
			teamID := uuid.New()
			expectedTeam := &models.Team{ID: teamID, TeamName: "SRE Core"}
			teamRepo.On("GetTeamByID", ctx, teamID).Return(expectedTeam, nil)

			team, err := teamSvc.GetTeam(ctx, teamID)
			Expect(err).NotTo(HaveOccurred())
			Expect(team).To(Equal(expectedTeam))
		})
	})

	Context("GetUserTeams", func() {
		It("should fetch user with teams", func() {
			userID := uuid.New()
			expectedUser := &models.User{ID: userID, Email: "user@test.com"}
			teamRepo.On("GetUserWithTeamsByID", ctx, userID).Return(expectedUser, nil)

			user, err := teamSvc.GetUserTeams(ctx, userID)
			Expect(err).NotTo(HaveOccurred())
			Expect(user).To(Equal(expectedUser))
		})
	})

	Context("AddMember", func() {
		var (
			teamID      uuid.UUID
			requesterID uuid.UUID
			targetID    uuid.UUID
		)

		BeforeEach(func() {
			teamID = uuid.New()
			requesterID = uuid.New()
			targetID = uuid.New()
		})

		It("should fail if requester is not team owner", func() {
			teamRepo.On("GetMemberRole", ctx, teamID, requesterID).Return("member", nil)

			err := teamSvc.AddMember(ctx, requesterID, teamID, targetID)
			Expect(err).To(Equal(customErrors.ErrUnauthorizedTeamOp))
		})

		It("should succeed and assign member role when requester is team owner", func() {
			teamRepo.On("GetMemberRole", ctx, teamID, requesterID).Return("owner", nil)
			teamRepo.On("AddTeamMember", ctx, mock.MatchedBy(func(m *models.TeamMember) bool {
				return m.TeamID == teamID && m.UserID == targetID && m.Role == "member"
			})).Return(nil)

			err := teamSvc.AddMember(ctx, requesterID, teamID, targetID)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("RemoveMember", func() {
		var (
			teamID      uuid.UUID
			ownerID     uuid.UUID
			memberID    uuid.UUID
			nonMemberID uuid.UUID
		)

		BeforeEach(func() {
			teamID = uuid.New()
			ownerID = uuid.New()
			memberID = uuid.New()
			nonMemberID = uuid.New()
		})

		It("should fail if requester is not owner", func() {
			teamRepo.On("GetMemberRole", ctx, teamID, memberID).Return("member", nil)

			err := teamSvc.RemoveMember(ctx, memberID, teamID, uuid.New())
			Expect(err).To(Equal(customErrors.ErrUnauthorizedTeamOp))
		})

		It("should fail if target user is not in team", func() {
			teamRepo.On("GetMemberRole", ctx, teamID, ownerID).Return("owner", nil)
			teamRepo.On("GetMemberRole", ctx, teamID, nonMemberID).Return("", gorm.ErrRecordNotFound)

			err := teamSvc.RemoveMember(ctx, ownerID, teamID, nonMemberID)
			Expect(err).To(Equal(customErrors.ErrUserNotInTeam))
		})

		It("should succeed when owner removes a member", func() {
			teamRepo.On("GetMemberRole", ctx, teamID, ownerID).Return("owner", nil)
			teamRepo.On("GetMemberRole", ctx, teamID, memberID).Return("member", nil)
			teamRepo.On("RemoveTeamMember", ctx, teamID, memberID).Return(nil)

			err := teamSvc.RemoveMember(ctx, ownerID, teamID, memberID)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("DeleteTeam", func() {
		var teamID uuid.UUID

		BeforeEach(func() {
			teamID = uuid.New()
		})

		It("should fail if user scope is not super_admin", func() {
			err := teamSvc.DeleteTeam(ctx, "engineer", teamID)
			Expect(err).To(Equal(customErrors.ErrSuperAdminRequired))
		})

		It("should succeed if user scope is super_admin", func() {
			teamRepo.On("DeleteTeam", ctx, teamID).Return(nil)

			err := teamSvc.DeleteTeam(ctx, "super_admin", teamID)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("AssignIncident & ListIncidents", func() {
		var (
			teamID uuid.UUID
			userID uuid.UUID
		)

		BeforeEach(func() {
			teamID = uuid.New()
			userID = uuid.New()
		})

		It("should fail AssignIncident if user is not in team", func() {
			teamRepo.On("GetMemberRole", ctx, teamID, userID).Return("", errors.New("not member"))

			inc, err := teamSvc.AssignIncident(ctx, userID, teamID, "High Latency", "OPEN", "Details")
			Expect(err).To(Equal(customErrors.ErrUnauthorizedTeamOp))
			Expect(inc).To(BeNil())
		})

		It("should succeed AssignIncident when user is in team", func() {
			teamRepo.On("GetMemberRole", ctx, teamID, userID).Return("member", nil)
			teamRepo.On("AssignTeamIncident", ctx, mock.MatchedBy(func(inc *models.TeamIncident) bool {
				return inc.TeamID == teamID && inc.Title == "High Latency"
			})).Return(nil)

			inc, err := teamSvc.AssignIncident(ctx, userID, teamID, "High Latency", "OPEN", "Details")
			Expect(err).NotTo(HaveOccurred())
			Expect(inc).NotTo(BeNil())
			Expect(inc.Title).To(Equal("High Latency"))
		})

		It("should ListIncidents when user is in team", func() {
			teamRepo.On("GetMemberRole", ctx, teamID, userID).Return("member", nil)
			expectedIncidents := []models.TeamIncident{
				{ID: uuid.New(), TeamID: teamID, Title: "Incident 1"},
			}
			teamRepo.On("ListTeamIncidents", ctx, teamID).Return(expectedIncidents, nil)

			incidents, err := teamSvc.ListIncidents(ctx, userID, teamID)
			Expect(err).NotTo(HaveOccurred())
			Expect(incidents).To(Equal(expectedIncidents))
		})
	})

	Context("GetIncident & UpdateIncidentStatus", func() {
		var (
			teamID     uuid.UUID
			userID     uuid.UUID
			incidentID uuid.UUID
		)

		BeforeEach(func() {
			teamID = uuid.New()
			userID = uuid.New()
			incidentID = uuid.New()
		})

		It("should fail GetIncident if requester is not in team", func() {
			mockInc := &models.TeamIncident{ID: incidentID, TeamID: teamID, Title: "Database Slow"}
			teamRepo.On("GetTeamIncidentByID", ctx, incidentID).Return(mockInc, nil)
			teamRepo.On("GetMemberRole", ctx, teamID, userID).Return("", errors.New("not member"))

			inc, err := teamSvc.GetIncident(ctx, userID, incidentID)
			Expect(err).To(Equal(customErrors.ErrUnauthorizedTeamOp))
			Expect(inc).To(BeNil())
		})

		It("should succeed GetIncident when requester is in team", func() {
			mockInc := &models.TeamIncident{ID: incidentID, TeamID: teamID, Title: "Database Slow"}
			teamRepo.On("GetTeamIncidentByID", ctx, incidentID).Return(mockInc, nil)
			teamRepo.On("GetMemberRole", ctx, teamID, userID).Return("member", nil)

			inc, err := teamSvc.GetIncident(ctx, userID, incidentID)
			Expect(err).NotTo(HaveOccurred())
			Expect(inc).To(Equal(mockInc))
		})

		It("should fail UpdateIncidentStatus when status is invalid", func() {
			inc, err := teamSvc.UpdateIncidentStatus(ctx, userID, incidentID, "INVALID_STATUS", "Title", "Details")
			Expect(err).To(Equal(customErrors.ErrInvalidIncidentStatus))
			Expect(inc).To(BeNil())
		})

		It("should succeed UpdateIncidentStatus and log status transition", func() {
			mockInc := &models.TeamIncident{
				ID:      incidentID,
				TeamID:  teamID,
				Title:   "Memory Leak",
				Status:  "OPEN",
				Details: "Initial details",
			}
			updatedInc := &models.TeamIncident{
				ID:      incidentID,
				TeamID:  teamID,
				Title:   "Memory Leak",
				Status:  "IN_PROGRESS",
				Details: "Investigating stack dump",
				History: []models.IncidentStatusHistory{
					{
						TeamIncidentID: incidentID,
						PreviousStatus: "OPEN",
						NewStatus:      "IN_PROGRESS",
					},
				},
			}

			teamRepo.On("GetTeamIncidentByID", ctx, incidentID).Return(mockInc, nil).Once()
			teamRepo.On("GetMemberRole", ctx, teamID, userID).Return("member", nil)
			teamRepo.On("UpdateTeamIncidentStatus", ctx, mock.MatchedBy(func(h *models.IncidentStatusHistory) bool {
				return h.TeamIncidentID == incidentID && h.PreviousStatus == "OPEN" && h.NewStatus == "IN_PROGRESS" && h.UpdatedBy == userID
			}), mock.MatchedBy(func(inc *models.TeamIncident) bool {
				return inc.ID == incidentID && inc.Status == "IN_PROGRESS"
			})).Return(nil)
			teamRepo.On("GetTeamIncidentByID", ctx, incidentID).Return(updatedInc, nil).Once()

			inc, err := teamSvc.UpdateIncidentStatus(ctx, userID, incidentID, "IN_PROGRESS", "Memory Leak", "Investigating stack dump")
			Expect(err).NotTo(HaveOccurred())
			Expect(inc).NotTo(BeNil())
			Expect(inc.Status).To(Equal("IN_PROGRESS"))
			Expect(len(inc.History)).To(Equal(1))
			Expect(inc.History[0].PreviousStatus).To(Equal("OPEN"))
			Expect(inc.History[0].NewStatus).To(Equal("IN_PROGRESS"))
		})
	})

	Context("SaveTeamInstruction validation", func() {
		var (
			teamID uuid.UUID
			userID uuid.UUID
		)

		BeforeEach(func() {
			teamID = uuid.New()
			userID = uuid.New()
		})

		It("should fail SaveTeamInstruction when instruction details are under 30 characters", func() {
			teamRepo.On("GetMemberRole", ctx, teamID, userID).Return("owner", nil)

			// short instruction details under 30 characters
			shortDetails := "too short details"
			inst, err := teamSvc.SaveTeamInstruction(ctx, userID, teamID, shortDetails)
			Expect(err).To(Equal(customErrors.ErrInstructionTooShort))
			Expect(inst).To(BeNil())
		})

		It("should succeed SaveTeamInstruction when instruction details meet 30 characters threshold", func() {
			teamRepo.On("GetMemberRole", ctx, teamID, userID).Return("owner", nil)
			teamRepo.On("GetTeamInstruction", ctx, teamID).Return((*models.Instruction)(nil), ([]models.InstructionLog)(nil), nil).Once()

			validDetails := "this is a valid team instruction string that has more than 30 characters"
			teamRepo.On("SaveTeamInstruction", ctx, mock.Anything, mock.Anything).Return(nil)

			expectedInst := &models.Instruction{
				ID:                 uuid.New(),
				TeamID:             teamID,
				CreatedBy:          userID,
				InstructionDetails: validDetails,
			}
			teamRepo.On("GetTeamInstruction", ctx, teamID).Return(expectedInst, ([]models.InstructionLog)(nil), nil).Once()

			inst, err := teamSvc.SaveTeamInstruction(ctx, userID, teamID, validDetails)
			Expect(err).NotTo(HaveOccurred())
			Expect(inst).NotTo(BeNil())
			Expect(inst.InstructionDetails).To(Equal(validDetails))
		})
	})

	Context("Runbook Service Operations", func() {
		var (
			teamID     uuid.UUID
			userID     uuid.UUID
			incidentID uuid.UUID
			runbookID  uuid.UUID
		)

		BeforeEach(func() {
			teamID = uuid.New()
			userID = uuid.New()
			incidentID = uuid.New()
			runbookID = uuid.New()
		})

		It("should CreateRunbook successfully", func() {
			teamRepo.On("CreateRunbook", ctx, mock.MatchedBy(func(rb *models.Runbook) bool {
				return rb.TeamID == teamID && rb.CreatedBy == userID && rb.Title == "Pod Restart Guide"
			})).Return(nil)

			rb, err := teamSvc.CreateRunbook(ctx, userID, teamID, incidentID, "Pod Restart Guide", "kubectl rollout restart")
			Expect(err).NotTo(HaveOccurred())
			Expect(rb).NotTo(BeNil())
			Expect(rb.Title).To(Equal("Pod Restart Guide"))
		})

		It("should UpdateRunbook and generate version log entry", func() {
			existingRb := &models.Runbook{
				ID:         runbookID,
				TeamID:     teamID,
				IncidentID: incidentID,
				Title:      "Original Title",
				Content:    "Original Content",
			}
			existingLogs := []models.RunbookLog{}

			teamRepo.On("GetRunbookLogs", ctx, runbookID).Return(existingLogs, nil)
			teamRepo.On("GetRunbookByID", ctx, runbookID).Return(existingRb, nil)

			updatedRb := &models.Runbook{
				ID:      runbookID,
				Title:   "Updated Title",
				Content: "Updated Content",
			}
			teamRepo.On("UpdateRunbook", ctx, runbookID, "Updated Title", "Updated Content", mock.MatchedBy(func(l *models.RunbookLog) bool {
				return l.RunbookID == runbookID && l.Version == 1 && l.OlderTitle == "Original Title"
			})).Return(updatedRb, nil)

			rb, err := teamSvc.UpdateRunbook(ctx, userID, runbookID, "Updated Title", "Updated Content")
			Expect(err).NotTo(HaveOccurred())
			Expect(rb).NotTo(BeNil())
			Expect(rb.Title).To(Equal("Updated Title"))
		})

		It("should GetRunbookLogs successfully", func() {
			logs := []models.RunbookLog{{ID: uuid.New(), RunbookID: runbookID, Version: 1}}
			teamRepo.On("GetRunbookLogs", ctx, runbookID).Return(logs, nil)

			res, err := teamSvc.GetRunbookLogs(ctx, runbookID)
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal(logs))
		})

		It("should DeprecateRunbook successfully", func() {
			deprecatedRb := &models.Runbook{ID: runbookID, Status: "deprecated"}
			teamRepo.On("DeprecateRunbook", ctx, runbookID).Return(deprecatedRb, nil)

			rb, err := teamSvc.DeprecateRunbook(ctx, runbookID)
			Expect(err).NotTo(HaveOccurred())
			Expect(rb.Status).To(Equal("deprecated"))
		})

		It("should GetIncidentContext successfully", func() {
			inc := &models.TeamIncident{ID: incidentID, Title: "Redis Failure"}
			alerts := []models.Alert{{ServiceName: "redis-service"}}

			teamRepo.On("GetIncidentContext", ctx, incidentID).Return(inc, alerts, nil)

			resInc, resAlerts, err := teamSvc.GetIncidentContext(ctx, incidentID)
			Expect(err).NotTo(HaveOccurred())
			Expect(resInc.Title).To(Equal("Redis Failure"))
			Expect(len(resAlerts)).To(Equal(1))
		})
	})
})
