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

		It("should return error when GetTeamByID fails", func() {
			teamID := uuid.New()
			teamRepo.On("GetTeamByID", ctx, teamID).Return(nil, errors.New("db error"))

			team, err := teamSvc.GetTeam(ctx, teamID)
			Expect(err).To(HaveOccurred())
			Expect(team).To(BeNil())
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

		It("should return error when GetUserWithTeamsByID fails", func() {
			userID := uuid.New()
			teamRepo.On("GetUserWithTeamsByID", ctx, userID).Return(nil, errors.New("db error"))

			user, err := teamSvc.GetUserTeams(ctx, userID)
			Expect(err).To(HaveOccurred())
			Expect(user).To(BeNil())
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

	Context("ListAllTeams", func() {
		It("should fail if user scope is not super_admin", func() {
			teams, err := teamSvc.ListAllTeams(ctx, "engineer")
			Expect(err).To(Equal(customErrors.ErrSuperAdminRequired))
			Expect(teams).To(BeNil())
		})

		It("should return all teams if user scope is super_admin", func() {
			expected := []models.Team{{ID: uuid.New(), TeamName: "DevOps"}}
			teamRepo.On("ListAllTeams", ctx).Return(expected, nil)

			teams, err := teamSvc.ListAllTeams(ctx, "super_admin")
			Expect(err).NotTo(HaveOccurred())
			Expect(teams).To(Equal(expected))
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
			teamRepo.On("GetUserWithTeamsByID", ctx, userID).Return(&models.User{Scope: "engineer"}, nil).Maybe()

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

		It("should fail AssignIncident when incident status is invalid", func() {
			teamRepo.On("GetMemberRole", ctx, teamID, userID).Return("member", nil).Once()

			_, err := teamSvc.AssignIncident(ctx, userID, teamID, "High Latency", "INVALID", "Details")
			Expect(err).To(Equal(customErrors.ErrInvalidIncidentStatus))
		})

		It("should ListIncidents when requester is in team", func() {
			teamRepo.On("GetMemberRole", ctx, teamID, userID).Return("member", nil)
			expectedIncidents := []models.TeamIncident{
				{ID: uuid.New(), TeamID: teamID, Title: "Incident 1"},
			}
			teamRepo.On("ListTeamIncidents", ctx, teamID).Return(expectedIncidents, nil)

			incidents, err := teamSvc.ListIncidents(ctx, userID, teamID)
			Expect(err).NotTo(HaveOccurred())
			Expect(incidents).To(Equal(expectedIncidents))
		})

		It("should fail ListIncidents when requester is not in team", func() {
			teamRepo.On("GetMemberRole", ctx, teamID, userID).Return("", errors.New("not member"))
			teamRepo.On("GetUserWithTeamsByID", ctx, userID).Return(&models.User{Scope: "engineer"}, nil).Maybe()

			incidents, err := teamSvc.ListIncidents(ctx, userID, teamID)
			Expect(err).To(Equal(customErrors.ErrUnauthorizedTeamOp))
			Expect(incidents).To(BeNil())
		})

		It("should fail GetIncident when GetTeamIncidentByID fails", func() {
			incID := uuid.New()
			teamRepo.On("GetTeamIncidentByID", ctx, incID).Return(nil, errors.New("db error")).Once()

			_, err := teamSvc.GetIncident(ctx, userID, incID)
			Expect(err).To(HaveOccurred())
		})

		It("should fail UpdateIncidentStatus when user is unauthorized", func() {
			incID := uuid.New()
			mockInc := &models.TeamIncident{ID: incID, TeamID: teamID, Title: "Memory Leak", Status: "OPEN"}
			teamRepo.On("GetTeamIncidentByID", ctx, incID).Return(mockInc, nil).Once()
			teamRepo.On("GetMemberRole", ctx, teamID, userID).Return("", errors.New("not member")).Once()
			teamRepo.On("GetUserWithTeamsByID", ctx, userID).Return(&models.User{Scope: "engineer"}, nil).Once()

			_, err := teamSvc.UpdateIncidentStatus(ctx, userID, incID, "IN_PROGRESS", "Memory Leak", "Investigating")
			Expect(err).To(Equal(customErrors.ErrUnauthorizedTeamOp))
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
			teamRepo.On("GetUserWithTeamsByID", ctx, userID).Return(&models.User{Scope: "engineer"}, nil).Maybe()

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
			teamRepo.On("GetTeamIncidentByIDOrNumber", ctx, incidentID.String()).Return(&models.TeamIncident{ID: incidentID, TeamID: teamID, CreatedBy: userID}, nil)
			teamRepo.On("GetTeamByID", ctx, teamID).Return(&models.Team{ID: teamID}, nil).Maybe()
			teamRepo.On("CreateRunbook", ctx, mock.MatchedBy(func(rb *models.Runbook) bool {
				return rb.TeamID == teamID && rb.CreatedBy == userID && rb.Title == "Pod Restart Guide"
			})).Return(nil)

			rb, err := teamSvc.CreateRunbook(ctx, userID, teamID, incidentID.String(), "Pod Restart Guide", "kubectl rollout restart")
			Expect(err).NotTo(HaveOccurred())
			Expect(rb).NotTo(BeNil())
			Expect(rb.Title).To(Equal("Pod Restart Guide"))
		})

		It("should CreateRunbook successfully using surrogate key INC-101", func() {
			teamRepo.On("GetTeamIncidentByIDOrNumber", ctx, "INC-101").Return(&models.TeamIncident{ID: incidentID, TeamID: teamID, CreatedBy: userID}, nil)
			teamRepo.On("GetTeamByID", ctx, teamID).Return(&models.Team{ID: teamID}, nil).Maybe()
			teamRepo.On("CreateRunbook", ctx, mock.MatchedBy(func(rb *models.Runbook) bool {
				return rb.TeamID == teamID && rb.CreatedBy == userID && rb.Title == "Pod Restart Guide" && *rb.IncidentID == incidentID
			})).Return(nil)

			rb, err := teamSvc.CreateRunbook(ctx, userID, teamID, "INC-101", "Pod Restart Guide", "kubectl rollout restart")
			Expect(err).NotTo(HaveOccurred())
			Expect(rb).NotTo(BeNil())
			Expect(rb.Title).To(Equal("Pod Restart Guide"))
			Expect(*rb.IncidentID).To(Equal(incidentID))
		})

		It("should UpdateRunbook and generate version log entry", func() {
			existingRb := &models.Runbook{
				ID:         runbookID,
				TeamID:     teamID,
				IncidentID: &incidentID,
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

		It("should fail CreateRunbook when incident is not found", func() {
			teamRepo.On("GetTeamIncidentByIDOrNumber", ctx, incidentID.String()).Return(nil, errors.New("inc not found"))
			_, err := teamSvc.CreateRunbook(ctx, userID, teamID, incidentID.String(), "title", "content")
			Expect(err).To(HaveOccurred())
		})

		It("should fail UpdateRunbook when runbook is not found", func() {
			teamRepo.On("GetRunbookLogs", ctx, runbookID).Return([]models.RunbookLog{}, nil)
			teamRepo.On("GetRunbookByID", ctx, runbookID).Return(nil, customErrors.ErrRunbookNotFound)
			_, err := teamSvc.UpdateRunbook(ctx, userID, runbookID, "Title", "Content")
			Expect(err).To(Equal(customErrors.ErrRunbookNotFound))
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
			alerts := []models.Alert{{ResourceInfo: `{"service":"redis-service"}`}}


			teamRepo.On("GetIncidentContext", ctx, incidentID).Return(inc, alerts, nil)

			resInc, resAlerts, err := teamSvc.GetIncidentContext(ctx, incidentID)
			Expect(err).NotTo(HaveOccurred())
			Expect(resInc.Title).To(Equal("Redis Failure"))
			Expect(len(resAlerts)).To(Equal(1))
		})

		It("should ListTeamIncidents and ListMembers successfully", func() {
			teamRepo.On("ListTeamIncidents", ctx, teamID).Return([]models.TeamIncident{{ID: incidentID}}, nil)
			teamRepo.On("GetMemberRole", ctx, teamID, userID).Return("member", nil)
			teamRepo.On("ListTeamMembers", ctx, teamID).Return([]models.TeamMember{{TeamID: teamID, UserID: userID}}, nil)

			incs, err := teamSvc.ListTeamIncidents(ctx, teamID)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(incs)).To(Equal(1))

			members, err := teamSvc.ListMembers(ctx, userID, teamID)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(members)).To(Equal(1))
		})

		It("should fail ListMembers and GetTeamInstruction when requester is not in team", func() {
			teamRepo.On("GetMemberRole", ctx, teamID, userID).Return("", errors.New("not member")).Twice()
			teamRepo.On("GetUserWithTeamsByID", ctx, userID).Return(&models.User{Scope: "engineer"}, nil).Maybe()

			members, err := teamSvc.ListMembers(ctx, userID, teamID)
			Expect(err).To(Equal(customErrors.ErrUnauthorizedTeamOp))
			Expect(members).To(BeNil())

			inst, logs, err := teamSvc.GetTeamInstruction(ctx, userID, teamID)
			Expect(err).To(Equal(customErrors.ErrUnauthorizedTeamOp))
			Expect(inst).To(BeNil())
			Expect(logs).To(BeNil())
		})

		It("should GetTeamInstruction and logs successfully", func() {
			inst := &models.Instruction{ID: uuid.New(), TeamID: teamID, InstructionDetails: "Follow instructions"}
			logs := []models.InstructionLog{{ID: uuid.New(), Version: 1}}
			teamRepo.On("GetMemberRole", ctx, teamID, userID).Return("member", nil)
			teamRepo.On("GetTeamInstruction", ctx, teamID).Return(inst, logs, nil)

			resInst, resLogs, err := teamSvc.GetTeamInstruction(ctx, userID, teamID)
			Expect(err).NotTo(HaveOccurred())
			Expect(resInst.InstructionDetails).To(Equal("Follow instructions"))
			Expect(len(resLogs)).To(Equal(1))
		})

		It("should GetRunbook and ListRunbooks successfully", func() {
			rb := &models.Runbook{ID: runbookID, TeamID: teamID, Title: "Runbook Title"}
			teamRepo.On("GetRunbookByID", ctx, runbookID).Return(rb, nil)
			teamRepo.On("ListRunbooks", ctx, teamID, "active").Return([]models.Runbook{*rb}, nil)

			resRb, err := teamSvc.GetRunbook(ctx, runbookID)
			Expect(err).NotTo(HaveOccurred())
			Expect(resRb.Title).To(Equal("Runbook Title"))

			rbs, err := teamSvc.ListRunbooks(ctx, teamID, "active")
			Expect(err).NotTo(HaveOccurred())
			Expect(len(rbs)).To(Equal(1))
		})

		It("should LinkAlertsToIncident and GetIncidentContextByIDOrNumber successfully", func() {
			alertID := uuid.New()
			mockAlertRepo := &mocks.IAlertRepository{}
			customTeamSvc := service.NewTeamService(teamRepo, mockAlertRepo)

			mockAlertRepo.On("RetrieveAlertbyID", ctx, alertID.String()).Return(&models.Alert{ID: alertID}, nil)
			mockAlertRepo.On("UpdateAlertIncidentID", ctx, alertID, incidentID).Return(nil)
			teamRepo.On("GetIncidentContextByIDOrNumber", ctx, "INC-101").Return(&models.TeamIncident{ID: incidentID, IncidentNumber: "INC-101"}, []models.Alert{{ID: alertID}}, nil)

			err := customTeamSvc.LinkAlertsToIncident(ctx, []string{alertID.String()}, incidentID)
			Expect(err).NotTo(HaveOccurred())

			inc, alerts, err := customTeamSvc.GetIncidentContextByIDOrNumber(ctx, "INC-101")
			Expect(err).NotTo(HaveOccurred())
			Expect(inc.IncidentNumber).To(Equal("INC-101"))
			Expect(len(alerts)).To(Equal(1))
		})

		It("should return error when CreateRunbook, DeprecateRunbook, GetIncidentContext, or GetIncidentContextByIDOrNumber repo fails", func() {
			teamRepo.On("GetTeamIncidentByIDOrNumber", ctx, incidentID.String()).Return(&models.TeamIncident{ID: incidentID, TeamID: teamID}, nil)
			teamRepo.On("GetTeamByID", ctx, teamID).Return(&models.Team{ID: teamID}, nil)
			teamRepo.On("CreateRunbook", ctx, mock.Anything).Return(errors.New("db error")).Once()
			_, err := teamSvc.CreateRunbook(ctx, userID, teamID, incidentID.String(), "Title", "Content")
			Expect(err).To(HaveOccurred())

			teamRepo.On("DeprecateRunbook", ctx, runbookID).Return(nil, errors.New("db error")).Once()
			_, err = teamSvc.DeprecateRunbook(ctx, runbookID)
			Expect(err).To(HaveOccurred())

			teamRepo.On("GetIncidentContext", ctx, incidentID).Return(nil, nil, errors.New("db error")).Once()
			_, _, err = teamSvc.GetIncidentContext(ctx, incidentID)
			Expect(err).To(HaveOccurred())

			teamRepo.On("GetIncidentContextByIDOrNumber", ctx, "INC-999").Return(nil, nil, errors.New("db error")).Once()
			_, _, err = teamSvc.GetIncidentContextByIDOrNumber(ctx, "INC-999")
			Expect(err).To(HaveOccurred())
		})

		It("should handle RetrieveAlertbyID and UpdateAlertIncidentID errors in LinkAlertsToIncident", func() {
			alertID1 := uuid.New()
			alertID2 := uuid.New()
			mockAlertRepo := &mocks.IAlertRepository{}
			customTeamSvc := service.NewTeamService(teamRepo, mockAlertRepo)

			mockAlertRepo.On("RetrieveAlertbyID", ctx, alertID1.String()).Return(nil, errors.New("alert not found")).Once()
			mockAlertRepo.On("RetrieveAlertbyID", ctx, alertID2.String()).Return(&models.Alert{ID: alertID2}, nil).Once()
			mockAlertRepo.On("UpdateAlertIncidentID", ctx, alertID2, incidentID).Return(errors.New("db update error")).Once()

			err := customTeamSvc.LinkAlertsToIncident(ctx, []string{alertID1.String(), alertID2.String(), " "}, incidentID)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return error when DeleteTeam, ListAllTeams, UpdateIncidentStatus, or ListRunbooks fails in repo", func() {
			teamRepo.On("DeleteTeam", ctx, teamID).Return(errors.New("db delete error")).Once()
			err := teamSvc.DeleteTeam(ctx, "super_admin", teamID)
			Expect(err).To(HaveOccurred())

			teamRepo.On("ListAllTeams", ctx).Return(nil, errors.New("db list error")).Once()
			_, err = teamSvc.ListAllTeams(ctx, "super_admin")
			Expect(err).To(HaveOccurred())

			mockInc := &models.TeamIncident{ID: incidentID, TeamID: teamID, Title: "Leak", Status: "OPEN"}
			teamRepo.On("GetTeamIncidentByID", ctx, incidentID).Return(mockInc, nil).Once()
			teamRepo.On("GetMemberRole", ctx, teamID, userID).Return("member", nil).Once()
			teamRepo.On("UpdateTeamIncidentStatus", ctx, mock.Anything, mock.Anything).Return(errors.New("db update error")).Once()
			_, err = teamSvc.UpdateIncidentStatus(ctx, userID, incidentID, "IN_PROGRESS", "Leak", "details")
			Expect(err).To(HaveOccurred())

			teamRepo.On("ListRunbooks", ctx, teamID, "active").Return(nil, errors.New("db runbooks error")).Once()
			_, err = teamSvc.ListRunbooks(ctx, teamID, "active")
			Expect(err).To(HaveOccurred())
		})

		It("should return error when SaveTeamInstruction or CreateRunbook fails in repository", func() {
			tID := uuid.New()
			uID := uuid.New()
			incID := uuid.New()
			rbID := uuid.New()

			existingInst := &models.Instruction{ID: uuid.New(), InstructionDetails: "old instruction details details"}
			teamRepo.On("GetMemberRole", ctx, tID, uID).Return("owner", nil).Once()
			teamRepo.On("GetTeamInstruction", ctx, tID).Return(existingInst, []models.InstructionLog{}, nil).Once()
			teamRepo.On("SaveTeamInstruction", ctx, mock.Anything, mock.Anything).Return(errors.New("instruction db err")).Once()
			_, err := teamSvc.SaveTeamInstruction(ctx, uID, tID, "This is a detailed instruction content that exceeds thirty characters.")
			Expect(err).To(HaveOccurred())

			teamRepo.On("ListTeamIncidents", ctx, tID).Return(nil, nil).Once()
			teamRepo.On("GetTeamByID", ctx, tID).Return(nil, errors.New("team not found")).Once()
			_, err = teamSvc.CreateRunbook(ctx, uID, tID, "", "Title", "Content")
			Expect(err).To(HaveOccurred())

			teamRepo.On("GetTeamIncidentByIDOrNumber", ctx, incID.String()).Return(nil, errors.New("inc not found")).Once()
			_, err = teamSvc.CreateRunbook(ctx, uID, tID, incID.String(), "Title", "Content")
			Expect(err).To(HaveOccurred())

			teamRepo.On("GetRunbookLogs", ctx, rbID).Return(nil, nil).Once()
			teamRepo.On("GetRunbookByID", ctx, rbID).Return(&models.Runbook{ID: rbID}, nil).Once()
			teamRepo.On("UpdateRunbook", ctx, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("db update error")).Once()
			_, err = teamSvc.UpdateRunbook(ctx, uID, rbID, "New Title", "New Content")
			Expect(err).To(HaveOccurred())
		})
	})
})
