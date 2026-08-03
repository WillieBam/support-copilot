package requests

// mcpToolsCallRequest is the json-rpc 2.0 envelope sent to fastmcp for tool invocation
type MCPToolsCallRequest struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      string         `json:"id"`
	Method  string         `json:"method"`
	Params  MCPToolsParams `json:"params"`
}

// mcpToolsParams carries the tool name and its arguments in the tools/call request
type MCPToolsParams struct {
	Name      string `json:"name"`
	Arguments any    `json:"arguments"`
}

// mcpToolsCallResponse is the json-rpc 2.0 response envelope returned by fastmcp
type MCPToolsCallResponse struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      string           `json:"id"`
	Result  MCPToolsResult   `json:"result"`
	Error   *MCPErrorPayload `json:"error"`
}

// mcpToolsResult holds the content blocks produced by the invoked tool
type MCPToolsResult struct {
	Content []MCPContentBlock `json:"content"`
	IsError bool              `json:"isError"`
}

// mcpContentBlock represents a single output block from a tool call result
type MCPContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// mcpErrorPayload carries the rpc error code and message when a tool call fails at the protocol level
type MCPErrorPayload struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ── mcp server 2 tool argument types ─────────────────────────────────────────

// mcp2CreateRunbookArgs is the argument struct for the create_runbook tool
type MCP2CreateRunbookArgs struct {
	TeamID     string `json:"team_id"`
	IncidentID string `json:"incident_id"`
	Title      string `json:"title"`
	Content    string `json:"content"`
}

// mcp2UpdateRunbookArgs is the argument struct for the update_runbook tool
type MCP2UpdateRunbookArgs struct {
	RunbookID string `json:"runbook_id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
}

// mcp2DeprecateRunbookArgs is the argument struct for the deprecate_runbook tool
type MCP2DeprecateRunbookArgs struct {
	RunbookID string `json:"runbook_id"`
}

// mcp2GetRunbookArgs is the argument struct for the get_runbook tool
type MCP2GetRunbookArgs struct {
	RunbookID string `json:"runbook_id"`
}

// mcp2ListRunbooksArgs is the argument struct for the list_runbooks tool
type MCP2ListRunbooksArgs struct {
	TeamID string `json:"team_id"`
	Status string `json:"status,omitempty"` // "active" or "deprecated"
}

// mcp2GetIncidentArgs is the argument struct for the get_incident tool
type MCP2GetIncidentArgs struct {
	IncidentID string `json:"incident_id"`
}

// mcp2ListIncidentsArgs is the argument struct for the list_incidents tool
type MCP2ListIncidentsArgs struct {
	TeamID string `json:"team_id"`
}

// mcp2LinkAlertIncidentArgs is the argument struct for the link_alert_to_incident tool
type MCP2LinkAlertIncidentArgs struct {
	AlertID       string `json:"alert_id"`
	IncidentID    string `json:"incident_id,omitempty"`
	IncidentTitle string `json:"incident_title,omitempty"`
}
