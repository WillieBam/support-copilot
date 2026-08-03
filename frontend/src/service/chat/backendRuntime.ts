import { useRef } from "react";
import {
  useLocalRuntime,
  type ChatModelAdapter,
  type ThreadMessage,
  type ChatModelRunOptions,
  type ChatModelRunResult,
} from "@assistant-ui/react";
// import apiClient from "../apiClient";
import { firebaseAuth } from "@/firebase";
import { exchangeToken } from "../auth/authService";

// const DEFAULT_API_BASE_URL = 'http://localhost:8080'
function extractText(message: ThreadMessage): string {
  return message.content
    .map((part) => {
      if (part.type === "text") return part.text;
      return "";
    })
    .filter((part) => part.trim().length > 0)
    .join("\n");
}

/** Returns the text of the latest user message (the prompt to send). */
function buildPrompt(messages: readonly ThreadMessage[]): string {
  const lastUserMessage = [...messages].reverse().find((m) => m.role === "user");
  if (!lastUserMessage) return "";
  return extractText(lastUserMessage).trim();
}

/**
 * conversation history to send to the backend.
 * Includes all messages BEFORE the latest user message so the LLM has
 * context from earlier turns (e.g. an alert ID mentioned two messages ago).
 * Only user and assistant turns are included; system messages are managed
 * server-side.
 */
function buildHistory(messages: readonly ThreadMessage[]): Array<{ role: string; content: string }> {
  const history: Array<{ role: string; content: string }> = [];

  // Find the index of the last user message
  let lastUserIdx = -1;
  for (let i = messages.length - 1; i >= 0; i--) {
    if (messages[i].role === "user") {
      lastUserIdx = i;
      break;
    }
  }

  // everything before the last user message goes into history
  for (let i = 0; i < lastUserIdx; i++) {
    const msg = messages[i];
    if (msg.role !== "user" && msg.role !== "assistant") continue;
    const text = extractText(msg).trim();
    if (!text) continue;
    history.push({ role: msg.role, content: text });
  }

  return history;
}

export interface UseBackendRuntimeOptions {
  teamId?: string | null;
  conversationId?: string | null;
  onConversationCreated?: (id: string) => void;
  onTitleGenerated?: (convId: string, title: string) => void;
  onFinish?: () => void;
}

export function useBackendRuntime(options?: UseBackendRuntimeOptions) {
  const optionsRef = useRef(options);
  optionsRef.current = options;

  const chatModelRef = useRef<ChatModelAdapter>({
    async *run({
      messages,
      abortSignal,
    }: ChatModelRunOptions): AsyncGenerator<ChatModelRunResult, void, unknown> {
      const API_BASE_URL =
        import.meta.env.VITE_API_BASE_URL || "http://localhost:8080";
      const ENDPOINT = `${API_BASE_URL}/query/chat`;
      const fetchOptions: RequestInit = {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          input: buildPrompt(messages),
          history: buildHistory(messages),
          team_id: optionsRef.current?.teamId || undefined,
          conversation_id: optionsRef.current?.conversationId || undefined,
        }),
        signal: abortSignal,
        credentials: "include",
      };

      try {
        let response = await fetch(ENDPOINT, fetchOptions);
        const user = firebaseAuth.currentUser;
        if (response.status == 401) {
          if (!user) {
            throw new Error("Unauthorized: No active user session");
          }
          try {
            await exchangeToken(user);
            response = await fetch(ENDPOINT, fetchOptions);
          } catch (refreshErr: any) {
            if (refreshErr.message !== "mfa_required") {
              await firebaseAuth.signOut().catch(() => {});
              window.location.href = "/login";
            }
            throw refreshErr;
          }
        }
        if (!response.ok) {
          throw new Error(`HTTP error! status ${response.status}`);
        }
        if (!response.body) throw new Error("No response body");
        const stream = response.body;
        if (!stream) throw new Error("Missing response body");
        const reader = stream.getReader();
        const decoder = new TextDecoder();

        let currentReasoning = "";
        let fullText = "";
        let currentConvId = optionsRef.current?.conversationId || undefined;

        while (true) {
          const { done, value } = await reader.read();
          if (done) break;

          const rawChunk = decoder.decode(value, { stream: true });
          const events = rawChunk.split("\n\n");

          for (const event of events) {
            if (!event.startsWith("data: ")) continue;

            const jsonString = event.replace("data: ", "").trim();
            if (!jsonString) continue;

            try {
              const parsed = JSON.parse(jsonString);
              if (parsed.type === "meta") {
                currentConvId = parsed.content;
                optionsRef.current?.onConversationCreated?.(parsed.content);
              } else if (parsed.type === "title") {
                if (currentConvId) {
                  optionsRef.current?.onTitleGenerated?.(currentConvId, parsed.content);
                }
              } else if (parsed.type === "text") {
                fullText += parsed.content;
              } else if (parsed.type === "reasoning") {
                currentReasoning += parsed.content + "\n";
              } else if (parsed.type === "drain") {
                // backend detected hallucinated content (e.g. embedded JSON
                // tool-call). it will discard everything accumulated so far so the
                // clean fallback response renders from a blank slate.
                fullText = "";
                currentReasoning = "";
              }

              const contentParts: any[] = [];
              if (currentReasoning.trim()) {
                contentParts.push({
                  type: "reasoning",
                  text: currentReasoning.trim(),
                });
              }
              if (fullText.trim() || contentParts.length === 0) {
                contentParts.push({
                  type: "text",
                  text: fullText,
                });
              }
              yield {
                content: contentParts,
              };
            } catch (e) {
              console.error("Error parsing chunk:", jsonString, e);
            }
          }
        }
        const finalContentParts: any[] = [];
        if (currentReasoning.trim()) {
          finalContentParts.push({ type: "reasoning", text: currentReasoning.trim() });
        }

        if (fullText.trim() || finalContentParts.length === 0) {
          finalContentParts.push({ type: "text", text: fullText });
        }

        yield {
          content: finalContentParts,
          status: { type: "complete", reason: "stop" } as const,
        };

        optionsRef.current?.onFinish?.();
        window.dispatchEvent(new CustomEvent('runbooks-updated'));
        return;

      } catch (error: any) {
        if (error.name === "AbortError") {
          console.log("Stream aborted by user");
          return;
        }
        throw new Error(error.message || "Backend request failed");
      }
    },
  });
  return { runtime: useLocalRuntime(chatModelRef.current) };
}

