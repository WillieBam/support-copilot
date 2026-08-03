import { useState } from 'react';
import { createRunbook } from '@/service/runbook/runbookService';

export function useCreateRunbookState(teamId: string, onSuccess: () => void, onClose: () => void) {
  const [title, setTitle] = useState('');
  const [incidentId, setIncidentId] = useState('');
  const [content, setContent] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!title.trim() || !content.trim()) {
      setError('Title and Content are required.');
      return;
    }

    setIsSubmitting(true);
    setError(null);
    try {
      await createRunbook(teamId, {
        title: title.trim(),
        incident_id: incidentId.trim() || undefined,
        content: content.trim(),
      });
      onSuccess();
      onClose();
    } catch (err: any) {
      setError(err?.response?.data?.error || 'Failed to create runbook.');
    } finally {
      setIsSubmitting(false);
    }
  };

  return {
    title,
    setTitle,
    incidentId,
    setIncidentId,
    content,
    setContent,
    isSubmitting,
    error,
    handleSubmit,
  };
}
