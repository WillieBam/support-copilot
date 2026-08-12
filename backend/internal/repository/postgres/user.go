package postgres

import (
	"context"
	"errors"

	"github.com/WillieBam/support_copilot/backend/internal/interfaces"
	"github.com/WillieBam/support_copilot/backend/types/models"
	customErrors "github.com/WillieBam/support_copilot/backend/utils/errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) interfaces.IUserRepository {
	return &userRepository{db: db}
}

func (u *userRepository) CreateUser(ctx context.Context, user *models.User) error {
	return u.db.WithContext(ctx).Create(user).Error
}

func (u *userRepository) GetUserByFirebaseUID(ctx context.Context, firebaseUid string) (*models.User, error) {
	var user models.User
	err := u.db.Where("firebase_uid = ?", firebaseUid).First(&user).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, customErrors.ErrUserNotFound
		}
		return nil, err
	}

	return &user, nil
}

func (u *userRepository) GetUserByUsernameOrEmail(ctx context.Context, identifier string) (*models.User, error) {
	var user models.User
	err := u.db.WithContext(ctx).Where("username = ? OR email = ?", identifier, identifier).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, customErrors.ErrUserNotFound
		}
		return nil, err
	}

	return &user, nil
}

func (u *userRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var user models.User
	err := u.db.WithContext(ctx).Where("id = ?", id).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, customErrors.ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (u *userRepository) UpdateUser(ctx context.Context, user *models.User) error {
	return u.db.WithContext(ctx).Save(user).Error
}

func (u *userRepository) UpsertUser(ctx context.Context, user *models.User) error {
	return u.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "email"}},
		DoUpdates: clause.AssignmentColumns([]string{"firebase_uid", "display_name"}),
	}).Create(user).Error
}

func (u *userRepository) SearchUsers(ctx context.Context, query string, limit int) ([]models.User, error) {
	if limit <= 0 {
		limit = 10
	}
	var users []models.User
	searchPattern := "%" + query + "%"
	err := u.db.WithContext(ctx).
		Where("email ILIKE ? OR display_name ILIKE ? OR username ILIKE ?", searchPattern, searchPattern, searchPattern).
		Limit(limit).
		Find(&users).Error
	return users, err
}
