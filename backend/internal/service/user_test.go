package service_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/WillieBam/support_copilot/backend/internal/interfaces"
	"github.com/WillieBam/support_copilot/backend/internal/mocks"
	"github.com/WillieBam/support_copilot/backend/internal/service"
	"github.com/WillieBam/support_copilot/backend/types/models"
	customErrors "github.com/WillieBam/support_copilot/backend/utils/errors"
	"github.com/google/uuid"
)

var _ = Describe("UserService", func() {
	var (
		mockUserRepo *mocks.IUserRepository
		userSvc      interfaces.IUserService
		ctx          context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		mockUserRepo = new(mocks.IUserRepository)
		userSvc = service.NewUserService(mockUserRepo)
	})

	Context("GetUserByFirebaseUID", func() {
		It("should delegate GetUserByFirebaseUID to user repository", func() {
			fbUID := "fb-123"
			expectedUser := &models.User{
				ID:          uuid.New(),
				FirebaseUID: &fbUID,
				Email:       "test@example.com",
			}
			mockUserRepo.On("GetUserByFirebaseUID", ctx, "fb-123").Return(expectedUser, nil)

			user, err := userSvc.GetUserByFirebaseUID(ctx, "fb-123")
			Expect(err).NotTo(HaveOccurred())
			Expect(user).To(Equal(expectedUser))
			mockUserRepo.AssertExpectations(GinkgoT())
		})
	})

	Context("DeactivateUser", func() {
		It("should fail if invalid user IDs are provided", func() {
			err := userSvc.DeactivateUser(ctx, uuid.Nil, uuid.New())
			Expect(err).To(HaveOccurred())

			err = userSvc.DeactivateUser(ctx, uuid.New(), uuid.Nil)
			Expect(err).To(HaveOccurred())
		})

		It("should fail if self-deactivation is attempted", func() {
			sameID := uuid.New()
			err := userSvc.DeactivateUser(ctx, sameID, sameID)
			Expect(err).To(Equal(customErrors.ErrSelfDeactivationNotAllowed))
		})

		It("should fail if requester is not super_admin", func() {
			reqID := uuid.New()
			targetID := uuid.New()
			engineerUser := &models.User{ID: reqID, Scope: "engineer"}

			mockUserRepo.On("GetUserByID", ctx, reqID).Return(engineerUser, nil)

			err := userSvc.DeactivateUser(ctx, reqID, targetID)
			Expect(err).To(Equal(customErrors.ErrSuperAdminRequired))
			mockUserRepo.AssertExpectations(GinkgoT())
		})

		It("should fail if target user is not found", func() {
			reqID := uuid.New()
			targetID := uuid.New()
			adminUser := &models.User{ID: reqID, Scope: "super_admin"}

			mockUserRepo.On("GetUserByID", ctx, reqID).Return(adminUser, nil)
			mockUserRepo.On("GetUserByID", ctx, targetID).Return(nil, customErrors.ErrUserNotFound)

			err := userSvc.DeactivateUser(ctx, reqID, targetID)
			Expect(err).To(Equal(customErrors.ErrUserNotFound))
			mockUserRepo.AssertExpectations(GinkgoT())
		})

		It("should set DeactivatedAt and update user when requester is super_admin", func() {
			reqID := uuid.New()
			targetID := uuid.New()
			adminUser := &models.User{ID: reqID, Scope: "super_admin"}
			targetUser := &models.User{ID: targetID, Scope: "engineer"}

			mockUserRepo.On("GetUserByID", ctx, reqID).Return(adminUser, nil)
			mockUserRepo.On("GetUserByID", ctx, targetID).Return(targetUser, nil)
			mockUserRepo.On("UpdateUser", ctx, targetUser).Return(nil)

			err := userSvc.DeactivateUser(ctx, reqID, targetID)
			Expect(err).NotTo(HaveOccurred())
			Expect(targetUser.DeactivatedAt).NotTo(BeNil())
			mockUserRepo.AssertExpectations(GinkgoT())
		})
	})
})
