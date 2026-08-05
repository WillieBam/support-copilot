import type { Conversation } from '@/types/conversation';
import { MessageSquare, ExternalLink, Plus, User as UserIcon, Clock } from 'lucide-react';

interface ChatHistoryPanelProps {
  conversations: Conversation[];
  selectedConvId: string | null;
  isLoading: boolean;
  onSelectConversation: (id: string) => void;
  onViewAll: () => void;
  onNewChat: () => void;
}

function formatDate(dateStr: string): string {
  if (!dateStr) return '';
  const d = new Date(dateStr);
  if(isNaN(d.getTime())) return dateStr;

  const now = new Date();
  const diffMs = now.getTime() - d.getTime();

  // handler future timestamp or slight clock skew
  if (diffMs <0) return 'Just now';

  const diffMins = Math.floor(diffMs / (1000 * 60));
  const diffHours = Math.floor(diffMs / (1000 * 60 * 60));
  const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));

  if (diffMins < 1) return 'Just now';
  if (diffMins < 60) return `${diffMins}m ago`;
  if (diffHours < 24) return `${diffHours}h ago`;
  if (diffDays === 1) return 'Yesterday';
  if (diffDays < 7) return `${diffDays}d ago`;
  return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' });

}

export function ChatHistoryPanel({
  conversations,
  selectedConvId,
  isLoading,
  onSelectConversation,
  onViewAll,
  onNewChat,
}: ChatHistoryPanelProps) {
  return (
    <div className="flex flex-col h-full overflow-hidden">
      {/* Header section with view all link */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-border bg-card/20 shrink-0">
        <div className="flex items-center gap-2">
          <MessageSquare className="w-4 h-4 text-emerald-500" />
          <span className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
            Chat History
          </span>
        </div>
        <button
          onClick={onViewAll}
          className="flex items-center gap-1 text-xs text-emerald-500 hover:text-emerald-400 font-medium transition-colors cursor-pointer"
          title="View full chat history"
        >
          View all <ExternalLink className="w-3 h-3" />
        </button>
      </div>

      {/* New Chat action button */}
      <div className="p-3 border-b border-border shrink-0">
        <button
          onClick={onNewChat}
          className="w-full flex items-center justify-center gap-2 bg-emerald-500/10 hover:bg-emerald-500/20 text-emerald-500 border border-emerald-500/30 rounded-[12px] py-2 text-xs font-medium transition-colors cursor-pointer"
        >
          <Plus className="w-4 h-4" /> Start New Chat
        </button>
      </div>

      {/* list of maximum 15 recent conversations */}
      <div className="flex-1 overflow-y-auto p-2 space-y-1">
        {isLoading ? (
          <div className="p-4 text-center text-xs text-muted-foreground animate-pulse">
            Loading chat history...
          </div>
        ) : conversations.length === 0 ? (
          <div className="p-4 text-center text-xs text-muted-foreground italic">
            No past conversations yet
          </div>
        ) : (
          conversations.slice(0, 15).map((conv) => {
            const isSelected = conv.id === selectedConvId;
            const userName = conv.user?.display_name || conv.user?.email || 'User';

            return (
              <button
                key={conv.id}
                onClick={() => onSelectConversation(conv.id)}
                className={`w-full text-left p-3 rounded-[12px] transition-all flex flex-col gap-1.5 cursor-pointer border ${
                  isSelected
                    ? 'bg-emerald-500/10 border-emerald-500/40 text-foreground'
                    : 'bg-transparent border-transparent hover:bg-muted/50 text-muted-foreground hover:text-foreground'
                }`}
              >
                <div className="flex items-center justify-between gap-2">
                  <span className="font-medium text-xs truncate flex-1">
                    {conv.title || 'Chat Conversation'}
                  </span>
                  <span className="text-[10px] text-muted-foreground shrink-0 flex items-center gap-1">
                    <Clock className="w-2.5 h-2.5" />
                    {formatDate(conv.created_at)}
                  </span>
                </div>
                <div className="flex items-center gap-1.5 text-[11px] text-muted-foreground/80 truncate">
                  <UserIcon className="w-3 h-3 text-emerald-500/70 shrink-0" />
                  <span className="truncate">{userName}</span>
                </div>
              </button>
            );
          })
        )}
      </div>
    </div>
  );
}
