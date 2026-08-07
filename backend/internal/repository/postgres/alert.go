package postgres

import (
	"context"
	"errors"

	"github.com/WillieBam/support_copilot/backend/internal/interfaces"
	"github.com/WillieBam/support_copilot/backend/types/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type alertRepository struct {
	db *gorm.DB
}

func NewAlertRepository(db *gorm.DB) interfaces.IAlertRepository {
	return &alertRepository{db: db}
}

func (a *alertRepository) StoreAlert(ctx context.Context, alert *models.Alert) error {
	result := a.db.WithContext(ctx).Create(&alert)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (a *alertRepository) RetrieveAlertbyID(ctx context.Context, id uuid.UUID) (*models.Alert, error) {
	var alert models.Alert
	if err := a.db.WithContext(ctx).First(&alert, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, errors.New("Internal Server Error")
	}
	return &alert, nil
}

// updatealertincidentid links an alert to a specific incident id
func (a *alertRepository) UpdateAlertIncidentID(ctx context.Context, alertID, incidentID uuid.UUID) error {
	return a.db.WithContext(ctx).
		Model(&models.Alert{}).
		Where("id = ?", alertID).
		Update("incident_id", incidentID).Error
}

func (a *alertRepository) ListAlerts(ctx context.Context, limit int) ([]*models.Alert, error) {
	var alerts []*models.Alert
	err := a.db.WithContext(ctx).Order("received_at DESC").Limit(limit).Find(&alerts).Error
	return alerts, err
}
