package service

import (
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
	promptpkg "github.com/WillieBam/support_copilot/backend/internal/prompt"
	"github.com/WillieBam/support_copilot/backend/internal/tools"
	"github.com/WillieBam/support_copilot/backend/types"
	"github.com/WillieBam/support_copilot/backend/types/models"
	"github.com/WillieBam/support_copilot/backend/types/requests"
	customErrors "github.com/WillieBam/support_copilot/backend/utils/errors"
	"github.com/google/uuid"
)

type AppService struct {
	alertRepo          interfaces.IAlertRepository
	llmClient          interfaces.ILLMClient
	mcpClient          interfaces.IMCPClient
	orchestrator       interfaces.IOrchestratorService
	toolRegistry       interfaces.IToolRegistry
	commandInterceptor interfaces.ICommandInterceptor
	intentClassifier   interfaces.IIntentClassifier
	convRepo           interfaces.IConversationRepository
	teamRepo           interfaces.ITeamRepository
}

func NewAppService(alertRepo interfaces.IAlertRepository, llmClient interfaces.ILLMClient, mcpClient interfaces.IMCPClient, opts ...interface{}) interfaces.IAppService {
	var registry interfaces.IToolRegistry
	var cmdInterceptor interfaces.ICommandInterceptor
	var intentCls interfaces.IIntentClassifier
	var convRepo interfaces.IConversationRepository
	var teamRepo interfaces.ITeamRepository
	var mcpClient2 interfaces.IMCP2Client

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
		if tm, ok := arg.(interfaces.ITeamRepository); ok && tm != nil {
			teamRepo = tm
		}
		if m2, ok := arg.(interfaces.IMCP2Client); ok && m2 != nil {
			mcpClient2 = m2
		}
	}

	orchestrator := NewOrchestratorService(alertRepo, mcpClient, mcpClient2, teamRepo)

	if registry == nil {
		tr := tools.NewToolRegistry()
		tools.RegisterDefaultTools(tr, orchestrator)
		registry = tr
	}

	if cmdInterceptor == nil {
		cmdInterceptor = command.NewCommandInterceptor(orchestrator)
	}

	if intentCls == nil {
		intentCls = classifier.NewIntentClassifier()
	}

	return &AppService{
		alertRepo:          alertRepo,
		llmClient:          llmClient,
		mcpClient:          mcpClient,
		orchestrator:       orchestrator,
		toolRegistry:       registry,
		commandInterceptor: cmdInterceptor,
		intentClassifier:   intentCls,
		convRepo:           convRepo,
		teamRepo:           teamRepo,
	}
}

func (s *AppService) IngestAlert(ctx context.Context, req *requests.AlertIngestRequest) error {
	if req == nil {
		return errors.New("alert ingest request cannot be nil")
	}
	slog.Info("[Alert Ingestion] Ingestion process started", "service", req.Resource.Service, "severity", req.Alert.Severity, "alert_info_id", req.Alert.ID)

	alertBytes, _ := json.Marshal(req.Alert)
	resourceBytes, _ := json.Marshal(req.Resource)
	metricsBytes, _ := json.Marshal(req.Metrics)
	bizBytes, _ := json.Marshal(req.BusinessContext)
	metaBytes, _ := json.Marshal(req.Metadata)

	alert := &models.Alert{
		ID:              uuid.New(),
		IncidentID:      req.IncidentID,
		AlertInfo:       string(alertBytes),
		ResourceInfo:    string(resourceBytes),
		Metrics:         string(metricsBytes),
		BusinessContext: string(bizBytes),
		Metadata:        string(metaBytes),
		ReceivedAt:      time.Now(),
	}

	if err := s.alertRepo.StoreAlert(ctx, alert); err != nil {
		slog.Error("[Alert Ingestion] Failed to store alert in database", "alert_id", alert.ID, "service", req.Resource.Service, "err", err)
		return err
	}

	slog.Info("[Alert Ingestion] Alert successfully stored in database", "alert_id", alert.ID, "service", req.Resource.Service)
	return nil
}

