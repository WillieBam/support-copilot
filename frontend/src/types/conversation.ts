export interface ConversationUser {
  id: string;
  email: string;
  display_name?: string;
}

export interface Conversation {
  id: string;
  team_id: string;
  user_id: string;
  user?: ConversationUser;
  title: string;
  created_at: string;
}

export interface ConversationMessage {
  id: string;
  conversation_id: string;
  sender: 'user' | 'assistant';
  content: string;
  created_at: string;
}
