import React from 'react';
import { X, BookOpen, Plus, AlertCircle } from 'lucide-react';
import { useCreateRunbookState } from './useCreateRunbookState';

interface CreateRunbookModalProps {
  teamId: string;
  onSuccess: () => void;
  onClose: () => void;
}

export const CreateRunbookModal: React.FC<CreateRunbookModalProps> = ({
  teamId,
  onSuccess,
  onClose,
}) => {
  const {
    title,
    setTitle,
    incidentId,
    setIncidentId,
    content,
    setContent,
    isSubmitting,
    error,
    handleSubmit,
  } = useCreateRunbookState(teamId, onSuccess, onClose);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm animate-in fade-in duration-200">
      <div className="bg-card border border-border rounded-[24px] w-full max-w-[560px] flex flex-col shadow-2xl overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-border bg-card/60">
          <div className="flex items-center gap-2">
            <BookOpen className="w-5 h-5 text-emerald-500" />
            <h2 className="text-base font-bold tracking-tight text-foreground">Create New Runbook</h2>
          </div>
          <button
            onClick={onClose}
            className="p-1 rounded-full text-muted-foreground hover:text-foreground hover:bg-muted transition-colors cursor-pointer"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Form Body */}
        <form onSubmit={handleSubmit} className="p-6 space-y-4">
          {error && (
            <div className="p-3 rounded-xl bg-red-500/10 border border-red-500/20 text-red-500 text-xs flex items-center gap-2">
              <AlertCircle className="w-4 h-4 shrink-0" />
              <span>{error}</span>
            </div>
          )}

          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">Runbook Title *</label>
            <input
              type="text"
              required
              placeholder="e.g. Database Connection Failover Procedure"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              className="w-full bg-background border border-border rounded-xl px-3.5 py-2 text-xs text-foreground focus:outline-none focus:ring-2 focus:ring-emerald-500/50"
            />
          </div>

          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">Associated Incident ID (Optional)</label>
            <input
              type="text"
              placeholder="e.g. inc-123 or UUID"
              value={incidentId}
              onChange={(e) => setIncidentId(e.target.value)}
              className="w-full bg-background border border-border rounded-xl px-3.5 py-2 text-xs text-foreground focus:outline-none focus:ring-2 focus:ring-emerald-500/50 font-mono"
            />
          </div>

          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">Runbook Content / Steps *</label>
            <textarea
              required
              rows={8}
              placeholder="Provide step-by-step resolution procedures..."
              value={content}
              onChange={(e) => setContent(e.target.value)}
              className="w-full bg-background border border-border rounded-xl p-3 text-xs text-foreground focus:outline-none focus:ring-2 focus:ring-emerald-500/50 font-mono resize-y"
            />
          </div>

          <div className="flex items-center justify-end gap-3 pt-2">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 text-xs font-medium text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={isSubmitting}
              className="flex items-center gap-1.5 bg-emerald-500 hover:bg-emerald-600 text-white text-xs px-4 py-2 rounded-xl font-semibold transition-colors cursor-pointer disabled:opacity-50 shadow-sm"
            >
              <Plus className="w-4 h-4" />
              {isSubmitting ? 'Creating...' : 'Create Runbook'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};
