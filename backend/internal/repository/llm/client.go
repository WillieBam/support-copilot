package llm

import (
	"bytes"
	"context"
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
	customErrors "github.com/WillieBam/support_copilot/backend/utils/errors"
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

	var resp *http.Response
	var lastErr error
	maxRetries := 2

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(attempt*500) * time.Millisecond
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payloadBytes))
		if err != nil {
			return nil, fmt.Errorf("failed to create LLM request context: %w", err)
		}
		httpReq.Header.Set("Content-Type", "application/json")

		resp, lastErr = c.httpClient.Do(httpReq)
		if lastErr != nil {
			if errors.Is(lastErr, context.Canceled) {
				return nil, fmt.Errorf("[STREAM ERROR]: client aborted stream")
			}
			continue
		}

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable {
			resp.Body.Close()
			lastErr = fmt.Errorf("LLM status %d", resp.StatusCode)
			continue
		}

		break
	}

	if lastErr != nil && resp == nil {
		return nil, fmt.Errorf("failed communicating with LLM: %w", lastErr)
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		slog.Error("OLLAMA CLIENT] Rate limit exceeded", "status", resp.StatusCode, "body", string(body))
		return nil, customErrors.ErrRateLimitExceeded
	}

	if resp.StatusCode == http.StatusServiceUnavailable {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		slog.Error("[OLLAMA CLIENT] Service unavailable", "status", resp.StatusCode, "body", string(body))
		return nil, customErrors.ErrServiceUnavailable
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("LLM returned status code %d: %s", resp.StatusCode, string(body))
	}
	defer resp.Body.Close()

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
			if streamChan != nil {
				streamChan <- types.StreamEvent{
					Type:    "text",
					Content: chunk.Message.Content,
				}
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
