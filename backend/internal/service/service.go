package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/WillieBam/support_copilot/backend/internal/classifier"
	"github.com/WillieBam/support_copilot/backend/internal/command"
	"github.com/WillieBam/support_copilot/backend/internal/interfaces"
	"github.com/WillieBam/support_copilot/backend/internal/tools"
	"github.com/WillieBam/support_copilot/backend/types"
	"github.com/WillieBam/support_copilot/backend/types/models"
	"github.com/WillieBam/support_copilot/backend/types/requests"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AppService struct {
	alertRepo          interfaces.IAlertRepository
	ollamaClient       interfaces.IOllamaClient
	mcpClient          interfaces.IMCPClient
	orchestrator       interfaces.IOrchestratorService
	toolRegistry       interfaces.IToolRegistry
	commandInterceptor interfaces.ICommandInterceptor
	intentClassifier   interfaces.IIntentClassifier
	convRepo           interfaces.IConversationRepository
}

func NewAppService(alertRepo interfaces.IAlertRepository, ollamaClient interfaces.IOllamaClient, mcpClient interfaces.IMCPClient, opts ...interface{}) interfaces.IAppService {
	orchestrator := NewOrchestratorService(alertRepo, mcpClient)

	var registry interfaces.IToolRegistry
	var cmdInterceptor interfaces.ICommandInterceptor
	var intentCls interfaces.IIntentClassifier
	var convRepo interfaces.IConversationRepository

	for _, arg := range opts {
		if tr, ok := arg.(interfaces.IToolRegistry); ok && tr != nil {
			registry = tr
		}
		if ci, ok := arg.(interfaces.ICommandInterceptor); ok && ci != nil {
			cmdInterceptor = ci
		}
		if ic, ok := arg.(interfaces.IIntentClassifier); ok && ic != nil {
			intentCls = ic
		}
		if cr, ok := arg.(interfaces.IConversationRepository); ok && cr != nil {
			convRepo = cr
		}
	}

	if registry == nil {
		tr := tools.NewToolRegistry()
		tools.RegisterDefaultTools(tr, orchestrator)
		registry = tr
	}

	if cmdInterceptor == nil {
		cmdInterceptor = command.NewCommandInterceptor()
	}

	if intentCls == nil {
		intentCls = classifier.NewIntentClassifier()
	}

	return &AppService{
		alertRepo:          alertRepo,
		ollamaClient:       ollamaClient,
		mcpClient:          mcpClient,
		orchestrator:       orchestrator,
		toolRegistry:       registry,
		commandInterceptor: cmdInterceptor,
		intentClassifier:   intentCls,
		convRepo:           convRepo,
	}
}

func (s *AppService) IngestAlert(ctx context.Context, incidentID uuid.UUID, serviceName, severity, metrics string) error {
	var buf bytes.Buffer
	if err := json.Compact(&buf, []byte(metrics)); err != nil {
		slog.Warn("metrics field is not valid JSON, storing raw value", "err", err)
		buf.WriteString(metrics)
	}

	alert := &models.Alert{
		ID:          uuid.New(),
		IncidentID:  incidentID,
		ServiceName: serviceName,
		Severity:    severity,
		Metrics:     buf.String(),
		ReceivedAt:  time.Now(),
	}
	return s.alertRepo.StoreAlert(ctx, alert)
}

func isValidToolCallArgs(toolName string, args map[string]interface{}) bool {
	if toolName == "validate_alert" {
		alertIDVal, exists := args["alert_id"]
		if !exists || alertIDVal == nil {
			return false
		}
		alertID, ok := alertIDVal.(string)
		if !ok {
			return false
		}
		alertID = strings.TrimSpace(alertID)
		if alertID == "" || alertID == "null" || alertID == "none" || alertID == "undefined" || alertID == "{alert_id}" || alertID == "00000000-0000-0000-0000-000000000000" {
			return false
		}
		if _, err := uuid.Parse(alertID); err != nil {
			return false
		}
	}
	return true
}

func (s *AppService) Intercept(ctx context.Context, prompt string) (*types.CommandResult, error) {
	if s.commandInterceptor != nil {
		return s.commandInterceptor.Intercept(ctx, prompt)
	}
	return &types.CommandResult{Handled: false}, nil
}