// formatFallbackMarkdown converts raw JSON tool results into clean markdown tables if LLM outputs empty content
func formatFallbackMarkdown(toolName, toolResult string) string {
	if toolName == "list_incidents" {
		var incs []struct {
			ID        string `json:"id"`
			Title     string `json:"title"`
			Status    string `json:"status"`
			CreatedAt string `json:"created_at"`
		}
		if err := json.Unmarshal([]byte(toolResult), &incs); err == nil && len(incs) > 0 {
			var sb strings.Builder
			sb.WriteString("\n\n### Team Incidents\n\n")
			sb.WriteString("| Title | Status | Incident ID | Created |\n")
			sb.WriteString("| --- | --- | --- | --- |\n")
			for _, inc := range incs {
				sb.WriteString(fmt.Sprintf("| %s | `%s` | `%s` | %s |\n", inc.Title, inc.Status, inc.ID, inc.CreatedAt))
			}
			return sb.String()
		}
	}
	if toolName == "get_incident" {
		var inc struct {
			IncidentID       string `json:"incident_id"`
			Title            string `json:"title"`
			Status           string `json:"status"`
			Age              string `json:"age"`
			Details          string `json:"details"`
			ExistingRunbooks []struct {
				ID    string `json:"id"`
				Title string `json:"title"`
			} `json:"existing_runbooks"`
		}
		if err := json.Unmarshal([]byte(toolResult), &inc); err == nil && inc.IncidentID != "" {
			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("\n\n### Incident Details: %s\n\n", inc.Title))
			sb.WriteString(fmt.Sprintf("- **Incident ID**: `%s`\n", inc.IncidentID))
			sb.WriteString(fmt.Sprintf("- **Status**: `%s`\n", inc.Status))
			if inc.Age != "" {
				sb.WriteString(fmt.Sprintf("- **Age**: %s\n", inc.Age))
			}
			if inc.Details != "" {
				sb.WriteString(fmt.Sprintf("\n#### Summary & Telemetry\n%s\n", inc.Details))
			}
			if len(inc.ExistingRunbooks) > 0 {
				sb.WriteString("\n#### Linked Runbooks\n")
				for _, rb := range inc.ExistingRunbooks {
					sb.WriteString(fmt.Sprintf("- **%s** (`%s`)\n", rb.Title, rb.ID))
				}
			}
			return sb.String()
		}
	}
	if toolName == "create_runbook" || toolName == "get_runbook" || toolName == "update_runbook" {
		var rb struct {
			ID         string `json:"id"`
			IncidentID string `json:"incident_id"`
			Title      string `json:"title"`
			Status     string `json:"status"`
			Content    string `json:"content"`
		}
		if err := json.Unmarshal([]byte(toolResult), &rb); err == nil && rb.ID != "" {
			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("\n\n### 📘 Operational Runbook: %s\n\n", rb.Title))
			sb.WriteString(fmt.Sprintf("- **Runbook ID**: `%s`\n", rb.ID))
			sb.WriteString(fmt.Sprintf("- **Status**: `%s`\n", rb.Status))
			if rb.IncidentID != "" {
				sb.WriteString(fmt.Sprintf("- **Incident ID**: `%s`\n", rb.IncidentID))
			}
			sb.WriteString("\n---\n\n")
			sb.WriteString(rb.Content)
			return sb.String()
		}
	}
	if strings.Contains(toolResult, `"error"`) {
		var errResp struct {
			Error string `json:"error"`
		}
		if err := json.Unmarshal([]byte(toolResult), &errResp); err == nil && errResp.Error != "" {
			return fmt.Sprintf("\n\n**Operational Request Failed** (`%s`): %s\n", toolName, errResp.Error)
		}
	}
	if len(toolResult) > 0 && len(toolResult) < 2000 {
		return fmt.Sprintf("\n\n### Tool Execution Result (`%s`)\n```json\n%s\n```", toolName, strings.TrimSpace(toolResult))
	}
	return fmt.Sprintf("\n\nSuccessfully retrieved operational data via `%s`.", toolName)
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
	}
	if toolName == "link_alert_to_incident" {
		alertStr, _ := args["alert_id"].(string)
		incidentStr, _ := args["incident_id"].(string)
		incidentTitle, _ := args["incident_title"].(string)

		alertStr = strings.TrimSpace(alertStr)
		incidentStr = strings.TrimSpace(incidentStr)
		incidentTitle = strings.TrimSpace(incidentTitle)

		if alertStr == "" {
			// Some models place the alert UUID in incident_id. Recover from that shape.
			if _, err := uuid.Parse(incidentStr); err == nil {
				alertStr = incidentStr
			}
		}
		if alertStr == "" {
			return false
		}
		if _, err := uuid.Parse(alertStr); err != nil {
			return false
		}

		if incidentStr == "" && incidentTitle == "" {
			return false
		}
	}
	if toolName == "get_incident" || toolName == "list_incidents" {
		for _, key := range []string{"incident_id", "team_id"} {
			if val, exists := args[key]; exists && val != nil {
				if str, ok := val.(string); ok && strings.TrimSpace(str) == "" {
					return false
				}
			}
		}
	}
	if toolName == "create_runbook" || toolName == "update_runbook" || toolName == "deprecate_runbook" || toolName == "get_runbook" || toolName == "list_runbooks" {
		for _, key := range []string{"team_id", "incident_id", "runbook_id", "title", "content"} {
			if val, exists := args[key]; exists && val != nil {
				str, ok := val.(string)
				if !ok || strings.TrimSpace(str) == "" {
					return false
				}
				// team_id and UUIDs must parse correctly
				if key == "team_id" || key == "incident_id" || key == "runbook_id" {
					if _, err := uuid.Parse(strings.TrimSpace(str)); err != nil {
						return false
					}
				}
			}
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

func (s *AppService) QueryStreamWithTools(ctx context.Context, prompt string, history []types.HistoryMessage, streamChan chan<- types.StreamEvent, opts ...interface{}) error {
	slog.Info("[APP SERVICE] QueryStreamWithTools started", "prompt", prompt)

	var teamID uuid.UUID
	var activeIncidentID uuid.UUID
	for _, opt := range opts {
		if id, ok := opt.(uuid.UUID); ok && id != uuid.Nil {
			if teamID == uuid.Nil {
				teamID = id
			} else {
				activeIncidentID = id
			}
		}
	}

	if teamID == uuid.Nil && s.teamRepo != nil {
		if incidents, err := s.teamRepo.ListTeamIncidents(ctx, uuid.Nil); err == nil && len(incidents) > 0 {
			teamID = incidents[0].TeamID
			slog.Info("[APP SERVICE] Resolved default team_id from team repository at service layer", "team_id", teamID)
		}
	}

	if teamID != uuid.Nil {
		ctx = command.WithTeamID(ctx, teamID)
	}
	if activeIncidentID != uuid.Nil {
		ctx = command.WithActiveIncidentID(ctx, activeIncidentID)
	}

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

	// emit instant reasoning status event so UI shows live activity immediately before calling Ollama
	streamChan <- types.StreamEvent{
		Type:    "reasoning",
		Content: "🧠 Analyzing prompt and evaluating available tools...\n",
	}

	systemPrompt := promptpkg.SystemPrompt

	if teamID != uuid.Nil || activeIncidentID != uuid.Nil {
		systemPrompt += "\n\n## Current Workspace Context"
		if teamID != uuid.Nil {
			systemPrompt += fmt.Sprintf("\n- Active Team ID: `%s`", teamID.String())
		}
		if activeIncidentID != uuid.Nil {
			systemPrompt += fmt.Sprintf("\n- Active Incident ID: `%s`", activeIncidentID.String())
		}
	}

	if teamID != uuid.Nil && s.teamRepo != nil {
		inst, _, err := s.teamRepo.GetTeamInstruction(ctx, teamID)
		if err == nil && inst != nil && strings.TrimSpace(inst.InstructionDetails) != "" {
			slog.Info("[APP SERVICE] Injecting custom team instructions into system prompt", "team_id", teamID)
			systemPrompt += fmt.Sprintf("\n\n## Team-Specific Custom Instructions\nThe engineering team for this workspace has provided the following custom instructions. You MUST strictly adhere to them in all your responses:\n%s", strings.TrimSpace(inst.InstructionDetails))
		}
	}

	// build the full multi-turn messages array:
	//   [system] + [history turns...] + [current user message]
	// this is to remain LLM full conversation context so it can remember context of a conversation
	messages := []requests.LLMMessage{
		{Role: "system", Content: systemPrompt},
	}
	for _, h := range history {
		if h.Role == "user" || h.Role == "assistant" {
			messages = append(messages, requests.LLMMessage{
				Role:    h.Role,
				Content: h.Content,
			})
		}
	}
	messages = append(messages, requests.LLMMessage{Role: "user", Content: prompt})

	// classify the user's intent to decide whether to expose tool
	// For conversational prompts the tool list is withheld entirely so the LLM
	// physically cannot make a tool call
	intent := s.intentClassifier.ClassifyWithHistory(prompt, history)
	slog.Info("[APP SERVICE] Intent classified", "intent", intent, "prompt", prompt)

	var availableTools []requests.LLMTool
	if intent == classifier.IntentTask {
		availableTools = s.toolRegistry.GetTools()
	} else {
		slog.Info("[APP SERVICE] Conversational intent detected — withholding tools from LLM request")
	}

	req := requests.LLMChatRequest{
		Messages: messages,
		Tools:    availableTools,
	}

	// Call LLM streaming with tool declarations dynamically provided by ToolRegistry
	assistantMsg, err := s.llmClient.QueryStreamWithTools(ctx, req, streamChan)
	if err != nil {
		slog.Error("[APP SERVICE] First pass QueryStreamWithTools failed", "err", err)
		if errors.Is(err, customErrors.ErrRateLimitExceeded) || errors.Is(err, customErrors.ErrServiceUnavailable) {
			streamChan <- types.StreamEvent{
				Type:    "text",
				Content: fmt.Sprintf("\n\n⚠️ **Service Notice**: %s\n", err.Error()),
			}
			return nil
		}
		return err
	}

	// detect when the LLM has emitted a raw JSON tool-call object as plain text (e.g. {"name":"create_runbook","parameters":{...}}).
	if assistantMsg != nil && classifier.LooksLikeEmbeddedToolCall(assistantMsg.Content) {
		slog.Info("[APP SERVICE] LLM emitted embedded JSON tool-call as text — parsing into tool call struct", "content", assistantMsg.Content)
		if parsedCall, parseErr := classifier.ParseEmbeddedToolCall(assistantMsg.Content); parseErr == nil {
			assistantMsg.ToolCalls = append(assistantMsg.ToolCalls, *parsedCall)
			assistantMsg.Content = ""
			streamChan <- types.StreamEvent{Type: "drain", Content: ""}
		} else {
			slog.Warn("[APP SERVICE] Failed to parse embedded JSON tool call, suppressing and falling back", "err", parseErr)
			streamChan <- types.StreamEvent{Type: "drain", Content: ""}
			fallbackReq := requests.LLMChatRequest{Messages: messages}
			_, fallbackErr := s.llmClient.QueryStreamWithTools(ctx, fallbackReq, streamChan)
			return fallbackErr
		}
	}

	// check if LLM requested a tool execution
	if assistantMsg != nil && len(assistantMsg.ToolCalls) > 0 {
		// emit reasoning event so React UI displays it immediately in the reasoning block
		for _, toolCall := range assistantMsg.ToolCalls {
			toolName := toolCall.Function.Name

			// Auto-populate team_id if omitted, empty, uuid.Nil, or not a valid UUID (e.g. Groq generated "your_team_id")
			if toolCall.Function.Arguments != nil && teamID != uuid.Nil {
				current, _ := toolCall.Function.Arguments["team_id"].(string)
				parsed, err := uuid.Parse(strings.TrimSpace(current))
				if err != nil || parsed == uuid.Nil {
					toolCall.Function.Arguments["team_id"] = teamID.String()
					slog.Info("[APP SERVICE] Auto-populated team_id for tool call", "tool", toolName, "team_id", teamID.String())
				}
			}

			slog.Info("[APP SERVICE] LLM triggered tool call", "tool", toolName, "args", toolCall.Function.Arguments)

			// pre-check tool arguments for dummy/invalid values BEFORE executing or emitting tool reasoning
			if !isValidToolCallArgs(toolName, toolCall.Function.Arguments) {
				slog.Warn("[APP SERVICE] Tool call skipped due to dummy or missing valid parameters", "tool", toolName, "args", toolCall.Function.Arguments)

				if strings.TrimSpace(assistantMsg.Content) == "" {
					slog.Info("[APP SERVICE] Falling back to direct text response without tools")
					fallbackReq := requests.LLMChatRequest{
						Messages: messages,
					}
					_, fallbackErr := s.llmClient.QueryStreamWithTools(ctx, fallbackReq, streamChan)
					return fallbackErr
				}
				continue
			}
			streamChan <- types.StreamEvent{
				Type:    "reasoning",
				Content: fmt.Sprintln("🔍 Accessing Database to get data... "),
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
					fallbackReq := requests.LLMChatRequest{
						Messages: messages,
					}
					_, fallbackErr := s.llmClient.QueryStreamWithTools(ctx, fallbackReq, streamChan)
					return fallbackErr
				}
				toolResult = fmt.Sprintf(`{"error": "%s"}`, err.Error())
			}

			slog.Info("[APP SERVICE] Tool result retrieved", "tool", toolName, "resultLen", len(toolResult))

			messages = append(messages, *assistantMsg)
			messages = append(messages, requests.LLMMessage{
				Role:    "tool",
				Content: toolResult,
			})

			// 2nd pass: stream llm's final synthesis based on the tool result
			slog.Info("[APP SERVICE] Starting Pass 2 synthesis with LLM...")
			secondReq := requests.LLMChatRequest{
				Messages: messages,
			}
			pass2Msg, err := s.llmClient.QueryStreamWithTools(ctx, secondReq, streamChan)
			if err != nil {
				slog.Error("[APP SERVICE] Pass 2 synthesis failed", "err", err)
			} else {
				contentLen := 0
				if pass2Msg != nil {
					contentLen = len(pass2Msg.Content)
				}
				slog.Info("[APP SERVICE] Pass 2 synthesis completed successfully", "contentLen", contentLen)
				if pass2Msg == nil || strings.TrimSpace(pass2Msg.Content) == "" {
					slog.Warn("[APP SERVICE] Pass 2 LLM generated empty content, emitting fallback summary")
					fallbackText := formatFallbackMarkdown(toolName, toolResult)
					streamChan <- types.StreamEvent{Type: "text", Content: fallbackText}
				}
			}
			return err
		}
	}

	return nil
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

	msg, err := s.llmClient.QueryStreamWithTools(ctx, requests.LLMChatRequest{
		Messages: []requests.LLMMessage{
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
