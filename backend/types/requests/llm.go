package requests

type LLMFunctionCall struct {
	Name             string                 `json:"name"`
	Arguments        map[string]interface{} `json:"arguments"`
	ThoughtSignature string                 `json:"thought_signature,omitempty"`
}

type LLMToolCall struct {
	ID       string          `json:"id,omitempty"`
	Function LLMFunctionCall `json:"function"`
}

type LLMMessage struct {
	Role      string        `json:"role"`
	Content   string        `json:"content"`
	ToolCalls []LLMToolCall `json:"tool_calls,omitempty"`
}

type LLMFunction struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

type LLMTool struct {
	Type     string      `json:"type"`
	Function LLMFunction `json:"function"`
}

type LLMOptions struct {
	NumCtx int `json:"num_ctx,omitempty"`
}

type LLMChatRequest struct {
	Model     string       `json:"model"`
	Messages  []LLMMessage `json:"messages"`
	Tools     []LLMTool    `json:"tools,omitempty"`
	Stream    bool         `json:"stream"`
	KeepAlive string       `json:"keep_alive,omitempty"`
	Options   *LLMOptions  `json:"options,omitempty"`
}

type LLMChatResponse struct {
	Model     string     `json:"model"`
	CreatedAt string     `json:"created_at"`
	Message   LLMMessage `json:"message"`
	Done      bool       `json:"done"`
}

// type aliases for backward compatibility during migration
type OllamaFunctionCall = LLMFunctionCall
type OllamaToolCall = LLMToolCall
type OllamaMessage = LLMMessage
type OllamaFunction = LLMFunction
type OllamaTool = LLMTool
type OllamaOptions = LLMOptions
type OllamaChatRequest = LLMChatRequest
type OllamaChatResponse = LLMChatResponse
