package endpoint_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/WillieBam/support_copilot/backend/types/models"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	"github.com/WillieBam/support_copilot/backend/internal/endpoint"
	"github.com/WillieBam/support_copilot/backend/internal/mocks"
	"github.com/WillieBam/support_copilot/backend/types"
	"github.com/WillieBam/support_copilot/backend/types/requests"
	"github.com/labstack/echo/v5"
)

var _ = Describe("Handler", func() {
	var (
		e           *echo.Echo
		mockAppSvc  *mocks.IAppService
		mockAuthSvc *mocks.IAuthService
		h           *endpoint.Handler
	)

	BeforeEach(func() {
		e = echo.New()
		mockAppSvc = &mocks.IAppService{}
		mockAuthSvc = &mocks.IAuthService{}
		h = endpoint.NewHandler(mockAppSvc, mockAuthSvc)
	})

	Context("TokenExchangeHandler", func() {
		It("should fail if request body is invalid", func() {
			req := httptest.NewRequest(http.MethodPost, "/token-exchange", strings.NewReader("invalid body"))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := h.TokenExchangeHandler(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusBadRequest))
		})

		It("should fail if firebase token is empty", func() {
			body, _ := json.Marshal(requests.TokenExchangeRequest{FirebaseToken: ""})
			req := httptest.NewRequest(http.MethodPost, "/token-exchange", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := h.TokenExchangeHandler(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusBadRequest))
		})

		It("should fail with status 403 when mfa is required", func() {
			body, _ := json.Marshal(requests.TokenExchangeRequest{FirebaseToken: "mfa-token"})
			req := httptest.NewRequest(http.MethodPost, "/token-exchange", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			mockAuthSvc.On("ExchangeToken", mock.Anything, "mfa-token").Return("", nil, errors.New("mfa_required"))

			err := h.TokenExchangeHandler(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusForbidden))
		})

		It("should return token and set cookie on successful exchange", func() {
			body, _ := json.Marshal(requests.TokenExchangeRequest{FirebaseToken: "valid-token"})
			req := httptest.NewRequest(http.MethodPost, "/token-exchange", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			claims := &types.Claims{
				FirebaseUID: "uid-123",
				RegisteredClaims: jwt.RegisteredClaims{
					ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
				},
			}
			mockAuthSvc.On("ExchangeToken", mock.Anything, "valid-token").Return("backend-token", claims, nil)

			err := h.TokenExchangeHandler(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusOK))

			// Check cookie
			cookies := rec.Result().Cookies()
			Expect(len(cookies)).To(Equal(1))
			Expect(cookies[0].Name).To(Equal("support_copilot_session"))
			Expect(cookies[0].Value).To(Equal("backend-token"))
		})
	})

	Context("Me", func() {
		It("should fail if unauthorized", func() {
			req := httptest.NewRequest(http.MethodGet, "/me", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := h.Me(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusUnauthorized))
		})

		It("should succeed and return user info when authorized", func() {
			req := httptest.NewRequest(http.MethodGet, "/me", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.Set("user_uid", "uid-123")
			c.Set("user_email", "user@test.com")

			err := h.Me(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusOK))

			var res map[string]interface{}
			err = json.Unmarshal(rec.Body.Bytes(), &res)
			Expect(err).NotTo(HaveOccurred())
			Expect(res["authenticated"]).To(BeTrue())
			Expect(res["user_uid"]).To(Equal("uid-123"))
			Expect(res["user_email"]).To(Equal("user@test.com"))
		})
	})

	Context("Query", func() {
		It("should return 401 if user is unauthorized", func() {
			req := httptest.NewRequest(http.MethodPost, "/query/chat", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := h.Query(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusUnauthorized))
		})

		It("should return 400 if input prompt is empty", func() {
			body, _ := json.Marshal(map[string]string{"input": ""})
			req := httptest.NewRequest(http.MethodPost, "/query/chat", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.Set("user_uid", "uid-123")

			err := h.Query(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusBadRequest))
		})

		It("should stream events and flush successfully", func() {
			body, _ := json.Marshal(map[string]string{"input": "what is AI?"})
			req := httptest.NewRequest(http.MethodPost, "/query/chat", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.Set("user_uid", "uid-123")

			// Setup mock stream channel
			mockAppSvc.On("QueryStreamWithTools", mock.Anything, "what is AI?", mock.Anything, mock.Anything).
				Return(nil).
				Run(func(args mock.Arguments) {
					ch := args.Get(3).(chan<- types.StreamEvent)
					ch <- types.StreamEvent{Type: "reasoning", Content: "thinking"}
					ch <- types.StreamEvent{Type: "text", Content: "AI is..."}
				})

			err := h.Query(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusOK))

			bodyStr := rec.Body.String()
			Expect(bodyStr).To(ContainSubstring(`data: {"type":"reasoning","content":"thinking"}`))
			Expect(bodyStr).To(ContainSubstring(`data: {"type":"text","content":"AI is..."}`))
		})
	})

	Context("Conversation helpers", func() {
		It("should create a conversation when user and team context are valid", func() {
			userSvc := &mocks.IUserService{}
			hWithUser := endpoint.NewHandler(mockAppSvc, mockAuthSvc, userSvc)
			teamID := uuid.New()
			userID := uuid.New()
			conv := &models.Conversation{ID: uuid.New(), TeamID: teamID, UserID: userID}

			body, _ := json.Marshal(map[string]any{"team_id": teamID.String()})
			req := httptest.NewRequest(http.MethodPost, "/api/conversations", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.Set("user_uid", "uid-123")

			userSvc.On("GetUserByFirebaseUID", mock.Anything, "uid-123").Return(&models.User{ID: userID}, nil)
			mockAppSvc.On("CreateConversation", mock.Anything, teamID, userID).Return(conv, nil)

			err := hWithUser.CreateConversation(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusOK))
		})

		It("should list team conversations with a parsed limit", func() {
			teamID := uuid.New()
			body := []models.Conversation{{ID: uuid.New(), TeamID: teamID}}
			mockAppSvc.On("ListTeamConversations", mock.Anything, teamID, 3).Return(body, nil)

			e.GET("/api/teams/:team_id/conversations", func(c *echo.Context) error {
				return h.ListTeamConversations(c)
			})
			req := httptest.NewRequest(http.MethodGet, "/api/teams/"+teamID.String()+"/conversations?limit=3", nil)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			Expect(rec.Code).To(Equal(http.StatusOK))
		})

		It("should return conversation messages when the conversation id is valid", func() {
			convID := uuid.New()
			msgList := []models.Message{{ID: uuid.New(), ConversationID: convID, Sender: "user"}}
			mockAppSvc.On("ListMessagesByConversation", mock.Anything, convID).Return(msgList, nil)

			e.GET("/api/conversations/:id/messages", func(c *echo.Context) error {
				return h.GetConversationMessages(c)
			})
			req := httptest.NewRequest(http.MethodGet, "/api/conversations/"+convID.String()+"/messages", nil)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			Expect(rec.Code).To(Equal(http.StatusOK))
		})

		It("should search users when query is long enough", func() {
			userSvc := &mocks.IUserService{}
			hWithUser := endpoint.NewHandler(mockAppSvc, mockAuthSvc, userSvc)
			userSvc.On("SearchUsers", mock.Anything, "ada", 10).Return([]models.User{{ID: uuid.New(), Email: "ada@example.com", DisplayName: "Ada"}}, nil)

			req := httptest.NewRequest(http.MethodGet, "/api/users/search?q=ada", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.Set("user_uid", "uid-123")

			err := hWithUser.SearchUsers(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusOK))
		})

		It("should handle error paths in Query, CreateConversation, ListTeamConversations, GetConversationMessages, and SearchUsers", func() {
			// Query invalid body
			cQueryBad := e.NewContext(httptest.NewRequest(http.MethodPost, "/api/query", strings.NewReader("invalid")), httptest.NewRecorder())
			err := h.Query(cQueryBad)
			Expect(err).NotTo(HaveOccurred())

			// CreateConversation invalid body
			cConvBad := e.NewContext(httptest.NewRequest(http.MethodPost, "/api/conversations", strings.NewReader("invalid")), httptest.NewRecorder())
			err = h.CreateConversation(cConvBad)
			Expect(err).NotTo(HaveOccurred())

			// ListTeamConversations invalid team_id
			cListBad := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder())
			cListBad.SetPathValues(echo.PathValues{{Name: "team_id", Value: "invalid"}})
			err = h.ListTeamConversations(cListBad)
			Expect(err).NotTo(HaveOccurred())

			// GetConversationMessages invalid id
			cMsgBad := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder())
			cMsgBad.SetPathValues(echo.PathValues{{Name: "id", Value: "invalid"}})
			err = h.GetConversationMessages(cMsgBad)
			Expect(err).NotTo(HaveOccurred())

			// SearchUsers q too short
			cSearchBad := e.NewContext(httptest.NewRequest(http.MethodGet, "/api/users/search?q=a", nil), httptest.NewRecorder())
			cSearchBad.Set("user_uid", "uid-123")
			userSvc := &mocks.IUserService{}
			hWithUser := endpoint.NewHandler(mockAppSvc, mockAuthSvc, userSvc)
			err = hWithUser.SearchUsers(cSearchBad)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should execute Query streaming with conversation creation, title generation, and SSE events", func() {
			userSvc := &mocks.IUserService{}
			hWithUser := endpoint.NewHandler(mockAppSvc, mockAuthSvc, userSvc)

			teamID := uuid.New()
			convID := uuid.New()
			incID := uuid.New()
			userID := uuid.New()
			user := &models.User{ID: userID, FirebaseUID: "uid-123"}

			userSvc.On("GetUserByFirebaseUID", mock.Anything, "uid-123").Return(user, nil)
			mockAppSvc.On("CreateConversation", mock.Anything, teamID, userID).Return(&models.Conversation{ID: convID, TeamID: teamID, UserID: userID}, nil)
			mockAppSvc.On("SaveMessage", mock.Anything, convID, "user", "What is the CPU usage?", "").Return(&models.Message{}, nil)
			mockAppSvc.On("SaveMessage", mock.Anything, convID, "assistant", "High usage", "").Return(&models.Message{}, nil)
			mockAppSvc.On("GenerateAndSaveTitle", mock.Anything, convID, "What is the CPU usage?", "High usage").Return("CPU Usage Query", nil)

			mockAppSvc.On("QueryStreamWithTools", mock.Anything, "What is the CPU usage?", mock.Anything, mock.Anything, teamID, incID).Run(func(args mock.Arguments) {
				ch := args.Get(3).(chan<- types.StreamEvent)
				ch <- types.StreamEvent{Type: "reasoning", Content: "Reasoning details"}
				ch <- types.StreamEvent{Type: "text", Content: "High usage"}
				ch <- types.StreamEvent{Type: "drain", Content: ""}
				ch <- types.StreamEvent{Type: "text", Content: "High usage"}
			}).Return(nil)

			reqBody := requests.ChatQueryRequest{
				Input:      "What is the CPU usage?",
				TeamID:     &teamID,
				IncidentID: &incID,
			}
			body, _ := json.Marshal(reqBody)
			req := httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.Set("user_uid", "uid-123")

			err := hWithUser.Query(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusOK))
		})

		It("should handle QueryStreamWithTools error in Query handler", func() {
			userSvc := &mocks.IUserService{}
			hWithUser := endpoint.NewHandler(mockAppSvc, mockAuthSvc, userSvc)
			convID := uuid.New()
			teamID := uuid.New()
			userID := uuid.New()
			user := &models.User{ID: userID, FirebaseUID: "uid-123"}

			userSvc.On("GetUserByFirebaseUID", mock.Anything, "uid-123").Return(user, nil)
			mockAppSvc.On("SaveMessage", mock.Anything, convID, "user", "error prompt", "").Return(&models.Message{}, nil)
			mockAppSvc.On("GetConversationByID", mock.Anything, convID).Return(&models.Conversation{ID: convID, TeamID: teamID}, nil)

			mockAppSvc.On("QueryStreamWithTools", mock.Anything, "error prompt", mock.Anything, mock.Anything, teamID).Return(errors.New("LLM timeout"))

			reqBody := requests.ChatQueryRequest{
				Input:          "error prompt",
				ConversationID: &convID,
			}
			body, _ := json.Marshal(reqBody)
			req := httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.Set("user_uid", "uid-123")

			err := hWithUser.Query(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusOK))
		})

		It("should handle 500 errors for CreateConversation, GetConversationMessages, and SearchUsers", func() {
			userSvc := &mocks.IUserService{}
			hWithUser := endpoint.NewHandler(mockAppSvc, mockAuthSvc, userSvc)
			userID := uuid.New()
			userSvc.On("GetUserByFirebaseUID", mock.Anything, "uid-123").Return(&models.User{ID: userID}, nil)

			// CreateConversation invalid team_id or service error
			cConvBadTeam := e.NewContext(httptest.NewRequest(http.MethodPost, "/api/conversations", strings.NewReader(`{"team_id":"invalid"}`)), httptest.NewRecorder())
			cConvBadTeam.Request().Header.Set("Content-Type", "application/json")
			cConvBadTeam.Set("user_uid", "uid-123")
			err := hWithUser.CreateConversation(cConvBadTeam)
			Expect(err).NotTo(HaveOccurred())

			teamID := uuid.New()
			cConvErr := e.NewContext(httptest.NewRequest(http.MethodPost, "/api/conversations", strings.NewReader(`{"team_id":"`+teamID.String()+`"}`)), httptest.NewRecorder())
			cConvErr.Request().Header.Set("Content-Type", "application/json")
			cConvErr.Set("user_uid", "uid-123")
			mockAppSvc.On("CreateConversation", mock.Anything, teamID, userID).Return(nil, errors.New("db error")).Once()
			err = hWithUser.CreateConversation(cConvErr)
			Expect(err).NotTo(HaveOccurred())

			// GetConversationMessages 500
			convID := uuid.New()
			cMsgErr := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder())
			cMsgErr.SetPathValues(echo.PathValues{{Name: "id", Value: convID.String()}})
			mockAppSvc.On("ListMessagesByConversation", mock.Anything, convID).Return(nil, errors.New("db error")).Once()
			err = h.GetConversationMessages(cMsgErr)
			Expect(err).NotTo(HaveOccurred())

			// SearchUsers 500
			cSearchErr := e.NewContext(httptest.NewRequest(http.MethodGet, "/api/users/search?q=engineer", nil), httptest.NewRecorder())
			cSearchErr.Set("user_uid", "uid-123")
			userSvc.On("SearchUsers", mock.Anything, "engineer", 10).Return(nil, errors.New("db error")).Once()
			err = hWithUser.SearchUsers(cSearchErr)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("IngestAlert", func() {
		It("should return 400 when body binding fails", func() {
			req := httptest.NewRequest(http.MethodPost, "/api/alerts", strings.NewReader("invalid body"))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := h.IngestAlert(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusBadRequest))
		})

		It("should return 400 when ServiceName is empty", func() {
			incID := uuid.New()
			reqBody := requests.AlertIngestRequest{
				IncidentID: &incID,
				Resource:   requests.ResourceInfo{Service: ""},
				Alert:      requests.AlertInfo{Severity: "high"},
			}
			body, _ := json.Marshal(reqBody)
			req := httptest.NewRequest(http.MethodPost, "/api/alerts", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := h.IngestAlert(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusBadRequest))
		})

		It("should return 500 when service IngestAlert fails", func() {
			incID := uuid.New()
			reqBody := requests.AlertIngestRequest{
				IncidentID: &incID,
				Resource:   requests.ResourceInfo{Service: "auth-service"},
				Alert:      requests.AlertInfo{Severity: "critical"},
				Metrics:    requests.MetricsInfo{ResponseLatency: 5000},
			}
			body, _ := json.Marshal(reqBody)
			req := httptest.NewRequest(http.MethodPost, "/api/alerts", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			mockAppSvc.On("IngestAlert", mock.Anything, mock.MatchedBy(func(r *requests.AlertIngestRequest) bool {
				return r != nil && r.Resource.Service == "auth-service"
			})).Return(errors.New("db error"))

			err := h.IngestAlert(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusInternalServerError))
		})

		It("should return 200 on successful alert ingestion with or without IncidentID", func() {
			incID := uuid.New()
			reqBody := requests.AlertIngestRequest{
				IncidentID: &incID,
				Resource:   requests.ResourceInfo{Service: "auth-service"},
				Alert:      requests.AlertInfo{Severity: "info"},
				Metrics:    requests.MetricsInfo{CPUUsage: 50},
			}
			body, _ := json.Marshal(reqBody)
			req := httptest.NewRequest(http.MethodPost, "/api/alerts", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			mockAppSvc.On("IngestAlert", mock.Anything, mock.MatchedBy(func(r *requests.AlertIngestRequest) bool {
				return r != nil && r.Resource.Service == "auth-service"
			})).Return(nil)

			err := h.IngestAlert(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Code).To(Equal(http.StatusOK))
		})

	})

})
