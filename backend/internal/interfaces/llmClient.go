package interfaces

import (
	"context"

	"github.com/WillieBam/support_copilot/backend/types"
	"github.com/WillieBam/support_copilot/backend/types/requests"
)

type ILLMClient interface {
	QueryStreamWithTools(ctx context.Context, req requests.LLMChatRequest, streamChan chan<- types.StreamEvent) (*requests.LLMMessage, error)
}
