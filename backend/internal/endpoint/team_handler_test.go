package endpoint_test

import (
	"bytes"
	"encoding/json"
	"errors"
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
	"gorm.io/gorm"
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

		It("should GetTeamIncidents successfully", func() {
			req := httptest.NewRequest(http.MethodGet, "/api/teams/"+teamID.String()+"/incidents", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPathValues(echo.PathValues{{Name: "team_id", Value: teamID.String()}})
			c.Set("user_uid", "fb-uid-123")

			incidents := []models.TeamIncident{{ID: incidentID, TeamID: teamID, Title: "Inc 1"}}
			mockUserSvc.On("GetUserByFirebaseUID", mock.Anything, "fb-uid-123").Return(testUser, nil)
			mockTeamSvc.On("ListIncidents", mock.Anything, userID, teamID).Return(incidents, nil)

			err := h.GetTeamIncidents(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusOK))
		})

		It("should GetTeamIncident successfully", func() {
			req := httptest.NewRequest(http.MethodGet, "/api/incidents/"+incidentID.String(), nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPathValues(echo.PathValues{{Name: "id", Value: incidentID.String()}})
			c.Set("user_uid", "fb-uid-123")

			inc := &models.TeamIncident{ID: incidentID, TeamID: teamID, Title: "Inc 1"}
			mockUserSvc.On("GetUserByFirebaseUID", mock.Anything, "fb-uid-123").Return(testUser, nil)
			mockTeamSvc.On("GetIncident", mock.Anything, userID, incidentID).Return(inc, nil)

			err := h.GetTeamIncident(c)
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

	Context("DeleteTeam", func() {
		It("should return 403 when user scope is engineer", func() {
			req := httptest.NewRequest(http.MethodDelete, "/api/teams/"+teamID.String(), nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPathValues(echo.PathValues{{Name: "team_id", Value: teamID.String()}})
			c.Set("user_uid", "fb-uid-123")

			mockUserSvc.On("GetUserByFirebaseUID", mock.Anything, "fb-uid-123").Return(testUser, nil)
			mockTeamSvc.On("DeleteTeam", mock.Anything, "engineer", teamID).Return(customErrors.ErrSuperAdminRequired)

			err := h.DeleteTeam(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusForbidden))
		})

		It("should return 200 when user scope is super_admin", func() {
			adminUser := &models.User{ID: userID, FirebaseUID: "fb-uid-123", Scope: "super_admin"}
			req := httptest.NewRequest(http.MethodDelete, "/api/admin/teams/"+teamID.String(), nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPathValues(echo.PathValues{{Name: "team_id", Value: teamID.String()}})
			c.Set("user_uid", "fb-uid-123")

			mockUserSvc.On("GetUserByFirebaseUID", mock.Anything, "fb-uid-123").Return(adminUser, nil)
			mockTeamSvc.On("DeleteTeam", mock.Anything, "super_admin", teamID).Return(nil)

			err := h.DeleteTeam(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusOK))
		})
	})

	Context("ListAllTeams", func() {
		It("should return 403 when user scope is engineer", func() {
			req := httptest.NewRequest(http.MethodGet, "/api/admin/teams", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.Set("user_uid", "fb-uid-123")

			mockUserSvc.On("GetUserByFirebaseUID", mock.Anything, "fb-uid-123").Return(testUser, nil)
			mockTeamSvc.On("ListAllTeams", mock.Anything, "engineer").Return(nil, customErrors.ErrSuperAdminRequired)

			err := h.ListAllTeams(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusForbidden))
		})

		It("should return 200 when user scope is super_admin", func() {
			adminUser := &models.User{ID: userID, FirebaseUID: "fb-uid-123", Scope: "super_admin"}
			req := httptest.NewRequest(http.MethodGet, "/api/admin/teams", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.Set("user_uid", "fb-uid-123")

			teams := []models.Team{{ID: teamID, TeamName: "DevOps"}}
			mockUserSvc.On("GetUserByFirebaseUID", mock.Anything, "fb-uid-123").Return(adminUser, nil)
			mockTeamSvc.On("ListAllTeams", mock.Anything, "super_admin").Return(teams, nil)

			err := h.ListAllTeams(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusOK))
		})
	})

	Context("getAuthenticatedUser and Handler Error Paths", func() {
		It("should fail getAuthenticatedUser on missing uid, nil service, or DB errors", func() {
			// missing user_uid
			cNoUID := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder())
			err := h.GetTeams(cNoUID)
			Expect(err).To(HaveOccurred())

			// nil userService
			hNilUser := endpoint.NewHandler(mockAppSvc, mockAuthSvc, nil, mockTeamSvc, mockDashSvc)
			cNilUser := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder())
			cNilUser.Set("user_uid", "fb-uid-123")
			err = hNilUser.GetTeams(cNilUser)
			Expect(err).To(HaveOccurred())

			// gorm.ErrRecordNotFound
			cNotFound := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder())
			cNotFound.Set("user_uid", "missing-uid")
			mockUserSvc.On("GetUserByFirebaseUID", mock.Anything, "missing-uid").Return(nil, gorm.ErrRecordNotFound)
			err = h.GetTeams(cNotFound)
			Expect(err).To(HaveOccurred())

			// generic DB error
			cErr := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder())
			cErr.Set("user_uid", "err-uid")
			mockUserSvc.On("GetUserByFirebaseUID", mock.Anything, "err-uid").Return(nil, errors.New("db crash"))
			err = h.GetTeams(cErr)
			Expect(err).To(HaveOccurred())
		})

		It("should handle validation and service errors for AddTeamMember and RemoveTeamMember", func() {
			mockUserSvc.On("GetUserByFirebaseUID", mock.Anything, "fb-uid-123").Return(testUser, nil)

			// AddTeamMember invalid team_id
			cAddBadTeam := e.NewContext(httptest.NewRequest(http.MethodPost, "/", nil), httptest.NewRecorder())
			cAddBadTeam.SetPathValues(echo.PathValues{{Name: "team_id", Value: "invalid"}})
			cAddBadTeam.Set("user_uid", "fb-uid-123")
			err := h.AddTeamMember(cAddBadTeam)
			Expect(err).NotTo(HaveOccurred())

			// AddTeamMember bind error
			cAddBadBind := e.NewContext(httptest.NewRequest(http.MethodPost, "/", strings.NewReader("bad")), httptest.NewRecorder())
			cAddBadBind.SetPathValues(echo.PathValues{{Name: "team_id", Value: teamID.String()}})
			cAddBadBind.Set("user_uid", "fb-uid-123")
			err = h.AddTeamMember(cAddBadBind)
			Expect(err).NotTo(HaveOccurred())

			// AddTeamMember zero user_id
			cAddZeroUser := e.NewContext(httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"user_id":"00000000-0000-0000-0000-000000000000"}`)), httptest.NewRecorder())
			cAddZeroUser.Request().Header.Set("Content-Type", "application/json")
			cAddZeroUser.SetPathValues(echo.PathValues{{Name: "team_id", Value: teamID.String()}})
			cAddZeroUser.Set("user_uid", "fb-uid-123")
			err = h.AddTeamMember(cAddZeroUser)
			Expect(err).NotTo(HaveOccurred())

			// AddTeamMember unauthorized error
			targetID := uuid.New()
			body, _ := json.Marshal(requests.AddTeamMemberRequest{UserID: targetID})
			cAddUnauth := e.NewContext(httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body)), httptest.NewRecorder())
			cAddUnauth.Request().Header.Set("Content-Type", "application/json")
			cAddUnauth.SetPathValues(echo.PathValues{{Name: "team_id", Value: teamID.String()}})
			cAddUnauth.Set("user_uid", "fb-uid-123")
			mockTeamSvc.On("AddMember", mock.Anything, userID, teamID, targetID).Return(customErrors.ErrUnauthorizedTeamOp).Once()
			err = h.AddTeamMember(cAddUnauth)
			Expect(err).NotTo(HaveOccurred())

			// RemoveTeamMember invalid team_id or user_id
			cRemBad := e.NewContext(httptest.NewRequest(http.MethodDelete, "/", nil), httptest.NewRecorder())
			cRemBad.SetPathValues(echo.PathValues{{Name: "team_id", Value: "invalid"}, {Name: "user_id", Value: "invalid"}})
			cRemBad.Set("user_uid", "fb-uid-123")
			err = h.RemoveTeamMember(cRemBad)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should handle validation and service errors for AssignTeamIncident, UpdateTeamIncidentStatus, and GetTeamIncident", func() {
			mockUserSvc.On("GetUserByFirebaseUID", mock.Anything, "fb-uid-123").Return(testUser, nil)

			// AssignTeamIncident invalid team_id or bind error
			cAssignBadTeam := e.NewContext(httptest.NewRequest(http.MethodPost, "/", nil), httptest.NewRecorder())
			cAssignBadTeam.SetPathValues(echo.PathValues{{Name: "team_id", Value: "invalid"}})
			cAssignBadTeam.Set("user_uid", "fb-uid-123")
			err := h.AssignTeamIncident(cAssignBadTeam)
			Expect(err).NotTo(HaveOccurred())

			cAssignBadBind := e.NewContext(httptest.NewRequest(http.MethodPost, "/", strings.NewReader("invalid")), httptest.NewRecorder())
			cAssignBadBind.SetPathValues(echo.PathValues{{Name: "team_id", Value: teamID.String()}})
			cAssignBadBind.Set("user_uid", "fb-uid-123")
			err = h.AssignTeamIncident(cAssignBadBind)
			Expect(err).NotTo(HaveOccurred())

			// AssignTeamIncident with AlertIDs linking
			bodyAssignAlerts, _ := json.Marshal(requests.AssignTeamIncidentRequest{Title: "Title", Status: "OPEN", AlertID: "alert-1", AlertIDs: []string{"alert-2"}})
			cAssignAlerts := e.NewContext(httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(bodyAssignAlerts)), httptest.NewRecorder())
			cAssignAlerts.Request().Header.Set("Content-Type", "application/json")
			cAssignAlerts.SetPathValues(echo.PathValues{{Name: "team_id", Value: teamID.String()}})
			cAssignAlerts.Set("user_uid", "fb-uid-123")

			createdInc := &models.TeamIncident{ID: incidentID, TeamID: teamID, Title: "Title"}
			mockTeamSvc.On("AssignIncident", mock.Anything, userID, teamID, "Title", "OPEN", "").Return(createdInc, nil).Once()
			mockTeamSvc.On("LinkAlertsToIncident", mock.Anything, []string{"alert-1", "alert-2"}, incidentID).Return(nil).Once()
			err = h.AssignTeamIncident(cAssignAlerts)
			Expect(err).NotTo(HaveOccurred())

			// AssignTeamIncident unauth (403) and 500
			bodyAssignUnauth, _ := json.Marshal(requests.AssignTeamIncidentRequest{Title: "Title", Status: "OPEN"})
			cAssignUnauth := e.NewContext(httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(bodyAssignUnauth)), httptest.NewRecorder())
			cAssignUnauth.Request().Header.Set("Content-Type", "application/json")
			cAssignUnauth.SetPathValues(echo.PathValues{{Name: "team_id", Value: teamID.String()}})
			cAssignUnauth.Set("user_uid", "fb-uid-123")
			mockTeamSvc.On("AssignIncident", mock.Anything, userID, teamID, "Title", "OPEN", "").Return(nil, customErrors.ErrUnauthorizedTeamOp).Once()
			err = h.AssignTeamIncident(cAssignUnauth)
			Expect(err).NotTo(HaveOccurred())

			// UpdateTeamIncidentStatus invalid id, bind error, 400, 403, 404, 500
			cUpdBadID := e.NewContext(httptest.NewRequest(http.MethodPut, "/", nil), httptest.NewRecorder())
			cUpdBadID.SetPathValues(echo.PathValues{{Name: "id", Value: "invalid"}})
			cUpdBadID.Set("user_uid", "fb-uid-123")
			err = h.UpdateTeamIncidentStatus(cUpdBadID)
			Expect(err).NotTo(HaveOccurred())

			bodyUpdValid, _ := json.Marshal(requests.UpdateIncidentStatusRequest{Status: "RESOLVED"})
			cUpdErr400 := e.NewContext(httptest.NewRequest(http.MethodPut, "/", bytes.NewReader(bodyUpdValid)), httptest.NewRecorder())
			cUpdErr400.Request().Header.Set("Content-Type", "application/json")
			cUpdErr400.SetPathValues(echo.PathValues{{Name: "id", Value: incidentID.String()}})
			cUpdErr400.Set("user_uid", "fb-uid-123")
			mockTeamSvc.On("UpdateIncidentStatus", mock.Anything, userID, incidentID, "RESOLVED", "", "").Return(nil, customErrors.ErrInvalidIncidentStatus).Once()
			err = h.UpdateTeamIncidentStatus(cUpdErr400)
			Expect(err).NotTo(HaveOccurred())

			cUpdErr403 := e.NewContext(httptest.NewRequest(http.MethodPut, "/", bytes.NewReader(bodyUpdValid)), httptest.NewRecorder())
			cUpdErr403.Request().Header.Set("Content-Type", "application/json")
			cUpdErr403.SetPathValues(echo.PathValues{{Name: "id", Value: incidentID.String()}})
			cUpdErr403.Set("user_uid", "fb-uid-123")
			mockTeamSvc.On("UpdateIncidentStatus", mock.Anything, userID, incidentID, "RESOLVED", "", "").Return(nil, customErrors.ErrUnauthorizedTeamOp).Once()
			err = h.UpdateTeamIncidentStatus(cUpdErr403)
			Expect(err).NotTo(HaveOccurred())

			cUpdErr404 := e.NewContext(httptest.NewRequest(http.MethodPut, "/", bytes.NewReader(bodyUpdValid)), httptest.NewRecorder())
			cUpdErr404.Request().Header.Set("Content-Type", "application/json")
			cUpdErr404.SetPathValues(echo.PathValues{{Name: "id", Value: incidentID.String()}})
			cUpdErr404.Set("user_uid", "fb-uid-123")
			mockTeamSvc.On("UpdateIncidentStatus", mock.Anything, userID, incidentID, "RESOLVED", "", "").Return(nil, gorm.ErrRecordNotFound).Once()
			err = h.UpdateTeamIncidentStatus(cUpdErr404)
			Expect(err).NotTo(HaveOccurred())

			cUpdErr500 := e.NewContext(httptest.NewRequest(http.MethodPut, "/", bytes.NewReader(bodyUpdValid)), httptest.NewRecorder())
			cUpdErr500.Request().Header.Set("Content-Type", "application/json")
			cUpdErr500.SetPathValues(echo.PathValues{{Name: "id", Value: incidentID.String()}})
			cUpdErr500.Set("user_uid", "fb-uid-123")
			mockTeamSvc.On("UpdateIncidentStatus", mock.Anything, userID, incidentID, "RESOLVED", "", "").Return(nil, errors.New("db crash")).Once()
			err = h.UpdateTeamIncidentStatus(cUpdErr500)
			Expect(err).NotTo(HaveOccurred())

			// GetTeamIncident invalid id or generic error
			cIncBadID := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder())
			cIncBadID.SetPathValues(echo.PathValues{{Name: "id", Value: "invalid"}})
			cIncBadID.Set("user_uid", "fb-uid-123")
			err = h.GetTeamIncident(cIncBadID)
			Expect(err).NotTo(HaveOccurred())

			cIncErr500 := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder())
			cIncErr500.SetPathValues(echo.PathValues{{Name: "id", Value: incidentID.String()}})
			cIncErr500.Set("user_uid", "fb-uid-123")
			mockTeamSvc.On("GetIncident", mock.Anything, userID, incidentID).Return(nil, errors.New("db error")).Once()
			err = h.GetTeamIncident(cIncErr500)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should handle validation and service errors for GetTeamInstruction and SaveTeamInstruction", func() {
			mockUserSvc.On("GetUserByFirebaseUID", mock.Anything, "fb-uid-123").Return(testUser, nil)

			// GetTeamInstruction invalid team_id or service error
			cInstBad := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder())
			cInstBad.SetPathValues(echo.PathValues{{Name: "team_id", Value: "invalid"}})
			cInstBad.Set("user_uid", "fb-uid-123")
			err := h.GetTeamInstruction(cInstBad)
			Expect(err).NotTo(HaveOccurred())

			cInstErr := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder())
			cInstErr.SetPathValues(echo.PathValues{{Name: "team_id", Value: teamID.String()}})
			cInstErr.Set("user_uid", "fb-uid-123")
			mockTeamSvc.On("GetTeamInstruction", mock.Anything, userID, teamID).Return(nil, nil, errors.New("db error")).Once()
			err = h.GetTeamInstruction(cInstErr)
			Expect(err).NotTo(HaveOccurred())

			// SaveTeamInstruction length invalid error or generic error
			bodySave, _ := json.Marshal(requests.SaveTeamInstructionRequest{InstructionDetails: "short"})
			cSaveShort := e.NewContext(httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(bodySave)), httptest.NewRecorder())
			cSaveShort.Request().Header.Set("Content-Type", "application/json")
			cSaveShort.SetPathValues(echo.PathValues{{Name: "team_id", Value: teamID.String()}})
			cSaveShort.Set("user_uid", "fb-uid-123")
			mockTeamSvc.On("SaveTeamInstruction", mock.Anything, userID, teamID, "short").Return(nil, customErrors.ErrInstructionTooShort).Once()
			err = h.SaveTeamInstruction(cSaveShort)
			Expect(err).NotTo(HaveOccurred())

			bodyValid, _ := json.Marshal(requests.SaveTeamInstructionRequest{InstructionDetails: "valid long instruction text details sample"})
			cSaveErr := e.NewContext(httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(bodyValid)), httptest.NewRecorder())
			cSaveErr.Request().Header.Set("Content-Type", "application/json")
			cSaveErr.SetPathValues(echo.PathValues{{Name: "team_id", Value: teamID.String()}})
			cSaveErr.Set("user_uid", "fb-uid-123")
			mockTeamSvc.On("SaveTeamInstruction", mock.Anything, userID, teamID, mock.Anything).Return(nil, errors.New("db error")).Once()
			err = h.SaveTeamInstruction(cSaveErr)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should handle 500 service errors across team handlers", func() {
			mockUserSvc.On("GetUserByFirebaseUID", mock.Anything, "fb-uid-123").Return(testUser, nil)

			// RemoveTeamMember 500
			targetID := uuid.New()
			cRemErr := e.NewContext(httptest.NewRequest(http.MethodDelete, "/", nil), httptest.NewRecorder())
			cRemErr.SetPathValues(echo.PathValues{{Name: "team_id", Value: teamID.String()}, {Name: "user_id", Value: targetID.String()}})
			cRemErr.Set("user_uid", "fb-uid-123")
			mockTeamSvc.On("RemoveMember", mock.Anything, userID, teamID, targetID).Return(errors.New("db error")).Once()
			err := h.RemoveTeamMember(cRemErr)
			Expect(err).NotTo(HaveOccurred())

			// DeleteTeam 500
			adminUser := &models.User{ID: userID, FirebaseUID: "fb-uid-123", Scope: "super_admin"}
			mockUserSvc.On("GetUserByFirebaseUID", mock.Anything, "admin-uid").Return(adminUser, nil)
			cDelErr := e.NewContext(httptest.NewRequest(http.MethodDelete, "/", nil), httptest.NewRecorder())
			cDelErr.SetPathValues(echo.PathValues{{Name: "team_id", Value: teamID.String()}})
			cDelErr.Set("user_uid", "admin-uid")
			mockTeamSvc.On("DeleteTeam", mock.Anything, "super_admin", teamID).Return(errors.New("db error")).Once()
			err = h.DeleteTeam(cDelErr)
			Expect(err).NotTo(HaveOccurred())

			// ListAllTeams 500
			cListErr := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder())
			cListErr.Set("user_uid", "admin-uid")
			mockTeamSvc.On("ListAllTeams", mock.Anything, "super_admin").Return(nil, errors.New("db error")).Once()
			err = h.ListAllTeams(cListErr)
			Expect(err).NotTo(HaveOccurred())

			// GetTeamIncidents 500
			cIncListErr := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder())
			cIncListErr.SetPathValues(echo.PathValues{{Name: "team_id", Value: teamID.String()}})
			cIncListErr.Set("user_uid", "fb-uid-123")
			mockTeamSvc.On("ListIncidents", mock.Anything, userID, teamID).Return(nil, errors.New("db error")).Once()
			err = h.GetTeamIncidents(cIncListErr)
			Expect(err).NotTo(HaveOccurred())

			// GetTeamMembers 500
			cMemErr := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder())
			cMemErr.SetPathValues(echo.PathValues{{Name: "team_id", Value: teamID.String()}})
			cMemErr.Set("user_uid", "fb-uid-123")
			mockTeamSvc.On("ListMembers", mock.Anything, userID, teamID).Return(nil, errors.New("db error")).Once()
			err = h.GetTeamMembers(cMemErr)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should handle 403 ErrUnauthorizedTeamOp and 500 errors for GetTeams, CreateTeam, GetTeamIncidents, GetTeamIncident", func() {
			mockUserSvc.On("GetUserByFirebaseUID", mock.Anything, "fb-uid-123").Return(testUser, nil)

			// GetTeams 500
			cGetTeamsErr := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder())
			cGetTeamsErr.Set("user_uid", "fb-uid-123")
			mockTeamSvc.On("GetUserTeams", mock.Anything, userID).Return(nil, errors.New("db error")).Once()
			err := h.GetTeams(cGetTeamsErr)
			Expect(err).NotTo(HaveOccurred())

			// CreateTeam 500
			cCreateTeamErr := e.NewContext(httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"team_name":"DevOps"}`)), httptest.NewRecorder())
			cCreateTeamErr.Request().Header.Set("Content-Type", "application/json")
			cCreateTeamErr.Set("user_uid", "fb-uid-123")
			mockTeamSvc.On("CreateTeam", mock.Anything, "DevOps", userID).Return(nil, errors.New("db error")).Once()
			err = h.CreateTeam(cCreateTeamErr)
			Expect(err).NotTo(HaveOccurred())

			// GetTeamIncidents 403
			cInc403 := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder())
			cInc403.SetPathValues(echo.PathValues{{Name: "team_id", Value: teamID.String()}})
			cInc403.Set("user_uid", "fb-uid-123")
			mockTeamSvc.On("ListIncidents", mock.Anything, userID, teamID).Return(nil, customErrors.ErrUnauthorizedTeamOp).Once()
			err = h.GetTeamIncidents(cInc403)
			Expect(err).NotTo(HaveOccurred())

			// GetTeamIncident 403
			cIncSingle403 := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder())
			cIncSingle403.SetPathValues(echo.PathValues{{Name: "id", Value: incidentID.String()}})
			cIncSingle403.Set("user_uid", "fb-uid-123")
			mockTeamSvc.On("GetIncident", mock.Anything, userID, incidentID).Return(nil, customErrors.ErrUnauthorizedTeamOp).Once()
			err = h.GetTeamIncident(cIncSingle403)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return 400 when team_id is invalid in GetTeamMembers, DeleteTeam, GetTeamIncidents", func() {
			mockUserSvc.On("GetUserByFirebaseUID", mock.Anything, "fb-uid-123").Return(testUser, nil)

			c1 := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder())
			c1.SetPathValues(echo.PathValues{{Name: "team_id", Value: "invalid"}})
			c1.Set("user_uid", "fb-uid-123")

			err := h.GetTeamMembers(c1)
			Expect(err).NotTo(HaveOccurred())

			c2 := e.NewContext(httptest.NewRequest(http.MethodDelete, "/", nil), httptest.NewRecorder())
			c2.SetPathValues(echo.PathValues{{Name: "team_id", Value: "invalid"}})
			c2.Set("user_uid", "fb-uid-123")
			err = h.DeleteTeam(c2)
			Expect(err).NotTo(HaveOccurred())

			c3 := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder())
			c3.SetPathValues(echo.PathValues{{Name: "team_id", Value: "invalid"}})
			c3.Set("user_uid", "fb-uid-123")
			err = h.GetTeamIncidents(c3)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
