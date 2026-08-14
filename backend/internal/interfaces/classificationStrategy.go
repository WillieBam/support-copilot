package interfaces

import (
	"github.com/WillieBam/support_copilot/backend/types"
)

// IClassificationStrategy defines an algorithm for classifying user prompt intent.
type IClassificationStrategy interface {
	Name() string
	Classify(prompt string, history []types.HistoryMessage) (types.Intent, float64, bool)
}
