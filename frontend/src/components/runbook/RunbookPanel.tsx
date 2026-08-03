import React from 'react';
import { BookOpen, Plus, Search, Filter, RefreshCw, Loader2, Clock, ChevronRight } from 'lucide-react';
import { useRunbookState } from './useRunbookState';
import { CreateRunbookModal } from './CreateRunbookModal';

interface RunbookPanelProps {
  teamId?: string | null;
  selectedRunbookId?: string | null;
  onSelectRunbookForThread: (runbookId: string) => void;
}

export const RunbookPanel: React.FC<RunbookPanelProps> = ({
  teamId,
  selectedRunbookId,
  onSelectRunbookForThread,
}) => {
  const {
    runbooks,
    isLoading,
    searchQuery,
    setSearchQuery,
    showAllStatuses,
    setShowAllStatuses,
    isCreateModalOpen,
    setIsCreateModalOpen,
    refreshRunbooks,
  } = useRunbookState(teamId);

  const formatDate = (isoString?: string) => {
    if (!isoString) return '';
    try {
      return new Date(isoString).toLocaleDateString('en-US', {
        month: 'short',
        day: 'numeric',
      });
    } catch {
      return '';
    }
  };

  return (
    <div className="flex flex-col h-full w-full bg-transparent overflow-hidden">
      {/* Header & Actions */}
      <div className="p-4 border-b border-border space-y-3 shrink-0">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <BookOpen className="w-5 h-5 text-emerald-500" />
            <h2 className="text-base font-bold tracking-tight text-foreground">Team Runbooks</h2>
          </div>
          <div className="flex items-center gap-1.5">
            <button
              onClick={refreshRunbooks}
              disabled={isLoading || !teamId}
              className="p-1.5 rounded-xl border border-border text-muted-foreground hover:text-foreground hover:bg-muted transition-colors disabled:opacity-50 cursor-pointer"
              title="Refresh runbooks"
            >
              <RefreshCw className={`w-3.5 h-3.5 ${isLoading ? 'animate-spin' : ''}`} />
            </button>
            {teamId && (
              <button
                onClick={() => setIsCreateModalOpen(true)}
                className="flex items-center gap-1.5 bg-emerald-500 hover:bg-emerald-600 text-white text-xs px-3 py-1.5 rounded-xl font-medium transition-colors cursor-pointer shadow-sm"
              >
                <Plus className="w-3.5 h-3.5" /> Create
              </button>
            )}
          </div>
        </div>

        {/* Search & Filter */}
        <div className="flex items-center gap-2">
          <div className="relative flex-1">
            <Search className="w-3.5 h-3.5 absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Search runbooks..."
              className="w-full bg-background border border-border rounded-xl pl-8 pr-3 py-1.5 text-xs focus:outline-none focus:ring-2 focus:ring-emerald-500/50"
            />
          </div>
          <button
            onClick={() => setShowAllStatuses(!showAllStatuses)}
            className={`flex items-center gap-1 px-2.5 py-1.5 text-xs border rounded-xl transition-colors cursor-pointer whitespace-nowrap ${
              showAllStatuses
                ? 'bg-emerald-500/10 border-emerald-500/30 text-emerald-500'
                : 'bg-background border-border text-muted-foreground hover:text-foreground'
            }`}
            title={showAllStatuses ? 'Showing All (incl. Deprecated)' : 'Showing Active Only'}
          >
            <Filter className="w-3 h-3" />
            {showAllStatuses ? 'All' : 'Active'}
          </button>
        </div>
      </div>

      {/* Runbook List */}
      <div className="flex-1 overflow-y-auto p-3 space-y-2.5">
        {!teamId ? (
          <div className="flex flex-col items-center justify-center h-full text-center p-6 text-muted-foreground">
            <BookOpen className="w-8 h-8 text-muted-foreground/40 mb-2" />
            <p className="text-xs">Select a team to view runbooks.</p>
          </div>
        ) : isLoading ? (
          <div className="flex flex-col items-center justify-center h-full py-12 text-muted-foreground">
            <Loader2 className="w-6 h-6 animate-spin text-emerald-500 mb-2" />
            <p className="text-xs">Loading runbooks...</p>
          </div>
        ) : runbooks.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-full text-center p-6 text-muted-foreground">
            <BookOpen className="w-8 h-8 text-muted-foreground/30 mb-2" />
            <p className="text-xs font-medium text-foreground">No runbooks found</p>
            <p className="text-[11px] mt-1 text-muted-foreground">
              {showAllStatuses
                ? 'No runbooks exist for this team.'
                : 'No active runbooks. Click "All" filter to view deprecated runbooks.'}
            </p>
          </div>
        ) : (
          runbooks.map((rb) => {
            const isSelected = selectedRunbookId === rb.id;
            const isDeprecated = rb.status === 'deprecated';

            return (
              <div
                key={rb.id}
                className={`p-3.5 rounded-xl border transition-all flex flex-col gap-2 group cursor-pointer ${
                  isSelected
                    ? 'bg-emerald-500/10 border-emerald-500/40 text-foreground'
                    : 'bg-card/70 hover:bg-muted/50 border-border'
                }`}
                onClick={() => onSelectRunbookForThread(rb.id)}
              >
                <div className="flex items-start justify-between gap-2">
                  <div className="flex items-center gap-1.5 flex-1 min-w-0">
                    <BookOpen className="w-3.5 h-3.5 text-emerald-500 shrink-0" />
                    <h3 className="text-xs font-semibold text-foreground group-hover:text-emerald-500 transition-colors truncate">
                      {rb.title}
                    </h3>
                  </div>
                  <span className="text-[10px] text-muted-foreground shrink-0 flex items-center gap-1">
                    <Clock className="w-3 h-3" /> {formatDate(rb.created_at)}
                  </span>
                </div>

                <p className="text-[11px] text-muted-foreground line-clamp-2">
                  {rb.content}
                </p>

                {/* Footer Status Badge */}
                <div className="flex items-center justify-between pt-1 border-t border-border/50 text-[10px]">
                  <span
                    className={`px-2 py-0.5 rounded-full font-medium ${
                      isDeprecated
                        ? 'bg-gray-500/10 text-gray-400 border border-gray-500/20'
                        : 'bg-emerald-500/10 text-emerald-500 border border-emerald-500/20'
                    }`}
                  >
                    {rb.status}
                  </span>

                  <span className="flex items-center gap-0.5 text-muted-foreground group-hover:text-emerald-500 font-medium transition-colors text-[11px]">
                    Open thread <ChevronRight className="w-3 h-3" />
                  </span>
                </div>
              </div>
            );
          })
        )}
      </div>

      {/* Create Modal */}
      {isCreateModalOpen && teamId && (
        <CreateRunbookModal
          teamId={teamId}
          onSuccess={refreshRunbooks}
          onClose={() => setIsCreateModalOpen(false)}
        />
      )}
    </div>
  );
};
