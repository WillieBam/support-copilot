package interfaces

import (
	"context"

	"github.com/WillieBam/support_copilot/backend/types/requests"
)

// ITool represents a pluggable tool that can be exposed to an LLM and executed.
type ITool interface {
	Name() string
	Definition() requests.LLMTool
	Execute(ctx context.Context, rawArgs string) (string, error)
}
