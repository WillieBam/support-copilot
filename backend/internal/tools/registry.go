package tools

import (
	"context"
	"fmt"
	"sync"

	"github.com/WillieBam/support_copilot/backend/internal/interfaces"
	"github.com/WillieBam/support_copilot/backend/types/requests"
)

type ToolHandler func(ctx context.Context, rawArgs string) (string, error)

type ToolDefinition struct {
	Tool    requests.LLMTool
	Handler ToolHandler
}

type ToolRegistry struct {
	mu    sync.RWMutex
	tools map[string]ToolDefinition
}

func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]ToolDefinition),
	}
}

func (r *ToolRegistry) Register(name string, tool requests.LLMTool, handler func(ctx context.Context, rawArgs string) (string, error)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[name] = ToolDefinition{
		Tool:    tool,
		Handler: handler,
	}
}

func (r *ToolRegistry) RegisterTool(tool interfaces.ITool) {
	r.Register(tool.Name(), tool.Definition(), tool.Execute)
}

func (r *ToolRegistry) GetTools() []requests.LLMTool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]requests.LLMTool, 0, len(r.tools))
	for _, def := range r.tools {
		result = append(result, def.Tool)
	}
	return result
}

func (r *ToolRegistry) Execute(ctx context.Context, name string, rawArgs string) (string, error) {
	r.mu.RLock()
	def, exists := r.tools[name]
	r.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("tool %q not registered in ToolRegistry", name)
	}
	return def.Handler(ctx, rawArgs)
}

// RegisterDefaultTools registers standard backend tools using concrete ITool structs
func RegisterDefaultTools(registry interfaces.IToolRegistry, orchestrator interfaces.IOrchestratorService) {
	defaultTools := []interfaces.ITool{
		NewValidateAlertTool(orchestrator),
		NewGetIncidentTool(orchestrator),
		NewCreateRunbookTool(orchestrator),
		NewUpdateRunbookTool(orchestrator),
		NewDeprecateRunbookTool(orchestrator),
		NewGetRunbookTool(orchestrator),
		NewListRunbooksTool(orchestrator),
		NewLinkAlertToIncidentTool(orchestrator),
	}

	for _, tool := range defaultTools {
		registry.RegisterTool(tool)
	}
}
