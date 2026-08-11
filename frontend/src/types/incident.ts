export interface IncidentStatusHistory {
    id: string;
    team_incident_id: string;
    updated_by: string;
    title: string;
    new_status: string;
    previous_status: string;
    details: string;
    updated_at: string;
}

export interface TeamIncident {
    id: string;
    team_id: string;
    created_by: string;
    title: string;
    status: string;
    details: string;
    created_at: string;
    assigned_at: string;
    resolved_at?: string | null;
    history?: IncidentStatusHistory[];
}

export interface CreateIncidentPayload {
    title: string;
    status?: string;
    details?: string;
}

export interface UpdateIncidentPayload {
    status: string;
    title?: string;
    details?: string;
}
