package postgres

import (
	"context"
	"errors"

	"github.com/WillieBam/support_copilot/backend/internal/interfaces"
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
	var team models.Team
	err := t.db.WithContext(ctx).Preload("Members.User").Where("id = ?", teamID).First(&team).Error
	if err != nil {
		return nil, err
	}
	return &team, nil
}

func (t *teamRepository) GetUserWithTeamsByID(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	var user models.User
	err := t.db.WithContext(ctx).Preload("Memberships.Team.Members.User").Preload("Memberships.Team.Members").Preload("Memberships.Team").Where("id = ?", userID).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
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
	var members []models.TeamMember
	err := t.db.WithContext(ctx).Preload("User").Where("team_id = ?", teamID).Find(&members).Error
	return members, err
}

func (t *teamRepository) AssignTeamIncident(ctx context.Context, incident *models.TeamIncident) error {
	return t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
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

func (t *teamRepository) ListTeamIncidents(ctx context.Context, teamID uuid.UUID) ([]models.TeamIncident, error) {
	var incidents []models.TeamIncident
	db := t.db.WithContext(ctx).Preload("History", func(db *gorm.DB) *gorm.DB {
		return db.Order("updated_at DESC")
	})
	if teamID != uuid.Nil {
		db = db.Where("team_id = ?", teamID)
	}
	err := db.Order("assigned_at DESC").Find(&incidents).Error
	return incidents, err
}

func (t *teamRepository) GetTeamIncidentByID(ctx context.Context, incidentID uuid.UUID) (*models.TeamIncident, error) {
	var incident models.TeamIncident
	err := t.db.WithContext(ctx).Preload("History", func(db *gorm.DB) *gorm.DB {
		return db.Order("updated_at DESC")
	}).Where("id = ?", incidentID).First(&incident).Error
	if err != nil {
		return nil, err
	}
	return &incident, nil
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
