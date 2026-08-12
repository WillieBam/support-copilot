package llm_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/WillieBam/support_copilot/backend/app/config"
	llm "github.com/WillieBam/support_copilot/backend/internal/repository/llm"
	"github.com/WillieBam/support_copilot/backend/types"
	"github.com/WillieBam/support_copilot/backend/types/requests"
)

var _ = Describe("OllamaClient", func() {
	Context("NewOllamaClient", func() {
		It("should initialize with defaults when config fields are empty", func() {
			cfg := &config.Config{}
			client := llm.NewOllamaClient(cfg)
			Expect(client).NotTo(BeNil())
		})

		It("should initialize with provided config values", func() {
			cfg := &config.Config{}
			cfg.LLM.BaseURL = "http://localhost:11434/"
			cfg.LLM.Model = "llama3.2:latest"

			client := llm.NewOllamaClient(cfg)
			Expect(client).NotTo(BeNil())
		})
	})

	Context("QueryStreamWithTools", func() {
		It("should stream responses successfully from LLM mock server", func() {
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				Expect(r.URL.Path).To(Equal("/api/chat"))
				Expect(r.Method).To(Equal(http.MethodPost))
				w.Header().Set("Content-Type", "application/x-ndjson")
				w.WriteHeader(http.StatusOK)

				flusher, ok := w.(http.Flusher)
				Expect(ok).To(BeTrue())

				fmt.Fprintln(w, `{"message":{"content":"Hello "},"done":false}`)
				flusher.Flush()

				fmt.Fprintln(w, `{"message":{"content":"World!"},"done":true}`)
				flusher.Flush()
			}))
			defer mockServer.Close()

			cfg := &config.Config{}
			cfg.LLM.BaseURL = mockServer.URL
			cfg.LLM.Model = "test-model"

			client := llm.NewOllamaClient(cfg)

			ch := make(chan types.StreamEvent, 10)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			req := requests.OllamaChatRequest{
				Messages: []requests.OllamaMessage{
					{Role: "user", Content: "Hello"},
				},
			}

			msg, err := client.QueryStreamWithTools(ctx, req, ch)
			close(ch)

			Expect(err).NotTo(HaveOccurred())
			Expect(msg.Content).To(Equal("Hello World!"))

			var events []types.StreamEvent
			for ev := range ch {
				events = append(events, ev)
			}

			Expect(len(events)).To(Equal(2))
			Expect(events[0].Content).To(Equal("Hello "))
			Expect(events[1].Content).To(Equal("World!"))
		})

		It("should return error when server returns non-200 status code", func() {
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("internal model error"))
			}))
			defer mockServer.Close()

			cfg := &config.Config{}
			cfg.LLM.BaseURL = mockServer.URL
			cfg.LLM.Model = "test-model"

			client := llm.NewOllamaClient(cfg)

			ch := make(chan types.StreamEvent, 10)
			req := requests.OllamaChatRequest{
				Messages: []requests.OllamaMessage{{Role: "user", Content: "Hello"}},
			}
			_, err := client.QueryStreamWithTools(context.Background(), req, ch)
			close(ch)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("status code 500"))
		})

		It("should return error when LLM returns an in-stream API error", func() {
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				fmt.Fprintln(w, `{"error":"model not found"}`)
			}))
			defer mockServer.Close()

			cfg := &config.Config{}
			cfg.LLM.BaseURL = mockServer.URL
			cfg.LLM.Model = "nonexistent-model"

			client := llm.NewOllamaClient(cfg)

			ch := make(chan types.StreamEvent, 10)
			req := requests.OllamaChatRequest{
				Messages: []requests.OllamaMessage{{Role: "user", Content: "Hello"}},
			}
			_, err := client.QueryStreamWithTools(context.Background(), req, ch)
			close(ch)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("ollama API error: model not found"))
		})

		It("should return error when context is canceled", func() {
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(100 * time.Millisecond)
			}))
			defer mockServer.Close()

			cfg := &config.Config{}
			cfg.LLM.BaseURL = mockServer.URL
			cfg.LLM.Model = "test-model"

			client := llm.NewOllamaClient(cfg)

			ch := make(chan types.StreamEvent, 10)
			ctx, cancel := context.WithCancel(context.Background())
			cancel() // cancel immediately

			req := requests.OllamaChatRequest{
				Messages: []requests.OllamaMessage{{Role: "user", Content: "Hello"}},
			}
			_, err := client.QueryStreamWithTools(ctx, req, ch)
			close(ch)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("STREAM ERROR"))
		})
	})

	Context("NewLLMClient Factory", func() {
		It("should return LLM client when provider is empty or ollama", func() {
			cfg := &config.Config{}
			client := llm.NewLLMClient(cfg)
			Expect(client).NotTo(BeNil())
		})

		It("should return OpenAI client when provider is gemini or openai", func() {
			cfg := &config.Config{}
			cfg.LLM.Provider = "gemini"
			cfg.LLM.APIKey = "test-gemini-key"
			client := llm.NewLLMClient(cfg)
			Expect(client).NotTo(BeNil())
		})
	})

	Context("OpenAIClient (Gemini & OpenAI API)", func() {
		It("should stream text response from Gemini/OpenAI SSE server", func() {
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				Expect(r.Header.Get("Authorization")).To(Equal("Bearer test-gemini-key"))
				Expect(r.Header.Get("Content-Type")).To(Equal("application/json"))
				w.Header().Set("Content-Type", "text/event-stream")
				w.WriteHeader(http.StatusOK)

				flusher, ok := w.(http.Flusher)
				Expect(ok).To(BeTrue())

				fmt.Fprintln(w, `data: {"choices":[{"delta":{"content":"Hello from "}}]}`)
				flusher.Flush()

				fmt.Fprintln(w, `data: {"choices":[{"delta":{"content":"Gemini!"}}]}`)
				flusher.Flush()

				fmt.Fprintln(w, `data: [DONE]`)
				flusher.Flush()
			}))
			defer mockServer.Close()

			cfg := &config.Config{}
			cfg.LLM.Provider = "gemini"
			cfg.LLM.BaseURL = mockServer.URL
			cfg.LLM.APIKey = "test-gemini-key"
			cfg.LLM.Model = "gemini-2.0-flash"

			client := llm.NewLLMClient(cfg)

			ch := make(chan types.StreamEvent, 10)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			req := requests.OllamaChatRequest{
				Messages: []requests.OllamaMessage{
					{Role: "user", Content: "Hello"},
				},
			}

			msg, err := client.QueryStreamWithTools(ctx, req, ch)
			close(ch)

			Expect(err).NotTo(HaveOccurred())
			Expect(msg.Content).To(Equal("Hello from Gemini!"))

			var events []types.StreamEvent
			for ev := range ch {
				events = append(events, ev)
			}

			Expect(len(events)).To(Equal(2))
			Expect(events[0].Content).To(Equal("Hello from "))
			Expect(events[1].Content).To(Equal("Gemini!"))
		})

		It("should fall back to reasoning content when Gemini/OpenAI omits content", func() {
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				w.WriteHeader(http.StatusOK)

				flusher, ok := w.(http.Flusher)
				Expect(ok).To(BeTrue())

				fmt.Fprintln(w, `data: {"choices":[{"delta":{"reasoning_content":"Final answer from Gemini."}}]}`)
				flusher.Flush()

				fmt.Fprintln(w, `data: [DONE]`)
				flusher.Flush()
			}))
			defer mockServer.Close()

			cfg := &config.Config{}
			cfg.LLM.Provider = "gemini"
			cfg.LLM.BaseURL = mockServer.URL
			cfg.LLM.APIKey = "test-gemini-key"
			cfg.LLM.Model = "gemini-2.0-flash"

			client := llm.NewLLMClient(cfg)

			ch := make(chan types.StreamEvent, 10)
			req := requests.OllamaChatRequest{
				Messages: []requests.OllamaMessage{{Role: "user", Content: "Hello"}},
			}

			msg, err := client.QueryStreamWithTools(context.Background(), req, ch)
			close(ch)

			Expect(err).NotTo(HaveOccurred())
			Expect(msg.Content).To(Equal("Final answer from Gemini."))

			var events []types.StreamEvent
			for ev := range ch {
				events = append(events, ev)
			}

			Expect(len(events)).To(Equal(1))
			Expect(events[0].Type).To(Equal("text"))
			Expect(events[0].Content).To(Equal("Final answer from Gemini."))
		})

		It("should parse tool calls from Gemini/OpenAI stream", func() {
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				w.WriteHeader(http.StatusOK)

				flusher, ok := w.(http.Flusher)
				Expect(ok).To(BeTrue())

				fmt.Fprintln(w, `data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"name":"create_runbook","arguments":"{\"title\":\"Test Runbook\"}"}}]}}]}`)
				flusher.Flush()

				fmt.Fprintln(w, `data: [DONE]`)
				flusher.Flush()
			}))
			defer mockServer.Close()

			cfg := &config.Config{}
			cfg.LLM.Provider = "gemini"
			cfg.LLM.BaseURL = mockServer.URL
			cfg.LLM.APIKey = "test-key"

			client := llm.NewOpenAIClient(cfg)

			ch := make(chan types.StreamEvent, 10)
			req := requests.OllamaChatRequest{
				Messages: []requests.OllamaMessage{{Role: "user", Content: "create a runbook"}},
			}

			msg, err := client.QueryStreamWithTools(context.Background(), req, ch)
			close(ch)

			Expect(err).NotTo(HaveOccurred())
			Expect(len(msg.ToolCalls)).To(Equal(1))
			Expect(msg.ToolCalls[0].Function.Name).To(Equal("create_runbook"))
			Expect(msg.ToolCalls[0].Function.Arguments["title"]).To(Equal("Test Runbook"))
		})

		It("should handle default factory provider, unknown provider, and TLSSkipVerify option", func() {
			cfgDefault := &config.Config{}
			clientDefault := llm.NewLLMClient(cfgDefault)
			Expect(clientDefault).NotTo(BeNil())

			cfgUnknown := &config.Config{}
			cfgUnknown.LLM.Provider = "unknown_provider"
			clientUnknown := llm.NewLLMClient(cfgUnknown)
			Expect(clientUnknown).NotTo(BeNil())

			cfgOpenAI := &config.Config{}
			cfgOpenAI.LLM.Provider = "openai"
			cfgOpenAI.LLM.Model = ""
			cfgOpenAI.LLM.Timeout = 0
			cfgOpenAI.LLM.TLSSkipVerify = true
			clientOpenAI := llm.NewOpenAIClient(cfgOpenAI)
			Expect(clientOpenAI).NotTo(BeNil())
		})

		It("should handle assistant and tool messages conversion in OpenAI client", func() {
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				w.WriteHeader(http.StatusOK)

				flusher, ok := w.(http.Flusher)
				Expect(ok).To(BeTrue())

				fmt.Fprintln(w, `data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_100","function":{"name":"get_incident","arguments":{"incident_id":"INC-101"}}}]}}]}`)
				flusher.Flush()
				fmt.Fprintln(w, `data: [DONE]`)
				flusher.Flush()
			}))
			defer mockServer.Close()

			cfg := &config.Config{}
			cfg.LLM.Provider = "openai"
			cfg.LLM.BaseURL = mockServer.URL

			client := llm.NewOpenAIClient(cfg)

			req := requests.LLMChatRequest{
				Messages: []requests.LLMMessage{
					{
						Role:    "assistant",
						Content: " ",
						ToolCalls: []requests.LLMToolCall{
							{
								ID: "call_100",
								Function: requests.LLMFunctionCall{
									Name:      "get_incident",
									Arguments: map[string]interface{}{"incident_id": "INC-101"},
								},
							},
						},
					},
					{
						Role:    "tool",
						Content: "incident details result",
					},
					{
						Role:    "tool",
						Content: "fallback tool call result",
					},
				},
			}

			msg, err := client.QueryStreamWithTools(context.Background(), req, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(msg).NotTo(BeNil())
		})

		It("should handle thought_signature and extra_content in OpenAI/Gemini stream", func() {
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				w.WriteHeader(http.StatusOK)

				flusher, ok := w.(http.Flusher)
				Expect(ok).To(BeTrue())

				// delta thought_signature
				fmt.Fprintln(w, `data: {"choices":[{"delta":{"thought_signature":"sig_delta_1"}}]}`)
				flusher.Flush()

				// tool call with function object arguments and extra content
				fmt.Fprintln(w, `data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"name":"validate_alert","arguments":{"alert_id":"123"},"extra_content":{"google":{"thought_signature":"sig_func_1"}}}}]}}]}`)
				flusher.Flush()

				fmt.Fprintln(w, `data: [DONE]`)
				flusher.Flush()
			}))
			defer mockServer.Close()

			cfg := &config.Config{}
			cfg.LLM.Provider = "gemini"
			cfg.LLM.BaseURL = mockServer.URL

			client := llm.NewOpenAIClient(cfg)
			msg, err := client.QueryStreamWithTools(context.Background(), requests.LLMChatRequest{}, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(msg.ToolCalls)).To(Equal(1))
			Expect(msg.ToolCalls[0].Function.ThoughtSignature).To(Equal("sig_func_1"))
		})

		It("should handle non-200 response and stream API errors in OpenAI client", func() {
			// non 200
			mockErrServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte("invalid api key"))
			}))
			defer mockErrServer.Close()

			cfg := &config.Config{}
			cfg.LLM.Provider = "openai"
			cfg.LLM.BaseURL = mockErrServer.URL

			client := llm.NewOpenAIClient(cfg)
			_, err := client.QueryStreamWithTools(context.Background(), requests.LLMChatRequest{}, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("status code 401"))

			// stream API error chunk
			mockStreamErrServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				w.WriteHeader(http.StatusOK)
				fmt.Fprintln(w, `data: {"error":{"message":"quota exceeded"}}`)
			}))
			defer mockStreamErrServer.Close()

			cfg.LLM.BaseURL = mockStreamErrServer.URL
			clientStream := llm.NewOpenAIClient(cfg)
			_, err = clientStream.QueryStreamWithTools(context.Background(), requests.LLMChatRequest{}, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("LLM API error: quota exceeded"))
		})

		It("should handle network failure and canceled context in OpenAI client", func() {
			mockClosedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
			mockClosedServer.Close()

			cfg := &config.Config{}
			cfg.LLM.Provider = "openai"
			cfg.LLM.BaseURL = mockClosedServer.URL

			client := llm.NewOpenAIClient(cfg)
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			_, err := client.QueryStreamWithTools(ctx, requests.LLMChatRequest{}, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("STREAM ERROR"))

			// general network error without cancel
			_, err = client.QueryStreamWithTools(context.Background(), requests.LLMChatRequest{}, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed communicating with LLM API"))
		})

		It("should handle tool calls and chunk decoding error in Ollama client", func() {
			// tool calls in chunk
			mockToolServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/x-ndjson")
				w.WriteHeader(http.StatusOK)
				fmt.Fprintln(w, `{"message":{"tool_calls":[{"function":{"name":"validate_alert","arguments":{"alert_id":"101"}}}]},"done":true}`)
			}))
			defer mockToolServer.Close()

			cfg := &config.Config{}
			cfg.LLM.BaseURL = mockToolServer.URL
			client := llm.NewOllamaClient(cfg)
			msg, err := client.QueryStreamWithTools(context.Background(), requests.LLMChatRequest{}, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(msg.ToolCalls)).To(Equal(1))

			// invalid JSON chunk error
			mockBadChunkServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				fmt.Fprintln(w, `{invalid-json-chunk}`)
			}))
			defer mockBadChunkServer.Close()

			cfg.LLM.BaseURL = mockBadChunkServer.URL
			clientBad := llm.NewOllamaClient(cfg)
			_, err = clientBad.QueryStreamWithTools(context.Background(), requests.LLMChatRequest{}, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error decoding ollama chunk"))

			// general network connection error
			mockClosedOllama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
			mockClosedOllama.Close()

			cfgNetErr := &config.Config{}
			cfgNetErr.LLM.BaseURL = mockClosedOllama.URL
			clientNetErr := llm.NewOllamaClient(cfgNetErr)
			_, err = clientNetErr.QueryStreamWithTools(context.Background(), requests.LLMChatRequest{}, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed communicating with LLM"))
		})

		It("should handle comments, non-data lines, empty choices, and invalid json in SSE stream", func() {
			mockSSEServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				w.WriteHeader(http.StatusOK)

				flusher, ok := w.(http.Flusher)
				Expect(ok).To(BeTrue())

				fmt.Fprintln(w, `: sse ping comment`)
				fmt.Fprintln(w, `event: ping`)
				fmt.Fprintln(w, `data: {invalid-json}`)
				fmt.Fprintln(w, `data: {"choices":[]}`)
				fmt.Fprintln(w, `data: {"choices":[{"delta":{"content":"Final SSE message"},"finish_reason":"stop"}]}`)
				flusher.Flush()
			}))
			defer mockSSEServer.Close()

			cfg := &config.Config{}
			cfg.LLM.Provider = "openai"
			cfg.LLM.BaseURL = mockSSEServer.URL

			client := llm.NewOpenAIClient(cfg)
			msg, err := client.QueryStreamWithTools(context.Background(), requests.LLMChatRequest{}, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(msg.Content).To(Equal("Final SSE message"))
		})

		It("should inject default KeepAlive and NumCtx when omitted in request", func() {
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/x-ndjson")
				w.WriteHeader(http.StatusOK)
				fmt.Fprintln(w, `{"message":{"content":"ok"},"done":true}`)
			}))
			defer mockServer.Close()

			cfg := &config.Config{}
			cfg.LLM.BaseURL = mockServer.URL
			cfg.LLM.KeepAlive = "5m"
			cfg.LLM.NumCtx = 4096

			client := llm.NewOllamaClient(cfg)
			ch := make(chan types.StreamEvent, 10)
			msg, err := client.QueryStreamWithTools(context.Background(), requests.LLMChatRequest{}, ch)
			close(ch)
			Expect(err).NotTo(HaveOccurred())
			Expect(msg.Content).To(Equal("ok"))
		})

		It("should convert non-string tool call arguments in OpenAI assistant request and stream chunk", func() {
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				w.WriteHeader(http.StatusOK)

				flusher, ok := w.(http.Flusher)
				Expect(ok).To(BeTrue())

				fmt.Fprintln(w, `data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"name":"validate_alert","arguments":{"alert_id":12345}}}]}}]}`)
				flusher.Flush()
				fmt.Fprintln(w, `data: [DONE]`)
				flusher.Flush()
			}))
			defer mockServer.Close()

			cfg := &config.Config{}
			cfg.LLM.Provider = "gemini"
			cfg.LLM.BaseURL = mockServer.URL

			client := llm.NewOpenAIClient(cfg)
			req := requests.LLMChatRequest{
				Messages: []requests.LLMMessage{
					{
						Role:    "assistant",
						Content: "calling tool",
						ToolCalls: []requests.LLMToolCall{
							{
								Function: requests.LLMFunctionCall{
									Name:             "test_fn",
									Arguments:        map[string]interface{}{"key": "val"},
									ThoughtSignature: "sig_thought",
								},
							},
						},
					},
				},
			}
			msg, err := client.QueryStreamWithTools(context.Background(), req, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(msg.ToolCalls)).To(Equal(1))
		})
	})
})
