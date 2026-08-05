package tools_test

import (
	"context"
	"testing"

	"github.com/WillieBam/support_copilot/backend/internal/mocks"
	"github.com/WillieBam/support_copilot/backend/internal/tools"
	"github.com/WillieBam/support_copilot/backend/types/requests"
)

// test tool registry registration and execution
func TestToolRegistry_RegisterAndExecute(t *testing.T) {
	tr := tools.NewToolRegistry()
	tool := requests.OllamaTool{
		Type: "function",
		Function: requests.OllamaFunction{
			Name: "test_tool",
		},
	}

	tr.Register("test_tool", tool, func(ctx context.Context, rawArgs string) (string, error) {
		return "ok", nil
	})

	toolsList := tr.GetTools()
	if len(toolsList) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(toolsList))
	}

	res, err := tr.Execute(context.Background(), "test_tool", "{}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res != "ok" {
		t.Fatalf("expected ok, got %s", res)
	}

	_, err = tr.Execute(context.Background(), "unknown_tool", "{}")
	if err == nil {
		t.Fatal("expected error for unregistered tool")
	}
}

// test register default tools using orchestrator mock
func TestRegisterDefaultTools(t *testing.T) {
	tr := tools.NewToolRegistry()
	mockOrchestrator := mocks.NewIOrchestratorService(t)

	tools.RegisterDefaultTools(tr, mockOrchestrator)

	registeredTools := tr.GetTools()
	if len(registeredTools) == 0 {
		t.Fatal("expected default tools to be registered")
	}
}
