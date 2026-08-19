package service

import (
	"context"
	"errors"
	"time"

	"github.com/WillieBam/support_copilot/backend/internal/interfaces"
	"github.com/WillieBam/support_copilot/backend/types/models"
	customErrors "github.com/WillieBam/support_copilot/backend/utils/errors"
	"github.com/google/uuid"
)

type userService struct {
	userRepo interfaces.IUserRepository
}

func NewUserService(userRepo interfaces.IUserRepository) interfaces.IUserService {
	return &userService{userRepo: userRepo}
}

func (s *userService) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	return s.userRepo.GetUserByID(ctx, id)
}

func (s *userService) GetUserByFirebaseUID(ctx context.Context, firebaseUid string) (*models.User, error) {
	return s.userRepo.GetUserByFirebaseUID(ctx, firebaseUid)
}

func (s *userService) SearchUsers(ctx context.Context, query string, limit int) ([]models.User, error) {
	return s.userRepo.SearchUsers(ctx, query, limit)
}

func (s *userService) DeactivateUser(ctx context.Context, requesterID, targetUserID uuid.UUID) error {
	if requesterID == uuid.Nil || targetUserID == uuid.Nil {
		return errors.New("invalid user ID")
	}
	if requesterID == targetUserID {
		return customErrors.ErrSelfDeactivationNotAllowed
	}

	requester, err := s.userRepo.GetUserByID(ctx, requesterID)
	if err != nil {
		return err
	}
	if requester.Scope != "super_admin" {
		return customErrors.ErrSuperAdminRequired
	}

	targetUser, err := s.userRepo.GetUserByID(ctx, targetUserID)
	if err != nil {
		return err
	}

	now := time.Now()
	targetUser.DeactivatedAt = &now
	return s.userRepo.UpdateUser(ctx, targetUser)
}
