import apiClient from '@/service/apiClient';
import type {UserWithTeams, AddMemberPayload, TeamMember, Team} from '@/types/team';
import type {UserSearchResult} from '@/types/user';

export const fetchUserTeams = async (): Promise<UserWithTeams> => {
    const response = await apiClient.get<UserWithTeams>('/api/teams/me');
    return response.data;
}

export const addTeamMember = async (teamId: string, payload: AddMemberPayload): Promise<void> => {
    await apiClient.post(`/api/teams/${teamId}/members`, payload)
}

export const createTeam = async (teamName: string): Promise<void> => {
    await apiClient.post('/api/teams', { team_name: teamName })
}

export const fetchTeamMembers = async (teamId: string): Promise<TeamMember[]> => {
    const response = await apiClient.get<TeamMember[]>(`/api/teams/${teamId}/members`);
    return response.data;
}

export const removeTeamMember = async (teamId: string, userId: string): Promise<void> => {
    await apiClient.delete(`/api/teams/${teamId}/members/${userId}`);
}

export const searchUsers = async (query: string): Promise<UserSearchResult[]> => {
    const response = await apiClient.get<UserSearchResult[]>('/api/users/search', {
        params: { q: query },
    });
    return response.data;
}

// deleteTeam removes a team permanently
export const deleteTeam = async (teamId: string): Promise<void> => {
    await apiClient.delete(`/api/admin/teams/${teamId}`);
}

// deactivateUser deactivates a user account (super_admin only)
export const deactivateUser = async (userId: string): Promise<void> => {
    await apiClient.post(`/api/admin/users/${userId}/deactivate`);
}

// fetchAllTeams retrieves all system teams for super_admin
export const fetchAllTeams = async (): Promise<Team[]> => {
    const response = await apiClient.get<Team[]>('/api/admin/teams');
    return response.data || [];
}