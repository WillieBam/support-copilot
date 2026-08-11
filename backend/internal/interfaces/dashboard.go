package interfaces

import (
	"context"

	"github.com/WillieBam/support_copilot/backend/types"
	"github.com/google/uuid"
)

// IDashboardRepository defines raw data access for the incident analytics dashboard
type IDashboardRepository interface {
	// GetIncidentTrend returns incident counts grouped by time bucket and status
	GetIncidentTrend(ctx context.Context, teamID uuid.UUID, timeframe string) ([]types.IncidentTrendPoint, error)
	// GetMTTRStats returns the avg resolution time in minutes and total resolved count for a team
	GetMTTRStats(ctx context.Context, teamID uuid.UUID) (avgMinutes float64, totalResolved int, err error)
	// GetBreachedIncidents returns incidents that exceeded the given sla target in minutes with pagination
	GetBreachedIncidents(ctx context.Context, teamID uuid.UUID, slaTargetMinutes int, limit, offset int) ([]types.BreachedIncident, error)
	// CountBreachedIncidents returns the total count of sla-breached incidents for mttr breach calculation
	CountBreachedIncidents(ctx context.Context, teamID uuid.UUID, slaTargetMinutes int) (int, error)
	// GetAllTeamsIncidentTrend returns incident counts across all teams grouped by time bucket and status
	GetAllTeamsIncidentTrend(ctx context.Context, timeframe string) ([]types.IncidentTrendPoint, error)
	// GetAllTeamsMTTRStats returns avg resolution time in minutes and total resolved count across all teams
	GetAllTeamsMTTRStats(ctx context.Context) (avgMinutes float64, totalResolved int, err error)
	// GetAllTeamsBreachedIncidents returns incidents that exceeded the given sla target across all teams
	GetAllTeamsBreachedIncidents(ctx context.Context, slaTargetMinutes int, limit, offset int) ([]types.BreachedIncident, error)
	// CountAllTeamsBreachedIncidents returns the total count of sla-breached incidents across all teams
	CountAllTeamsBreachedIncidents(ctx context.Context, slaTargetMinutes int) (int, error)
}

// IDashboardService defines business-level analytics operations for the incident dashboard
type IDashboardService interface {
	// GetIncidentTrend validates timeframe and membership then returns bucketed trend data
	GetIncidentTrend(ctx context.Context, requesterID, teamID uuid.UUID, userScope, timeframe string) ([]types.IncidentTrendPoint, error)
	// GetMTTR validates membership then computes mttr and sla compliance for the team
	GetMTTR(ctx context.Context, requesterID, teamID uuid.UUID, userScope string, slaTargetMinutes int) (*types.MTTRResult, error)
	// GetBreachedIncidents validates membership then returns sla-breached incidents with pagination
	GetBreachedIncidents(ctx context.Context, requesterID, teamID uuid.UUID, userScope string, slaTargetMinutes, limit, offset int) ([]types.BreachedIncident, error)
	// GetAllTeamsIncidentTrend returns combined trend data for all teams
	GetAllTeamsIncidentTrend(ctx context.Context, userScope, timeframe string) ([]types.IncidentTrendPoint, error)
	// GetAllTeamsMTTR returns combined mttr metrics for all teams
	GetAllTeamsMTTR(ctx context.Context, userScope string, slaTargetMinutes int) (*types.MTTRResult, error)
	// GetAllTeamsBreachedIncidents returns combined breached incidents for all teams
	GetAllTeamsBreachedIncidents(ctx context.Context, userScope string, slaTargetMinutes, limit, offset int) ([]types.BreachedIncident, error)
}
