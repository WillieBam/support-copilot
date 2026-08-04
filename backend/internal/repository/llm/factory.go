package llm

import (
	"strings"

	"github.com/WillieBam/support_copilot/backend/app/config"
	"github.com/WillieBam/support_copilot/backend/internal/interfaces"
)

func NewLLMClient(cfg *config.Config) interfaces.ILLMClient {
	provider := strings.ToLower(strings.TrimSpace(cfg.LLM.Provider))
	if provider == "" {
		provider = "ollama"
	}

	switch provider {
	case "gemini", "openai", "openai_compatible", "groq", "deepseek", "openrouter":
		return NewOpenAIClient(cfg)
	case "ollama":
		return NewOllamaClient(cfg)
	default:
		return NewOllamaClient(cfg)
	}
}
