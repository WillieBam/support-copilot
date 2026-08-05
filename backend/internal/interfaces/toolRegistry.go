package interfaces

import (
	"context"

	"github.com/WillieBam/support_copilot/backend/types/requests"
)

type IToolRegistry interface {
	Register(name string, tool requests.OllamaTool, handler func(ctx context.Context, rawArgs string) (string, error))
	GetTools() []requests.OllamaTool
	Execute(ctx context.Context, name string, rawArgs string) (string, error)
}
