package mcp_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/WillieBam/support_copilot/backend/app/config"
	"github.com/WillieBam/support_copilot/backend/internal/repository/mcp"
	"github.com/WillieBam/support_copilot/backend/types/requests"
)

// buildMCPResponse wraps a tool result inside a JSON-RPC 2.0 response envelope,
// with the tool payload serialised as a text content block (as FastMCP does).
func buildMCPResponse(result any) map[string]any {
	text, _ := json.Marshal(result)
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      "1",
		"result": map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": string(text)},
			},
			"isError": false,
		},
	}
}

var _ = Describe("McpOneClient", func() {
	Context("NewMcpOneClient", func() {
		It("should set default host and port when config is empty", func() {
			cfg := &config.Config{}
			client := mcp.NewMcpOneClient(cfg)
			Expect(client).NotTo(BeNil())
		})

		It("should set host and port from config", func() {
			cfg := &config.Config{}
			cfg.MCP1.Host = "127.0.0.1"
			cfg.MCP1.Port = "9000"

			client := mcp.NewMcpOneClient(cfg)
			Expect(client).NotTo(BeNil())
		})
	})

	Context("DetectAnomalies", func() {
		It("should detect anomalies successfully from mock MCP server", func() {
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify path and method match FastMCP streamable-http transport.
				Expect(r.URL.Path).To(Equal("/mcp"))
				Expect(r.Method).To(Equal(http.MethodPost))

				// verify the json-rpc envelope is correct
				var rpcReq requests.MCPToolsCallRequest
				err := json.NewDecoder(r.Body).Decode(&rpcReq)
				Expect(err).NotTo(HaveOccurred())
				Expect(rpcReq.JSONRPC).To(Equal("2.0"))
				Expect(rpcReq.Method).To(Equal("tools/call"))
				Expect(rpcReq.Params.Name).To(Equal("detect_anomalies"))

				res := requests.AnomalyDetectionResponse{
					Status: 1,
					Label:  "Normal",
					Engine: "IsolationForest",
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(buildMCPResponse(res))
			}))
			defer mockServer.Close()

			u, err := url.Parse(mockServer.URL)
			Expect(err).NotTo(HaveOccurred())

			cfg := &config.Config{}
			cfg.MCP1.Host = u.Hostname()
			cfg.MCP1.Port = u.Port()

			client := mcp.NewMcpOneClient(cfg)

			anomalyReq := requests.AnomalyDetectionRequest{
				CpuUsage:            95.5,
				MemoryUsage:         88.0,
				ResponseLatency:     150.0,
				AvailabilityPercent: 99.9,
			}

			res, err := client.DetectAnomalies(context.Background(), anomalyReq)
			Expect(err).NotTo(HaveOccurred())
			Expect(res).NotTo(BeNil())
			Expect(res.Label).To(Equal("Normal"))
			Expect(res.Engine).To(Equal("IsolationForest"))
		})

		It("should handle non-200 response from MCP server", func() {
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("invalid metric input"))
			}))
			defer mockServer.Close()

			u, err := url.Parse(mockServer.URL)
			Expect(err).NotTo(HaveOccurred())

			cfg := &config.Config{}
			cfg.MCP1.Host = u.Hostname()
			cfg.MCP1.Port = u.Port()

			client := mcp.NewMcpOneClient(cfg)

			anomalyReq := requests.AnomalyDetectionRequest{
				CpuUsage: 99.0,
			}

			res, err := client.DetectAnomalies(context.Background(), anomalyReq)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("mcp server returned bad status code 400"))
			Expect(res).To(BeNil())
		})

		It("should handle JSON-RPC error response from MCP server", func() {
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]any{
					"jsonrpc": "2.0",
					"id":      "1",
					"error": map[string]any{
						"code":    -32601,
						"message": "Method not found",
					},
				})
			}))
			defer mockServer.Close()

			u, err := url.Parse(mockServer.URL)
			Expect(err).NotTo(HaveOccurred())

			cfg := &config.Config{}
			cfg.MCP1.Host = u.Hostname()
			cfg.MCP1.Port = u.Port()

			client := mcp.NewMcpOneClient(cfg)

			anomalyReq := requests.AnomalyDetectionRequest{
				CpuUsage: 50.0,
			}

			res, err := client.DetectAnomalies(context.Background(), anomalyReq)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("mcp server returned rpc error"))
			Expect(res).To(BeNil())
		})

		It("should handle decoding failure when MCP server returns non-JSON body", func() {
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("non-json body"))
			}))
			defer mockServer.Close()

			u, err := url.Parse(mockServer.URL)
			Expect(err).NotTo(HaveOccurred())

			cfg := &config.Config{}
			cfg.MCP1.Host = u.Hostname()
			cfg.MCP1.Port = u.Port()

			client := mcp.NewMcpOneClient(cfg)

			anomalyReq := requests.AnomalyDetectionRequest{
				CpuUsage: 50.0,
			}

			res, err := client.DetectAnomalies(context.Background(), anomalyReq)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed decoding MCP JSON-RPC envelope"))
			Expect(res).To(BeNil())
		})
	})
})
