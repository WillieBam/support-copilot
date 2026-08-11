package service

import (
	"context"
	"log/slog"

	"github.com/WillieBam/support_copilot/backend/internal/interfaces"
	"github.com/WillieBam/support_copilot/backend/types"
	customErrors "github.com/WillieBam/support_copilot/backend/utils/errors"
	"github.com/google/uuid"
)

type dashboardService struct {
	dashboardRepo interfaces.IDashboardRepository
	teamRepo      interfaces.ITeamRepository
}

// NewDashboardService creates a dashboardService backed by the given repositories
func NewDashboardService(dashboardRepo interfaces.IDashboardRepository, teamRepo interfaces.ITeamRepository) interfaces.IDashboardService {
	return &dashboardService{
		dashboardRepo: dashboardRepo,
		teamRepo:      teamRepo,
	}
}

// isSuperAdmin checks if the user scope grants unrestricted team access
func isSuperAdmin(userScope string) bool {
	return userScope == "super_admin"
}

// validateAccess ensures the requester is a member of the team, unless they are a super_admin
func (s *dashboardService) validateAccess(ctx context.Context, requesterID, teamID uuid.UUID, userScope string) error {
	if isSuperAdmin(userScope) {
		return nil
	}
	_, err := s.teamRepo.GetMemberRole(ctx, teamID, requesterID)
	if err != nil {
		return customErrors.ErrDashboardUnauthorized
	}
	return nil
}

// GetIncidentTrend validates the timeframe param and membership then returns bucketed trend data
func (s *dashboardService) GetIncidentTrend(ctx context.Context, requesterID, teamID uuid.UUID, userScope, timeframe string) ([]types.IncidentTrendPoint, error) {
	switch timeframe {
	case "day", "month", "year":
	default:
		return nil, customErrors.ErrInvalidTimeframe
	}
	if err := s.validateAccess(ctx, requesterID, teamID, userScope); err != nil {
		return nil, err
	}
	slog.InfoContext(ctx, "[dashboard-svc] GetIncidentTrend", "team_id", teamID, "timeframe", timeframe)
	results, err := s.dashboardRepo.GetIncidentTrend(ctx, teamID, timeframe)
	if err != nil {
		slog.ErrorContext(ctx, "[dashboard-svc] GetIncidentTrend: repository error", "team_id", teamID, "error", err)
		return nil, err
	}
	return results, nil
}

// GetMTTR validates membership then computes mttr and sla compliance metrics
func (s *dashboardService) GetMTTR(ctx context.Context, requesterID, teamID uuid.UUID, userScope string, slaTargetMinutes int) (*types.MTTRResult, error) {
	if slaTargetMinutes <= 0 {
		return nil, customErrors.ErrInvalidSLATarget
	}
	if err := s.validateAccess(ctx, requesterID, teamID, userScope); err != nil {
		return nil, err
	}
	slog.InfoContext(ctx, "[dashboard-svc] GetMTTR", "team_id", teamID, "sla_target_minutes", slaTargetMinutes)

	avgMinutes, totalResolved, err := s.dashboardRepo.GetMTTRStats(ctx, teamID)
	if err != nil {
		slog.ErrorContext(ctx, "[dashboard-svc] GetMTTR: stats query failed", "team_id", teamID, "error", err)
		return nil, err
	}

	// count breaches in a dedicated query to avoid over-fetching rows
	breachCount, err := s.dashboardRepo.CountBreachedIncidents(ctx, teamID, slaTargetMinutes)
	if err != nil {
		slog.ErrorContext(ctx, "[dashboard-svc] GetMTTR: breach count query failed", "team_id", teamID, "error", err)
		return nil, err
	}

	var complianceRate float64
	if totalResolved > 0 {
		complianceRate = float64(totalResolved-breachCount) / float64(totalResolved) * 100
	}

	return &types.MTTRResult{
		MTTRMinutes:    avgMinutes,
		TotalResolved:  totalResolved,
		SLABreaches:    breachCount,
		ComplianceRate: complianceRate,
	}, nil
}

