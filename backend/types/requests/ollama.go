package requests

type OllamaFunctionCall struct {
	Name             string                 `json:"name"`
	Arguments        map[string]interface{} `json:"arguments"`
	ThoughtSignature string                 `json:"thought_signature,omitempty"`
}

type OllamaToolCall struct {
	ID       string             `json:"id,omitempty"`
	Function OllamaFunctionCall `json:"function"`
}

type OllamaMessage struct {
	Role      string           `json:"role"`
	Content   string           `json:"content"`
	ToolCalls []OllamaToolCall `json:"tool_calls,omitempty"`
}

type OllamaFunction struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

type OllamaTool struct {
	Type     string         `json:"type"`
	Function OllamaFunction `json:"function"`
}

type OllamaOptions struct {
	NumCtx int `json:"num_ctx,omitempty"`
}

type OllamaChatRequest struct {
	Model     string          `json:"model"`
	Messages  []OllamaMessage `json:"messages"`
	Tools     []OllamaTool    `json:"tools,omitempty"`
	Stream    bool            `json:"stream"`
	KeepAlive string          `json:"keep_alive,omitempty"`
	Options   *OllamaOptions  `json:"options,omitempty"`
}

type OllamaChatResponse struct {
	Model     string        `json:"model"`
	CreatedAt string        `json:"created_at"`
	Message   OllamaMessage `json:"message"`
	Done      bool          `json:"done"`
}
