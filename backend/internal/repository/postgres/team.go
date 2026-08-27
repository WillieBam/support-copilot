package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/WillieBam/support_copilot/backend/internal/interfaces"
	"github.com/WillieBam/support_copilot/backend/types"
	"github.com/WillieBam/support_copilot/backend/types/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type teamRepository struct {
	db *gorm.DB
}

func NewTeamRepository(db *gorm.DB) interfaces.ITeamRepository {
	return &teamRepository{db: db}
}

// CreateTeamWithOwner atomically creates a team and assigns the owner within a DB transaction.
func (t *teamRepository) CreateTeamWithOwner(ctx context.Context, team *models.Team, ownerID uuid.UUID) error {
	return t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(team).Error; err != nil {
			return err
		}
		member := models.TeamMember{
			ID:     uuid.New(),
			TeamID: team.ID,
			UserID: ownerID,
			Role:   "owner",
		}
		return tx.Create(&member).Error
	})
}

func (t *teamRepository) GetTeamByID(ctx context.Context, teamID uuid.UUID) (*models.Team, error) {
	var rows []types.TeamWithMemberRow
	rawSQL := `SELECT 
		t.id AS team_id, t.team_name, t.created_at AS team_created_at,
		tm.id AS member_id, tm.user_id AS member_user_id, tm.role AS member_role,
		u.email AS user_email, u.display_name AS user_display_name, u.scope AS user_scope
	FROM teams t
	LEFT JOIN team_members tm ON tm.team_id = t.id
	LEFT JOIN users u ON u.id = tm.user_id
	WHERE t.id = ?`

	if err := t.db.WithContext(ctx).Raw(rawSQL, teamID).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	first := rows[0]
	team := &models.Team{
		ID:        first.TeamID,
		TeamName:  first.TeamName,
		CreatedAt: first.TeamCreatedAt,
		Members:   []models.TeamMember{},
	}

	for _, r := range rows {
		if r.MemberID != nil && r.MemberUserID != nil {
			email := ""
			if r.UserEmail != nil {
				email = *r.UserEmail
			}
			displayName := ""
			if r.UserDisplayName != nil {
				displayName = *r.UserDisplayName
			}
			scope := ""
			if r.UserScope != nil {
				scope = *r.UserScope
			}
			role := "member"
			if r.MemberRole != nil {
				role = *r.MemberRole
			}
			team.Members = append(team.Members, models.TeamMember{
				ID:     *r.MemberID,
				TeamID: first.TeamID,
				UserID: *r.MemberUserID,
				Role:   role,
				User: &models.User{
					ID:          *r.MemberUserID,
					Email:       email,
					DisplayName: displayName,
					Scope:       scope,
				},
			})
		}
	}
	return team, nil
}

func (t *teamRepository) GetUserWithTeamsByID(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	var rows []types.UserWithTeamMembershipRow
	rawSQL := `SELECT 
		u.id AS user_id, u.firebase_uid, u.username, u.email, u.display_name,
		u.created_at AS user_created_at, u.deactivated_at, u.scope,
		tm.id AS membership_id, tm.team_id, tm.role AS membership_role,
		t.team_name, t.created_at AS team_created_at
	FROM users u
	LEFT JOIN team_members tm ON tm.user_id = u.id
	LEFT JOIN teams t ON t.id = tm.team_id
	WHERE u.id = ?`

	if err := t.db.WithContext(ctx).Raw(rawSQL, userID).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	first := rows[0]
	user := &models.User{
		ID:            first.UserID,
		FirebaseUID:   first.FirebaseUID,
		Username:      first.Username,
		Email:         first.Email,
		DisplayName:   first.DisplayName,
		CreatedAt:     first.UserCreatedAt,
		DeactivatedAt: first.DeactivatedAt,
		Scope:         first.Scope,
		Memberships:   []models.TeamMember{},
	}

	for _, r := range rows {
		if r.MembershipID != nil && r.TeamID != nil {
			teamName := ""
			if r.TeamName != nil {
				teamName = *r.TeamName
			}
			teamCreatedAt := time.Time{}
			if r.TeamCreatedAt != nil {
				teamCreatedAt = *r.TeamCreatedAt
			}
			role := "member"
			if r.MembershipRole != nil {
				role = *r.MembershipRole
			}
			user.Memberships = append(user.Memberships, models.TeamMember{
				ID:     *r.MembershipID,
				TeamID: *r.TeamID,
				UserID: first.UserID,
				Role:   role,
				Team: &models.Team{
					ID:        *r.TeamID,
					TeamName:  teamName,
					CreatedAt: teamCreatedAt,
				},
			})
		}
	}
	return user, nil
}

func (t *teamRepository) AddTeamMember(ctx context.Context, member *models.TeamMember) error {
	return t.db.WithContext(ctx).Create(member).Error
}

