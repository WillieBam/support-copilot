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

// TestRegisterDefaultTools tests default tool handler execution
func TestRegisterDefaultTools(t *testing.T) {
	tr := tools.NewToolRegistry()
	mockOrchestrator := mocks.NewIOrchestratorService(t)

	tools.RegisterDefaultTools(tr, mockOrchestrator)

	registeredTools := tr.GetTools()
	if len(registeredTools) == 0 {
		t.Fatal("expected default tools to be registered")
	}

	ctx := context.Background()

	mockOrchestrator.On("ExecuteValidateAlertRaw", ctx, "{}").Return("validated", nil)
	res, err := tr.Execute(ctx, "validate_alert", "{}")
	if err != nil || res != "validated" {
		t.Fatalf("validate_alert failed: %v", err)
	}

	mockOrchestrator.On("ExecuteGetIncidentRaw", ctx, "{}").Return("incident", nil)
	res, err = tr.Execute(ctx, "get_incident", "{}")
	if err != nil || res != "incident" {
		t.Fatalf("get_incident failed: %v", err)
	}

	mockOrchestrator.On("ExecuteCreateRunbookRaw", ctx, "{}").Return("created", nil)
	res, err = tr.Execute(ctx, "create_runbook", "{}")
	if err != nil || res != "created" {
		t.Fatalf("create_runbook failed: %v", err)
	}

	mockOrchestrator.On("ExecuteUpdateRunbookRaw", ctx, "{}").Return("updated", nil)
	res, err = tr.Execute(ctx, "update_runbook", "{}")
	if err != nil || res != "updated" {
		t.Fatalf("update_runbook failed: %v", err)
	}

	mockOrchestrator.On("ExecuteDeprecateRunbookRaw", ctx, "{}").Return("deprecated", nil)
	res, err = tr.Execute(ctx, "deprecate_runbook", "{}")
	if err != nil || res != "deprecated" {
		t.Fatalf("deprecate_runbook failed: %v", err)
	}

	mockOrchestrator.On("ExecuteGetRunbookRaw", ctx, "{}").Return("runbook", nil)
	res, err = tr.Execute(ctx, "get_runbook", "{}")
	if err != nil || res != "runbook" {
		t.Fatalf("get_runbook failed: %v", err)
	}

	mockOrchestrator.On("ExecuteListRunbooksRaw", ctx, "{}").Return("runbooks", nil)
	res, err = tr.Execute(ctx, "list_runbooks", "{}")
	if err != nil || res != "runbooks" {
		t.Fatalf("list_runbooks failed: %v", err)
	}

	mockOrchestrator.On("ExecuteLinkAlertToIncidentRaw", ctx, "{}").Return("linked", nil)
	res, err = tr.Execute(ctx, "link_alert_to_incident", "{}")
	if err != nil || res != "linked" {
		t.Fatalf("link_alert_to_incident failed: %v", err)
	}
}
