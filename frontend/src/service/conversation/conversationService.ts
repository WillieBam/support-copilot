import apiClient from '@/service/apiClient';
import type { Conversation, ConversationMessage } from '@/types/conversation';

export const fetchTeamConversations = async (teamId: string, limit?: number): Promise<Conversation[]> => {
  const response = await apiClient.get<Conversation[]>(`/api/teams/${teamId}/conversations`, {
    params: limit ? { limit } : undefined,
  });
  return response.data;
};

export const fetchConversationMessages = async (convId: string): Promise<ConversationMessage[]> => {
  const response = await apiClient.get<ConversationMessage[]>(`/api/conversations/${convId}/messages`);
  return response.data;
};
