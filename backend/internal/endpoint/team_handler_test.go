package endpoint_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/WillieBam/support_copilot/backend/internal/endpoint"
	"github.com/WillieBam/support_copilot/backend/internal/mocks"
	"github.com/WillieBam/support_copilot/backend/types/models"
	"github.com/WillieBam/support_copilot/backend/types/requests"
	customErrors "github.com/WillieBam/support_copilot/backend/utils/errors"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

var _ = Describe("TeamHandler", func() {
	var (
		e           *echo.Echo
		mockAppSvc  *mocks.IAppService
		mockAuthSvc *mocks.IAuthService
		mockUserSvc *mocks.IUserService
		mockTeamSvc *mocks.ITeamService
		mockDashSvc *mocks.IDashboardService
		h           *endpoint.Handler
		testUser    *models.User
		userID      uuid.UUID
		teamID      uuid.UUID
		incidentID  uuid.UUID
	)

	BeforeEach(func() {
		e = echo.New()
		mockAppSvc = &mocks.IAppService{}
		mockAuthSvc = &mocks.IAuthService{}
		mockUserSvc = &mocks.IUserService{}
		mockTeamSvc = &mocks.ITeamService{}
		mockDashSvc = &mocks.IDashboardService{}

		h = endpoint.NewHandler(
			mockAppSvc,
			mockAuthSvc,
			mockUserSvc,
			mockTeamSvc,
			mockDashSvc,
		)

		userID = uuid.New()
		teamID = uuid.New()
		incidentID = uuid.New()

		testUser = &models.User{
			ID:          userID,
			FirebaseUID: "fb-uid-123",
			Email:       "engineer@test.com",
			DisplayName: "Test Engineer",
			Scope:       "engineer",
		}
	})

	Context("CreateTeam", func() {
		It("should fail if user is unauthorized", func() {
			req := httptest.NewRequest(http.MethodPost, "/api/teams", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := h.CreateTeam(c)
			Expect(err).To(HaveOccurred())
		})

		It("should fail if request payload is invalid", func() {
			req := httptest.NewRequest(http.MethodPost, "/api/teams", strings.NewReader("invalid"))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.Set("user_uid", "fb-uid-123")

			mockUserSvc.On("GetUserByFirebaseUID", mock.Anything, "fb-uid-123").Return(testUser, nil)

			err := h.CreateTeam(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusBadRequest))
		})

		It("should return 400 when team name validation fails", func() {
			body, _ := json.Marshal(requests.CreateTeamRequest{TeamName: ""})
			req := httptest.NewRequest(http.MethodPost, "/api/teams", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.Set("user_uid", "fb-uid-123")

			mockUserSvc.On("GetUserByFirebaseUID", mock.Anything, "fb-uid-123").Return(testUser, nil)
			mockTeamSvc.On("CreateTeam", mock.Anything, "", userID).Return(nil, customErrors.ErrTeamNameRequired)

			err := h.CreateTeam(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusBadRequest))
		})

		It("should return 201 on successful team creation", func() {
			body, _ := json.Marshal(requests.CreateTeamRequest{TeamName: "DevOps Rescuers"})
			req := httptest.NewRequest(http.MethodPost, "/api/teams", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.Set("user_uid", "fb-uid-123")

			createdTeam := &models.Team{
				ID:       teamID,
				TeamName: "DevOps Rescuers",
			}
			mockUserSvc.On("GetUserByFirebaseUID", mock.Anything, "fb-uid-123").Return(testUser, nil)
			mockTeamSvc.On("CreateTeam", mock.Anything, "DevOps Rescuers", userID).Return(createdTeam, nil)

			err := h.CreateTeam(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusCreated))
		})
	})

	Context("GetTeams", func() {
		It("should return user teams list successfully", func() {
			req := httptest.NewRequest(http.MethodGet, "/api/teams/me", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.Set("user_uid", "fb-uid-123")

			mockUserSvc.On("GetUserByFirebaseUID", mock.Anything, "fb-uid-123").Return(testUser, nil)
			mockTeamSvc.On("GetUserTeams", mock.Anything, userID).Return(testUser, nil)

			err := h.GetTeams(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusOK))
		})
	})

	Context("Team Members Management", func() {
		It("should fail GetTeamMembers if team_id param is invalid", func() {
			req := httptest.NewRequest(http.MethodGet, "/api/teams/invalid/members", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPathValues(echo.PathValues{{Name: "team_id", Value: "invalid-uuid"}})
			c.Set("user_uid", "fb-uid-123")

			mockUserSvc.On("GetUserByFirebaseUID", mock.Anything, "fb-uid-123").Return(testUser, nil)

			err := h.GetTeamMembers(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusBadRequest))
		})

		It("should return team members list on GetTeamMembers success", func() {
			req := httptest.NewRequest(http.MethodGet, "/api/teams/"+teamID.String()+"/members", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPathValues(echo.PathValues{{Name: "team_id", Value: teamID.String()}})
			c.Set("user_uid", "fb-uid-123")

			members := []models.TeamMember{{ID: uuid.New(), TeamID: teamID, UserID: userID, Role: "owner"}}
			mockUserSvc.On("GetUserByFirebaseUID", mock.Anything, "fb-uid-123").Return(testUser, nil)
			mockTeamSvc.On("ListMembers", mock.Anything, userID, teamID).Return(members, nil)

			err := h.GetTeamMembers(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusOK))
		})

		It("should AddTeamMember successfully", func() {
			targetUserID := uuid.New()
			body, _ := json.Marshal(requests.AddTeamMemberRequest{
				UserID: targetUserID,
			})
			req := httptest.NewRequest(http.MethodPost, "/api/teams/"+teamID.String()+"/members", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPathValues(echo.PathValues{{Name: "team_id", Value: teamID.String()}})
			c.Set("user_uid", "fb-uid-123")

			mockUserSvc.On("GetUserByFirebaseUID", mock.Anything, "fb-uid-123").Return(testUser, nil)
			mockTeamSvc.On("AddMember", mock.Anything, userID, teamID, targetUserID).Return(nil)

			err := h.AddTeamMember(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusCreated))
		})

		It("should RemoveTeamMember successfully", func() {
			targetUserID := uuid.New()
			req := httptest.NewRequest(http.MethodDelete, "/api/teams/"+teamID.String()+"/members/"+targetUserID.String(), nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPathValues(echo.PathValues{
				{Name: "team_id", Value: teamID.String()},
				{Name: "user_id", Value: targetUserID.String()},
			})
			c.Set("user_uid", "fb-uid-123")

			mockUserSvc.On("GetUserByFirebaseUID", mock.Anything, "fb-uid-123").Return(testUser, nil)
			mockTeamSvc.On("RemoveMember", mock.Anything, userID, teamID, targetUserID).Return(nil)

			err := h.RemoveTeamMember(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusOK))
		})
	})

	Context("Incident Handlers", func() {
		It("should AssignTeamIncident successfully", func() {
			body, _ := json.Marshal(requests.AssignTeamIncidentRequest{
				Title:   "High Latency Alert",
				Status:  "OPEN",
				Details: "Latency > 5s",
			})
			req := httptest.NewRequest(http.MethodPost, "/api/teams/"+teamID.String()+"/incidents", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPathValues(echo.PathValues{{Name: "team_id", Value: teamID.String()}})
			c.Set("user_uid", "fb-uid-123")

			createdInc := &models.TeamIncident{
				ID:        incidentID,
				TeamID:    teamID,
				CreatedBy: userID,
				Title:     "High Latency Alert",
				Status:    "OPEN",
			}
			mockUserSvc.On("GetUserByFirebaseUID", mock.Anything, "fb-uid-123").Return(testUser, nil)
			mockTeamSvc.On("AssignIncident", mock.Anything, userID, teamID, "High Latency Alert", "OPEN", "Latency > 5s").
				Return(createdInc, nil)

			err := h.AssignTeamIncident(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusCreated))
		})

		It("should UpdateTeamIncidentStatus successfully", func() {
			body, _ := json.Marshal(requests.UpdateIncidentStatusRequest{
				Status:  "RESOLVED",
				Title:   "High Latency Alert Fixed",
				Details: "Scaled pods",
			})
			req := httptest.NewRequest(http.MethodPut, "/api/incidents/"+incidentID.String(), bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPathValues(echo.PathValues{{Name: "id", Value: incidentID.String()}})
			c.Set("user_uid", "fb-uid-123")

			updatedInc := &models.TeamIncident{
				ID:     incidentID,
				Title:  "High Latency Alert Fixed",
				Status: "RESOLVED",
			}
			mockUserSvc.On("GetUserByFirebaseUID", mock.Anything, "fb-uid-123").Return(testUser, nil)
			mockTeamSvc.On("UpdateIncidentStatus", mock.Anything, userID, incidentID, "RESOLVED", "High Latency Alert Fixed", "Scaled pods").
				Return(updatedInc, nil)

			err := h.UpdateTeamIncidentStatus(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusOK))
		})
	})

	Context("Instruction Handlers", func() {
		It("should GetTeamInstruction successfully", func() {
			req := httptest.NewRequest(http.MethodGet, "/api/teams/"+teamID.String()+"/instruction", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPathValues(echo.PathValues{{Name: "team_id", Value: teamID.String()}})
			c.Set("user_uid", "fb-uid-123")

			inst := &models.Instruction{
				ID:                 uuid.New(),
				TeamID:             teamID,
				InstructionDetails: "Always run kubectl describe pod before scaling deployment",
			}
			logs := []models.InstructionLog{}
			mockUserSvc.On("GetUserByFirebaseUID", mock.Anything, "fb-uid-123").Return(testUser, nil)
			mockTeamSvc.On("GetTeamInstruction", mock.Anything, userID, teamID).Return(inst, logs, nil)

			err := h.GetTeamInstruction(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusOK))
		})

		It("should SaveTeamInstruction successfully", func() {
			details := "Always run kubectl describe pod before scaling deployment"
			body, _ := json.Marshal(requests.SaveTeamInstructionRequest{
				InstructionDetails: details,
			})
			req := httptest.NewRequest(http.MethodPost, "/api/teams/"+teamID.String()+"/instruction", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPathValues(echo.PathValues{{Name: "team_id", Value: teamID.String()}})
			c.Set("user_uid", "fb-uid-123")

			updatedInst := &models.Instruction{
				ID:                 uuid.New(),
				TeamID:             teamID,
				InstructionDetails: details,
			}
			mockUserSvc.On("GetUserByFirebaseUID", mock.Anything, "fb-uid-123").Return(testUser, nil)
			mockTeamSvc.On("SaveTeamInstruction", mock.Anything, userID, teamID, details).Return(updatedInst, nil)

			err := h.SaveTeamInstruction(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusOK))
		})
	})

	Context("SearchUsers", func() {
		It("should SearchUsers successfully", func() {
			req := httptest.NewRequest(http.MethodGet, "/api/users/search?q=engineer", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.Set("user_uid", "fb-uid-123")

			foundUsers := []models.User{*testUser}
			mockUserSvc.On("GetUserByFirebaseUID", mock.Anything, "fb-uid-123").Return(testUser, nil)
			mockUserSvc.On("SearchUsers", mock.Anything, "engineer", 10).Return(foundUsers, nil)

			err := h.SearchUsers(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusOK))
		})
	})
})
