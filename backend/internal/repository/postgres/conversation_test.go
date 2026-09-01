package postgres_test

import (
	"context"
	"errors"
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

	Context("CreateMessage & ListMessagesByConversation", func() {
		It("should insert a message successfully", func() {
			msg := &models.Message{
				ID:             uuid.New(),
				ConversationID: uuid.New(),
				Sender:         "user",
				Content:        "Hello, copilot!",
				CreatedAt:      time.Now(),
			}

			mock.ExpectBegin()
			mock.ExpectQuery(`INSERT INTO "messages"`).
				WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
				WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(msg.ID))
			mock.ExpectCommit()

			err := convRepo.CreateMessage(ctx, msg)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should list messages by conversation ID", func() {
			convID := uuid.New()
			rows := sqlmock.NewRows([]string{"id", "conversation_id", "sender_type", "content"}).
				AddRow(uuid.New(), convID, "user", "Hello")

			mock.ExpectQuery(`SELECT \* FROM "messages" WHERE conversation_id = \$1 ORDER BY created_at ASC`).
				WithArgs(convID).
				WillReturnRows(rows)

			msgs, err := convRepo.ListMessagesByConversation(ctx, convID)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(msgs)).To(Equal(1))
		})
	})

	Context("ListTeamConversations", func() {
		It("should list conversations by team ID with limit", func() {
			teamID := uuid.New()
			rows := sqlmock.NewRows([]string{"id", "team_id", "team_incident_id", "user_id", "title", "created_at", "user_email", "user_display_name", "user_scope"}).
				AddRow(uuid.New(), teamID, nil, uuid.New(), "Chat 1", time.Now(), "test@user.com", "Test User", "engineer")

			mock.ExpectQuery(`SELECT (.+) FROM conversations c LEFT JOIN users u ON (.+) WHERE c\.team_id = \$1 ORDER BY c\.created_at DESC LIMIT \$2`).
				WithArgs(teamID, 10).
				WillReturnRows(rows)

			convs, err := convRepo.ListTeamConversations(ctx, teamID, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(convs)).To(Equal(1))
		})

		It("should list conversations by team ID without limit when limit <= 0", func() {
			teamID := uuid.New()
			rows := sqlmock.NewRows([]string{"id", "team_id", "team_incident_id", "user_id", "title", "created_at", "user_email", "user_display_name", "user_scope"}).
				AddRow(uuid.New(), teamID, nil, uuid.New(), "Chat 1", time.Now(), "test@user.com", "Test User", "engineer")

			mock.ExpectQuery(`SELECT (.+) FROM conversations c LEFT JOIN users u ON (.+) WHERE c\.team_id = \$1 ORDER BY c\.created_at DESC`).
				WithArgs(teamID).
				WillReturnRows(rows)

			convs, err := convRepo.ListTeamConversations(ctx, teamID, 0)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(convs)).To(Equal(1))
		})

		It("should return error when ListTeamConversations fails", func() {
			teamID := uuid.New()
			mock.ExpectQuery(`SELECT (.+) FROM conversations c LEFT JOIN users u ON (.+) WHERE c\.team_id = \$1 ORDER BY c\.created_at DESC`).
				WithArgs(teamID).
				WillReturnError(errors.New("db error"))

			convs, err := convRepo.ListTeamConversations(ctx, teamID, 0)
			Expect(err).To(HaveOccurred())
			Expect(convs).To(BeNil())
		})
	})

	Context("GetConversationByID", func() {
		It("should fetch conversation by ID successfully with messages", func() {
			convID := uuid.New()
			userID := uuid.New()
			msgID := uuid.New()
			now := time.Now()
			sender := "user"
			content := "Hello"

			rows := sqlmock.NewRows([]string{"conv_id", "team_id", "team_incident_id", "user_id", "title", "conv_created_at", "user_email", "user_display_name", "user_scope", "message_id", "parent_message_id", "message_sender", "message_content", "message_created_at"}).
				AddRow(convID, uuid.New(), nil, userID, "Test Conv", now, "test@example.com", "User", "engineer", &msgID, nil, &sender, &content, &now)

			mock.ExpectQuery(`SELECT (.+) FROM conversations c LEFT JOIN users u ON (.+) LEFT JOIN messages m ON (.+) WHERE c\.id = \$1 ORDER BY m\.created_at ASC`).
				WithArgs(convID).
				WillReturnRows(rows)

			conv, err := convRepo.GetConversationByID(ctx, convID)
			Expect(err).NotTo(HaveOccurred())
			Expect(conv).NotTo(BeNil())
			Expect(conv.ID).To(Equal(convID))
			Expect(len(conv.Messages)).To(Equal(1))
			Expect(conv.Messages[0].Content).To(Equal("Hello"))
		})

		It("should return error when conversation not found", func() {
			convID := uuid.New()
			mock.ExpectQuery(`SELECT (.+) FROM conversations c LEFT JOIN users u ON (.+) LEFT JOIN messages m ON (.+) WHERE c\.id = \$1 ORDER BY m\.created_at ASC`).
				WithArgs(convID).
				WillReturnRows(sqlmock.NewRows([]string{"conv_id"}))

			conv, err := convRepo.GetConversationByID(ctx, convID)
			Expect(err).To(HaveOccurred())
			Expect(conv).To(BeNil())
		})

		It("should return error when GetConversationByID query fails", func() {
			convID := uuid.New()
			mock.ExpectQuery(`SELECT (.+) FROM conversations c LEFT JOIN users u ON (.+) LEFT JOIN messages m ON (.+) WHERE c\.id = \$1 ORDER BY m\.created_at ASC`).
				WithArgs(convID).
				WillReturnError(errors.New("db error"))

			conv, err := convRepo.GetConversationByID(ctx, convID)
			Expect(err).To(HaveOccurred())
			Expect(conv).To(BeNil())
		})
	})
})
