package service

import (
	"context"

	"github.com/WillieBam/support_copilot/backend/internal/interfaces"
	"github.com/WillieBam/support_copilot/backend/types/models"
)

type userService struct {
	userRepo interfaces.IUserRepository
}

func NewUserService(userRepo interfaces.IUserRepository) interfaces.IUserService {
	return &userService{userRepo: userRepo}
}

func (s *userService) GetUserByFirebaseUID(ctx context.Context, firebaseUid string) (*models.User, error) {
	return s.userRepo.GetUserByFirebaseUID(ctx, firebaseUid)
}

func (s *userService) SearchUsers(ctx context.Context, query string, limit int) ([]models.User, error) {
	return s.userRepo.SearchUsers(ctx, query, limit)
}
