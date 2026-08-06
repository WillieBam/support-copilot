package types

import "time"

// IncidentTrendPoint is a single time-bucket data point returned by the trend query
type IncidentTrendPoint struct {
	TimeBucket string `json:"time_bucket"`
	Status     string `json:"status"`
	Count      int    `json:"count"`
}

// MTTRResult holds mean time to resolve and sla compliance metrics
type MTTRResult struct {
	MTTRMinutes    float64 `json:"mttr_minutes"`
	TotalResolved  int     `json:"total_resolved"`
	SLABreaches    int     `json:"sla_breaches"`
	ComplianceRate float64 `json:"compliance_rate"`
}

// BreachedIncident is an enriched projection for incidents that exceeded the sla target
type BreachedIncident struct {
	ID              string     `json:"id"`
	Title           string     `json:"title"`
	CreatedAt       time.Time  `json:"created_at"`
	ResolvedAt      *time.Time `json:"resolved_at"`
	DurationMinutes float64    `json:"duration_minutes"`
}
