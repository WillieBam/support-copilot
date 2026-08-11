import apiClient from '@/service/apiClient';
import type { Runbook, RunbookLog, CreateRunbookPayload, UpdateRunbookPayload } from '@/types/runbook';

export async function fetchTeamRunbooks(teamId: string, status = 'active'): Promise<Runbook[]> {
  const response = await apiClient.get<Runbook[]>(`/api/teams/${teamId}/runbooks`, {
    params: { status },
  });
  return response.data || [];
}

export async function fetchRunbookById(id: string): Promise<Runbook> {
  const response = await apiClient.get<Runbook>(`/api/runbooks/${id}`);
  return response.data;
}

export async function fetchRunbookLogs(id: string): Promise<RunbookLog[]> {
  const response = await apiClient.get<RunbookLog[]>(`/api/runbooks/${id}/logs`);
  return response.data || [];
}

export async function createRunbook(teamId: string, payload: CreateRunbookPayload): Promise<Runbook> {
  const response = await apiClient.post<Runbook>(`/api/teams/${teamId}/runbooks`, payload);
  window.dispatchEvent(new CustomEvent('runbooks-updated'));
  return response.data;
}

export async function updateRunbook(id: string, payload: UpdateRunbookPayload): Promise<Runbook> {
  const response = await apiClient.patch<Runbook>(`/api/runbooks/${id}`, payload);
  window.dispatchEvent(new CustomEvent('runbooks-updated'));
  return response.data;
}

export async function deprecateRunbook(id: string): Promise<Runbook> {
  const response = await apiClient.patch<Runbook>(`/api/runbooks/${id}/deprecate`);
  window.dispatchEvent(new CustomEvent('runbooks-updated'));
  return response.data;
}
