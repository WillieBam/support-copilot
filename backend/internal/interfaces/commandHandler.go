package interfaces

import (
	"context"

	"github.com/WillieBam/support_copilot/backend/types"
)

// ICommandHandler represents a modular slash command handler.
type ICommandHandler interface {
	Command() string
	Description() string
	Handle(ctx context.Context, prompt string) (*types.CommandResult, error)
}
