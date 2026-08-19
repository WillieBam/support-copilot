package classifier

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/WillieBam/support_copilot/backend/internal/interfaces"
	"github.com/WillieBam/support_copilot/backend/types"
	"github.com/WillieBam/support_copilot/backend/types/requests"
)

// Re-export types.Intent and its constants for backward compatibility
type Intent = types.Intent

const (
	IntentConversational = types.IntentConversational
	IntentTask           = types.IntentTask
)

// embeddedToolCallPattern detects when the LLM emits a raw JSON tool-call
// object as plain text content instead of via the proper tool_calls mechanism
var embeddedToolCallPattern = regexp.MustCompile(
	`(?s)^\s*\{\s*"(name|function)"\s*:\s*"[^"]+"\s*,\s*"(parameters|arguments)"\s*:\s*\{`,
)

// LooksLikeEmbeddedToolCall returns true when content appears to be a raw
// JSON tool-call emitted by the LLM as text rather than through the
// tool_calls field. Such content should be parsed into a proper tool call
func LooksLikeEmbeddedToolCall(content string) bool {
	return embeddedToolCallPattern.MatchString(strings.TrimSpace(content))
}

type rawEmbeddedCall struct {
	Name         string                 `json:"name"`
	FunctionName string                 `json:"function"`
	Parameters   map[string]interface{} `json:"parameters"`
	Arguments    map[string]interface{} `json:"arguments"`
}

// ParseEmbeddedToolCall extracts and parses an embedded JSON tool call from text content
func ParseEmbeddedToolCall(content string) (*requests.LLMToolCall, error) {
	s := strings.TrimSpace(content)
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start == -1 || end == -1 || start >= end {
		return nil, fmt.Errorf("no json object found in text content")
	}
	jsonStr := s[start : end+1]

	var raw rawEmbeddedCall
	if err := json.Unmarshal([]byte(jsonStr), &raw); err != nil {
		return nil, fmt.Errorf("failed to unmarshal embedded tool call json: %w", err)
	}

	name := raw.Name
	if name == "" {
		name = raw.FunctionName
	}
	if name == "" {
		return nil, fmt.Errorf("embedded tool call missing tool name")
	}

	args := raw.Parameters
	if len(args) == 0 {
		args = raw.Arguments
	}
	if args == nil {
		args = make(map[string]interface{})
	}

	return &requests.LLMToolCall{
		Function: requests.LLMFunctionCall{
			Name:      name,
			Arguments: args,
		},
	}, nil
}

// IntentClassifier classifies a prompt as conversational or task-oriented
// using configured classification strategies. It implements interfaces.IIntentClassifier
type IntentClassifier struct {
	strategies []interfaces.IClassificationStrategy
}

// NewIntentClassifier returns a new IntentClassifier with default strategies
func NewIntentClassifier(strategies ...interfaces.IClassificationStrategy) *IntentClassifier {
	if len(strategies) == 0 {
		strategies = []interfaces.IClassificationStrategy{
			NewRegexRuleStrategy(),
		}
	}
	return &IntentClassifier{
		strategies: strategies,
	}
}

// Classify checks the prompt and returns the detected Intent
func (c *IntentClassifier) Classify(prompt string) Intent {
	return c.ClassifyWithHistory(prompt, nil)
}

// ClassifyWithHistory classifies a prompt taking the conversation history into account
func (c *IntentClassifier) ClassifyWithHistory(prompt string, history []types.HistoryMessage) Intent {
	for _, strategy := range c.strategies {
		intent, confidence, matched := strategy.Classify(prompt, history)
		if matched && confidence >= 0.8 {
			return intent
		}
	}
	return types.IntentTask
}
