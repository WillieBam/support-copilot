package postgres

import (
	"context"

	"github.com/WillieBam/support_copilot/backend/internal/interfaces"
	"github.com/WillieBam/support_copilot/backend/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type dashboardRepository struct {
	db *gorm.DB
}

// NewDashboardRepository creates a postgres-backed IDashboardRepository
func NewDashboardRepository(db *gorm.DB) interfaces.IDashboardRepository {
	return &dashboardRepository{db: db}
}

// GetIncidentTrend aggregates incident counts grouped by date_trunc time bucket and status
func (r *dashboardRepository) GetIncidentTrend(ctx context.Context, teamID uuid.UUID, timeframe string) ([]types.IncidentTrendPoint, error) {
	var results []types.IncidentTrendPoint
	sql := `
		SELECT
			DATE_TRUNC(?, created_at)::text AS time_bucket,
			status,
			COUNT(*) AS count
		FROM team_incidents
		WHERE team_id = ?
		GROUP BY time_bucket, status
		ORDER BY time_bucket ASC
	`
	err := r.db.WithContext(ctx).Raw(sql, timeframe, teamID).Scan(&results).Error
	return results, err
}

// GetMTTRStats returns the average resolution time and total resolved count for the team
func (r *dashboardRepository) GetMTTRStats(ctx context.Context, teamID uuid.UUID) (float64, int, error) {
	type rawResult struct {
		AvgMinutes float64
		Total      int
	}
	var res rawResult
	sql := `
		SELECT
			COALESCE(AVG(EXTRACT(EPOCH FROM (resolved_at - created_at)) / 60), 0) AS avg_minutes,
			COUNT(*) AS total
		FROM team_incidents
		WHERE team_id = ?
		  AND status = 'RESOLVED'
		  AND resolved_at IS NOT NULL
	`
	err := r.db.WithContext(ctx).Raw(sql, teamID).Scan(&res).Error
	return res.AvgMinutes, res.Total, err
}

// GetBreachedIncidents fetches resolved incidents where resolution time exceeded the sla target
func (r *dashboardRepository) GetBreachedIncidents(ctx context.Context, teamID uuid.UUID, slaTargetMinutes int, limit, offset int) ([]types.BreachedIncident, error) {
	var results []types.BreachedIncident
	sql := `
		SELECT
			id::text,
			title,
			created_at,
			resolved_at,
			EXTRACT(EPOCH FROM (resolved_at - created_at)) / 60 AS duration_minutes
		FROM team_incidents
		WHERE team_id = ?
		  AND status = 'RESOLVED'
		  AND resolved_at IS NOT NULL
		  AND EXTRACT(EPOCH FROM (resolved_at - created_at)) / 60 > ?
		ORDER BY duration_minutes DESC
		LIMIT ? OFFSET ?
	`
	err := r.db.WithContext(ctx).Raw(sql, teamID, slaTargetMinutes, limit, offset).Scan(&results).Error
	return results, err
}

// CountBreachedIncidents returns the total number of sla-breached resolved incidents
func (r *dashboardRepository) CountBreachedIncidents(ctx context.Context, teamID uuid.UUID, slaTargetMinutes int) (int, error) {
	var count int
	sql := `
		SELECT COUNT(*) FROM team_incidents
		WHERE team_id = ?
		  AND status = 'RESOLVED'
		  AND resolved_at IS NOT NULL
		  AND EXTRACT(EPOCH FROM (resolved_at - created_at)) / 60 > ?
	`
	err := r.db.WithContext(ctx).Raw(sql, teamID, slaTargetMinutes).Scan(&count).Error
	return count, err
}

// GetAllTeamsIncidentTrend aggregates incident counts across all teams
func (r *dashboardRepository) GetAllTeamsIncidentTrend(ctx context.Context, timeframe string) ([]types.IncidentTrendPoint, error) {
	var results []types.IncidentTrendPoint
	sql := `
		SELECT
			DATE_TRUNC(?, created_at)::text AS time_bucket,
			status,
			COUNT(*) AS count
		FROM team_incidents
		GROUP BY time_bucket, status
		ORDER BY time_bucket ASC
	`
	err := r.db.WithContext(ctx).Raw(sql, timeframe).Scan(&results).Error
	return results, err
}

// GetAllTeamsMTTRStats returns resolution time and total resolved count across all teams
func (r *dashboardRepository) GetAllTeamsMTTRStats(ctx context.Context) (float64, int, error) {
	type rawResult struct {
		AvgMinutes float64
		Total      int
	}
	var res rawResult
	sql := `
		SELECT
			COALESCE(AVG(EXTRACT(EPOCH FROM (resolved_at - created_at)) / 60), 0) AS avg_minutes,
			COUNT(*) AS total
		FROM team_incidents
		WHERE status = 'RESOLVED'
		  AND resolved_at IS NOT NULL
	`
	err := r.db.WithContext(ctx).Raw(sql).Scan(&res).Error
	return res.AvgMinutes, res.Total, err
}

// GetAllTeamsBreachedIncidents fetches resolved incidents that exceeded sla target across all teams
func (r *dashboardRepository) GetAllTeamsBreachedIncidents(ctx context.Context, slaTargetMinutes int, limit, offset int) ([]types.BreachedIncident, error) {
	var results []types.BreachedIncident
	sql := `
		SELECT
			id::text,
			title,
			created_at,
			resolved_at,
			EXTRACT(EPOCH FROM (resolved_at - created_at)) / 60 AS duration_minutes
		FROM team_incidents
		WHERE status = 'RESOLVED'
		  AND resolved_at IS NOT NULL
		  AND EXTRACT(EPOCH FROM (resolved_at - created_at)) / 60 > ?
		ORDER BY duration_minutes DESC
		LIMIT ? OFFSET ?
	`
	err := r.db.WithContext(ctx).Raw(sql, slaTargetMinutes, limit, offset).Scan(&results).Error
	return results, err
}

// CountAllTeamsBreachedIncidents returns total count of sla-breached resolved incidents across all teams
func (r *dashboardRepository) CountAllTeamsBreachedIncidents(ctx context.Context, slaTargetMinutes int) (int, error) {
	var count int
	sql := `
		SELECT COUNT(*) FROM team_incidents
		WHERE status = 'RESOLVED'
		  AND resolved_at IS NOT NULL
		  AND EXTRACT(EPOCH FROM (resolved_at - created_at)) / 60 > ?
	`
	err := r.db.WithContext(ctx).Raw(sql, slaTargetMinutes).Scan(&count).Error
	return count, err
}