func (s *AppService) QueryStreamWithTools(ctx context.Context, prompt string, history []types.HistoryMessage, streamChan chan<- types.StreamEvent) error {
	slog.Info("[APP SERVICE] QueryStreamWithTools started", "prompt", prompt)

	res, err := s.Intercept(ctx, prompt)
	if err != nil {
		slog.Error("[APP SERVICE] Command interceptor error", "err", err)
		return err
	}
	if res != nil && res.Handled {
		slog.Info("[APP SERVICE] Prompt intercepted by command parser", "prompt", prompt)
		if res.Message != "" {
			streamChan <- types.StreamEvent{
				Type:    "text",
				Content: res.Message,
			}
		}
		return nil
	}

	systemPrompt := `You are a Support Copilot that helps support engineers resolve production incidents.

## Behaviour Rules
- Respond conversationally (no tools) when the user sends greetings, acknowledgments,
  sign-offs, or short social messages such as "ok", "thanks", "bye", "got it", "alright",
  "yes", "no", or any similar phrase.
- Call tools ONLY when the user explicitly provides an alert ID (UUID format) or clearly
  requests alert validation or system inspection.
- If you are uncertain whether a tool call is appropriate, respond with plain text and ask
  the user for the alert ID instead of calling the tool with a placeholder value.
- Never fabricate alert IDs or call tools with placeholder values such as "null", "none",
  or "00000000-0000-0000-0000-000000000000".
- When the conversation is winding down (e.g. the user says "thanks", "bye", "ok"), reply
  with a short, friendly closing message and do not call any tools.`

	// build the full multi-turn messages array:
	//   [system] + [history turns...] + [current user message]
	// this is to remain LLM full conversation context so it can remember context of a conversation
	messages := []requests.OllamaMessage{
		{Role: "system", Content: systemPrompt},
	}
	for _, h := range history {
		if h.Role == "user" || h.Role == "assistant" {
			messages = append(messages, requests.OllamaMessage{
				Role:    h.Role,
				Content: h.Content,
			})
		}
	}
	messages = append(messages, requests.OllamaMessage{Role: "user", Content: prompt})

	// classify the user's intent to decide whether to expose tool
	// For conversational prompts the tool list is withheld entirely so the LLM
	// physically cannot make a tool call
	intent := s.intentClassifier.Classify(prompt)
	slog.Info("[APP SERVICE] Intent classified", "intent", intent, "prompt", prompt)

	var availableTools []requests.OllamaTool
	if intent == classifier.IntentTask {
		availableTools = s.toolRegistry.GetTools()
	} else {
		slog.Info("[APP SERVICE] Conversational intent detected — withholding tools from Ollama request")
	}

	req := requests.OllamaChatRequest{
		Messages: messages,
		Tools:    availableTools,
	}

	// Call Ollama streaming with tool declarations dynamically provided by ToolRegistry
	assistantMsg, err := s.ollamaClient.QueryStreamWithTools(ctx, req, streamChan)
	if err != nil {
		slog.Error("[APP SERVICE] First pass QueryStreamWithTools failed", "err", err)
		return err
	}

	// detect when the LLM has emitted a raw JSON tool-call object as plain text (e.g. {"name":"greet","parameters":{"message":"I"}}).
	// checking here is to suppress that kind of response
	if assistantMsg != nil && classifier.LooksLikeEmbeddedToolCall(assistantMsg.Content) {
		slog.Warn("[APP SERVICE] LLM emitted embedded JSON tool-call as text — suppressing and falling back", "content", assistantMsg.Content)
		streamChan <- types.StreamEvent{Type: "drain", Content: ""}
		fallbackReq := requests.OllamaChatRequest{Messages: messages}
		_, fallbackErr := s.ollamaClient.QueryStreamWithTools(ctx, fallbackReq, streamChan)
		return fallbackErr
	}

	// check if LLM requested a tool execution
	if assistantMsg != nil && len(assistantMsg.ToolCalls) > 0 {
		// emit reasoning event so React UI displays it immediately in the reasoning block
		for _, toolCall := range assistantMsg.ToolCalls {
			toolName := toolCall.Function.Name
			slog.Info("[APP SERVICE] Ollama triggered tool call", "tool", toolName, "args", toolCall.Function.Arguments)

			// pre-check tool arguments for dummy/invalid values BEFORE executing or emitting tool reasoning
			if !isValidToolCallArgs(toolName, toolCall.Function.Arguments) {
				slog.Warn("[APP SERVICE] Tool call skipped due to dummy or missing valid parameters", "tool", toolName, "args", toolCall.Function.Arguments)

				if strings.TrimSpace(assistantMsg.Content) == "" {
					slog.Info("[APP SERVICE] Falling back to direct text response without tools")
					fallbackReq := requests.OllamaChatRequest{
						Messages: messages,
					}
					_, fallbackErr := s.ollamaClient.QueryStreamWithTools(ctx, fallbackReq, streamChan)
					return fallbackErr
				}
				continue
			}
			streamChan <- types.StreamEvent{
				Type:    "reasoning",
				Content: fmt.Sprintln("🔍 Accessing Database to get alert... "),
			}

			streamChan <- types.StreamEvent{
				Type:    "reasoning",
				Content: fmt.Sprintf("🔍 Intercepted tool call: %s. Executing tool...\n", toolName),
			}

			argsBytes, _ := json.Marshal(toolCall.Function.Arguments)
			toolResult, err := s.toolRegistry.Execute(ctx, toolName, string(argsBytes))
			if err != nil {
				slog.Warn("[APP SERVICE] Tool execution failed via ToolRegistry", "tool", toolName, "err", err)

				// if tool call failed (e.g. invalid alert_id "null") and no content was streamed yet,
				// fall back to a conversational text stream without tools to avoid sending raw error noise to user
				if strings.TrimSpace(assistantMsg.Content) == "" {
					slog.Info("[APP SERVICE] Falling back to direct text response without tools")
					fallbackReq := requests.OllamaChatRequest{
						Messages: messages,
					}
					_, fallbackErr := s.ollamaClient.QueryStreamWithTools(ctx, fallbackReq, streamChan)
					return fallbackErr
				}
				toolResult = fmt.Sprintf(`{"error": "%s"}`, err.Error())
			}

			slog.Info("[APP SERVICE] Tool result retrieved", "tool", toolName, "resultLen", len(toolResult))

			messages = append(messages, *assistantMsg)
			messages = append(messages, requests.OllamaMessage{
				Role:    "tool",
				Content: toolResult,
			})

			// 2nd Pass: Stream Ollama's final synthesis based on the tool result
			secondReq := requests.OllamaChatRequest{
				Messages: messages,
			}
			_, err = s.ollamaClient.QueryStreamWithTools(ctx, secondReq, streamChan)
			return err
		}
	}

	return nil
}

