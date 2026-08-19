import React from 'react';
import { ArrowLeft, AlertTriangle, Clock, Save, Loader2, User, History, CheckCircle2, AlertCircle, ShieldAlert } from 'lucide-react';
import { useIncidentThreadState } from './useIncidentThreadState';
import { useTeam } from '@/context/TeamContext';

interface IncidentThreadViewProps {
  incidentId: string;
  activeTeamId?: string | null;
  onBack: () => void;
  onIncidentUpdated?: () => void;
}

export const IncidentThreadView: React.FC<IncidentThreadViewProps> = ({
  incidentId,
  activeTeamId,
  onBack,
  onIncidentUpdated,
}) => {
  const { getUserDisplayName } = useTeam();
  const {
    incident,
    isLoading,
    isSubmitting,
    error,
    authError,
    successMsg,
    title,
    setTitle,
    status,
    setStatus,
    details,
    setDetails,
    handleUpdate,
  } = useIncidentThreadState(incidentId, activeTeamId, onIncidentUpdated);

  const getStatusBadgeClass = (s: string) => {
    switch (s?.toUpperCase()) {
      case 'OPEN':
        return 'bg-red-500/10 text-red-500 border-red-500/20';
      case 'IN_PROGRESS':
        return 'bg-amber-500/10 text-amber-500 border-amber-500/20';
      case 'RESOLVED':
        return 'bg-emerald-500/10 text-emerald-500 border-emerald-500/20';
      case 'CLOSED':
        return 'bg-gray-500/10 text-gray-400 border-gray-500/20';
      default:
        return 'bg-muted text-muted-foreground border-border';
    }
  };

  const formatDate = (isoString?: string) => {
    if (!isoString) return '';
    try {
      return new Date(isoString).toLocaleString('en-MY', {
        timeZone: 'Asia/Kuala_Lumpur',
        month: 'short',
        day: 'numeric',
        year: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
        hour12: true,
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
          {incident && (
            <div className="flex items-center gap-2">
              <span className="px-2.5 py-0.5 text-xs font-mono font-bold rounded-lg bg-emerald-500/10 text-emerald-500 border border-emerald-500/20">
                {incident.incident_number || `INC-${incident.id.slice(0, 6)}`}
              </span>
              <span className={`px-3 py-1 rounded-full text-xs font-semibold border ${getStatusBadgeClass(incident.status)}`}>
                Incident ({incident.status})
              </span>
            </div>
          )}
        </div>
      </div>

      {/* Main Workspace Thread Content */}
      <div className="flex-1 min-h-0 overflow-y-auto p-6 space-y-6">
        {isLoading ? (
          <div className="p-12 text-center text-sm text-muted-foreground animate-pulse">
            Loading incident thread...
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
        ) : !incident ? (
          <div className="p-12 text-center text-sm text-muted-foreground italic">
            Incident not found or removed.
          </div>
        ) : (
          <div className="max-w-3xl mx-auto space-y-6">
            {/* Header Card */}
            <div className="bg-card border border-border rounded-[20px] p-6 shadow-sm space-y-4">
              <div className="flex items-start justify-between gap-4">
                <div className="flex items-center gap-3">
                  <div className="p-2.5 bg-red-500/10 border border-red-500/20 rounded-xl text-red-500">
                    <AlertTriangle className="w-6 h-6" />
                  </div>
                  <div>
                    <div className="flex items-center gap-2">
                      <span className="px-2 py-0.5 text-xs font-mono font-bold rounded-md bg-emerald-500/10 text-emerald-500 border border-emerald-500/20">
                        {incident.incident_number || `INC-${incident.id.slice(0, 6)}`}
                      </span>
                      <h1 className="text-xl font-bold text-foreground tracking-tight">{incident.title}</h1>
                    </div>
                    <p className="text-xs text-muted-foreground mt-1 flex items-center gap-2">
                      <User className="w-3.5 h-3.5" /> Created by: <span className="text-foreground font-medium">{getUserDisplayName(incident.created_by)}</span>
                    </p>
                  </div>
                </div>
                <span className={`px-3 py-1 text-xs font-semibold rounded-full border shrink-0 ${getStatusBadgeClass(incident.status)}`}>
                  {incident.status}
                </span>
              </div>

              <div className="flex items-center gap-4 pt-3 border-t border-border/60 text-xs text-muted-foreground">
                <span className="flex items-center gap-1.5">
                  <Clock className="w-3.5 h-3.5" /> Created: {formatDate(incident.assigned_at)}
                </span>
              </div>
            </div>

            {/* Update / Edit State Form */}
            <form onSubmit={handleUpdate} className="bg-card border border-border rounded-[20px] p-6 py-12 shadow-sm space-y-4">
              <h2 className="text-sm font-bold text-foreground flex items-center gap-2">
                <Save className="w-4 h-4 text-emerald-500" /> Update Incident Status & Resolution Details
              </h2>

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <div>
                  <label className="block text-xs font-medium mb-1.5 text-muted-foreground">Incident Title</label>
                  <input
                    type="text"
                    value={title}
                    onChange={(e) => setTitle(e.target.value)}
                    className="w-full bg-background border border-border rounded-xl px-3.5 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500/50"
                    disabled={isSubmitting}
                    required
                  />
                </div>

                <div>
                  <label className="block text-xs font-medium mb-1.5 text-muted-foreground">Current Status</label>
                  <select
                    value={status}
                    onChange={(e) => setStatus(e.target.value)}
                    className="w-full bg-background border border-border rounded-xl px-3.5 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500/50"
                    disabled={isSubmitting}
                  >
                    <option value="OPEN">OPEN</option>
                    <option value="IN_PROGRESS">IN_PROGRESS</option>
                    <option value="RESOLVED">RESOLVED</option>
                    <option value="CLOSED">CLOSED</option>
                  </select>
                </div>
              </div>

              <div>
                <label className="block text-xs font-medium mb-1.5 text-muted-foreground">Details & Resolution Notes</label>
                <textarea
                  value={details}
                  onChange={(e) => setDetails(e.target.value)}
                  rows={12}
                  placeholder="Add details, diagnostic notes, root cause, or resolution steps..."
                  className="w-full bg-background border border-border rounded-xl p-4 text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500/50 min-h-[300px] resize-y"
                  disabled={isSubmitting}
                />
              </div>

              {error && (
                <div className="p-3 bg-red-500/10 border border-red-500/20 text-red-500 rounded-xl text-xs flex items-center gap-2">
                  <AlertCircle className="w-4 h-4 shrink-0" /> {error}
                </div>
              )}

              {successMsg && (
                <div className="p-3 bg-emerald-500/10 border border-emerald-500/20 text-emerald-500 rounded-xl text-xs flex items-center gap-2">
                  <CheckCircle2 className="w-4 h-4 shrink-0" /> {successMsg}
                </div>
              )}

              <div className="flex justify-end pt-1">
                <button
                  type="submit"
                  disabled={isSubmitting}
                  className="flex items-center gap-2 bg-emerald-500 hover:bg-emerald-600 text-white px-4 py-2 rounded-xl text-xs font-medium transition-colors disabled:opacity-50 cursor-pointer shadow-sm"
                >
                  {isSubmitting ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <Save className="w-3.5 h-3.5" />}
                  Save Status & Notes
                </button>
              </div>
            </form>

            {/* Status Change History Timeline */}
            <div className="bg-card border border-border rounded-[20px] p-6 shadow-sm space-y-4">
              <h2 className="text-sm font-bold text-foreground flex items-center gap-2">
                <History className="w-4 h-4 text-emerald-500" /> Status Transition History
              </h2>

              {!incident.history || incident.history.length === 0 ? (
                <p className="text-xs text-muted-foreground italic bg-muted/20 p-4 rounded-xl text-center">
                  No status transition logs recorded yet.
                </p>
              ) : (
                <div className="relative border-l-2 border-border ml-3 pl-5 space-y-3 pt-1">
                  {incident.history.map((item) => (
                    <div key={item.id} className="relative group">
                      {/* Timeline dot */}
                      <div className="absolute left-[27px] top-3.5 w-3 h-3 rounded-full bg-emerald-500 border-2 border-card" />

                      <div className="bg-background border border-border p-3.5 rounded-xl flex flex-wrap items-center justify-between gap-3">
                        <div className="flex flex-wrap items-center gap-3">
                          <div className="flex items-center gap-2 text-xs font-semibold text-foreground">
                            <span className="text-muted-foreground font-normal">Status:</span>
                            {item.previous_status ? (
                              <span className="flex items-center gap-1.5">
                                <span className={`px-2 py-0.5 text-[10px] rounded-full border ${getStatusBadgeClass(item.previous_status)}`}>
                                  {item.previous_status}
                                </span>
                                <span className="text-muted-foreground font-normal">→</span>
                                <span className={`px-2 py-0.5 text-[10px] rounded-full border ${getStatusBadgeClass(item.new_status)}`}>
                                  {item.new_status}
                                </span>
                              </span>
                            ) : (
                              <span className={`px-2 py-0.5 text-[10px] rounded-full border ${getStatusBadgeClass(item.new_status)}`}>
                                {item.new_status}
                              </span>
                            )}
                          </div>

                          <div className="flex items-center gap-1.5 text-xs text-muted-foreground border-l border-border pl-3">
                            <User className="w-3.5 h-3.5 text-emerald-500" />
                            <span>Updated by: <strong className="text-foreground font-medium">{getUserDisplayName(item.updated_by)}</strong></span>
                          </div>
                        </div>

                        <span className="text-[11px] text-muted-foreground flex items-center gap-1">
                          <Clock className="w-3 h-3" /> {formatDate(item.updated_at)}
                        </span>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
};
