package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/WillieBam/support_copilot/backend/app/config"
	"github.com/WillieBam/support_copilot/backend/internal/interfaces"
	"github.com/WillieBam/support_copilot/backend/types"
	"github.com/WillieBam/support_copilot/backend/types/requests"
)

type ollamaClient struct {
	httpClient    *http.Client
	ollamaBaseUrl string
	ollamaModel   string
	keepAlive     string
	numCtx        int
}

func NewOllamaClient(cfg *config.Config) interfaces.IOllamaClient {
	baseUrl := strings.TrimRight(cfg.Ollama.BaseURL, "/")
	if baseUrl == "" {
		baseUrl = "http://localhost:11434"
	}

	model := strings.TrimSpace(cfg.Ollama.Model)
	if model == "" {
		model = "llama3.2:latest"
	}

	timeout := cfg.Ollama.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}

	keepAlive := cfg.Ollama.KeepAlive
	if keepAlive == "" {
		keepAlive = "30m"
	}

	numCtx := cfg.Ollama.NumCtx
	if numCtx <= 0 {
		numCtx = 2048
	}

	return &ollamaClient{
		httpClient:    &http.Client{Timeout: timeout},
		ollamaBaseUrl: baseUrl,
		ollamaModel:   model,
		keepAlive:     keepAlive,
		numCtx:        numCtx,
	}

}

func (c *ollamaClient) QueryStreamWithTools(ctx context.Context, req requests.OllamaChatRequest, streamChan chan<- types.StreamEvent) (*requests.OllamaMessage, error) {
	if req.Model == "" {
		req.Model = c.ollamaModel
	}
	if req.KeepAlive == "" {
		req.KeepAlive = c.keepAlive
	}
	if req.Options == nil && c.numCtx > 0 {
		req.Options = &requests.OllamaOptions{NumCtx: c.numCtx}
	}
	req.Stream = true

	url := c.ollamaBaseUrl + "/api/chat"
	payloadBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Ollama chat request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request context for Ollama: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, fmt.Errorf("[STREAM ERROR]: client aborted stream")
		}
		return nil, fmt.Errorf("failed communicating with Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama returned status code %d: %s", resp.StatusCode, string(body))
	}

	decoder := json.NewDecoder(resp.Body)
	var accumulatedToolCalls []requests.OllamaToolCall
	var fullContent strings.Builder

	for {
		var chunk struct {
			Message struct {
				Role      string                    `json:"role"`
				Content   string                    `json:"content"`
				ToolCalls []requests.OllamaToolCall `json:"tool_calls"`
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

	return &requests.OllamaMessage{
		Role:      "assistant",
		Content:   fullContent.String(),
		ToolCalls: accumulatedToolCalls,
	}, nil
}
