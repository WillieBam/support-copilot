import { useState, useEffect, useCallback } from 'react';
import type { TeamIncident } from '@/types/incident';
import { fetchIncidentById, updateIncidentStatus } from '@/service/incident/incidentService';

export function useIncidentThreadState(
  incidentId: string,
  activeTeamId?: string | null,
  onIncidentUpdated?: () => void
) {
  const [incident, setIncident] = useState<TeamIncident | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [authError, setAuthError] = useState<string | null>(null);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);

  // Edit form state
  const [title, setTitle] = useState('');
  const [status, setStatus] = useState('OPEN');
  const [details, setDetails] = useState('');

  const loadIncident = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    setAuthError(null);
    try {
      const data = await fetchIncidentById(incidentId);
      if (activeTeamId && data.team_id && data.team_id !== activeTeamId) {
        setAuthError('Unauthorized: You are not authorized to view incidents belonging to another team.');
        setIncident(null);
        return;
      }
      setIncident(data);
      setTitle(data.title || '');
      setStatus(data.status || 'OPEN');
      setDetails(data.details || '');
    } catch (err: any) {
      setError(err?.response?.data?.error || 'Failed to load incident details');
    } finally {
      setIsLoading(false);
    }
  }, [incidentId, activeTeamId]);

  useEffect(() => {
    loadIncident();
  }, [loadIncident]);

  const handleUpdate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!title.trim()) return;

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
      setSuccessMsg('Incident status and details updated successfully!');
      onIncidentUpdated?.();
      window.dispatchEvent(new CustomEvent('incidents-updated'));
      setTimeout(() => setSuccessMsg(null), 3000);
    } catch (err: any) {
      setError(err?.response?.data?.error || 'Failed to update incident status');
    } finally {
      setIsSubmitting(false);
    }
  };

  return {
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
    refreshIncident: loadIncident,
  };
}
