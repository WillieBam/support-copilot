export interface IncidentTrendPoint {
  time_bucket: string;
  status: string;
  count: number;
}

export interface MTTRResult {
  mttr_minutes: number;
  total_resolved: number;
  sla_breaches: number;
  compliance_rate: number;
}

export interface BreachedIncident {
  id: string;
  title: string;
  created_at: string;
  resolved_at: string | null;
  duration_minutes: number;
}
