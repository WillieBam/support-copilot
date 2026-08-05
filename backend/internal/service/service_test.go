package service_test

import (
	"context"
	"errors"

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
		mockLLM *mocks.ILLMClient
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
	})

	Context("IngestAlert", func() {
		It("should successfully store alert in repository", func() {
			incidentID := uuid.New()
			mockAlertRepo.On("StoreAlert", mock.Anything, mock.Anything).Return(nil)

			err := appSvc.IngestAlert(ctx, &incidentID, "auth-service", "critical", "cpu_util > 90%")
			Expect(err).NotTo(HaveOccurred())
			mockAlertRepo.AssertExpectations(GinkgoT())
		})

		It("should return error if repository fails to store alert", func() {
			incidentID := uuid.New()
			mockAlertRepo.On("StoreAlert", mock.Anything, mock.Anything).Return(errors.New("db error"))

			err := appSvc.IngestAlert(ctx, &incidentID, "auth-service", "critical", "cpu_util > 90%")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("db error"))
			mockAlertRepo.AssertExpectations(GinkgoT())
		})
	})

	
})
