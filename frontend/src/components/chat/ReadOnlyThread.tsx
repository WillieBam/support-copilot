import type { ConversationMessage } from '@/types/conversation';
import { ArrowLeft, Eye, Bot, User, Brain } from 'lucide-react';

interface ReadOnlyThreadProps {
  messages: ConversationMessage[];
  isLoading: boolean;
  onBack: () => void;
}

export function ReadOnlyThread({ messages, isLoading, onBack }: ReadOnlyThreadProps) {
  return (
    <div className="flex flex-col h-full w-full overflow-hidden bg-transparent">
      {/* Top Banner / Navigation */}
      <div className="flex items-center justify-between px-6 py-3 border-b border-border bg-card/40 shrink-0">
        <button
          onClick={onBack}
          className="flex items-center gap-2 text-xs font-medium text-muted-foreground hover:text-foreground transition-colors cursor-pointer bg-muted/40 hover:bg-muted px-3 py-1.5 rounded-[10px]"
        >
          <ArrowLeft className="w-4 h-4" /> Back to live chat
        </button>
        <div className="flex items-center gap-1.5 px-3 py-1 bg-amber-500/10 border border-amber-500/30 text-amber-500 rounded-full text-xs font-semibold">
          <Eye className="w-3.5 h-3.5" /> Read-Only Mode
        </div>
      </div>

      {/* Messages Scroll View */}
      <div className="flex-1 overflow-y-auto p-6 space-y-6">
        {isLoading ? (
          <div className="p-12 text-center text-sm text-muted-foreground animate-pulse">
            Loading chat conversation...
          </div>
        ) : messages.length === 0 ? (
          <div className="p-12 text-center text-sm text-muted-foreground italic">
            No messages found in this conversation.
          </div>
        ) : (
          messages.map((msg) => {
            const isUser = msg.sender === 'user';

            return (
              <div
                key={msg.id}
                className={`flex gap-3 max-w-[85%] ${
                  isUser ? 'ml-auto flex-row-reverse' : 'mr-auto flex-row'
                }`}
              >
                {/* Avatar icon */}
                <div
                  className={`w-8 h-8 rounded-full flex items-center justify-center shrink-0 border text-xs font-bold ${
                    isUser
                      ? 'bg-emerald-500/20 text-emerald-500 border-emerald-500/40'
                      : 'bg-muted text-muted-foreground border-border'
                  }`}
                >
                  {isUser ? <User className="w-4 h-4" /> : <Bot className="w-4 h-4" />}
                </div>

                {/* Message Bubble */}
                <div className="flex flex-col gap-2">
                  {/* Optional Reasoning Steps */}
                  {msg.reasoning_steps && (
                    <div className="p-3 bg-muted/30 border border-border/60 rounded-[14px] text-xs text-muted-foreground font-mono space-y-1">
                      <div className="flex items-center gap-1.5 font-sans font-semibold text-emerald-500">
                        <Brain className="w-3.5 h-3.5" /> Reasoning Steps
                      </div>
                      <p className="whitespace-pre-wrap">{msg.reasoning_steps}</p>
                    </div>
                  )}

                  {/* Message Content */}
                  <div
                    className={`p-4 rounded-[16px] text-sm whitespace-pre-wrap leading-relaxed shadow-sm ${
                      isUser
                        ? 'bg-emerald-600 text-white rounded-tr-none'
                        : 'bg-card border border-border text-foreground rounded-tl-none'
                    }`}
                  >
                    {msg.content}
                  </div>
                </div>
              </div>
            );
          })
        )}
      </div>

      {/* Disabled Read-Only Footer Banner */}
      <div className="p-4 border-t border-border bg-card/40 text-center shrink-0">
        <p className="text-xs text-muted-foreground">
          📖 This conversation is read-only. Start a new chat session to send messages.
        </p>
      </div>
    </div>
  );
}
