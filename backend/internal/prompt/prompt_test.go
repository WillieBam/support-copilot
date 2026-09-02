package prompt_test

import (
	"strings"
	"testing"

	"github.com/WillieBam/support_copilot/backend/internal/prompt"
)

// test embedded system prompt loading
func TestSystemPrompt(t *testing.T) {
	if strings.TrimSpace(prompt.SystemPrompt) == "" {
		t.Fatal("expected system prompt to be loaded and non-empty")
	}

	if !strings.Contains(prompt.SystemPrompt, "ALERT CLASSIFICATION & DIAGNOSIS") {
		t.Fatal("expected system prompt to contain alert classification & diagnosis section")
	}
}
