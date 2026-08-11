import React, { useState } from 'react';
import { AlertCircle, Loader2, X } from 'lucide-react';
import { createTeamIncident } from '@/service/incident/incidentService';

interface CreateIncidentModalProps {
  teamId: string;
  onSuccess: () => void;
  onClose: () => void;
}

export const CreateIncidentModal: React.FC<CreateIncidentModalProps> = ({
  teamId,
  onSuccess,
  onClose,
}) => {
  const [title, setTitle] = useState('');
  const [status, setStatus] = useState('OPEN');
  const [details, setDetails] = useState('');
  const [alertId, setAlertId] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!title.trim()) {
      setError('Title is required');
      return;
    }

    setIsSubmitting(true);
    setError(null);

    try {
      await createTeamIncident(teamId, {
        title: title.trim(),
        status,
        details: details.trim(),
        alert_id: alertId.trim() || undefined,
      });
      onSuccess();
      onClose();
    } catch (err: any) {
      setError(err?.response?.data?.error || 'Failed to create incident');
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="fixed inset-0 z-[100] flex items-center justify-center bg-background/80 backdrop-blur-sm p-4">
      <div className="relative w-full max-w-lg bg-card border border-border rounded-[20px] shadow-2xl p-6 transition-all">
        <button
          onClick={onClose}
          className="absolute top-4 right-4 text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
        >
          <X className="w-5 h-5" />
        </button>

        <div className="mb-6">
          <div className="w-12 h-12 bg-red-500/10 border border-red-500/20 rounded-2xl flex items-center justify-center mb-4">
            <AlertCircle className="w-6 h-6 text-red-500" />
          </div>
          <h2 className="text-xl font-bold tracking-tight">Create Incident</h2>
          <p className="text-sm text-muted-foreground mt-1">
            Track and assign a new production incident to your team.
          </p>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-1.5 text-foreground">Incident Title *</label>
            <input
              type="text"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="e.g. High CPU spike on Payment Gateway"
              className="w-full bg-background border border-border rounded-xl px-4 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500/50"
              disabled={isSubmitting}
              required
            />
          </div>

          <div>
            <label className="block text-sm font-medium mb-1.5 text-foreground">Triggering Alert ID (Optional)</label>
            <input
              type="text"
              value={alertId}
              onChange={(e) => setAlertId(e.target.value)}
              placeholder="e.g. ALT-100 or alert UUID"
              className="w-full bg-background border border-border rounded-xl px-4 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500/50"
              disabled={isSubmitting}
            />
            <p className="text-xs text-muted-foreground mt-1">Provide alert id(e.g.12345678) to link alert upon creation.</p>
          </div>

          <div>
            <label className="block text-sm font-medium mb-1.5 text-foreground">Initial Status</label>
            <select
              value={status}
              onChange={(e) => setStatus(e.target.value)}
              className="w-full bg-background border border-border rounded-xl px-4 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500/50"
              disabled={isSubmitting}
            >
              <option value="OPEN">OPEN</option>
              <option value="IN_PROGRESS">IN_PROGRESS</option>
              <option value="RESOLVED">RESOLVED</option>
              <option value="CLOSED">CLOSED</option>
            </select>
          </div>

          <div>
            <label className="block text-sm font-medium mb-1.5 text-foreground">Details & Context</label>
            <textarea
              value={details}
              onChange={(e) => setDetails(e.target.value)}
              placeholder="Describe symptoms, impacted services, metrics, or diagnostic notes..."
              rows={4}
              className="w-full bg-background border border-border rounded-xl p-4 text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500/50 resize-none"
              disabled={isSubmitting}
            />
          </div>

          {error && (
            <div className="p-3 bg-red-500/10 border border-red-500/20 text-red-500 rounded-xl text-sm">
              {error}
            </div>
          )}

          <div className="flex justify-end gap-3 pt-4 border-t border-border">
            <button
              type="button"
              onClick={onClose}
              disabled={isSubmitting}
              className="px-4 py-2 text-sm font-medium text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={isSubmitting || !title.trim()}
              className="flex items-center justify-center min-w-[130px] bg-emerald-500 hover:bg-emerald-600 text-white px-5 py-2.5 rounded-xl text-sm font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer shadow-md"
            >
              {isSubmitting ? <Loader2 className="w-4 h-4 animate-spin" /> : 'Create Incident'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};
