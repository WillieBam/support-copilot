import { useState, useEffect, useCallback } from 'react';
import type { TeamIncident } from '@/types/incident';
import { fetchTeamIncidents } from '@/service/incident/incidentService';

export function useIncidentState(teamId?: string | null) {
  const [incidents, setIncidents] = useState<TeamIncident[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [showAllStatuses, setShowAllStatuses] = useState(false);
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);

  const loadIncidents = useCallback(async () => {
    if (!teamId) {
      setIncidents([]);
      return;
    }
    setIsLoading(true);
    try {
      const data = await fetchTeamIncidents(teamId);
      setIncidents(data || []);
    } catch {
      setIncidents([]);
    } finally {
      setIsLoading(false);
    }
  }, [teamId]);

  useEffect(() => {
    loadIncidents();

    const handleIncidentsUpdated = () => {
      loadIncidents();
    };

    window.addEventListener('incidents-updated', handleIncidentsUpdated);
    return () => {
      window.removeEventListener('incidents-updated', handleIncidentsUpdated);
    };
  }, [loadIncidents]);

  const filteredIncidents = incidents.filter((inc) => {
    const statusUpper = inc.status?.toUpperCase() || '';
    const isActive = statusUpper === 'OPEN' || statusUpper === 'IN_PROGRESS' || statusUpper === 'IN PRORESS';

    if (!showAllStatuses && !isActive) {
      return false;
    }

    if (searchQuery.trim()) {
      const q = searchQuery.toLowerCase();
      const matchTitle = inc.title?.toLowerCase().includes(q);
      const matchDetails = inc.details?.toLowerCase().includes(q);
      const matchStatus = inc.status?.toLowerCase().includes(q);
      return matchTitle || matchDetails || matchStatus;
    }

    return true;
  });

  const displayedIncidents = !showAllStatuses && !searchQuery.trim()
    ? filteredIncidents.slice(0, 5)
    : filteredIncidents;

  return {
    teamId,
    incidents: displayedIncidents,
    isLoading,
    searchQuery,
    setSearchQuery,
    showAllStatuses,
    setShowAllStatuses,
    isCreateModalOpen,
    setIsCreateModalOpen,
    refreshIncidents: loadIncidents,
  };
}
