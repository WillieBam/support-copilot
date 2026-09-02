import React, { useState, useEffect } from 'react';
import { X, Search, Link2, Loader2, AlertCircle, Check, Clock } from 'lucide-react';
import { fetchRecentAlerts, linkAlertsToIncident } from '@/service/alert/alertService';
import type { AlertItem } from '@/types/alert';

interface LinkAlertModalProps {
  incidentId: string;
  alreadyLinkedIds: Set<string>;
  onSuccess: () => void;
  onClose: () => void;
}

export const LinkAlertModal: React.FC<LinkAlertModalProps> = ({
  incidentId,
  alreadyLinkedIds,
  onSuccess,
  onClose,
}) => {
  const [alerts, setAlerts] = useState<AlertItem[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const loadAlerts = async () => {
      try {
        setIsLoading(true);
        const data = await fetchRecentAlerts(100);
        setAlerts(data);
      } catch (err: any) {
        setError(err?.response?.data?.error || 'Failed to load recent alerts');
      } finally {
        setIsLoading(false);
      }
    };
    loadAlerts();
  }, []);

  const getAlertId = (a: AlertItem): string => {
    return a.id || (a as any)?.ID || '';
  };

  const parseAlertDetails = (a: AlertItem) => {
    let service = 'unknown-service';
    let severity = 'INFO';
    let message = '';

    try {
      const rawResource = a.resource_info || (a as any)?.ResourceInfo;
      if (rawResource) {
        const res = typeof rawResource === 'string' ? JSON.parse(rawResource) : rawResource;
        if (res?.service) service = String(res.service);
      }
      const rawAlert = a.alert_info || (a as any)?.AlertInfo;
      if (rawAlert) {
        const alert = typeof rawAlert === 'string' ? JSON.parse(rawAlert) : rawAlert;
        if (alert?.severity) severity = String(alert.severity);
        if (alert?.message) message = String(alert.message);
      }
    } catch {
      // ignore JSON parse fallback
    }

    return { service, severity, message };
  };

  const getSeverityBadgeClass = (s?: string) => {
    switch (s?.toUpperCase()) {
      case 'CRITICAL':
        return 'bg-red-500/10 text-red-500 border-red-500/20';
      case 'HIGH':
      case 'WARNING':
        return 'bg-amber-500/10 text-amber-500 border-amber-500/20';
      case 'MEDIUM':
      case 'LOW':
        return 'bg-blue-500/10 text-blue-500 border-blue-500/20';
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
        hour: '2-digit',
        minute: '2-digit',
        hour12: true,
      });
    } catch {
      return isoString;
    }
  };

  const toggleSelect = (id: string) => {
    if (!id) return;
    const next = new Set(selectedIds);
    if (next.has(id)) {
      next.delete(id);
    } else {
      next.add(id);
    }
    setSelectedIds(next);
  };

  const filteredAlerts = (alerts || []).filter((a) => {
    const alertId = getAlertId(a);
    if (!alertId || alreadyLinkedIds.has(alertId)) return false;
    const { service, severity, message } = parseAlertDetails(a);
    const q = (searchQuery || '').trim().toLowerCase();
    if (!q) return true;
    return (
      (service || '').toLowerCase().includes(q) ||
      (severity || '').toLowerCase().includes(q) ||
      (message || '').toLowerCase().includes(q) ||
      alertId.toLowerCase().includes(q)
    );
  });

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (selectedIds.size === 0) return;

    try {
      setIsSubmitting(true);
      setError(null);
      await linkAlertsToIncident(incidentId, {
        alert_ids: Array.from(selectedIds),
      });
      onSuccess();
      onClose();
    } catch (err: any) {
      setError(err?.response?.data?.error || 'Failed to link selected alerts');
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-xs p-4">
      <div className="bg-card border border-border rounded-2xl w-full max-w-2xl shadow-xl flex flex-col max-h-[85vh] overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-border">
          <div className="flex items-center gap-2">
            <Link2 className="w-5 h-5 text-emerald-500" />
            <h2 className="text-base font-bold text-foreground">Link Alerts to Incident</h2>
          </div>
          <button
            onClick={onClose}
            className="p-1 rounded-lg text-muted-foreground hover:text-foreground hover:bg-muted transition-colors cursor-pointer"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* Search */}
        <div className="p-4 border-b border-border bg-muted/20">
          <div className="relative">
            <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Filter by service name, severity, or message..."
              className="w-full bg-background border border-border rounded-xl pl-9 pr-4 py-2 text-xs focus:outline-none focus:ring-2 focus:ring-emerald-500/50"
            />
          </div>
        </div>

        {/* Content List */}
        <div className="flex-1 overflow-y-auto p-4 space-y-2">
          {error && (
            <div className="p-3 bg-red-500/10 border border-red-500/20 text-red-500 rounded-xl text-xs flex items-center gap-2">
              <AlertCircle className="w-4 h-4 shrink-0" /> {error}
            </div>
          )}

          {isLoading ? (
            <div className="py-12 flex flex-col items-center justify-center text-muted-foreground text-xs gap-2">
              <Loader2 className="w-6 h-6 animate-spin text-emerald-500" />
              Loading unlinked alerts...
            </div>
          ) : filteredAlerts.length === 0 ? (
            <div className="py-12 text-center text-muted-foreground text-xs italic">
              {searchQuery ? 'No alerts matched your search query.' : 'No available unlinked alerts found.'}
            </div>
          ) : (
            filteredAlerts.map((a) => {
              const alertId = getAlertId(a);
              const isSelected = selectedIds.has(alertId);
              const { service, severity, message } = parseAlertDetails(a);
              const incidentId = a.incident_id || (a as any)?.IncidentID;
              const receivedAt = a.received_at || (a as any)?.ReceivedAt;

              return (
                <div
                  key={alertId}
                  onClick={() => toggleSelect(alertId)}
                  className={`p-3 rounded-xl border transition-all cursor-pointer flex items-center justify-between gap-3 ${
                    isSelected
                      ? 'bg-emerald-500/10 border-emerald-500/40'
                      : 'bg-background hover:bg-muted/40 border-border'
                  }`}
                >
                  <div className="flex items-center gap-3 min-w-0">
                    <div
                      className={`w-4 h-4 rounded-md border flex items-center justify-center shrink-0 transition-colors ${
                        isSelected
                          ? 'bg-emerald-500 border-emerald-500 text-white'
                          : 'border-border bg-card'
                      }`}
                    >
                      {isSelected && <Check className="w-3 h-3" />}
                    </div>

                    <div className="min-w-0 space-y-1">
                      <div className="flex items-center gap-2">
                        <span className="font-semibold text-xs text-foreground truncate">
                          {service}
                        </span>
                        <span
                          className={`px-2 py-0.5 text-[10px] font-semibold rounded-full border ${getSeverityBadgeClass(
                            severity
                          )}`}
                        >
                          {severity}
                        </span>
                        {incidentId && (
                          <span className="text-[10px] text-muted-foreground bg-muted px-1.5 py-0.2 rounded">
                            linked to other
                          </span>
                        )}
                      </div>
                      {message && (
                        <p className="text-[11px] text-muted-foreground line-clamp-1">
                          {message}
                        </p>
                      )}
                    </div>
                  </div>

                  <span className="text-[10px] text-muted-foreground shrink-0 flex items-center gap-1">
                    <Clock className="w-3 h-3" /> {formatDate(receivedAt)}
                  </span>
                </div>
              );
            })
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-between px-6 py-4 border-t border-border bg-muted/10">
          <span className="text-xs text-muted-foreground">
            {selectedIds.size} alert(s) selected
          </span>
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={onClose}
              className="px-3.5 py-1.5 rounded-xl border border-border text-xs font-medium text-muted-foreground hover:text-foreground hover:bg-muted transition-colors cursor-pointer"
            >
              Cancel
            </button>
            <button
              type="button"
              onClick={handleSubmit}
              disabled={selectedIds.size === 0 || isSubmitting}
              className="flex items-center gap-1.5 bg-emerald-500 hover:bg-emerald-600 text-white text-xs px-4 py-1.5 rounded-xl font-medium transition-colors disabled:opacity-50 cursor-pointer shadow-sm"
            >
              {isSubmitting ? (
                <Loader2 className="w-3.5 h-3.5 animate-spin" />
              ) : (
                <Link2 className="w-3.5 h-3.5" />
              )}
              Link to Incident
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};
