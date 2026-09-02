package interfaces

import (
	"context"

	"github.com/WillieBam/support_copilot/backend/types/models"
	"github.com/google/uuid"
)

type IAlertRepository interface {
	StoreAlert(ctx context.Context, alert *models.Alert) error
	RetrieveAlertbyID(ctx context.Context, id string) (*models.Alert, error)
	UpdateAlertIncidentID(ctx context.Context, alertID, incidentID uuid.UUID) error
	UnlinkAlertFromIncident(ctx context.Context, alertID, incidentID uuid.UUID) error
	ListAlertsForIncident(ctx context.Context, incidentID uuid.UUID) ([]*models.Alert, error)
	ListAlerts(ctx context.Context, limit int) ([]*models.Alert, error)
}
