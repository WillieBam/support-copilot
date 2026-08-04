package interfaces

import (
	"context"

	"github.com/WillieBam/support_copilot/backend/types"
	"github.com/WillieBam/support_copilot/backend/types/requests"
)

type ILLMClient interface {
	QueryStreamWithTools(ctx context.Context, req requests.OllamaChatRequest, streamChan chan<- types.StreamEvent) (*requests.OllamaMessage, error)
}

type IOllamaClient = ILLMClient
