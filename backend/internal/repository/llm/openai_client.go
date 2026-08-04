package llm

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/WillieBam/support_copilot/backend/app/config"
	"github.com/WillieBam/support_copilot/backend/internal/interfaces"
	"github.com/WillieBam/support_copilot/backend/types"
	"github.com/WillieBam/support_copilot/backend/types/requests"
)

type openAIClient struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	model      string
}

func NewOpenAIClient(cfg *config.Config) interfaces.ILLMClient {
	baseURL := strings.TrimRight(cfg.LLM.BaseURL, "/")
	provider := strings.ToLower(cfg.LLM.Provider)

	if baseURL == "" || strings.HasPrefix(baseURL, "http://localhost:11434") {
		if provider == "gemini" {
			baseURL = "https://generativelanguage.googleapis.com/v1beta/openai"
		} else if provider == "openai" {
			baseURL = "https://api.openai.com/v1"
		}
	}

	if !strings.HasSuffix(baseURL, "/chat/completions") {
		baseURL = baseURL + "/chat/completions"
	}

	apiKey := cfg.LLM.APIKey

	model := cfg.LLM.Model
	if model == "" {
		if provider == "gemini" {
			model = "gemini-3.5-flash-lite"
		} else {
			model = "gpt-4o-mini"
		}
	}

	timeout := cfg.LLM.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}

	return &openAIClient{
		httpClient: newHTTPClient(timeout, cfg.LLM.TLSSkipVerify),
		baseURL:    baseURL,
		apiKey:     apiKey,
		model:      model,
	}
}

// newHTTPClient builds an http.Client with optional TLS verification disabled.
// Only set skipVerify=true in environments where the provider's certificate
// cannot be validated (e.g. corporate proxies, self-signed certs).
func newHTTPClient(timeout time.Duration, skipVerify bool) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if skipVerify {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // intentional, opt-in only
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
}

type googleExtraContent struct {
	ThoughtSignature string `json:"thought_signature,omitempty"`
}

type extraContent struct {
	Google *googleExtraContent `json:"google,omitempty"`
}

type openAIFunctionCall struct {
	Name             string        `json:"name"`
	Arguments        string        `json:"arguments"`
	ThoughtSignature string        `json:"thought_signature,omitempty"`
	ExtraContent     *extraContent `json:"extra_content,omitempty"`
}

type openAIToolCall struct {
	ID               string             `json:"id"`
	Type             string             `json:"type"`
	Function         openAIFunctionCall `json:"function"`
	ThoughtSignature string             `json:"thought_signature,omitempty"`
	ExtraContent     *extraContent      `json:"extra_content,omitempty"`
}

