import { useState, useEffect, useCallback } from 'react';
import type { Runbook } from '@/types/runbook';
import { fetchRunbookById, updateRunbook, deprecateRunbook } from '@/service/runbook/runbookService';

export function useRunbookThreadState(
  runbookId: string | null,
  activeTeamId?: string | null,
  onBack?: () => void,
  onRunbookUpdated?: () => void
) {
  const [runbook, setRunbook] = useState<Runbook | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [isEditing, setIsEditing] = useState(false);
  const [authError, setAuthError] = useState<string | null>(null);

  // Edit fields
  const [editTitle, setEditTitle] = useState('');
  const [editContent, setEditContent] = useState('');
  const [isSaving, setIsSaving] = useState(false);

  const loadRunbook = useCallback(async () => {
    if (!runbookId) {
      setRunbook(null);
      setAuthError(null);
      return;
    }
    setIsLoading(true);
    setAuthError(null);
    try {
      const data = await fetchRunbookById(runbookId);
      if (activeTeamId && data.team_id !== activeTeamId) {
        setAuthError('Unauthorized: You are not authorized to view runbooks belonging to another team.');
        setRunbook(null);
      } else {
        setRunbook(data);
        setEditTitle(data.title);
        setEditContent(data.content);
      }
    } catch {
      setRunbook(null);
    } finally {
      setIsLoading(false);
    }
  }, [runbookId, activeTeamId]);

  useEffect(() => {
    loadRunbook();
  }, [loadRunbook]);

  const handleSave = async () => {
    if (!runbookId) return;
    setIsSaving(true);
    try {
      const updated = await updateRunbook(runbookId, {
        title: editTitle,
        content: editContent,
      });
      setRunbook(updated);
      setIsEditing(false);
      if (onRunbookUpdated) onRunbookUpdated();
    } catch (err) {
      console.error('Failed to update runbook', err);
    } finally {
      setIsSaving(false);
    }
  };

  const handleDeprecate = async () => {
    if (!runbookId) return;
    setIsSaving(true);
    try {
      const updated = await deprecateRunbook(runbookId);
      setRunbook(updated);
      if (onRunbookUpdated) onRunbookUpdated();
    } catch (err) {
      console.error('Failed to deprecate runbook', err);
    } finally {
      setIsSaving(false);
    }
  };

  return {
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
    onBack,
  };
}