func (t *teamRepository) RemoveTeamMember(ctx context.Context, teamID, userID uuid.UUID) error {
	res := t.db.WithContext(ctx).Where("team_id = ? AND user_id = ?", teamID, userID).Delete(&models.TeamMember{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (t *teamRepository) DeleteTeam(ctx context.Context, teamID uuid.UUID) error {
	res := t.db.WithContext(ctx).Where("id = ?", teamID).Delete(&models.Team{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (t *teamRepository) GetMemberRole(ctx context.Context, teamID, userID uuid.UUID) (string, error) {
	var member models.TeamMember
	err := t.db.WithContext(ctx).Where("team_id = ? AND user_id = ?", teamID, userID).First(&member).Error
	if err != nil {
		return "", err
	}
	return member.Role, nil
}

func (t *teamRepository) ListTeamMembers(ctx context.Context, teamID uuid.UUID) ([]models.TeamMember, error) {
	var rows []types.TeamMemberWithUserRow
	rawSQL := `SELECT 
		tm.id, tm.team_id, tm.user_id, tm.role,
		u.email AS user_email, u.display_name AS user_display_name, u.scope AS user_scope,
		u.totp_enabled AS user_totp_enabled, u.created_at AS user_created_at
	FROM team_members tm
	JOIN users u ON u.id = tm.user_id
	WHERE tm.team_id = ?
	ORDER BY tm.id ASC`

	if err := t.db.WithContext(ctx).Raw(rawSQL, teamID).Scan(&rows).Error; err != nil {
		return nil, err
	}

	members := make([]models.TeamMember, len(rows))
	for i, r := range rows {
		members[i] = models.TeamMember{
			ID:     r.ID,
			TeamID: r.TeamID,
			UserID: r.UserID,
			Role:   r.Role,
			User: &models.User{
				ID:          r.UserID,
				Email:       r.UserEmail,
				DisplayName: r.UserDisplayName,
				Scope:       r.UserScope,
				TOTPEnabled: r.UserTotpEnabled,
				CreatedAt:   r.UserCreatedAt,
			},
		}
	}
	return members, nil
}

func (t *teamRepository) AssignTeamIncident(ctx context.Context, incident *models.TeamIncident) error {
	return t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if strings.TrimSpace(incident.IncidentNumber) == "" {
			var lastIncident models.TeamIncident
			var nextNum int64 = 101
			if err := tx.Where("incident_number IS NOT NULL AND incident_number != ''").Order("incident_number DESC").First(&lastIncident).Error; err == nil && lastIncident.IncidentNumber != "" {
				var parsedNum int64
				if _, err := fmt.Sscanf(lastIncident.IncidentNumber, "INC-%d", &parsedNum); err == nil && parsedNum >= 100 {
					nextNum = parsedNum + 1
				}
			} else {
				var count int64
				_ = tx.Model(&models.TeamIncident{}).Count(&count)
				nextNum = 101 + count
			}
			incident.IncidentNumber = fmt.Sprintf("INC-%d", nextNum)
		}
		if err := tx.Create(incident).Error; err != nil {
			return err
		}
		initialHistory := models.IncidentStatusHistory{
			ID:             uuid.New(),
			TeamIncidentID: incident.ID,
			UpdatedBy:      incident.CreatedBy,
			Title:          incident.Title,
			NewStatus:      incident.Status,
			PreviousStatus: "",
			Details:        incident.Details,
			UpdatedAt:      incident.AssignedAt,
		}
		return tx.Create(&initialHistory).Error
	})
}

func mapIncidentRows(rows []types.TeamIncidentWithHistoryRow) []models.TeamIncident {
	incMap := make(map[uuid.UUID]*models.TeamIncident)
	var orderedIncidents []*models.TeamIncident

	for _, r := range rows {
		inc, exists := incMap[r.IncidentID]
		if !exists {
			newInc := &models.TeamIncident{
				ID:             r.IncidentID,
				IncidentNumber: r.IncidentNumber,
				TeamID:         r.TeamID,
				CreatedBy:      r.CreatedBy,
				Title:          r.Title,
				Status:         r.Status,
				Details:        r.Details,
				CreatedAt:      r.CreatedAt,
				AssignedAt:     r.AssignedAt,
				ResolvedAt:     r.ResolvedAt,
				History:        []models.IncidentStatusHistory{},
			}
			incMap[r.IncidentID] = newInc
			orderedIncidents = append(orderedIncidents, newInc)
			inc = newInc
		}

		if r.HistoryID != nil {
			title := ""
			if r.HistoryTitle != nil {
				title = *r.HistoryTitle
			}
			newStatus := ""
			if r.HistoryNewStatus != nil {
				newStatus = *r.HistoryNewStatus
			}
			prevStatus := ""
			if r.HistoryPreviousStatus != nil {
				prevStatus = *r.HistoryPreviousStatus
			}
			details := ""
			if r.HistoryDetails != nil {
				details = *r.HistoryDetails
			}
			updatedAt := time.Time{}
			if r.HistoryUpdatedAt != nil {
				updatedAt = *r.HistoryUpdatedAt
			}
			updatedBy := uuid.Nil
			if r.HistoryUpdatedBy != nil {
				updatedBy = *r.HistoryUpdatedBy
			}

			inc.History = append(inc.History, models.IncidentStatusHistory{
				ID:             *r.HistoryID,
				TeamIncidentID: r.IncidentID,
				UpdatedBy:      updatedBy,
				Title:          title,
				NewStatus:      newStatus,
				PreviousStatus: prevStatus,
				Details:        details,
				UpdatedAt:      updatedAt,
			})
		}
	}

	result := make([]models.TeamIncident, len(orderedIncidents))
	for i, inc := range orderedIncidents {
		result[i] = *inc
	}
	return result
}

func (t *teamRepository) ListTeamIncidents(ctx context.Context, teamID uuid.UUID) ([]models.TeamIncident, error) {
	var rows []types.TeamIncidentWithHistoryRow
	var rawSQL string
	var args []interface{}

	if teamID != uuid.Nil {
		rawSQL = `SELECT 
			i.id AS incident_id, i.incident_number, i.team_id, i.created_by, i.title,
			i.status, i.details, i.created_at, i.assigned_at, i.resolved_at,
			h.id AS history_id, h.updated_by AS history_updated_by, h.title AS history_title,
			h.new_status AS history_new_status, h.previous_status AS history_previous_status,
			h.details AS history_details, h.updated_at AS history_updated_at
		FROM team_incidents i
		LEFT JOIN incident_status_histories h ON h.team_incident_id = i.id
		WHERE i.team_id = ?
		ORDER BY i.assigned_at DESC, h.updated_at DESC`
		args = append(args, teamID)
	} else {
		rawSQL = `SELECT 
			i.id AS incident_id, i.incident_number, i.team_id, i.created_by, i.title,
			i.status, i.details, i.created_at, i.assigned_at, i.resolved_at,
			h.id AS history_id, h.updated_by AS history_updated_by, h.title AS history_title,
			h.new_status AS history_new_status, h.previous_status AS history_previous_status,
			h.details AS history_details, h.updated_at AS history_updated_at
		FROM team_incidents i
		LEFT JOIN incident_status_histories h ON h.team_incident_id = i.id
		ORDER BY i.assigned_at DESC, h.updated_at DESC`
	}

	if err := t.db.WithContext(ctx).Raw(rawSQL, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}

	return mapIncidentRows(rows), nil
}

func (t *teamRepository) GetTeamIncidentByID(ctx context.Context, incidentID uuid.UUID) (*models.TeamIncident, error) {
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
	ORDER BY h.updated_at DESC`

	if err := t.db.WithContext(ctx).Raw(rawSQL, incidentID).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	incidents := mapIncidentRows(rows)
	if len(incidents) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &incidents[0], nil
}

func (t *teamRepository) GetTeamIncidentByIDOrNumber(ctx context.Context, idOrNumber string) (*models.TeamIncident, error) {
	clean := strings.TrimSpace(idOrNumber)
	if clean == "" {
		return nil, errors.New("incident ID or number is required")
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
		ORDER BY h.updated_at DESC`
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
		ORDER BY h.updated_at DESC`
		args = append(args, clean)
	}

	if err := t.db.WithContext(ctx).Raw(rawSQL, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	incidents := mapIncidentRows(rows)
	if len(incidents) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &incidents[0], nil
}

func (t *teamRepository) UpdateTeamIncidentStatus(ctx context.Context, history *models.IncidentStatusHistory, updatedInc *models.TeamIncident) error {
	return t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		updates := map[string]interface{}{
			"status":  updatedInc.Status,
			"title":   updatedInc.Title,
			"details": updatedInc.Details,
		}
		// persist resolved_at when set by the service
		if updatedInc.ResolvedAt != nil {
			updates["resolved_at"] = updatedInc.ResolvedAt
		}
		if err := tx.Model(updatedInc).Updates(updates).Error; err != nil {
			return err
		}
		return tx.Create(history).Error
	})
}

func (t *teamRepository) GetTeamInstruction(ctx context.Context, teamID uuid.UUID) (*models.Instruction, []models.InstructionLog, error) {
	var inst models.Instruction
	err := t.db.WithContext(ctx).Where("team_id = ?", teamID).First(&inst).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, []models.InstructionLog{}, nil
		}
		return nil, nil, err
	}

	var logs []models.InstructionLog
	err = t.db.WithContext(ctx).Where("instruction_id = ?", inst.ID).Order("version DESC").Find(&logs).Error
	if logs == nil {
		logs = []models.InstructionLog{}
	}
	return &inst, logs, err
}

func (t *teamRepository) SaveTeamInstruction(ctx context.Context, instruction *models.Instruction, log *models.InstructionLog) error {
	return t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing models.Instruction
		err := tx.Where("team_id = ?", instruction.TeamID).First(&existing).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return tx.Create(instruction).Error
			}
			return err
		}

		if log != nil {
			log.InstructionID = existing.ID
			if err := tx.Create(log).Error; err != nil {
				return err
			}
		}

		existing.InstructionDetails = instruction.InstructionDetails
		existing.CreatedBy = instruction.CreatedBy
		return tx.Save(&existing).Error
	})
}

// ListAllTeams returns all teams in the database
func (t *teamRepository) ListAllTeams(ctx context.Context) ([]models.Team, error) {
	var teams []models.Team
	err := t.db.WithContext(ctx).Order("team_name ASC").Find(&teams).Error
	return teams, err
}
