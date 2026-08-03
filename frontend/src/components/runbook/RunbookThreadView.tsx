import React from 'react';
import { ArrowLeft, BookOpen, Bot, FileText, Check, Edit3, Trash2, Calendar, Hash, ShieldAlert } from 'lucide-react';
import { useRunbookThreadState } from './useRunbookThreadState';

interface RunbookThreadViewProps {
  runbookId: string;
  activeTeamId?: string | null;
  onBack: () => void;
  onRunbookUpdated?: () => void;
}

export const RunbookThreadView: React.FC<RunbookThreadViewProps> = ({
  runbookId,
  activeTeamId,
  onBack,
  onRunbookUpdated,
}) => {
  const {
    runbook,
    isLoading,
    authError,
    isEditing,
    setIsEditing,
    editTitle,
    setEditTitle,
    editContent,
    setEditContent,
    isSaving,
    handleSave,
    handleDeprecate,
  } = useRunbookThreadState(runbookId, activeTeamId, onBack, onRunbookUpdated);

  const formatDate = (isoString?: string) => {
    if (!isoString) return '';
    try {
      return new Date(isoString).toLocaleString('en-US', {
        month: 'short',
        day: 'numeric',
        year: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
      });
    } catch {
      return isoString;
    }
  };

  return (
    <div className="flex flex-col h-full w-full overflow-hidden bg-transparent">
      {/* Top Banner Navigation */}
      <div className="flex items-center justify-between px-6 py-3 border-b border-border bg-card/40 shrink-0">
        <button
          onClick={onBack}
          className="flex items-center gap-2 text-xs font-medium text-muted-foreground hover:text-foreground transition-colors cursor-pointer bg-muted/40 hover:bg-muted px-3 py-1.5 rounded-[10px]"
        >
          <ArrowLeft className="w-4 h-4" /> Back to live chat
        </button>

        <div className="flex items-center gap-2">
          {runbook && (
            <span
              className={`px-3 py-1 rounded-full text-xs font-semibold border ${
                runbook.status === 'deprecated'
                  ? 'bg-gray-500/10 border-gray-500/30 text-gray-400'
                  : 'bg-emerald-500/10 border-emerald-500/30 text-emerald-500'
              }`}
            >
              Runbook ({runbook.status})
            </span>
          )}
        </div>
      </div>

      {/* Messages Scroll View */}
      <div className="flex-1 min-h-0 overflow-y-auto p-6 space-y-6">
        {isLoading ? (
          <div className="p-12 text-center text-sm text-muted-foreground animate-pulse">
            Loading runbook thread...
          </div>
        ) : authError ? (
          <div className="p-12 flex flex-col items-center justify-center text-center space-y-3">
            <ShieldAlert className="w-10 h-10 text-red-500/80" />
            <p className="text-sm font-semibold text-foreground">{authError}</p>
            <button
              onClick={onBack}
              className="text-xs text-emerald-500 hover:text-emerald-400 font-medium underline cursor-pointer"
            >
              Return to chat
            </button>
          </div>
        ) : !runbook ? (
          <div className="p-12 text-center text-sm text-muted-foreground italic">
            Runbook not found or removed.
          </div>
        ) : (
          <div className="space-y-6 max-w-[800px] mx-auto">
            {/* System Metadata Bubble */}
            <div className="flex gap-3 mr-auto flex-row">
              <div className="w-8 h-8 rounded-full flex items-center justify-center shrink-0 border text-xs font-bold bg-emerald-500/20 text-emerald-500 border-emerald-500/40">
                <BookOpen className="w-4 h-4" />
              </div>

              <div className="flex flex-col gap-2 w-full">
                <div className="p-4 rounded-[16px] bg-card border border-border text-foreground rounded-tl-none space-y-2 shadow-sm">
                  <div className="flex items-center justify-between border-b border-border/60 pb-2">
                    <span className="text-xs font-bold uppercase tracking-wider text-emerald-500 flex items-center gap-1.5">
                      <FileText className="w-3.5 h-3.5" /> Operational Runbook
                    </span>
                    <span className="text-[11px] text-muted-foreground flex items-center gap-1">
                      <Calendar className="w-3 h-3" /> {formatDate(runbook.created_at)}
                    </span>
                  </div>

                  {isEditing ? (
                    <input
                      type="text"
                      value={editTitle}
                      onChange={(e) => setEditTitle(e.target.value)}
                      className="w-full text-base font-bold bg-background border border-border rounded-lg px-3 py-1.5 text-foreground focus:outline-none focus:ring-2 focus:ring-emerald-500"
                    />
                  ) : (
                    <h2 className="text-base font-bold text-foreground tracking-tight">
                      {runbook.title}
                    </h2>
                  )}

                  {runbook.incident_id && (
                    <div className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-md bg-muted text-[11px] font-mono text-muted-foreground">
                      <Hash className="w-3 h-3 text-emerald-500" /> Incident: {runbook.incident_id}
                    </div>
                  )}
                </div>
              </div>
            </div>

            {/* Assistant Runbook Content Bubble */}
            <div className="flex gap-3 mr-auto flex-row">
              <div className="w-8 h-8 rounded-full flex items-center justify-center shrink-0 border text-xs font-bold bg-muted text-muted-foreground border-border">
                <Bot className="w-4 h-4" />
              </div>

              <div className="flex flex-col gap-2 w-full">
                <div className="p-5 rounded-[16px] bg-card border border-border text-foreground rounded-tl-none leading-relaxed shadow-sm">
                  {isEditing ? (
                    <textarea
                      rows={12}
                      value={editContent}
                      onChange={(e) => setEditContent(e.target.value)}
                      className="w-full text-sm font-mono bg-background border border-border rounded-lg p-3 text-foreground focus:outline-none focus:ring-2 focus:ring-emerald-500 resize-y"
                    />
                  ) : (
                    <div className="text-sm whitespace-pre-wrap font-sans space-y-2">
                      {runbook.content}
                    </div>
                  )}
                </div>
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Action Footer Bar */}
      {runbook && (
        <div className="p-4 border-t border-border bg-card/40 flex items-center justify-between shrink-0 px-6">
          <div className="flex items-center gap-2">
            {isEditing ? (
              <>
                <button
                  onClick={handleSave}
                  disabled={isSaving}
                  className="flex items-center gap-1.5 bg-emerald-500 hover:bg-emerald-600 text-white text-xs px-4 py-2 rounded-xl font-semibold transition-colors cursor-pointer disabled:opacity-50"
                >
                  <Check className="w-3.5 h-3.5" /> {isSaving ? 'Saving...' : 'Save Changes'}
                </button>
                <button
                  onClick={() => setIsEditing(false)}
                  disabled={isSaving}
                  className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground px-3 py-2 rounded-xl transition-colors cursor-pointer"
                >
                  Cancel
                </button>
              </>
            ) : (
              <button
                onClick={() => setIsEditing(true)}
                className="flex items-center gap-1.5 bg-muted/60 hover:bg-muted text-foreground text-xs px-3.5 py-2 rounded-xl font-medium border border-border transition-colors cursor-pointer"
              >
                <Edit3 className="w-3.5 h-3.5 text-emerald-500" /> Edit Content
              </button>
            )}
          </div>

          {runbook.status === 'active' && !isEditing && (
            <button
              onClick={handleDeprecate}
              disabled={isSaving}
              className="flex items-center gap-1.5 bg-red-500/10 hover:bg-red-500/20 text-red-500 text-xs px-3.5 py-2 rounded-xl font-medium border border-red-500/20 transition-colors cursor-pointer disabled:opacity-50"
            >
              <Trash2 className="w-3.5 h-3.5" /> Deprecate Runbook
            </button>
          )}
        </div>
      )}
    </div>
  );
};
