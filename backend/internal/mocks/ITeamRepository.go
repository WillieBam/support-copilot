package mocks

import (
	context "context"

	models "github.com/WillieBam/support_copilot/backend/types/models"
	uuid "github.com/google/uuid"
	mock "github.com/stretchr/testify/mock"
)

// ITeamRepository is a mock type for the ITeamRepository type
type ITeamRepository struct {
	mock.Mock
}

func (_m *ITeamRepository) CreateTeamWithOwner(ctx context.Context, team *models.Team, ownerID uuid.UUID) error {
	ret := _m.Called(ctx, team, ownerID)
	return ret.Error(0)
}

func (_m *ITeamRepository) GetTeamByID(ctx context.Context, teamID uuid.UUID) (*models.Team, error) {
	ret := _m.Called(ctx, teamID)
	if ret.Get(0) != nil {
		return ret.Get(0).(*models.Team), ret.Error(1)
	}
	return nil, ret.Error(1)
}

func (_m *ITeamRepository) GetUserWithTeamsByID(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	ret := _m.Called(ctx, userID)
	if ret.Get(0) != nil {
		return ret.Get(0).(*models.User), ret.Error(1)
	}
	return nil, ret.Error(1)
}

func (_m *ITeamRepository) AddTeamMember(ctx context.Context, member *models.TeamMember) error {
	ret := _m.Called(ctx, member)
	return ret.Error(0)
}

func (_m *ITeamRepository) RemoveTeamMember(ctx context.Context, teamID, userID uuid.UUID) error {
	ret := _m.Called(ctx, teamID, userID)
	return ret.Error(0)
}

func (_m *ITeamRepository) DeleteTeam(ctx context.Context, teamID uuid.UUID) error {
	ret := _m.Called(ctx, teamID)
	return ret.Error(0)
}

func (_m *ITeamRepository) GetMemberRole(ctx context.Context, teamID, userID uuid.UUID) (string, error) {
	ret := _m.Called(ctx, teamID, userID)
	return ret.String(0), ret.Error(1)
}

func (_m *ITeamRepository) ListTeamMembers(ctx context.Context, teamID uuid.UUID) ([]models.TeamMember, error) {
	ret := _m.Called(ctx, teamID)
	if ret.Get(0) != nil {
		return ret.Get(0).([]models.TeamMember), ret.Error(1)
	}
	return nil, ret.Error(1)
}

func (_m *ITeamRepository) AssignTeamIncident(ctx context.Context, incident *models.TeamIncident) error {
	ret := _m.Called(ctx, incident)
	return ret.Error(0)
}

func (_m *ITeamRepository) ListTeamIncidents(ctx context.Context, teamID uuid.UUID) ([]models.TeamIncident, error) {
	ret := _m.Called(ctx, teamID)
	if ret.Get(0) != nil {
		return ret.Get(0).([]models.TeamIncident), ret.Error(1)
	}
	return nil, ret.Error(1)
}

func (_m *ITeamRepository) GetTeamIncidentByID(ctx context.Context, incidentID uuid.UUID) (*models.TeamIncident, error) {
	ret := _m.Called(ctx, incidentID)
	if ret.Get(0) != nil {
		return ret.Get(0).(*models.TeamIncident), ret.Error(1)
	}
	return nil, ret.Error(1)
}

func (_m *ITeamRepository) UpdateTeamIncidentStatus(ctx context.Context, history *models.IncidentStatusHistory, updatedInc *models.TeamIncident) error {
	ret := _m.Called(ctx, history, updatedInc)
	return ret.Error(0)
}

func (_m *ITeamRepository) GetTeamInstruction(ctx context.Context, teamID uuid.UUID) (*models.Instruction, []models.InstructionLog, error) {
	ret := _m.Called(ctx, teamID)
	var inst *models.Instruction
	if ret.Get(0) != nil {
		inst = ret.Get(0).(*models.Instruction)
	}
	var logs []models.InstructionLog
	if ret.Get(1) != nil {
		logs = ret.Get(1).([]models.InstructionLog)
	}
	return inst, logs, ret.Error(2)
}

func (_m *ITeamRepository) SaveTeamInstruction(ctx context.Context, instruction *models.Instruction, log *models.InstructionLog) error {
	ret := _m.Called(ctx, instruction, log)
	return ret.Error(0)
}

func NewITeamRepository(t interface {
	mock.TestingT
	Cleanup(func())
}) *ITeamRepository {
	mock := &ITeamRepository{}
	mock.Mock.Test(t)
	t.Cleanup(func() { mock.AssertExpectations(t) })
	return mock
}
