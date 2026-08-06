export interface Runbook {
  id: string;
  team_id: string;
  incident_id?: string;
  created_by?: string;
  title: string;
  content: string;
  status: 'active' | 'deprecated';
  created_at: string;
  updated_at: string;
}

export interface RunbookLog {
  id: string;
  runbook_id: string;
  incident_id: string;
  team_id: string;
  updated_by: string;
  older_title: string;
  older_content: string;
  version: number;
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
