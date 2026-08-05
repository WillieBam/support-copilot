package command

import (
	"context"

	"github.com/WillieBam/support_copilot/backend/types"
	"github.com/google/uuid"
)

func WithTeamID(ctx context.Context, teamID uuid.UUID) context.Context {
	return context.WithValue(ctx, types.TeamIDContextKey, teamID)
}

// GetTeamID extracts team_id from context
func GetTeamID(ctx context.Context) (uuid.UUID, bool) {
	val, ok := ctx.Value(types.TeamIDContextKey).(uuid.UUID)
	return val, ok && val != uuid.Nil
}

// WithActiveIncidentID injects active_incident_id into context
func WithActiveIncidentID(ctx context.Context, incidentID uuid.UUID) context.
	Context {
	return context.WithValue(ctx, types.ActiveIncidentIDContextKey, incidentID)
}

// GetActiveIncidentID extracts active_incident_id from context
func GetActiveIncidentID(ctx context.Context) (uuid.UUID, bool) {
	val, ok := ctx.Value(types.ActiveIncidentIDContextKey).(uuid.UUID)
	return val, ok && val != uuid.Nil
}
