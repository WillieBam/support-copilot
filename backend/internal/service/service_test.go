package service_test

import (
	"context"
	"errors"

	"github.com/WillieBam/support_copilot/backend/types/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	"github.com/WillieBam/support_copilot/backend/internal/classifier"
	"github.com/WillieBam/support_copilot/backend/internal/interfaces"
	"github.com/WillieBam/support_copilot/backend/internal/mocks"
	"github.com/WillieBam/support_copilot/backend/internal/service"
	"github.com/WillieBam/support_copilot/backend/types"
	"github.com/WillieBam/support_copilot/backend/types/requests"
)

var _ = Describe("AppService (Streaming & Alerts)", func() {
	var (
		appSvc        interfaces.IAppService
		mockAlertRepo *mocks.IAlertRepository
		mockLLM       *mocks.ILLMClient
		mockMcpOne    *mocks.IMCPClient
		ctx           context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		mockAlertRepo = &mocks.IAlertRepository{}
		mockLLM = &mocks.ILLMClient{}
		mockMcpOne = &mocks.IMCPClient{}

		appSvc = service.NewAppService(mockAlertRepo, mockLLM, mockMcpOne)
	})

	Context("QueryStream", func() {
		It("should connect to Ollama server and stream token events correctly", func() {
			mockLLM.On("QueryStreamWithTools", mock.Anything, mock.Anything, mock.Anything).
				Return(&requests.OllamaMessage{Role: "assistant", Content: "Hello world!"}, nil).
				Run(func(args mock.Arguments) {
					streamChan := args.Get(2).(chan<- types.StreamEvent)
					streamChan <- types.StreamEvent{Type: "text", Content: "Hello"}
					streamChan <- types.StreamEvent{Type: "text", Content: " world!"}
				})

			streamChan := make(chan types.StreamEvent, 10)

			err := appSvc.QueryStreamWithTools(ctx, "hello test", nil, streamChan)
			Expect(err).NotTo(HaveOccurred())
			close(streamChan)

			var events []types.StreamEvent
			for ev := range streamChan {
				events = append(events, ev)
			}

			Expect(len(events)).To(Equal(3))
			Expect(events[0].Type).To(Equal("reasoning"))
			Expect(events[0].Content).To(ContainSubstring("Analyzing prompt"))
			Expect(events[1].Type).To(Equal("text"))
			Expect(events[1].Content).To(Equal("Hello"))
			Expect(events[2].Type).To(Equal("text"))
			Expect(events[2].Content).To(Equal(" world!"))
			mockLLM.AssertExpectations(GinkgoT())
		})

		It("should return an error if the server returns non-200 status code", func() {
			mockLLM.On("QueryStreamWithTools", mock.Anything, mock.Anything, mock.Anything).
				Return(nil, errors.New("Ollama returned status 500: ollama internal error"))

			streamChan := make(chan types.StreamEvent, 10)

			err := appSvc.QueryStreamWithTools(ctx, "hello test", nil, streamChan)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Ollama returned status 500"))
			mockLLM.AssertExpectations(GinkgoT())
		})

		It("should return an error if client cancels the context", func() {
			mockLLM.On("QueryStreamWithTools", mock.Anything, mock.Anything, mock.Anything).
				Return(nil, context.Canceled)

			cancelCtx, cancel := context.WithCancel(ctx)
			cancel() // cancel immediately

			streamChan := make(chan types.StreamEvent, 10)

			err := appSvc.QueryStreamWithTools(cancelCtx, "hello test", nil, streamChan)
			Expect(err).To(HaveOccurred())
			mockLLM.AssertExpectations(GinkgoT())
		})

		It("should fallback to direct text response if tool call receives dummy alert_id null", func() {
			toolCallMsg := &requests.OllamaMessage{
				Role: "assistant",
				ToolCalls: []requests.OllamaToolCall{
					{
						Function: requests.OllamaFunctionCall{
							Name: "validate_alert",
							Arguments: map[string]interface{}{
								"alert_id": "null",
							},
						},
					},
				},
			}
			// First call returns tool call with alert_id "null"
			mockLLM.On("QueryStreamWithTools", mock.Anything, mock.Anything, mock.Anything).Return(toolCallMsg, nil).Once()

			// Fallback call
			fallbackMsg := &requests.OllamaMessage{Role: "assistant", Content: "You're welcome!"}
			mockLLM.On("QueryStreamWithTools", mock.Anything, mock.Anything, mock.Anything).Return(fallbackMsg, nil).Once()

			streamChan := make(chan types.StreamEvent, 10)

			err := appSvc.QueryStreamWithTools(ctx, "alright , thanks", nil, streamChan)
			Expect(err).NotTo(HaveOccurred())
			close(streamChan)

			mockLLM.AssertExpectations(GinkgoT())
		})

		It("should intercept /quit prompt and halt processing without invoking Ollama", func() {
			streamChan := make(chan types.StreamEvent, 10)

			err := appSvc.QueryStreamWithTools(ctx, "/quit", nil, streamChan)
			Expect(err).NotTo(HaveOccurred())
			close(streamChan)

			var events []types.StreamEvent
			for ev := range streamChan {
				events = append(events, ev)
			}

			Expect(len(events)).To(Equal(1))
			Expect(events[0].Type).To(Equal("text"))
			Expect(events[0].Content).To(ContainSubstring("LLM processing stopped by /quit command"))

			// Assert Ollama was never called
			mockLLM.AssertNotCalled(GinkgoT(), "QueryStreamWithTools", mock.Anything, mock.Anything, mock.Anything)
		})

		It("should delegate to mock ICommandInterceptor when provided", func() {
			mockInterceptor := &mocks.ICommandInterceptor{}
			customAppSvc := service.NewAppService(mockAlertRepo, mockLLM, mockMcpOne, mockInterceptor)

			mockInterceptor.On("Intercept", mock.Anything, "/quit").Return(&types.CommandResult{
				Handled: true,
				Message: "Custom quit message",
			}, nil)

			streamChan := make(chan types.StreamEvent, 10)

			err := customAppSvc.QueryStreamWithTools(ctx, "/quit", nil, streamChan)
			Expect(err).NotTo(HaveOccurred())
			close(streamChan)

			var events []types.StreamEvent
			for ev := range streamChan {
				events = append(events, ev)
			}

			Expect(len(events)).To(Equal(1))
			Expect(events[0].Content).To(Equal("Custom quit message"))
			mockInterceptor.AssertExpectations(GinkgoT())
		})

		It("should intercept /incident prompt and return result without calling LLM", func() {
			mockInterceptor := &mocks.ICommandInterceptor{}
			customAppSvc := service.NewAppService(mockAlertRepo, mockLLM, mockMcpOne, mockInterceptor)

			mockInterceptor.On("Intercept", mock.Anything, "/incident redis").Return(&types.CommandResult{
				Handled: true,
				Message: "found matching incident for redis",
			}, nil)

			streamChan := make(chan types.StreamEvent, 10)
			err := customAppSvc.QueryStreamWithTools(ctx, "/incident redis", nil, streamChan)
			Expect(err).NotTo(HaveOccurred())
			close(streamChan)

			var events []types.StreamEvent
			for ev := range streamChan {
				events = append(events, ev)
			}

			Expect(len(events)).To(Equal(1))
			Expect(events[0].Type).To(Equal("text"))
			Expect(events[0].Content).To(Equal("found matching incident for redis"))
			mockLLM.AssertNotCalled(GinkgoT(), "QueryStreamWithTools", mock.Anything, mock.Anything, mock.Anything)
			mockInterceptor.AssertExpectations(GinkgoT())
		})

		It("should intercept /runbook prompt and return result without calling LLM", func() {
			mockInterceptor := &mocks.ICommandInterceptor{}
			customAppSvc := service.NewAppService(mockAlertRepo, mockLLM, mockMcpOne, mockInterceptor)

			mockInterceptor.On("Intercept", mock.Anything, "/runbook database").Return(&types.CommandResult{
				Handled: true,
				Message: "found matching runbook for database",
			}, nil)

			streamChan := make(chan types.StreamEvent, 10)
			err := customAppSvc.QueryStreamWithTools(ctx, "/runbook database", nil, streamChan)
			Expect(err).NotTo(HaveOccurred())
			close(streamChan)

			var events []types.StreamEvent
			for ev := range streamChan {
				events = append(events, ev)
			}

			Expect(len(events)).To(Equal(1))
			Expect(events[0].Type).To(Equal("text"))
			Expect(events[0].Content).To(Equal("found matching runbook for database"))
			mockLLM.AssertNotCalled(GinkgoT(), "QueryStreamWithTools", mock.Anything, mock.Anything, mock.Anything)
			mockInterceptor.AssertExpectations(GinkgoT())
		})

		It("should withhold tools from Ollama when intent is conversational (ok byebye)", func() {
			// Expect Ollama to be called with an empty tools slice
			mockLLM.On("QueryStreamWithTools", mock.Anything,
				mock.MatchedBy(func(req requests.OllamaChatRequest) bool {
					return len(req.Tools) == 0
				}),
				mock.Anything,
			).Return(&requests.OllamaMessage{Role: "assistant", Content: "Goodbye!"}, nil).Once()

			streamChan := make(chan types.StreamEvent, 10)
			err := appSvc.QueryStreamWithTools(ctx, "ok byebye", nil, streamChan)
			Expect(err).NotTo(HaveOccurred())
			mockLLM.AssertExpectations(GinkgoT())
		})

		It("should expose tools to Ollama when intent is task (validate alert uuid)", func() {
			// Expect Ollama to be called with at least one tool in the list
			mockLLM.On("QueryStreamWithTools", mock.Anything,
				mock.MatchedBy(func(req requests.OllamaChatRequest) bool {
					return len(req.Tools) > 0
				}),
				mock.Anything,
			).Return(&requests.OllamaMessage{Role: "assistant", Content: "Validating..."}, nil).Once()

			streamChan := make(chan types.StreamEvent, 10)
			err := appSvc.QueryStreamWithTools(ctx, "validate alert 550e8400-e29b-41d4-a716-446655440000", nil, streamChan)
			Expect(err).NotTo(HaveOccurred())
			mockLLM.AssertExpectations(GinkgoT())
		})

		It("should withhold tools when a mock IIntentClassifier returns IntentConversational", func() {
			mockCls := &mocks.IIntentClassifier{}
			mockCls.On("ClassifyWithHistory", "thanks mate", mock.Anything).Return(classifier.IntentConversational)

			customAppSvc := service.NewAppService(mockAlertRepo, mockLLM, mockMcpOne, mockCls)

			mockLLM.On("QueryStreamWithTools", mock.Anything,
				mock.MatchedBy(func(req requests.OllamaChatRequest) bool {
					return len(req.Tools) == 0
				}),
				mock.Anything,
			).Return(&requests.OllamaMessage{Role: "assistant", Content: "You're welcome!"}, nil).Once()

			streamChan := make(chan types.StreamEvent, 10)
			err := customAppSvc.QueryStreamWithTools(ctx, "thanks mate", nil, streamChan)
			Expect(err).NotTo(HaveOccurred())
			mockCls.AssertExpectations(GinkgoT())
			mockLLM.AssertExpectations(GinkgoT())
		})

		It("should execute tool calls during first pass and stream second pass LLM response", func() {
			mockToolReg := &mocks.IToolRegistry{}
			mockToolReg.On("GetTools").Return([]requests.LLMTool{})
			customAppSvc := service.NewAppService(mockAlertRepo, mockLLM, mockMcpOne, mockToolReg)

			firstPassMsg := &requests.OllamaMessage{
				Role: "assistant",
				ToolCalls: []requests.OllamaToolCall{
					{
						Function: requests.OllamaFunctionCall{
							Name:      "get_incident",
							Arguments: map[string]interface{}{"incident_id": "INC-101"},
						},
					},
				},
			}
			mockLLM.On("QueryStreamWithTools", mock.Anything, mock.Anything, mock.Anything).Return(firstPassMsg, nil).Once()

			mockToolReg.On("Execute", mock.Anything, "get_incident", mock.Anything).Return(`{"incident_id":"INC-101","title":"CPU Spike","status":"OPEN"}`, nil)

			secondPassMsg := &requests.OllamaMessage{Role: "assistant", Content: "Incident INC-101 details retrieved"}
			mockLLM.On("QueryStreamWithTools", mock.Anything, mock.Anything, mock.Anything).Return(secondPassMsg, nil).Once()

			streamChan := make(chan types.StreamEvent, 10)
			err := customAppSvc.QueryStreamWithTools(ctx, "get incident INC-101", nil, streamChan)
			Expect(err).NotTo(HaveOccurred())
			close(streamChan)
		})

		It("should format fallback markdown when second pass returns empty content", func() {
			mockToolReg := &mocks.IToolRegistry{}
			mockToolReg.On("GetTools").Return([]requests.LLMTool{})
			customAppSvc := service.NewAppService(mockAlertRepo, mockLLM, mockMcpOne, mockToolReg)

			firstPassMsg := &requests.OllamaMessage{
				Role: "assistant",
				ToolCalls: []requests.OllamaToolCall{
					{
						Function: requests.OllamaFunctionCall{
							Name:      "get_incident",
							Arguments: map[string]interface{}{"incident_id": "INC-101"},
						},
					},
				},
			}
			mockLLM.On("QueryStreamWithTools", mock.Anything, mock.Anything, mock.Anything).Return(firstPassMsg, nil).Once()

			mockToolReg.On("Execute", mock.Anything, "get_incident", mock.Anything).Return(`{"incident_id":"INC-101","title":"CPU Spike","status":"OPEN","age":"1h","details":"telemetry","existing_runbooks":[{"id":"RB-1","title":"Restart Guide"}]}`, nil)

			secondPassMsg := &requests.OllamaMessage{Role: "assistant", Content: ""}
			mockLLM.On("QueryStreamWithTools", mock.Anything, mock.Anything, mock.Anything).Return(secondPassMsg, nil).Once()

			streamChan := make(chan types.StreamEvent, 10)
			err := customAppSvc.QueryStreamWithTools(ctx, "get incident INC-101", nil, streamChan)
			Expect(err).NotTo(HaveOccurred())
			close(streamChan)

			var events []types.StreamEvent
			for ev := range streamChan {
				events = append(events, ev)
			}
			Expect(len(events)).To(BeNumerically(">", 0))
		})

		It("should format fallback markdown for list_incidents and create_runbook tool calls", func() {
			mockToolReg := &mocks.IToolRegistry{}
			mockToolReg.On("GetTools").Return([]requests.LLMTool{})
			customAppSvc := service.NewAppService(mockAlertRepo, mockLLM, mockMcpOne, mockToolReg)

			// Spec for list_incidents fallback
			firstPassMsgList := &requests.OllamaMessage{
				Role: "assistant",
				ToolCalls: []requests.OllamaToolCall{
					{Function: requests.OllamaFunctionCall{Name: "list_incidents", Arguments: map[string]interface{}{"team_id": "team-1"}}},
				},
			}
			mockLLM.On("QueryStreamWithTools", mock.Anything, mock.Anything, mock.Anything).Return(firstPassMsgList, nil).Once()
			mockToolReg.On("Execute", mock.Anything, "list_incidents", mock.Anything).Return(`[{"id":"INC-101","title":"Memory Spike","status":"OPEN","created_at":"2026-08-11"}]`, nil)
			mockLLM.On("QueryStreamWithTools", mock.Anything, mock.Anything, mock.Anything).Return(&requests.OllamaMessage{Role: "assistant", Content: ""}, nil).Once()

			streamChanList := make(chan types.StreamEvent, 10)
			err := customAppSvc.QueryStreamWithTools(ctx, "list team incidents", nil, streamChanList)
			Expect(err).NotTo(HaveOccurred())
			close(streamChanList)

			// Spec for create_runbook fallback
			firstPassMsgRb := &requests.OllamaMessage{
				Role: "assistant",
				ToolCalls: []requests.OllamaToolCall{
					{Function: requests.OllamaFunctionCall{Name: "create_runbook", Arguments: map[string]interface{}{"team_id": "t-1", "incident_id": "i-1", "title": "Guide", "content": "steps"}}},
				},
			}
			mockLLM.On("QueryStreamWithTools", mock.Anything, mock.Anything, mock.Anything).Return(firstPassMsgRb, nil).Once()
			mockToolReg.On("Execute", mock.Anything, "create_runbook", mock.Anything).Return(`{"id":"RB-101","incident_id":"INC-1","title":"Guide","status":"active","content":"rollout"}` , nil)
			mockLLM.On("QueryStreamWithTools", mock.Anything, mock.Anything, mock.Anything).Return(&requests.OllamaMessage{Role: "assistant", Content: ""}, nil).Once()

			streamChanRb := make(chan types.StreamEvent, 10)
			err = customAppSvc.QueryStreamWithTools(ctx, "create runbook guide", nil, streamChanRb)
			Expect(err).NotTo(HaveOccurred())
			close(streamChanRb)
		})

		It("should skip tool execution when tool call arguments are invalid or dummy", func() {
			// Spec for validate_alert invalid alert_id (non-string, undefined, zero-uuid)
			dummyMsg := &requests.OllamaMessage{
				Role: "assistant",
				ToolCalls: []requests.OllamaToolCall{
					{Function: requests.OllamaFunctionCall{Name: "validate_alert", Arguments: map[string]interface{}{}}},
					{Function: requests.OllamaFunctionCall{Name: "validate_alert", Arguments: map[string]interface{}{"alert_id": nil}}},
					{Function: requests.OllamaFunctionCall{Name: "validate_alert", Arguments: map[string]interface{}{"alert_id": 12345}}},
					{Function: requests.OllamaFunctionCall{Name: "validate_alert", Arguments: map[string]interface{}{"alert_id": "none"}}},
					{Function: requests.OllamaFunctionCall{Name: "validate_alert", Arguments: map[string]interface{}{"alert_id": "{alert_id}"}}},
					{Function: requests.OllamaFunctionCall{Name: "validate_alert", Arguments: map[string]interface{}{"alert_id": "00000000-0000-0000-0000-000000000000"}}},
					{Function: requests.OllamaFunctionCall{Name: "link_alert_to_incident", Arguments: map[string]interface{}{"alert_id": "invalid-uuid"}}},
					{Function: requests.OllamaFunctionCall{Name: "link_alert_to_incident", Arguments: map[string]interface{}{"alert_id": uuid.New().String()}}}, // missing incident_id and incident_title
					{Function: requests.OllamaFunctionCall{Name: "get_incident", Arguments: map[string]interface{}{"incident_id": "  "}}},
					{Function: requests.OllamaFunctionCall{Name: "create_runbook", Arguments: map[string]interface{}{"title": ""}}},
					{Function: requests.OllamaFunctionCall{Name: "create_runbook", Arguments: map[string]interface{}{"team_id": "not-a-uuid"}}},
				},
			}
			mockLLM.On("QueryStreamWithTools", mock.Anything, mock.Anything, mock.Anything).Return(dummyMsg, nil).Once()
			mockLLM.On("QueryStreamWithTools", mock.Anything, mock.Anything, mock.Anything).Return(&requests.OllamaMessage{Role: "assistant", Content: "Fallback text"}, nil).Once()

			streamChan := make(chan types.StreamEvent, 10)
			err := appSvc.QueryStreamWithTools(ctx, "do task", nil, streamChan)
			Expect(err).NotTo(HaveOccurred())
			close(streamChan)
		})
	})

	Context("Conversation helpers", func() {
		It("should create a conversation when the repository is configured", func() {
			mockConvRepo := &mocks.IConversationRepository{}
			customAppSvc := service.NewAppService(mockAlertRepo, mockLLM, mockMcpOne, mockConvRepo)
			teamID := uuid.New()
			userID := uuid.New()

			mockConvRepo.On("CreateConversation", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
				conv := args.Get(1).(*models.Conversation)
				Expect(conv.TeamID).To(Equal(teamID))
				Expect(conv.UserID).To(Equal(userID))
			})

			conv, err := customAppSvc.CreateConversation(ctx, teamID, userID)
			Expect(err).NotTo(HaveOccurred())
			Expect(conv).NotTo(BeNil())
			Expect(conv.Title).To(Equal("New Conversation"))
		})

		It("should return an error when conversation repository is not configured", func() {
			conv, err := appSvc.CreateConversation(ctx, uuid.New(), uuid.New())
			Expect(err).To(HaveOccurred())
			Expect(conv).To(BeNil())

			c, err := appSvc.GetConversationByID(ctx, uuid.New())
			Expect(err).To(HaveOccurred())
			Expect(c).To(BeNil())

			convs, err := appSvc.ListTeamConversations(ctx, uuid.New(), 10)
			Expect(err).To(HaveOccurred())
			Expect(convs).To(BeNil())

			msgs, err := appSvc.ListMessagesByConversation(ctx, uuid.New())
			Expect(err).To(HaveOccurred())
			Expect(msgs).To(BeNil())
		})

		It("should retrieve conversation, team conversations, and messages when repository is configured", func() {
			mockConvRepo := &mocks.IConversationRepository{}
			customAppSvc := service.NewAppService(mockAlertRepo, mockLLM, mockMcpOne, mockConvRepo)
			convID := uuid.New()
			teamID := uuid.New()

			mockConvRepo.On("GetConversationByID", mock.Anything, convID).Return(&models.Conversation{ID: convID, Title: "Test Conv"}, nil)
			mockConvRepo.On("ListTeamConversations", mock.Anything, teamID, 10).Return([]models.Conversation{{ID: convID, Title: "Test Conv"}}, nil)
			mockConvRepo.On("ListMessagesByConversation", mock.Anything, convID).Return([]models.Message{{ID: uuid.New(), Content: "Hello"}}, nil)

			conv, err := customAppSvc.GetConversationByID(ctx, convID)
			Expect(err).NotTo(HaveOccurred())
			Expect(conv.Title).To(Equal("Test Conv"))

			convs, err := customAppSvc.ListTeamConversations(ctx, teamID, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(convs)).To(Equal(1))

			msgs, err := customAppSvc.ListMessagesByConversation(ctx, convID)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(msgs)).To(Equal(1))
		})

		It("should save messages and generate a title when the repository and LLM are available", func() {
			mockConvRepo := &mocks.IConversationRepository{}
			customAppSvc := service.NewAppService(mockAlertRepo, mockLLM, mockMcpOne, mockConvRepo)
			convID := uuid.New()

			mockConvRepo.On("CreateMessage", mock.Anything, mock.Anything).Return(nil)
			mockConvRepo.On("UpdateConversationTitle", mock.Anything, convID, "Summarized topic").Return(nil)
			mockLLM.On("QueryStreamWithTools", mock.Anything, mock.Anything, mock.Anything).Return(&requests.OllamaMessage{Role: "assistant", Content: "Summarized topic"}, nil)

			msg, err := customAppSvc.SaveMessage(ctx, convID, "assistant", "hello", "")
			Expect(err).NotTo(HaveOccurred())
			Expect(msg).NotTo(BeNil())

			title, err := customAppSvc.GenerateAndSaveTitle(ctx, convID, "hello", "hi")
			Expect(err).NotTo(HaveOccurred())
			Expect(title).To(Equal("Summarized topic"))
		})

		It("should return error when SaveMessage or GenerateAndSaveTitle is called without convRepo or llmClient", func() {
			msg, err := appSvc.SaveMessage(ctx, uuid.New(), "user", "content", "")
			Expect(err).To(HaveOccurred())
			Expect(msg).To(BeNil())

			title, err := appSvc.GenerateAndSaveTitle(ctx, uuid.New(), "prompt", "resp")
			Expect(err).To(HaveOccurred())
			Expect(title).To(BeEmpty())
		})
	})

	Context("IngestAlert", func() {
		It("should successfully store alert in repository", func() {
			incidentID := uuid.New()
			mockAlertRepo.On("StoreAlert", mock.Anything, mock.Anything).Return(nil)

			req := &requests.AlertIngestRequest{
				IncidentID: &incidentID,
				Resource:   requests.ResourceInfo{Service: "auth-service"},
				Alert:      requests.AlertInfo{Severity: "critical"},
			}
			err := appSvc.IngestAlert(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			mockAlertRepo.AssertExpectations(GinkgoT())
		})

		It("should return error if repository fails to store alert", func() {
			incidentID := uuid.New()
			mockAlertRepo.On("StoreAlert", mock.Anything, mock.Anything).Return(errors.New("db error"))

			req := &requests.AlertIngestRequest{
				IncidentID: &incidentID,
				Resource:   requests.ResourceInfo{Service: "auth-service"},
				Alert:      requests.AlertInfo{Severity: "critical"},
			}
			err := appSvc.IngestAlert(ctx, req)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("db error"))
			mockAlertRepo.AssertExpectations(GinkgoT())
		})
	})

})
