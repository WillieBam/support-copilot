package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/WillieBam/support_copilot/backend/app/config"
	"github.com/WillieBam/support_copilot/backend/internal/interfaces"
	"github.com/WillieBam/support_copilot/backend/types/requests"
)

type mcp2Client struct {
	httpClient *http.Client
	mcpBaseUrl string
}

func NewMcpTwoClient(cfg *config.Config) interfaces.IMCP2Client {
	host := cfg.MCP2.Host
	if host == "" {
		host = "localhost"
	}
	port := cfg.MCP2.Port
	if port == "" {
		port = "9000"
	}

	return &mcp2Client{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		mcpBaseUrl: fmt.Sprintf("http://%s:%s", host, port),
	}
}

// callTool sends a json-rpc tools/call request and returns raw json text result
func (m *mcp2Client) callTool(ctx context.Context, toolName string, args any) (string, error) {
	envelope := requests.MCPToolsCallRequest{
		JSONRPC: "2.0",
		ID:      "1",
		Method:  "tools/call",
		Params:  requests.MCPToolsParams{Name: toolName, Arguments: args},
	}
	payload, err := json.Marshal(envelope)
	if err != nil {
		return "", fmt.Errorf("failed to marshal mcp2 tool request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.mcpBaseUrl+"/mcp", bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("failed creating request for mcp2: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed communicating with mcp_server_2: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("mcp2 server returned status %d: %s", resp.StatusCode, string(body))
	}
	var rpcResp requests.MCPToolsCallResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return "", fmt.Errorf("failed decoding mcp2 json-rpc envelope: %w", err)
	}
	if rpcResp.Error != nil {
		return "", fmt.Errorf("mcp2 rpc error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}
	if rpcResp.Result.IsError || len(rpcResp.Result.Content) == 0 {
		return "", fmt.Errorf("mcp2 tool %q returned an error result", toolName)
	}
	return rpcResp.Result.Content[0].Text, nil
}

func (m *mcp2Client) CreateRunbook(ctx context.Context, args requests.MCP2CreateRunbookArgs) (string, error) {
	return m.callTool(ctx, "create_runbook", args)
}

func (m *mcp2Client) UpdateRunbook(ctx context.Context, args requests.MCP2UpdateRunbookArgs) (string, error) {
	return m.callTool(ctx, "update_runbook", args)
}

func (m *mcp2Client) DeprecateRunbook(ctx context.Context, args requests.MCP2DeprecateRunbookArgs) (string, error) {
	return m.callTool(ctx, "deprecate_runbook", args)
}

func (m *mcp2Client) GetRunbook(ctx context.Context, args requests.MCP2GetRunbookArgs) (string, error) {
	return m.callTool(ctx, "get_runbook", args)
}

func (m *mcp2Client) ListRunbooks(ctx context.Context, args requests.MCP2ListRunbooksArgs) (string, error) {
	return m.callTool(ctx, "list_runbooks", args)
}

func (m *mcp2Client) GetIncident(ctx context.Context, args requests.MCP2GetIncidentArgs) (string, error) {
	return m.callTool(ctx, "get_incident", args)
}

func (m *mcp2Client) ListIncidents(ctx context.Context, args requests.MCP2ListIncidentsArgs) (string, error) {
	return m.callTool(ctx, "list_incidents", args)
}
