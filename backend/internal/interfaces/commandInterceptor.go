package interfaces

import (
	"context"

	"github.com/WillieBam/support_copilot/backend/types"
)

type ICommandInterceptor interface {
	Intercept(ctx context.Context, prompt string) (*types.CommandResult, error)
	RegisterHandler(handler ICommandHandler)
}
