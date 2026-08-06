import apiClient from '@/service/apiClient';
import type { IncidentTrendPoint, MTTRResult, BreachedIncident } from '@/types/analytics';

// fetchIncidentTrend retrieves incident frequency trends grouped by timeframe
export const fetchIncidentTrend = async (
  teamId: string,
  timeframe: 'day' | 'month' | 'year'
): Promise<IncidentTrendPoint[]> => {
  const response = await apiClient.get<IncidentTrendPoint[]>('/api/dashboard/incidents/trend', {
    params: { team_id: teamId, timeframe },
  });
  return response.data || [];
};

// fetchMTTR calculates mttr metrics and sla compliance for the selected team
export const fetchMTTR = async (
  teamId: string,
  slaTargetMinutes: number
): Promise<MTTRResult> => {
  const response = await apiClient.get<MTTRResult>('/api/dashboard/mttr', {
    params: { team_id: teamId, sla_target_minutes: slaTargetMinutes },
  });
  return response.data;
};

// fetchBreachedIncidents lists resolved incidents that breached or exceeded the specified sla threshold
export const fetchBreachedIncidents = async (
  teamId: string,
  slaTargetMinutes: number
): Promise<BreachedIncident[]> => {
  const response = await apiClient.get<BreachedIncident[]>('/api/dashboard/incidents/breached', {
    params: { team_id: teamId, sla_target_minutes: slaTargetMinutes },
  });
  return response.data || [];
};
