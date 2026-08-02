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

type mcp1Client struct {
	httpClient *http.Client
	mcpBaseUrl string
}

func NewMcpOneClient(cfg *config.Config) interfaces.IMCPClient {
	host := cfg.MCP1.Host
	if host == "" {
		host = "localhost"
	}

	port := cfg.MCP1.Port
	if port == "" {
		port = "9000"
	}

	baseUrl := fmt.Sprintf("http://%s:%s", host, port)

	return &mcp1Client{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		mcpBaseUrl: baseUrl,
	}
}

func (m *mcp1Client) DetectAnomalies(ctx context.Context, anomalyReq requests.AnomalyDetectionRequest) (*requests.AnomalyDetectionResponse, error) {
	// FastMCP (streamable-http) exposes a single JSON-RPC 2.0 endpoint at /mcp.
	// Tools are invoked via method="tools/call" — there are no per-tool REST paths.
	mcpURL := fmt.Sprintf("%s/mcp", m.mcpBaseUrl)

	envelope := requests.MCPToolsCallRequest{
		JSONRPC: "2.0",
		ID:      "1",
		Method:  "tools/call",
		Params: requests.MCPToolsParams{
			Name:      "detect_anomalies",
			Arguments: anomalyReq,
		},
	}

	payloadBytes, err := json.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal MCP payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, mcpURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed creating request context for MCP: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed communicating with mcp_server_1: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("mcp server returned bad status code %d: %s", resp.StatusCode, string(body))
	}

	var rpcResp requests.MCPToolsCallResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return nil, fmt.Errorf("failed decoding MCP JSON-RPC envelope: %w", err)
	}

	if rpcResp.Error != nil {
		return nil, fmt.Errorf("mcp server returned rpc error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	if rpcResp.Result.IsError || len(rpcResp.Result.Content) == 0 {
		return nil, fmt.Errorf("mcp tool returned an error result")
	}

	// the tool result is returned as a text/json content block
	textContent := rpcResp.Result.Content[0].Text
	var anomalyResp requests.AnomalyDetectionResponse
	if err := json.Unmarshal([]byte(textContent), &anomalyResp); err != nil {
		return nil, fmt.Errorf("failed decoding anomaly response payload: %w", err)
	}

	return &anomalyResp, nil
}
