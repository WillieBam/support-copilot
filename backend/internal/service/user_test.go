package service_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/WillieBam/support_copilot/backend/internal/interfaces"
	"github.com/WillieBam/support_copilot/backend/internal/mocks"
	"github.com/WillieBam/support_copilot/backend/internal/service"
	"github.com/WillieBam/support_copilot/backend/types/models"
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
			expectedUser := &models.User{
				ID:          uuid.New(),
				FirebaseUID: "fb-123",
				Email:       "test@example.com",
			}
			mockUserRepo.On("GetUserByFirebaseUID", ctx, "fb-123").Return(expectedUser, nil)

			user, err := userSvc.GetUserByFirebaseUID(ctx, "fb-123")
			Expect(err).NotTo(HaveOccurred())
			Expect(user).To(Equal(expectedUser))
			mockUserRepo.AssertExpectations(GinkgoT())
		})
	})

	Context("SearchUsers", func() {
		It("should delegate SearchUsers to user repository", func() {
			expectedUsers := []models.User{
				{ID: uuid.New(), Email: "user1@example.com"},
			}
			mockUserRepo.On("SearchUsers", ctx, "user1", 10).Return(expectedUsers, nil)

			users, err := userSvc.SearchUsers(ctx, "user1", 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(users).To(Equal(expectedUsers))
			mockUserRepo.AssertExpectations(GinkgoT())
		})
	})
})
