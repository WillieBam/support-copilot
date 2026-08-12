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

var _ = Describe("McpTwoClient", func() {
	Context("NewMcpTwoClient", func() {
		It("should set default host and port when config is empty", func() {
			cfg := &config.Config{}
			client := mcp.NewMcpTwoClient(cfg)
			Expect(client).NotTo(BeNil())
		})

		It("should set host and port from config", func() {
			cfg := &config.Config{}
			cfg.MCP2.Host = "127.0.0.1"
			cfg.MCP2.Port = "9000"

			client := mcp.NewMcpTwoClient(cfg)
			Expect(client).NotTo(BeNil())
		})
	})

	Context("Tool Calls", func() {
		It("should call create_runbook successfully", func() {
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				Expect(r.URL.Path).To(Equal("/mcp"))
				Expect(r.Method).To(Equal(http.MethodPost))

				var rpcReq requests.MCPToolsCallRequest
				err := json.NewDecoder(r.Body).Decode(&rpcReq)
				Expect(err).NotTo(HaveOccurred())
				Expect(rpcReq.Method).To(Equal("tools/call"))
				Expect(rpcReq.Params.Name).To(Equal("create_runbook"))

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(buildMCPResponse(map[string]any{
					"id":     "rb-123",
					"status": "active",
				}))
			}))
			defer mockServer.Close()

			u, err := url.Parse(mockServer.URL)
			Expect(err).NotTo(HaveOccurred())

			cfg := &config.Config{}
			cfg.MCP2.Host = u.Hostname()
			cfg.MCP2.Port = u.Port()

			client := mcp.NewMcpTwoClient(cfg)

			res, err := client.CreateRunbook(context.Background(), requests.MCP2CreateRunbookArgs{
				TeamID:     "team-1",
				IncidentID: "inc-1",
				Title:      "Test Runbook",
				Content:    "Content",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(ContainSubstring("rb-123"))
		})

		It("should call update_runbook successfully", func() {
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var rpcReq requests.MCPToolsCallRequest
				_ = json.NewDecoder(r.Body).Decode(&rpcReq)
				Expect(rpcReq.Params.Name).To(Equal("update_runbook"))

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(buildMCPResponse(map[string]any{"updated": true}))
			}))
			defer mockServer.Close()

			u, _ := url.Parse(mockServer.URL)
			cfg := &config.Config{}
			cfg.MCP2.Host = u.Hostname()
			cfg.MCP2.Port = u.Port()

			client := mcp.NewMcpTwoClient(cfg)
			res, err := client.UpdateRunbook(context.Background(), requests.MCP2UpdateRunbookArgs{
				RunbookID: "rb-123",
				Title:     "New Title",
				Content:   "New Content",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(ContainSubstring("updated"))
		})

		It("should call deprecate_runbook successfully", func() {
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var rpcReq requests.MCPToolsCallRequest
				_ = json.NewDecoder(r.Body).Decode(&rpcReq)
				Expect(rpcReq.Params.Name).To(Equal("deprecate_runbook"))

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(buildMCPResponse(map[string]any{"status": "deprecated"}))
			}))
			defer mockServer.Close()

			u, _ := url.Parse(mockServer.URL)
			cfg := &config.Config{}
			cfg.MCP2.Host = u.Hostname()
			cfg.MCP2.Port = u.Port()

			client := mcp.NewMcpTwoClient(cfg)
			res, err := client.DeprecateRunbook(context.Background(), requests.MCP2DeprecateRunbookArgs{
				RunbookID: "rb-123",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(ContainSubstring("deprecated"))
		})

		It("should call get_runbook successfully", func() {
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var rpcReq requests.MCPToolsCallRequest
				_ = json.NewDecoder(r.Body).Decode(&rpcReq)
				Expect(rpcReq.Params.Name).To(Equal("get_runbook"))

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(buildMCPResponse(map[string]any{"id": "rb-123"}))
			}))
			defer mockServer.Close()

			u, _ := url.Parse(mockServer.URL)
			cfg := &config.Config{}
			cfg.MCP2.Host = u.Hostname()
			cfg.MCP2.Port = u.Port()

			client := mcp.NewMcpTwoClient(cfg)
			res, err := client.GetRunbook(context.Background(), requests.MCP2GetRunbookArgs{
				RunbookID: "rb-123",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(ContainSubstring("rb-123"))
		})

		It("should call list_runbooks successfully", func() {
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var rpcReq requests.MCPToolsCallRequest
				_ = json.NewDecoder(r.Body).Decode(&rpcReq)
				Expect(rpcReq.Params.Name).To(Equal("list_runbooks"))

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(buildMCPResponse([]map[string]any{{"id": "rb-1"}}))
			}))
			defer mockServer.Close()

			u, _ := url.Parse(mockServer.URL)
			cfg := &config.Config{}
			cfg.MCP2.Host = u.Hostname()
			cfg.MCP2.Port = u.Port()

			client := mcp.NewMcpTwoClient(cfg)
			res, err := client.ListRunbooks(context.Background(), requests.MCP2ListRunbooksArgs{
				TeamID: "team-1",
				Status: "active",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(ContainSubstring("rb-1"))
		})

		It("should call get_incident successfully", func() {
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var rpcReq requests.MCPToolsCallRequest
				_ = json.NewDecoder(r.Body).Decode(&rpcReq)
				Expect(rpcReq.Params.Name).To(Equal("get_incident"))

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(buildMCPResponse(map[string]any{"incident_id": "inc-1"}))
			}))
			defer mockServer.Close()

			u, _ := url.Parse(mockServer.URL)
			cfg := &config.Config{}
			cfg.MCP2.Host = u.Hostname()
			cfg.MCP2.Port = u.Port()

			client := mcp.NewMcpTwoClient(cfg)
			res, err := client.GetIncident(context.Background(), requests.MCP2GetIncidentArgs{
				IncidentID: "inc-1",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(ContainSubstring("inc-1"))
		})

		It("should call list_incidents successfully", func() {
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var rpcReq requests.MCPToolsCallRequest
				_ = json.NewDecoder(r.Body).Decode(&rpcReq)
				Expect(rpcReq.Params.Name).To(Equal("list_incidents"))

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(buildMCPResponse([]map[string]any{{"id": "inc-1"}}))
			}))
			defer mockServer.Close()

			u, _ := url.Parse(mockServer.URL)
			cfg := &config.Config{}
			cfg.MCP2.Host = u.Hostname()
			cfg.MCP2.Port = u.Port()

			client := mcp.NewMcpTwoClient(cfg)
			res, err := client.ListIncidents(context.Background(), requests.MCP2ListIncidentsArgs{
				TeamID: "team-1",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(ContainSubstring("inc-1"))
		})

		It("should return [] when list_incidents returns empty content", func() {
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]any{
					"jsonrpc": "2.0",
					"id":      "1",
					"result": map[string]any{
						"content": []any{},
						"isError": false,
					},
				})
			}))
			defer mockServer.Close()

			u, _ := url.Parse(mockServer.URL)
			cfg := &config.Config{}
			cfg.MCP2.Host = u.Hostname()
			cfg.MCP2.Port = u.Port()

			client := mcp.NewMcpTwoClient(cfg)
			res, err := client.ListIncidents(context.Background(), requests.MCP2ListIncidentsArgs{
				TeamID: "team-1",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal("[]"))
		})

		It("should handle rpc error from mcp2 server", func() {
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
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

			u, _ := url.Parse(mockServer.URL)
			cfg := &config.Config{}
			cfg.MCP2.Host = u.Hostname()
			cfg.MCP2.Port = u.Port()

			client := mcp.NewMcpTwoClient(cfg)
			res, err := client.ListIncidents(context.Background(), requests.MCP2ListIncidentsArgs{
				TeamID: "team-1",
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("mcp2 rpc error"))
			Expect(res).To(BeEmpty())
		})

		It("should handle non-200 HTTP status response", func() {
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("internal server error"))
			}))
			defer mockServer.Close()

			u, _ := url.Parse(mockServer.URL)
			cfg := &config.Config{}
			cfg.MCP2.Host = u.Hostname()
			cfg.MCP2.Port = u.Port()

			client := mcp.NewMcpTwoClient(cfg)
			res, err := client.GetRunbook(context.Background(), requests.MCP2GetRunbookArgs{RunbookID: "rb-1"})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("mcp2 server returned status 500"))
			Expect(res).To(BeEmpty())
		})

		It("should handle malformed JSON response body decoding error", func() {
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("invalid-json"))
			}))
			defer mockServer.Close()

			u, _ := url.Parse(mockServer.URL)
			cfg := &config.Config{}
			cfg.MCP2.Host = u.Hostname()
			cfg.MCP2.Port = u.Port()

			client := mcp.NewMcpTwoClient(cfg)
			res, err := client.GetRunbook(context.Background(), requests.MCP2GetRunbookArgs{RunbookID: "rb-1"})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed decoding mcp2 json-rpc envelope"))
			Expect(res).To(BeEmpty())
		})

		It("should handle tool error result with message content", func() {
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]any{
					"jsonrpc": "2.0",
					"id":      "1",
					"result": map[string]any{
						"isError": true,
						"content": []map[string]any{
							{"type": "text", "text": "runbook not found"},
						},
					},
				})
			}))
			defer mockServer.Close()

			u, _ := url.Parse(mockServer.URL)
			cfg := &config.Config{}
			cfg.MCP2.Host = u.Hostname()
			cfg.MCP2.Port = u.Port()

			client := mcp.NewMcpTwoClient(cfg)
			res, err := client.GetRunbook(context.Background(), requests.MCP2GetRunbookArgs{RunbookID: "rb-1"})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("runbook not found"))
			Expect(res).To(BeEmpty())
		})

		It("should handle tool error result with empty content message", func() {
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]any{
					"jsonrpc": "2.0",
					"id":      "1",
					"result": map[string]any{
						"isError": true,
						"content": []map[string]any{},
					},
				})
			}))
			defer mockServer.Close()

			u, _ := url.Parse(mockServer.URL)
			cfg := &config.Config{}
			cfg.MCP2.Host = u.Hostname()
			cfg.MCP2.Port = u.Port()

			client := mcp.NewMcpTwoClient(cfg)
			res, err := client.GetRunbook(context.Background(), requests.MCP2GetRunbookArgs{RunbookID: "rb-1"})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unknown error"))
			Expect(res).To(BeEmpty())
		})
		It("should handle HTTP client connection error when context is canceled", func() {
			cfg := &config.Config{}
			cfg.MCP2.Host = "127.0.0.1"
			cfg.MCP2.Port = "9999"

			client := mcp.NewMcpTwoClient(cfg)
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			res, err := client.GetRunbook(ctx, requests.MCP2GetRunbookArgs{RunbookID: "rb-1"})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed communicating with mcp_server_2"))
			Expect(res).To(BeEmpty())
		})
	})
})
