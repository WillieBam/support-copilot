package postgres_test

import (
	"context"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/WillieBam/support_copilot/backend/internal/interfaces"
	postgresRepo "github.com/WillieBam/support_copilot/backend/internal/repository/postgres"
	"github.com/WillieBam/support_copilot/backend/types/models"
)

var _ = Describe("ConversationRepository", func() {
	var (
		gormDB   *gorm.DB
		mock     sqlmock.Sqlmock
		convRepo interfaces.IConversationRepository
		ctx      context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		db, sqlMock, err := sqlmock.New()
		Expect(err).NotTo(HaveOccurred())

		mock = sqlMock
		dialector := postgres.New(postgres.Config{
			Conn: db,
		})

		gormDB, err = gorm.Open(dialector, &gorm.Config{})
		Expect(err).NotTo(HaveOccurred())

		convRepo = postgresRepo.NewConversationRepository(gormDB)
	})

	AfterEach(func() {
		Expect(mock.ExpectationsWereMet()).To(Succeed())
	})

	Context("CreateConversation", func() {
		It("should successfully insert a conversation", func() {
			conv := &models.Conversation{
				ID:        uuid.New(),
				TeamID:    uuid.New(),
				UserID:    uuid.New(),
				Title:     "test chat",
				CreatedAt: time.Now(),
			}

			mock.ExpectBegin()
			mock.ExpectQuery(`INSERT INTO "conversations"`).
				WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
				WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(conv.ID))
			mock.ExpectCommit()

			err := convRepo.CreateConversation(ctx, conv)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("UpdateConversationTitle", func() {
		It("should update title successfully", func() {
			convID := uuid.New()
			newTitle := "updated title"

			mock.ExpectBegin()
			mock.ExpectExec(`UPDATE "conversations" SET "title"=\$1 WHERE id = \$2`).
				WithArgs(newTitle, convID).
				WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectCommit()

			err := convRepo.UpdateConversationTitle(ctx, convID, newTitle)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
