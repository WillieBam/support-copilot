import React, { useState, useEffect, useCallback } from 'react';
import { Bell, Plus, Unlink, Loader2, AlertCircle, CheckCircle2, Clock } from 'lucide-react';
import { fetchIncidentAlerts, unlinkAlertFromIncident } from '@/service/alert/alertService';
import { LinkAlertModal } from './LinkAlertModal';
import type { AlertItem } from '@/types/alert';

interface IncidentAlertsSectionProps {
  incidentId: string;
}

export const IncidentAlertsSection: React.FC<IncidentAlertsSectionProps> = ({ incidentId }) => {
  const [alerts, setAlerts] = useState<AlertItem[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [unlinkingId, setUnlinkingId] = useState<string | null>(null);
  const [isLinkModalOpen, setIsLinkModalOpen] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);

  const loadAlerts = useCallback(async () => {
    try {
      setIsLoading(true);
      setError(null);
      const data = await fetchIncidentAlerts(incidentId);
      setAlerts(data);
    } catch (err: any) {
      setError(err?.response?.data?.error || 'Failed to fetch linked alerts');
    } finally {
      setIsLoading(false);
    }
  }, [incidentId]);

  useEffect(() => {
    loadAlerts();
  }, [loadAlerts]);

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
      // fallback
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
        year: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
        hour12: true,
      });
    } catch {
      return isoString;
    }
  };

  const handleUnlink = async (alertId: string, serviceName: string) => {
    if (!alertId) return;
    try {
      setUnlinkingId(alertId);
      setError(null);
      await unlinkAlertFromIncident(incidentId, alertId);
      setSuccessMsg(`Unlinked alert (${serviceName}) successfully.`);
      setTimeout(() => setSuccessMsg(null), 3500);
      await loadAlerts();
    } catch (err: any) {
      setError(err?.response?.data?.error || 'Failed to unlink alert');
    } finally {
      setUnlinkingId(null);
    }
  };

  const alreadyLinkedIds = new Set((alerts || []).map(getAlertId).filter(Boolean));

  return (
    <div className="bg-card border border-border rounded-[20px] p-6 shadow-sm space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Bell className="w-4 h-4 text-emerald-500" />
          <h2 className="text-sm font-bold text-foreground">
            Linked Telemetry Alerts
          </h2>
          <span className="px-2 py-0.5 text-xs font-semibold rounded-full bg-emerald-500/10 text-emerald-500 border border-emerald-500/20">
            {alerts.length}
          </span>
        </div>

        <button
          type="button"
          onClick={() => setIsLinkModalOpen(true)}
          className="flex items-center gap-1.5 bg-emerald-500 hover:bg-emerald-600 text-white text-xs px-3 py-1.5 rounded-xl font-medium transition-colors cursor-pointer shadow-sm"
        >
          <Plus className="w-3.5 h-3.5" /> Link Alerts
        </button>
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

      {/* Alert List */}
      {isLoading ? (
        <div className="p-6 flex flex-col items-center justify-center text-muted-foreground text-xs gap-2">
          <Loader2 className="w-5 h-5 animate-spin text-emerald-500" />
          Loading associated alerts...
        </div>
      ) : alerts.length === 0 ? (
        <div className="p-6 border border-dashed border-border rounded-xl text-center space-y-2">
          <p className="text-xs text-muted-foreground">
            No telemetry alerts are currently linked to this incident.
          </p>
          <button
            type="button"
            onClick={() => setIsLinkModalOpen(true)}
            className="text-xs text-emerald-500 hover:text-emerald-400 font-medium underline cursor-pointer"
          >
            Attach relevant alerts
          </button>
        </div>
      ) : (
        <div className="space-y-2.5">
          {alerts.map((a) => {
            const alertId = getAlertId(a);
            const { service, severity, message } = parseAlertDetails(a);
            const isUnlinking = unlinkingId === alertId;
            const receivedAt = a.received_at || (a as any)?.ReceivedAt;

            return (
              <div
                key={alertId || Math.random()}
                className="bg-background border border-border p-4 rounded-xl flex items-start justify-between gap-4 group hover:border-emerald-500/30 transition-colors"
              >
                <div className="space-y-1.5 min-w-0 flex-1">
                  <div className="flex items-center gap-2 flex-wrap">
                    <span className="font-semibold text-xs text-foreground">
                      {service}
                    </span>
                    <span
                      className={`px-2 py-0.5 text-[10px] font-semibold rounded-full border ${getSeverityBadgeClass(
                        severity
                      )}`}
                    >
                      {severity}
                    </span>
                    {alertId && (
                      <span className="text-[10px] text-muted-foreground font-mono">
                        ID: {alertId.slice(0, 8)}...
                      </span>
                    )}
                  </div>

                  {message && (
                    <p className="text-xs text-muted-foreground leading-relaxed">
                      {message}
                    </p>
                  )}

                  <div className="flex items-center gap-3 text-[11px] text-muted-foreground pt-1">
                    <span className="flex items-center gap-1">
                      <Clock className="w-3 h-3" /> Received: {formatDate(receivedAt)}
                    </span>
                  </div>
                </div>

                <button
                  type="button"
                  onClick={() => handleUnlink(alertId, service)}
                  disabled={isUnlinking || !alertId}
                  title="Unlink alert from this incident"
                  className="flex items-center gap-1 text-xs text-muted-foreground hover:text-red-500 hover:bg-red-500/10 px-2.5 py-1.5 rounded-lg border border-border hover:border-red-500/30 transition-all cursor-pointer disabled:opacity-50 shrink-0"
                >
                  {isUnlinking ? (
                    <Loader2 className="w-3.5 h-3.5 animate-spin text-red-500" />
                  ) : (
                    <Unlink className="w-3.5 h-3.5" />
                  )}
                  <span className="text-[11px]">Unlink</span>
                </button>
              </div>
            );
          })}
        </div>
      )}

      {/* Link Modal */}
      {isLinkModalOpen && (
        <LinkAlertModal
          incidentId={incidentId}
          alreadyLinkedIds={alreadyLinkedIds}
          onSuccess={loadAlerts}
          onClose={() => setIsLinkModalOpen(false)}
        />
      )}
    </div>
  );
};
