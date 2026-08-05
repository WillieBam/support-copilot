package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/WillieBam/support_copilot/backend/app/config"
	"github.com/WillieBam/support_copilot/backend/internal/interfaces"
	"github.com/WillieBam/support_copilot/backend/types"
	"github.com/WillieBam/support_copilot/backend/types/requests"
)

type ollamaClient struct {
	httpClient *http.Client
	baseURL    string
	model      string
	keepAlive  string
	numCtx     int
}

func NewOllamaClient(cfg *config.Config) interfaces.ILLMClient {
	baseUrl := strings.TrimRight(cfg.LLM.BaseURL, "/")

	model := strings.TrimSpace(cfg.LLM.Model)

	timeout := cfg.LLM.Timeout

	keepAlive := cfg.LLM.KeepAlive

	numCtx := cfg.LLM.NumCtx

	return &ollamaClient{
		httpClient: newHTTPClient(timeout, cfg.LLM.TLSSkipVerify),
		baseURL:    baseUrl,
		model:      model,
		keepAlive:  keepAlive,
		numCtx:     numCtx,
	}
}

func (c *ollamaClient) QueryStreamWithTools(ctx context.Context, req requests.LLMChatRequest, streamChan chan<- types.StreamEvent) (*requests.LLMMessage, error) {
	if req.Model == "" {
		req.Model = c.model
	}
	if req.KeepAlive == "" {
		req.KeepAlive = c.keepAlive
	}
	if req.Options == nil && c.numCtx > 0 {
		req.Options = &requests.LLMOptions{NumCtx: c.numCtx}
	}
	req.Stream = true

	url := c.baseURL + "/api/chat"
	payloadBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal LLM chat request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM request context: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, fmt.Errorf("[STREAM ERROR]: client aborted stream")
		}
		return nil, fmt.Errorf("failed communicating with LLM: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("LLM returned status code %d: %s", resp.StatusCode, string(body))
	}

	decoder := json.NewDecoder(resp.Body)
	var accumulatedToolCalls []requests.LLMToolCall
	var fullContent strings.Builder

	for {
		var chunk struct {
			Message struct {
				Role      string                 `json:"role"`
				Content   string                 `json:"content"`
				ToolCalls []requests.LLMToolCall `json:"tool_calls"`
			} `json:"message"`
			Done  bool   `json:"done"`
			Error string `json:"error"`
		}

		if err := decoder.Decode(&chunk); err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("error decoding ollama chunk: %w", err)
		}

		if chunk.Error != "" {
			return nil, fmt.Errorf("ollama API error: %s", chunk.Error)
		}

		if chunk.Message.Content != "" {
			fullContent.WriteString(chunk.Message.Content)
			streamChan <- types.StreamEvent{
				Type:    "text",
				Content: chunk.Message.Content,
			}
		}

		if len(chunk.Message.ToolCalls) > 0 {
			accumulatedToolCalls = append(accumulatedToolCalls, chunk.Message.ToolCalls...)
		}

		if chunk.Done {
			break
		}
	}

	return &requests.LLMMessage{
		Role:      "assistant",
		Content:   fullContent.String(),
		ToolCalls: accumulatedToolCalls,
	}, nil
}
