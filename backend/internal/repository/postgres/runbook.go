package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/WillieBam/support_copilot/backend/types/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (t *teamRepository) CreateRunbook(ctx context.Context, runbook *models.Runbook) error {
	return t.db.WithContext(ctx).Create(runbook).Error
}

func (t *teamRepository) UpdateRunbook(ctx context.Context, runbookID uuid.UUID, title, content string) (*models.Runbook, error) {
	var rb models.Runbook
	if err := t.db.WithContext(ctx).Where("id = ?", runbookID).First(&rb).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, err
	}
	if err := t.db.WithContext(ctx).Model(&rb).Updates(map[string]interface{}{
		"title":      title,
		"content":    content,
		"updated_at": time.Now(),
	}).Error; err != nil {
		return nil, err
	}
	return &rb, nil
}

func (t *teamRepository) DeprecateRunbook(ctx context.Context, runbookID uuid.UUID) (*models.Runbook, error) {
	var rb models.Runbook
	if err := t.db.WithContext(ctx).Where("id = ?", runbookID).First(&rb).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, err
	}
	if err := t.db.WithContext(ctx).Model(&rb).Updates(map[string]interface{}{
		"status":     "deprecated",
		"updated_at": time.Now(),
	}).Error; err != nil {
		return nil, err
	}
	rb.Status = "deprecated"
	return &rb, nil
}

func (t *teamRepository) GetRunbookByID(ctx context.Context, runbookID uuid.UUID) (*models.Runbook, error) {
	var rb models.Runbook
	if err := t.db.WithContext(ctx).Where("id = ?", runbookID).First(&rb).Error; err != nil {
		return nil, err
	}
	return &rb, nil
}

func (t *teamRepository) ListRunbooks(ctx context.Context, teamID uuid.UUID, status string) ([]models.Runbook, error) {
	var runbooks []models.Runbook
	query := t.db.WithContext(ctx).Where("team_id = ?", teamID)
	if status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}
	if err := query.Order("created_at DESC").Find(&runbooks).Error; err != nil {
		return nil, err
	}
	if runbooks == nil {
		runbooks = []models.Runbook{}
	}
	return runbooks, nil
}

func (t *teamRepository) GetRunbooksByIncidentID(ctx context.Context, incidentID uuid.UUID) ([]models.Runbook, error) {
	var runbooks []models.Runbook
	if err := t.db.WithContext(ctx).
		Where("incident_id = ? AND status = 'active'", incidentID).
		Order("created_at DESC").
		Find(&runbooks).Error; err != nil {
		return nil, err
	}
	if runbooks == nil {
		runbooks = []models.Runbook{}
	}
	return runbooks, nil
}

// GetIncidentContext fetches a TeamIncident with its full status history (ASC order)
// plus all associated alerts via the root IncidentID — used by the MCP KB incident context endpoint.
func (t *teamRepository) GetIncidentContext(ctx context.Context, teamIncidentID uuid.UUID) (*models.TeamIncident, []models.Alert, error) {
	var incident models.TeamIncident
	if err := t.db.WithContext(ctx).
		Preload("History", func(db *gorm.DB) *gorm.DB {
			return db.Order("updated_at ASC")
		}).
		Where("id = ?", teamIncidentID).
		First(&incident).Error; err != nil {
		return nil, nil, err
	}

	var alerts []models.Alert
	if err := t.db.WithContext(ctx).
		Where("incident_id = ?", incident.ID).
		Order("received_at DESC").
		Find(&alerts).Error; err != nil {
		return nil, nil, err
	}
	if alerts == nil {
		alerts = []models.Alert{}
	}
	return &incident, alerts, nil
}
