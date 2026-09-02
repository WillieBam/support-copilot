package postgres

import (
	"context"
	"errors"

	"github.com/WillieBam/support_copilot/backend/internal/interfaces"
	"github.com/WillieBam/support_copilot/backend/types/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

// RetrieveAlertbyID fetches alert by primary key or alert info id
func (a *alertRepository) RetrieveAlertbyID(ctx context.Context, id string) (*models.Alert, error) {
	var alert models.Alert
	if parsedUUID, err := uuid.Parse(id); err == nil {
		if err := a.db.WithContext(ctx).First(&alert, "id = ? OR alert_info LIKE ?", parsedUUID, "%\"id\":\""+id+"\"%").Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, err
			}
			return nil, errors.New("Internal Server Error")
		}
		return &alert, nil
	}
	if err := a.db.WithContext(ctx).First(&alert, "alert_info LIKE ?", "%\"id\":\""+id+"\"%").Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, errors.New("Internal Server Error")
	}
	return &alert, nil
}

func (a *alertRepository) UpdateAlertIncidentID(ctx context.Context, alertID, incidentID uuid.UUID) error {
	joinRecord := models.AlertIncident{
		AlertID:    alertID,
		IncidentID: incidentID,
		LinkedBy:   "human_ui",
	}
	if err := a.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&joinRecord).Error; err != nil {
		return err
	}

	return a.db.WithContext(ctx).
		Model(&models.Alert{}).
		Where("id = ? AND incident_id IS NULL", alertID).
		Update("incident_id", incidentID).Error
}

func (a *alertRepository) UnlinkAlertFromIncident(ctx context.Context, alertID, incidentID uuid.UUID) error {
	if err := a.db.WithContext(ctx).
		Where("alert_id = ? AND incident_id = ?", alertID, incidentID).
		Delete(&models.AlertIncident{}).Error; err != nil {
		return err
	}

	return a.db.WithContext(ctx).
		Model(&models.Alert{}).
		Where("id = ? AND incident_id = ?", alertID, incidentID).
		Update("incident_id", nil).Error
}

func (a *alertRepository) ListAlertsForIncident(ctx context.Context, incidentID uuid.UUID) ([]*models.Alert, error) {
	var alerts []*models.Alert
	query := `SELECT DISTINCT a.*
	FROM alerts a
	LEFT JOIN alert_incidents ai ON ai.alert_id = a.id
	WHERE a.incident_id = ? OR ai.incident_id = ?
	ORDER BY a.received_at DESC`

	if err := a.db.WithContext(ctx).Raw(query, incidentID, incidentID).Scan(&alerts).Error; err != nil {
		return nil, err
	}
	if alerts == nil {
		alerts = []*models.Alert{}
	}
	return alerts, nil
}

func (a *alertRepository) ListAlerts(ctx context.Context, limit int) ([]*models.Alert, error) {
	var alerts []*models.Alert
	err := a.db.WithContext(ctx).Order("received_at DESC").Limit(limit).Find(&alerts).Error
	return alerts, err
}

