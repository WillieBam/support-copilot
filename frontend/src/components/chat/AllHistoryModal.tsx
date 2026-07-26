import { useState } from 'react';
import type { Conversation } from '@/types/conversation';
import { X, Search, MessageSquare, User, Clock, ChevronRight } from 'lucide-react';

interface AllHistoryModalProps {
  isOpen: boolean;
  conversations: Conversation[];
  isLoading: boolean;
  onClose: () => void;
  onSelectConversation: (id: string) => void;
}

function formatDate(dateStr: string): string {
  try {
    const d = new Date(dateStr);
    return d.toLocaleString(undefined, {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  } catch {
    return dateStr;
  }
}

export function AllHistoryModal({
  isOpen,
  conversations,
  isLoading,
  onClose,
  onSelectConversation,
}: AllHistoryModalProps) {
  const [searchQuery, setSearchQuery] = useState('');

  if (!isOpen) return null;

  const filteredConversations = conversations.filter((c) => {
    const query = searchQuery.toLowerCase();
    const titleMatch = c.title?.toLowerCase().includes(query);
    const userMatch = c.user?.email?.toLowerCase().includes(query) || c.user?.display_name?.toLowerCase().includes(query);
    return titleMatch || userMatch;
  });

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-background/80 backdrop-blur-sm animate-in fade-in duration-200">
      <div className="w-full max-w-[640px] max-h-[80vh] bg-card border border-border rounded-[20px] shadow-2xl flex flex-col overflow-hidden">
        {/* Modal Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-border bg-card/60">
          <div className="flex items-center gap-3">
            <div className="w-9 h-9 rounded-full bg-emerald-500/10 border border-emerald-500/30 flex items-center justify-center text-emerald-500">
              <MessageSquare className="w-5 h-5" />
            </div>
            <div>
              <h2 className="text-base font-bold text-foreground tracking-tight">
                Team Chat History
              </h2>
              <p className="text-xs text-muted-foreground">
                All saved conversations for this team
              </p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="w-8 h-8 rounded-full flex items-center justify-center text-muted-foreground hover:text-foreground hover:bg-muted transition-colors cursor-pointer"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* Search bar */}
        <div className="p-4 border-b border-border bg-muted/20">
          <div className="relative">
            <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
            <input
              type="text"
              placeholder="Search conversations by title or user..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full pl-9 pr-4 py-2 bg-background border border-border rounded-[12px] text-xs text-foreground placeholder:text-muted-foreground focus:outline-none focus:border-emerald-500 transition-colors"
            />
          </div>
        </div>

        {/* Conversations list */}
        <div className="flex-1 overflow-y-auto p-4 space-y-2 max-h-[500px]">
          {isLoading ? (
            <div className="p-8 text-center text-sm text-muted-foreground animate-pulse">
              Loading team history...
            </div>
          ) : filteredConversations.length === 0 ? (
            <div className="p-8 text-center text-sm text-muted-foreground italic">
              {searchQuery ? 'No matching conversations found' : 'No saved conversations'}
            </div>
          ) : (
            filteredConversations.map((conv) => {
              const userName = conv.user?.display_name || conv.user?.email || 'User';

              return (
                <button
                  key={conv.id}
                  onClick={() => onSelectConversation(conv.id)}
                  className="w-full text-left p-4 rounded-[14px] bg-muted/30 border border-border/60 hover:border-emerald-500/40 hover:bg-emerald-500/5 transition-all flex items-center justify-between group cursor-pointer"
                >
                  <div className="flex flex-col gap-1.5 flex-1 min-w-0 pr-4">
                    <span className="font-semibold text-sm text-foreground truncate group-hover:text-emerald-500 transition-colors">
                      {conv.title || 'Chat Conversation'}
                    </span>
                    <div className="flex items-center gap-4 text-xs text-muted-foreground">
                      <span className="flex items-center gap-1">
                        <User className="w-3 h-3 text-emerald-500/70" />
                        <span className="truncate">{userName}</span>
                      </span>
                      <span className="flex items-center gap-1">
                        <Clock className="w-3 h-3" />
                        <span>{formatDate(conv.created_at)}</span>
                      </span>
                    </div>
                  </div>
                  <ChevronRight className="w-4 h-4 text-muted-foreground group-hover:text-emerald-500 group-hover:translate-x-0.5 transition-all shrink-0" />
                </button>
              );
            })
          )}
        </div>
      </div>
    </div>
  );
}
