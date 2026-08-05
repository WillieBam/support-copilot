package prompt

import _ "embed"

// system prompt loaded at compile time via go embed
//go:embed system_prompt.md
var SystemPrompt string
