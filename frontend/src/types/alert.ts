export interface AlertItem {
  id: string;
  incident_id?: string | null;
  alert_info: string;
  resource_info: string;
  metrics?: string;
  business_context?: string;
  metadata?: string;
  received_at: string;
}

export interface ParsedAlertDisplay {
  id: string;
  incidentId?: string | null;
  serviceName: string;
  severity: string;
  message: string;
  receivedAt: string;
}

export interface LinkAlertsPayload {
  alert_ids?: string[];
  alert_id?: string;
}
