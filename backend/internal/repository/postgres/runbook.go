package postgres

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/WillieBam/support_copilot/backend/types"
	"github.com/WillieBam/support_copilot/backend/types/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (t *teamRepository) CreateRunbook(ctx context.Context, runbook *models.Runbook) error {
	return t.db.WithContext(ctx).Create(runbook).Error
}

func (t *teamRepository) UpdateRunbook(ctx context.Context, runbookID uuid.UUID, title, content string, log *models.RunbookLog) (*models.Runbook, error) {
	var rb models.Runbook
	err := t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ?", runbookID).First(&rb).Error; err != nil {
			return err
		}
		if log != nil {
			log.RunbookID = rb.ID
			log.IncidentID = rb.IncidentID
			log.TeamID = rb.TeamID
			if err := tx.Create(log).Error; err != nil {
				return err
			}
		}
		if err := tx.Model(&rb).Updates(map[string]interface{}{
			"title":      title,
			"content":    content,
			"updated_at": time.Now(),
		}).Error; err != nil {
			return err
		}
		rb.Title = title
		rb.Content = content
		return nil
	})
	if err != nil {
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

// GetRunbookLogs returns version history logs for a runbook ordered by version desc
func (t *teamRepository) GetRunbookLogs(ctx context.Context, runbookID uuid.UUID) ([]models.RunbookLog, error) {
	var logs []models.RunbookLog
	if err := t.db.WithContext(ctx).Where("runbook_id = ?", runbookID).Order("version DESC").Find(&logs).Error; err != nil {
		return nil, err
	}
	if logs == nil {
		logs = []models.RunbookLog{}
	}
	return logs, nil
}

// GetIncidentContext fetches a TeamIncident with its full status history (ASC order)
// plus all associated alerts via the root IncidentID — used by the MCP KB incident context endpoint
func (t *teamRepository) GetIncidentContext(ctx context.Context, teamIncidentID uuid.UUID) (*models.TeamIncident, []models.Alert, error) {
	var rows []types.TeamIncidentWithHistoryRow
	rawSQL := `SELECT 
		i.id AS incident_id, i.incident_number, i.team_id, i.created_by, i.title,
		i.status, i.details, i.created_at, i.assigned_at, i.resolved_at,
		h.id AS history_id, h.updated_by AS history_updated_by, h.title AS history_title,
		h.new_status AS history_new_status, h.previous_status AS history_previous_status,
		h.details AS history_details, h.updated_at AS history_updated_at
	FROM team_incidents i
	LEFT JOIN incident_status_histories h ON h.team_incident_id = i.id
	WHERE i.id = ?
	ORDER BY h.updated_at ASC`

	if err := t.db.WithContext(ctx).Raw(rawSQL, teamIncidentID).Scan(&rows).Error; err != nil {
		return nil, nil, err
	}
	if len(rows) == 0 {
		return nil, nil, gorm.ErrRecordNotFound
	}

	incidents := mapIncidentRows(rows)
	if len(incidents) == 0 {
		return nil, nil, gorm.ErrRecordNotFound
	}
	incident := &incidents[0]

	var alerts []models.Alert
	alertQuery := `SELECT DISTINCT a.*
	FROM alerts a
	LEFT JOIN alert_incidents ai ON ai.alert_id = a.id
	WHERE a.incident_id = ? OR ai.incident_id = ?
	ORDER BY a.received_at DESC`

	if err := t.db.WithContext(ctx).Raw(alertQuery, incident.ID, incident.ID).Scan(&alerts).Error; err != nil {
		return nil, nil, err
	}
	if alerts == nil {
		alerts = []models.Alert{}
	}
	return incident, alerts, nil
}

func (t *teamRepository) GetIncidentContextByIDOrNumber(ctx context.Context, idOrNumber string) (*models.TeamIncident, []models.Alert, error) {
	clean := strings.TrimSpace(idOrNumber)
	if clean == "" {
		return nil, nil, errors.New("incident ID or number is required")
	}

	var rows []types.TeamIncidentWithHistoryRow
	var rawSQL string
	var args []interface{}

	if parsedUUID, err := uuid.Parse(clean); err == nil {
		rawSQL = `SELECT 
			i.id AS incident_id, i.incident_number, i.team_id, i.created_by, i.title,
			i.status, i.details, i.created_at, i.assigned_at, i.resolved_at,
			h.id AS history_id, h.updated_by AS history_updated_by, h.title AS history_title,
			h.new_status AS history_new_status, h.previous_status AS history_previous_status,
			h.details AS history_details, h.updated_at AS history_updated_at
		FROM team_incidents i
		LEFT JOIN incident_status_histories h ON h.team_incident_id = i.id
		WHERE i.id = ? OR LOWER(i.incident_number) = LOWER(?)
		ORDER BY h.updated_at ASC`
		args = append(args, parsedUUID, clean)
	} else {
		rawSQL = `SELECT 
			i.id AS incident_id, i.incident_number, i.team_id, i.created_by, i.title,
			i.status, i.details, i.created_at, i.assigned_at, i.resolved_at,
			h.id AS history_id, h.updated_by AS history_updated_by, h.title AS history_title,
			h.new_status AS history_new_status, h.previous_status AS history_previous_status,
			h.details AS history_details, h.updated_at AS history_updated_at
		FROM team_incidents i
		LEFT JOIN incident_status_histories h ON h.team_incident_id = i.id
		WHERE LOWER(i.incident_number) = LOWER(?)
		ORDER BY h.updated_at ASC`
		args = append(args, clean)
	}

	if err := t.db.WithContext(ctx).Raw(rawSQL, args...).Scan(&rows).Error; err != nil {
		return nil, nil, err
	}
	if len(rows) == 0 {
		return nil, nil, gorm.ErrRecordNotFound
	}

	incidents := mapIncidentRows(rows)
	if len(incidents) == 0 {
		return nil, nil, gorm.ErrRecordNotFound
	}
	incident := &incidents[0]

	var alerts []models.Alert
	alertQuery := `SELECT DISTINCT a.*
	FROM alerts a
	LEFT JOIN alert_incidents ai ON ai.alert_id = a.id
	WHERE a.incident_id = ? OR ai.incident_id = ?
	ORDER BY a.received_at DESC`

	if err := t.db.WithContext(ctx).Raw(alertQuery, incident.ID, incident.ID).Scan(&alerts).Error; err != nil {
		return nil, nil, err
	}
	if alerts == nil {
		alerts = []models.Alert{}
	}
	return incident, alerts, nil
}
