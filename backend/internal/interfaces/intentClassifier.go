package interfaces

import (
	"github.com/WillieBam/support_copilot/backend/types"
)

// IIntentClassifier classifies a user prompt as conversational or task-oriented.
// The concrete implementation lives in internal/classifier; this interface allows
// AppService to accept mock classifiers in tests.
type IIntentClassifier interface {
	Classify(prompt string) types.Intent
	ClassifyWithHistory(prompt string, history []types.HistoryMessage) types.Intent
}
