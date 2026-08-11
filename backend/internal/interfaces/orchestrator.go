package interfaces

import (
	"context"

	"github.com/WillieBam/support_copilot/backend/types/responses"
)

// iorchestratorservice defines execution handlers for mcp tools
type IOrchestratorService interface {
	ExecuteListAlertsRaw(ctx context.Context) (string, error)
	ExecuteValidateAlert(ctx context.Context, alertID string) (*responses.CombinedValidationResult, error)
	ExecuteValidateAlertRaw(ctx context.Context, rawArgs string) (string, error)
	ExecuteGetIncidentRaw(ctx context.Context, rawArgs string) (string, error)
	ExecuteListIncidentsRaw(ctx context.Context, rawArgs string) (string, error)
	ExecuteCreateRunbookRaw(ctx context.Context, rawArgs string) (string, error)
	ExecuteUpdateRunbookRaw(ctx context.Context, rawArgs string) (string, error)
	ExecuteDeprecateRunbookRaw(ctx context.Context, rawArgs string) (string, error)
	ExecuteGetRunbookRaw(ctx context.Context, rawArgs string) (string, error)
	ExecuteListRunbooksRaw(ctx context.Context, rawArgs string) (string, error)
	ExecuteLinkAlertToIncidentRaw(ctx context.Context, rawArgs string) (string, error)
}
