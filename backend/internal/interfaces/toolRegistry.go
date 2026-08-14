package interfaces

import (
	"context"

	"github.com/WillieBam/support_copilot/backend/types/requests"
)

type IToolRegistry interface {
	Register(name string, tool requests.LLMTool, handler func(ctx context.Context, rawArgs string) (string, error))
	RegisterTool(tool ITool)
	GetTools() []requests.LLMTool
	Execute(ctx context.Context, name string, rawArgs string) (string, error)
}
