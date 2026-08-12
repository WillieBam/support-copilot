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
	customErrors "github.com/WillieBam/support_copilot/backend/utils/errors"
)

var _ = Describe("UserRepository", func() {
	var (
		gormDB   *gorm.DB
		mock     sqlmock.Sqlmock
		userRepo interfaces.IUserRepository
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

		userRepo = postgresRepo.NewUserRepository(gormDB)
	})

	AfterEach(func() {
		Expect(mock.ExpectationsWereMet()).To(Succeed())
	})

	Context("CreateUser", func() {
		It("should successfully insert a user", func() {
			fbUID := "uid-123"
			user := &models.User{
				FirebaseUID: &fbUID,
				Email:       "user@example.com",
				DisplayName: "Test User",
				CreatedAt:   time.Now(),
				Scope:       "engineer",
			}

			mock.ExpectBegin()
			mock.ExpectQuery(`INSERT INTO "users"`).
				WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
				WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(uuid.New(), time.Now()))
			mock.ExpectCommit()

			err := userRepo.CreateUser(ctx, user)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return an error if insert fails", func() {
			fbUID := "uid-123"
			user := &models.User{
				FirebaseUID: &fbUID,
				Email:       "user@example.com",
				Scope:       "engineer",
			}

			mock.ExpectBegin()
			mock.ExpectQuery(`INSERT INTO "users"`).
				WillReturnError(errors.New("db error"))
			mock.ExpectRollback()

			err := userRepo.CreateUser(ctx, user)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("db error"))
		})
	})

	Context("GetUserByFirebaseUID", func() {
		It("should retrieve a user if they exist", func() {
			firebaseUid := "uid-123"
			uid := uuid.New()

			rows := sqlmock.NewRows([]string{"id", "firebase_uid", "username", "password_hash", "totp_secret", "totp_enabled", "email", "display_name", "scope"}).
				AddRow(uid, firebaseUid, nil, "", "", false, "user@example.com", "Test User", "engineer")

			mock.ExpectQuery(`SELECT \* FROM "users" WHERE firebase_uid = \$1`).
				WithArgs(firebaseUid, 1).
				WillReturnRows(rows)

			user, err := userRepo.GetUserByFirebaseUID(ctx, firebaseUid)
			Expect(err).NotTo(HaveOccurred())
			Expect(user).NotTo(BeNil())
			Expect(*user.FirebaseUID).To(Equal(firebaseUid))
			Expect(user.Email).To(Equal("user@example.com"))
		})

		It("should return error if user not found", func() {
			firebaseUid := "nonexistent"

			mock.ExpectQuery(`SELECT \* FROM "users" WHERE firebase_uid = \$1`).
				WithArgs(firebaseUid, 1).
				WillReturnError(gorm.ErrRecordNotFound)

			user, err := userRepo.GetUserByFirebaseUID(ctx, firebaseUid)
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, customErrors.ErrUserNotFound)).To(BeTrue())
			Expect(user).To(BeNil())
		})
	})

	Context("UpsertUser & SearchUsers", func() {
		It("should execute upsert query successfully", func() {
			fbUID := "uid-123"
			user := &models.User{
				FirebaseUID: &fbUID,
				Email:       "user@example.com",
				DisplayName: "Updated Name",
				CreatedAt:   time.Now(),
				Scope:       "engineer",
			}

			mock.ExpectBegin()
			mock.ExpectQuery(`INSERT INTO "users"`).
				WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
				WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New()))
			mock.ExpectCommit()

			err := userRepo.UpsertUser(ctx, user)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should search users by email or display name", func() {
			rows := sqlmock.NewRows([]string{"id", "email", "display_name"}).
				AddRow(uuid.New(), "john@example.com", "John Doe")

			mock.ExpectQuery(`SELECT \* FROM "users" WHERE email ILIKE \$1 OR display_name ILIKE \$2 OR username ILIKE \$3 LIMIT \$4`).
				WithArgs("%john%", "%john%", "%john%", 10).
				WillReturnRows(rows)

			users, err := userRepo.SearchUsers(ctx, "john", 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(users)).To(Equal(1))
			Expect(users[0].Email).To(Equal("john@example.com"))
		})
	})
})
