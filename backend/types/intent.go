package types

// Intent labels the nature of a user prompt.
type Intent string

const (
	// IntentConversational is used for social/acknowledgement message that
	// does not require tool execution e.g. "ok", "thanks", "bye"
	IntentConversational Intent = "conversational"

	// IntentTask indicates the user is requesting an operation that may
	// require tool execution e.g. providing an alert ID for validation
	IntentTask Intent = "task"
)
