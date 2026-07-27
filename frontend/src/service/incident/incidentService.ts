import apiClient from '@/service/apiClient';
import type { TeamIncident, CreateIncidentPayload, UpdateIncidentPayload } from '@/types/incident';

export const fetchTeamIncidents = async (teamId: string): Promise<TeamIncident[]> => {
    const response = await apiClient.get<TeamIncident[]>(`/api/teams/${teamId}/incidents`);
    return response.data || [];
};

export const fetchIncidentById = async (incidentId: string): Promise<TeamIncident> => {
    const response = await apiClient.get<TeamIncident>(`/api/incidents/${incidentId}`);
    return response.data;
};

export const createTeamIncident = async (teamId: string, payload: CreateIncidentPayload): Promise<TeamIncident> => {
    const response = await apiClient.post<TeamIncident>(`/api/teams/${teamId}/incidents`, payload);
    return response.data;
};

export const updateIncidentStatus = async (incidentId: string, payload: UpdateIncidentPayload): Promise<TeamIncident> => {
    const response = await apiClient.put<TeamIncident>(`/api/incidents/${incidentId}`, payload);
    return response.data;
};
