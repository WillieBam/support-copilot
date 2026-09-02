import apiClient from '@/service/apiClient';
import type { AlertItem, LinkAlertsPayload } from '@/types/alert';

export const fetchIncidentAlerts = async (incidentId: string): Promise<AlertItem[]> => {
  const response = await apiClient.get<AlertItem[]>(`/api/incidents/${incidentId}/alerts`);
  return response.data || [];
};

export const linkAlertsToIncident = async (incidentId: string, payload: LinkAlertsPayload): Promise<void> => {
  await apiClient.post(`/api/incidents/${incidentId}/alerts`, payload);
};

export const unlinkAlertFromIncident = async (incidentId: string, alertId: string): Promise<void> => {
  await apiClient.delete(`/api/incidents/${incidentId}/alerts/${alertId}`);
};

export const fetchRecentAlerts = async (limit = 50): Promise<AlertItem[]> => {
  const response = await apiClient.get<AlertItem[]>(`/api/alerts?limit=${limit}`);
  return response.data || [];
};
