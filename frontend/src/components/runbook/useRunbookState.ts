import { useState, useEffect, useCallback } from 'react';
import type { Runbook } from '@/types/runbook';
import { fetchTeamRunbooks } from '@/service/runbook/runbookService';

export function useRunbookState(teamId?: string | null) {
  const [runbooks, setRunbooks] = useState<Runbook[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [showAllStatuses, setShowAllStatuses] = useState(false);

  // Modal states
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);

  const loadRunbooks = useCallback(async () => {
    if (!teamId) {
      setRunbooks([]);
      return;
    }
    setIsLoading(true);
    try {
      const statusParam = showAllStatuses ? '' : 'active';
      const data = await fetchTeamRunbooks(teamId, statusParam);
      setRunbooks(data || []);
    } catch {
      setRunbooks([]);
    } finally {
      setIsLoading(false);
    }
  }, [teamId, showAllStatuses]);

  useEffect(() => {
    loadRunbooks();

    const handleRunbooksUpdated = () => {
      loadRunbooks();
    };

    window.addEventListener('runbooks-updated', handleRunbooksUpdated);
    return () => {
      window.removeEventListener('runbooks-updated', handleRunbooksUpdated);
    };
  }, [loadRunbooks]);

  const filteredRunbooks = runbooks.filter((rb) => {
    if (searchQuery.trim()) {
      const q = searchQuery.toLowerCase();
      const matchTitle = rb.title?.toLowerCase().includes(q);
      const matchContent = rb.content?.toLowerCase().includes(q);
      const matchStatus = rb.status?.toLowerCase().includes(q);
      return matchTitle || matchContent || matchStatus;
    }
    return true;
  });

  return {
    teamId,
    runbooks: filteredRunbooks,
    isLoading,
    searchQuery,
    setSearchQuery,
    showAllStatuses,
    setShowAllStatuses,
    isCreateModalOpen,
    setIsCreateModalOpen,
    refreshRunbooks: loadRunbooks,
  };
}