type openAIMessage struct {
	Role       string           `json:"role"`
	Content    interface{}      `json:"content"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

type openAITool struct {
	Type     string                  `json:"type"`
	Function requests.OllamaFunction `json:"function"`
}

type openAIChatRequest struct {
	Model    string          `json:"model"`
	Messages []openAIMessage `json:"messages"`
	Tools    []openAITool    `json:"tools,omitempty"`
	Stream   bool            `json:"stream"`
}

type openAIStreamChunk struct {
	Choices []struct {
		Delta struct {
			Role             string        `json:"role"`
			Content          string        `json:"content"`
			ThoughtSignature string        `json:"thought_signature"`
			ReasoningContent string        `json:"reasoning_content"`
			ExtraContent     *extraContent `json:"extra_content"`
			ToolCalls        []struct {
				Index            int           `json:"index"`
				ID               string        `json:"id"`
				Type             string        `json:"type"`
				ThoughtSignature string        `json:"thought_signature"`
				ExtraContent     *extraContent `json:"extra_content"`
				Function         struct {
					Name             string        `json:"name"`
					Arguments        interface{}   `json:"arguments"`
					ThoughtSignature string        `json:"thought_signature"`
					ExtraContent     *extraContent `json:"extra_content"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (c *openAIClient) QueryStreamWithTools(ctx context.Context, req requests.OllamaChatRequest, streamChan chan<- types.StreamEvent) (*requests.OllamaMessage, error) {
	model := req.Model
	if model == "" {
		model = c.model
	}

	var openAITools []openAITool
	if len(req.Tools) > 0 {
		for _, t := range req.Tools {
			openAITools = append(openAITools, openAITool{
				Type:     "function",
				Function: t.Function,
			})
		}
	}

	var openAIMessages []openAIMessage
	var lastToolCallIDs []string

	for _, msg := range req.Messages {
		converted := openAIMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}

		if msg.Role == "assistant" && len(msg.ToolCalls) > 0 {
			if strings.TrimSpace(msg.Content) == "" {
				converted.Content = nil
			}
			lastToolCallIDs = nil
			var tcs []openAIToolCall
			for i, tc := range msg.ToolCalls {
				toolID := tc.ID
				if toolID == "" {
					toolID = fmt.Sprintf("call_%d_%d", time.Now().UnixNano(), i)
				}
				lastToolCallIDs = append(lastToolCallIDs, toolID)

				argsStr := "{}"
				if tc.Function.Arguments != nil {
					b, _ := json.Marshal(tc.Function.Arguments)
					argsStr = string(b)
				}

				var extra *extraContent
				if tc.Function.ThoughtSignature != "" {
					extra = &extraContent{
						Google: &googleExtraContent{
							ThoughtSignature: tc.Function.ThoughtSignature,
						},
					}
				}

				fnCall := openAIFunctionCall{
					Name:             tc.Function.Name,
					Arguments:        argsStr,
					ThoughtSignature: tc.Function.ThoughtSignature, // empty = omitted by omitempty
					ExtraContent:     extra,
				}

				tcs = append(tcs, openAIToolCall{
					ID:               toolID,
					Type:             "function",
					Function:         fnCall,
					ThoughtSignature: tc.Function.ThoughtSignature, // empty = omitted by omitempty
					ExtraContent:     extra,
				})
			}
			converted.ToolCalls = tcs
		} else if msg.Role == "tool" {
			if len(lastToolCallIDs) > 0 {
				converted.ToolCallID = lastToolCallIDs[0]
				lastToolCallIDs = lastToolCallIDs[1:]
			} else {
				converted.ToolCallID = fmt.Sprintf("call_tool_%d", time.Now().UnixNano())
			}
		}

		openAIMessages = append(openAIMessages, converted)
	}

	openAIReq := openAIChatRequest{
		Model:    model,
		Messages: openAIMessages,
		Tools:    openAITools,
		Stream:   true,
	}

	payloadBytes, err := json.Marshal(openAIReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal OpenAI chat request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request context for LLM: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			slog.Error("[OPENAI CLIENT] Stream context canceled", "err", err)
			return nil, fmt.Errorf("[STREAM ERROR]: client aborted stream")
		}
		slog.Error("[OPENAI CLIENT] HTTP request to LLM failed", "url", c.baseURL, "err", err)
		return nil, fmt.Errorf("failed communicating with LLM API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		slog.Error("[OPENAI CLIENT] LLM API error response", "status", resp.StatusCode, "body", string(body))
		return nil, fmt.Errorf("LLM API returned status code %d: %s", resp.StatusCode, string(body))
	}

	reader := bufio.NewReader(resp.Body)
	var fullContent strings.Builder
	type toolCallAcc struct {
		id               string
		name             string
		argsBuilder      strings.Builder
		thoughtSignature string
	}
	accToolCalls := make(map[int]*toolCallAcc)
	// Thought signature may arrive in a leading delta chunk BEFORE the tool_calls delta.
	// We track the most recent per-stream thought_signature and attach it to subsequent tool calls.
	var pendingThoughtSig string

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("error reading stream line: %w", err)
		}

		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, ":") {
			// Skip empty lines or SSE keepalive comments
			continue
		}

		if !strings.HasPrefix(line, "data:") {
			continue
		}

		dataStr := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if dataStr == "[DONE]" {
			break
		}

		var chunk openAIStreamChunk
		if err := json.Unmarshal([]byte(dataStr), &chunk); err != nil {
			continue
		}

		if chunk.Error != nil && chunk.Error.Message != "" {
			return nil, fmt.Errorf("LLM API error: %s", chunk.Error.Message)
		}

		if len(chunk.Choices) == 0 {
			continue
		}

		delta := chunk.Choices[0].Delta

		// Capture thought_signature at the delta level (Gemini sends it before tool_calls, or in extra_content).
		if delta.ThoughtSignature != "" {
			pendingThoughtSig = delta.ThoughtSignature
		} else if delta.ExtraContent != nil && delta.ExtraContent.Google != nil && delta.ExtraContent.Google.ThoughtSignature != "" {
			pendingThoughtSig = delta.ExtraContent.Google.ThoughtSignature
		}

		streamText := delta.Content
		if streamText == "" {
			streamText = delta.ReasoningContent
		}
		if streamText != "" {
			fullContent.WriteString(streamText)
			if streamChan != nil {
				streamChan <- types.StreamEvent{
					Type:    "text",
					Content: streamText,
				}
			}
		}

		finishReason := chunk.Choices[0].FinishReason

		for _, tc := range delta.ToolCalls {
			idx := tc.Index
			if _, exists := accToolCalls[idx]; !exists {
				accToolCalls[idx] = &toolCallAcc{}
			}
			if tc.ID != "" {
				accToolCalls[idx].id = tc.ID
			}
			if tc.Function.Name != "" {
				accToolCalls[idx].name = tc.Function.Name
			}

			// Collect thought_signature from all possible locations, in priority order:
			// 1. tool_calls[i].thought_signature or extra_content
			// 2. tool_calls[i].function.thought_signature or extra_content
			// 3. delta-level thought_signature (accumulated in pendingThoughtSig)
			sig := tc.ThoughtSignature
			if sig == "" && tc.ExtraContent != nil && tc.ExtraContent.Google != nil {
				sig = tc.ExtraContent.Google.ThoughtSignature
			}
			if sig == "" {
				sig = tc.Function.ThoughtSignature
			}
			if sig == "" && tc.Function.ExtraContent != nil && tc.Function.ExtraContent.Google != nil {
				sig = tc.Function.ExtraContent.Google.ThoughtSignature
			}
			if sig == "" {
				sig = pendingThoughtSig
			}
			if sig != "" {
				accToolCalls[idx].thoughtSignature = sig
			}

			if tc.Function.Arguments != nil {
				switch v := tc.Function.Arguments.(type) {
				case string:
					accToolCalls[idx].argsBuilder.WriteString(v)
				default:
					b, _ := json.Marshal(v)
					accToolCalls[idx].argsBuilder.Write(b)
				}
			}
		}

		if finishReason != "" {
			break
		}
	}

	var finalToolCalls []requests.OllamaToolCall
	for i := 0; i < len(accToolCalls); i++ {
		acc, exists := accToolCalls[i]
		if !exists || acc.name == "" {
			continue
		}
		if acc.thoughtSignature == "" && pendingThoughtSig != "" {
			acc.thoughtSignature = pendingThoughtSig
		}
		rawArgs := strings.TrimSpace(acc.argsBuilder.String())
		argsMap := make(map[string]interface{})
		if rawArgs != "" {
			_ = json.Unmarshal([]byte(rawArgs), &argsMap)
		}

		finalToolCalls = append(finalToolCalls, requests.OllamaToolCall{
			ID: acc.id,
			Function: requests.OllamaFunctionCall{
				Name:             acc.name,
				Arguments:        argsMap,
				ThoughtSignature: acc.thoughtSignature, // only set if genuinely received
			},
		})
	}

	return &requests.OllamaMessage{
		Role:      "assistant",
		Content:   fullContent.String(),
		ToolCalls: finalToolCalls,
	}, nil
}
