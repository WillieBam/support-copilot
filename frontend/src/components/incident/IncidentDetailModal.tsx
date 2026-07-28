import React, { useState, useEffect } from 'react';
import { AlertCircle, History, Clock, Save, Loader2, X, CheckCircle2, User } from 'lucide-react';
import type { TeamIncident } from '@/types/incident';
import { fetchIncidentById, updateIncidentStatus } from '@/service/incident/incidentService';

interface IncidentDetailModalProps {
  incidentId: string;
  onClose: () => void;
  onUpdated: () => void;
}

export const IncidentDetailModal: React.FC<IncidentDetailModalProps> = ({
  incidentId,
  onClose,
  onUpdated,
}) => {
  const [incident, setIncident] = useState<TeamIncident | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);

  // Edit form state
  const [title, setTitle] = useState('');
  const [status, setStatus] = useState('OPEN');
  const [details, setDetails] = useState('');

  const loadIncident = async () => {
    setIsLoading(true);
    setError(null);
    try {
      const data = await fetchIncidentById(incidentId);
      setIncident(data);
      setTitle(data.title || '');
      setStatus(data.status || 'OPEN');
      setDetails(data.details || '');
    } catch (err: any) {
      setError(err?.response?.data?.error || 'Failed to load incident details');
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    loadIncident();
  }, [incidentId]);

  const handleUpdate = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSubmitting(true);
    setError(null);
    setSuccessMsg(null);

    try {
      const updated = await updateIncidentStatus(incidentId, {
        status,
        title: title.trim(),
        details: details.trim(),
      });
      setIncident(updated);
      setTitle(updated.title || '');
      setStatus(updated.status || 'OPEN');
      setDetails(updated.details || '');
      setSuccessMsg('Incident updated successfully!');
      onUpdated();
      setTimeout(() => setSuccessMsg(null), 3000);
    } catch (err: any) {
      setError(err?.response?.data?.error || 'Failed to update incident status');
    } finally {
      setIsSubmitting(false);
    }
  };

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
      return new Date(isoString).toLocaleString('en-US', {
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
      });
    } catch {
      return isoString;
    }
  };

  return (
    <div className="fixed inset-0 z-[100] flex items-center justify-center bg-background/80 backdrop-blur-sm p-4 overflow-y-auto">
      <div className="relative w-full max-w-2xl bg-card border border-border rounded-[20px] shadow-2xl p-6 transition-all my-8 max-h-[90vh] flex flex-col">
        <button
          onClick={onClose}
          className="absolute top-4 right-4 text-muted-foreground hover:text-foreground transition-colors cursor-pointer z-10"
        >
          <X className="w-5 h-5" />
        </button>

        {isLoading ? (
          <div className="flex flex-col items-center justify-center py-16 text-muted-foreground">
            <Loader2 className="w-8 h-8 animate-spin text-emerald-500 mb-3" />
            <p className="text-sm">Loading incident details...</p>
          </div>
        ) : incident ? (
          <div className="flex flex-col flex-1 overflow-y-auto pr-1 space-y-6">
            {/* Header section */}
            <div className="border-b border-border pb-4 pr-8">
              <div className="flex items-center gap-3 mb-2">
                <span className={`px-3 py-1 text-xs font-semibold rounded-full border ${getStatusBadgeClass(incident.status)}`}>
                  {incident.status}
                </span>
                <span className="text-xs text-muted-foreground flex items-center gap-1">
                  <Clock className="w-3.5 h-3.5" /> Assigned: {formatDate(incident.assigned_at)}
                </span>
              </div>
              <h2 className="text-xl font-bold text-foreground">{incident.title}</h2>
              <p className="text-xs text-muted-foreground mt-1 flex items-center gap-1">
                <User className="w-3.5 h-3.5" /> Assigned By: {incident.assigned_by}
              </p>
            </div>

            {/* Update / Edit Form */}
            <form onSubmit={handleUpdate} className="space-y-4 bg-muted/30 border border-border p-5 rounded-2xl">
              <h3 className="text-sm font-semibold text-foreground flex items-center gap-2">
                <Save className="w-4 h-4 text-emerald-500" /> Update Incident State & Details
              </h3>

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <div>
                  <label className="block text-xs font-medium mb-1 text-muted-foreground">Title</label>
                  <input
                    type="text"
                    value={title}
                    onChange={(e) => setTitle(e.target.value)}
                    className="w-full bg-background border border-border rounded-xl px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500/50"
                    disabled={isSubmitting}
                    required
                  />
                </div>

                <div>
                  <label className="block text-xs font-medium mb-1 text-muted-foreground">Status</label>
                  <select
                    value={status}
                    onChange={(e) => setStatus(e.target.value)}
                    className="w-full bg-background border border-border rounded-xl px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500/50"
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
                <label className="block text-xs font-medium mb-1 text-muted-foreground">Details & Resolution Notes</label>
                <textarea
                  value={details}
                  onChange={(e) => setDetails(e.target.value)}
                  rows={3}
                  placeholder="Add details, diagnostic notes, root cause, or resolution steps..."
                  className="w-full bg-background border border-border rounded-xl p-3 text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500/50 resize-none"
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

            {/* Status History Timeline */}
            <div className="pt-2">
              <h3 className="text-sm font-semibold text-foreground flex items-center gap-2 mb-3">
                <History className="w-4 h-4 text-emerald-500" /> Status Change History Timeline
              </h3>

              {!incident.history || incident.history.length === 0 ? (
                <p className="text-xs text-muted-foreground italic bg-muted/20 p-4 rounded-xl text-center">
                  No status transition logs recorded yet.
                </p>
              ) : (
                <div className="relative border-l-2 border-border ml-3 pl-5 space-y-4">
                  {incident.history.map((item) => (
                    <div key={item.id} className="relative group">
                      {/* Timeline dot */}
                      <div className="absolute -left-[27px] top-1 w-3 h-3 rounded-full bg-emerald-500 border-2 border-card" />

                      <div className="bg-card/70 border border-border p-3.5 rounded-xl">
                        <div className="flex flex-wrap items-center justify-between gap-2 mb-1">
                          <div className="flex items-center gap-2">
                            <span className="text-xs font-semibold text-foreground">
                              {item.previous_status ? `${item.previous_status} → ` : 'Initial: '}
                              <span className={`px-2 py-0.5 text-[10px] rounded-full border ${getStatusBadgeClass(item.new_status)}`}>
                                {item.new_status}
                              </span>
                            </span>
                          </div>
                          <span className="text-[11px] text-muted-foreground flex items-center gap-1">
                            <Clock className="w-3 h-3" /> {formatDate(item.updated_at)}
                          </span>
                        </div>

                        {item.details && (
                          <p className="text-xs text-muted-foreground mt-1 whitespace-pre-wrap">
                            {item.details}
                          </p>
                        )}
                        <span className="text-[10px] text-muted-foreground/70 block mt-1">
                          Updated by: {item.updated_by}
                        </span>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        ) : (
          <div className="p-8 text-center text-muted-foreground">
            Incident not found.
          </div>
        )}
      </div>
    </div>
  );
};
