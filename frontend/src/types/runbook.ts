export interface Runbook {
  id: string;
  team_id: string;
  incident_id?: string;
  title: string;
  content: string;
  status: 'active' | 'deprecated';
  created_at: string;
  updated_at: string;
}

export interface CreateRunbookPayload {
  incident_id?: string;
  title: string;
  content: string;
}

export interface UpdateRunbookPayload {
  title?: string;
  content?: string;
}