// GetBreachedIncidents validates membership then returns sla-breached incidents with pagination
func (s *dashboardService) GetBreachedIncidents(ctx context.Context, requesterID, teamID uuid.UUID, userScope string, slaTargetMinutes, limit, offset int) ([]types.BreachedIncident, error) {
	if slaTargetMinutes < 0 {
		return nil, customErrors.ErrInvalidSLATarget
	}
	if limit <= 0 {
		limit = 50
	}
	if err := s.validateAccess(ctx, requesterID, teamID, userScope); err != nil {
		return nil, err
	}
	slog.InfoContext(ctx, "[dashboard-svc] GetBreachedIncidents", "team_id", teamID, "sla_target_minutes", slaTargetMinutes, "limit", limit, "offset", offset)
	results, err := s.dashboardRepo.GetBreachedIncidents(ctx, teamID, slaTargetMinutes, limit, offset)
	if err != nil {
		slog.ErrorContext(ctx, "[dashboard-svc] GetBreachedIncidents: repository error", "team_id", teamID, "error", err)
		return nil, err
	}
	return results, nil
}

// GetAllTeamsIncidentTrend validates super_admin scope then returns combined trend data for all teams
func (s *dashboardService) GetAllTeamsIncidentTrend(ctx context.Context, userScope, timeframe string) ([]types.IncidentTrendPoint, error) {
	if !isSuperAdmin(userScope) {
		return nil, customErrors.ErrSuperAdminRequired
	}
	switch timeframe {
	case "day", "month", "year":
	default:
		return nil, customErrors.ErrInvalidTimeframe
	}
	slog.InfoContext(ctx, "[dashboard-svc] GetAllTeamsIncidentTrend", "timeframe", timeframe)
	results, err := s.dashboardRepo.GetAllTeamsIncidentTrend(ctx, timeframe)
	if err != nil {
		slog.ErrorContext(ctx, "[dashboard-svc] GetAllTeamsIncidentTrend: repository error", "error", err)
		return nil, err
	}
	return results, nil
}

// GetAllTeamsMTTR validates super_admin scope then computes combined mttr metrics for all teams
func (s *dashboardService) GetAllTeamsMTTR(ctx context.Context, userScope string, slaTargetMinutes int) (*types.MTTRResult, error) {
	if !isSuperAdmin(userScope) {
		return nil, customErrors.ErrSuperAdminRequired
	}
	if slaTargetMinutes <= 0 {
		return nil, customErrors.ErrInvalidSLATarget
	}
	slog.InfoContext(ctx, "[dashboard-svc] GetAllTeamsMTTR", "sla_target_minutes", slaTargetMinutes)

	avgMinutes, totalResolved, err := s.dashboardRepo.GetAllTeamsMTTRStats(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "[dashboard-svc] GetAllTeamsMTTR: stats query failed", "error", err)
		return nil, err
	}

	breachCount, err := s.dashboardRepo.CountAllTeamsBreachedIncidents(ctx, slaTargetMinutes)
	if err != nil {
		slog.ErrorContext(ctx, "[dashboard-svc] GetAllTeamsMTTR: breach count query failed", "error", err)
		return nil, err
	}

	var complianceRate float64
	if totalResolved > 0 {
		complianceRate = float64(totalResolved-breachCount) / float64(totalResolved) * 100
	}

	return &types.MTTRResult{
		MTTRMinutes:    avgMinutes,
		TotalResolved:  totalResolved,
		SLABreaches:    breachCount,
		ComplianceRate: complianceRate,
	}, nil
}

// GetAllTeamsBreachedIncidents validates super_admin scope then returns combined breached incidents for all teams
func (s *dashboardService) GetAllTeamsBreachedIncidents(ctx context.Context, userScope string, slaTargetMinutes, limit, offset int) ([]types.BreachedIncident, error) {
	if !isSuperAdmin(userScope) {
		return nil, customErrors.ErrSuperAdminRequired
	}
	if slaTargetMinutes < 0 {
		return nil, customErrors.ErrInvalidSLATarget
	}
	if limit <= 0 {
		limit = 50
	}
	slog.InfoContext(ctx, "[dashboard-svc] GetAllTeamsBreachedIncidents", "sla_target_minutes", slaTargetMinutes, "limit", limit, "offset", offset)
	results, err := s.dashboardRepo.GetAllTeamsBreachedIncidents(ctx, slaTargetMinutes, limit, offset)
	if err != nil {
		slog.ErrorContext(ctx, "[dashboard-svc] GetAllTeamsBreachedIncidents: repository error", "error", err)
		return nil, err
	}
	return results, nil
}
