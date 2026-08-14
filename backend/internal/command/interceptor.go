package command

import (
	"context"
	"strings"
	"sync"

	"github.com/WillieBam/support_copilot/backend/internal/interfaces"
	"github.com/WillieBam/support_copilot/backend/types"
)

// CommandHandler is the function signature for legacy slash command handlers.
type CommandHandler func(ctx context.Context, prompt string) (*types.CommandResult, error)

type CommandInterceptor struct {
	mu           sync.RWMutex
	handlers     map[string]interfaces.ICommandHandler
	orchestrator interfaces.IOrchestratorService
}

func NewCommandInterceptor(orchestrator ...interfaces.IOrchestratorService) interfaces.ICommandInterceptor {
	ci := &CommandInterceptor{
		handlers: make(map[string]interfaces.ICommandHandler),
	}
	if len(orchestrator) > 0 {
		ci.orchestrator = orchestrator[0]
	}

	ci.RegisterHandler(NewQuitCommandHandler())
	ci.RegisterHandler(NewIncidentCommandHandler(ci.orchestrator))
	ci.RegisterHandler(NewRunbookCommandHandler(ci.orchestrator))
	ci.RegisterHandler(NewAlertCommandHandler(ci.orchestrator))
	ci.RegisterHandler(NewHelpCommandHandler(ci))

	return ci
}

func (ci *CommandInterceptor) RegisterHandler(handler interfaces.ICommandHandler) {
	ci.mu.Lock()
	defer ci.mu.Unlock()
	ci.handlers[strings.ToLower(handler.Command())] = handler
}

func (ci *CommandInterceptor) RegisterCommand(command string, handler CommandHandler) {
	ci.RegisterHandler(NewFuncCommandHandler(strings.ToLower(command), "", handler))
}

func (ci *CommandInterceptor) GetHandlers() map[string]interfaces.ICommandHandler {
	ci.mu.RLock()
	defer ci.mu.RUnlock()
	copyMap := make(map[string]interfaces.ICommandHandler, len(ci.handlers))
	for k, v := range ci.handlers {
		copyMap[k] = v
	}
	return copyMap
}

// Intercept is the function to check prompt against all registered commands
func (ci *CommandInterceptor) Intercept(ctx context.Context, prompt string) (*types.CommandResult, error) {
	ci.mu.RLock()
	defer ci.mu.RUnlock()

	trimmed := strings.ToLower(strings.TrimSpace(prompt))
	for cmd, handler := range ci.handlers {
		if trimmed == cmd || strings.HasPrefix(trimmed, cmd+" ") || strings.HasPrefix(trimmed, cmd) {
			return handler.Handle(ctx, prompt)
		}
	}
	return &types.CommandResult{Handled: false}, nil
}