func (s *AppService) RetrieveAlert(ctx context.Context, id uuid.UUID) (*models.Alert, error) {
	alert, err := s.alertRepo.RetrieveAlertbyID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("alert not found")
		}
		return nil, err
	}
	return alert, nil
}

// createconversation initializes a new conversation entry for team and user
func (s *AppService) CreateConversation(ctx context.Context, teamID, userID uuid.UUID) (*models.Conversation, error) {
	if s.convRepo == nil {
		return nil, errors.New("conversation repository not configured")
	}
	conv := &models.Conversation{
		ID:        uuid.New(),
		TeamID:    teamID,
		UserID:    userID,
		Title:     "New Conversation",
		CreatedAt: time.Now(),
	}
	if err := s.convRepo.CreateConversation(ctx, conv); err != nil {
		return nil, err
	}
	return conv, nil
}

// getconversationbyid retrieves a conversation by id
func (s *AppService) GetConversationByID(ctx context.Context, id uuid.UUID) (*models.Conversation, error) {
	if s.convRepo == nil {
		return nil, errors.New("conversation repository not configured")
	}
	return s.convRepo.GetConversationByID(ctx, id)
}

// listteamconversations retrieves team conversations up to limit
func (s *AppService) ListTeamConversations(ctx context.Context, teamID uuid.UUID, limit int) ([]models.Conversation, error) {
	if s.convRepo == nil {
		return nil, errors.New("conversation repository not configured")
	}
	return s.convRepo.ListTeamConversations(ctx, teamID, limit)
}

// savemessage persists a chat message in the repository
func (s *AppService) SaveMessage(ctx context.Context, convID uuid.UUID, sender, content, reasoning string) (*models.Message, error) {
	if s.convRepo == nil {
		return nil, errors.New("conversation repository not configured")
	}
	msg := &models.Message{
		ID:             uuid.New(),
		ConversationID: convID,
		Sender:         sender,
		Content:        content,
		ReasoningSteps: reasoning,
		CreatedAt:      time.Now(),
	}
	if err := s.convRepo.CreateMessage(ctx, msg); err != nil {
		return nil, err
	}
	return msg, nil
}

// listmessagesbyconversation fetches all messages for a conversation
func (s *AppService) ListMessagesByConversation(ctx context.Context, convID uuid.UUID) ([]models.Message, error) {
	if s.convRepo == nil {
		return nil, errors.New("conversation repository not configured")
	}
	return s.convRepo.ListMessagesByConversation(ctx, convID)
}

// generateandsavetitle generates a session title from user prompt and assistant response
func (s *AppService) GenerateAndSaveTitle(ctx context.Context, convID uuid.UUID, userPrompt, assistantReply string) (string, error) {
	if s.convRepo == nil {
		return "", errors.New("conversation repository not configured")
	}

	titlePrompt := fmt.Sprintf(
		"Based on this user query and assistant response, summarize the conversation topic into a short title (5 words or less). Return ONLY the title text, no quotes or additional words.\n\nUser: %s\n\nAssistant: %s",
		userPrompt, assistantReply,
	)

	titleChan := make(chan types.StreamEvent, 64)
	go func() {
		// drain stream events during background title generation
		for range titleChan {
		}
	}()

	msg, err := s.ollamaClient.QueryStreamWithTools(ctx, requests.OllamaChatRequest{
		Messages: []requests.OllamaMessage{
			{Role: "system", Content: "You generate concise conversation titles. Output only the title text."},
			{Role: "user", Content: titlePrompt},
		},
	}, titleChan)
	close(titleChan)

	if err != nil {
		slog.Error("[APP SERVICE] Failed to generate conversation title", "err", err, "conv_id", convID)
		return "", err
	}

	title := strings.TrimSpace(msg.Content)
	title = strings.Trim(title, `"'`)
	if title == "" {
		title = "Chat Conversation"
	}
	if len(title) > 255 {
		title = title[:255]
	}

	slog.Info("[APP SERVICE] Generated title for conversation", "conv_id", convID, "title", title)
	err = s.convRepo.UpdateConversationTitle(ctx, convID, title)
	return title, err
}

