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

		It("should return generic db error if query fails", func() {
			firebaseUid := "uid-err"

			mock.ExpectQuery(`SELECT \* FROM "users" WHERE firebase_uid = \$1`).
				WithArgs(firebaseUid, 1).
				WillReturnError(errors.New("connection failed"))

			user, err := userRepo.GetUserByFirebaseUID(ctx, firebaseUid)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("connection failed"))
			Expect(user).To(BeNil())
		})
	})

	Context("GetUserByUsernameOrEmail", func() {
		It("should retrieve a user by username or email successfully", func() {
			uid := uuid.New()
			rows := sqlmock.NewRows([]string{"id", "username", "email", "display_name", "scope"}).
				AddRow(uid, "john_doe", "john@example.com", "John Doe", "engineer")

			mock.ExpectQuery(`SELECT \* FROM "users" WHERE username = \$1 OR email = \$2 ORDER BY "users"\."id" LIMIT \$3`).
				WithArgs("john_doe", "john_doe", 1).
				WillReturnRows(rows)

			user, err := userRepo.GetUserByUsernameOrEmail(ctx, "john_doe")
			Expect(err).NotTo(HaveOccurred())
			Expect(user).NotTo(BeNil())
			Expect(user.Email).To(Equal("john@example.com"))
		})

		It("should return ErrUserNotFound when user not found by identifier", func() {
			mock.ExpectQuery(`SELECT \* FROM "users" WHERE username = \$1 OR email = \$2 ORDER BY "users"\."id" LIMIT \$3`).
				WithArgs("unknown", "unknown", 1).
				WillReturnError(gorm.ErrRecordNotFound)

			user, err := userRepo.GetUserByUsernameOrEmail(ctx, "unknown")
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, customErrors.ErrUserNotFound)).To(BeTrue())
			Expect(user).To(BeNil())
		})

		It("should return generic error when database fails", func() {
			mock.ExpectQuery(`SELECT \* FROM "users" WHERE username = \$1 OR email = \$2 ORDER BY "users"\."id" LIMIT \$3`).
				WithArgs("unknown", "unknown", 1).
				WillReturnError(errors.New("db failure"))

			user, err := userRepo.GetUserByUsernameOrEmail(ctx, "unknown")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("db failure"))
			Expect(user).To(BeNil())
		})
	})

	Context("GetUserByID", func() {
		It("should retrieve a user by UUID successfully", func() {
			uid := uuid.New()
			rows := sqlmock.NewRows([]string{"id", "email", "display_name"}).
				AddRow(uid, "test@example.com", "Test User")

			mock.ExpectQuery(`SELECT \* FROM "users" WHERE id = \$1 ORDER BY "users"\."id" LIMIT \$2`).
				WithArgs(uid, 1).
				WillReturnRows(rows)

			user, err := userRepo.GetUserByID(ctx, uid)
			Expect(err).NotTo(HaveOccurred())
			Expect(user).NotTo(BeNil())
			Expect(user.ID).To(Equal(uid))
		})

		It("should return ErrUserNotFound when user ID does not exist", func() {
			uid := uuid.New()
			mock.ExpectQuery(`SELECT \* FROM "users" WHERE id = \$1 ORDER BY "users"\."id" LIMIT \$2`).
				WithArgs(uid, 1).
				WillReturnError(gorm.ErrRecordNotFound)

			user, err := userRepo.GetUserByID(ctx, uid)
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, customErrors.ErrUserNotFound)).To(BeTrue())
			Expect(user).To(BeNil())
		})

		It("should return database error when query fails", func() {
			uid := uuid.New()
			mock.ExpectQuery(`SELECT \* FROM "users" WHERE id = \$1 ORDER BY "users"\."id" LIMIT \$2`).
				WithArgs(uid, 1).
				WillReturnError(errors.New("db error"))

			user, err := userRepo.GetUserByID(ctx, uid)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("db error"))
			Expect(user).To(BeNil())
		})
	})

	Context("UpdateUser", func() {
		It("should update user successfully", func() {
			user := &models.User{
				ID:          uuid.New(),
				Email:       "update@example.com",
				DisplayName: "Updated",
			}

			mock.ExpectBegin()
			mock.ExpectExec(`UPDATE "users"`).
				WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectCommit()

			err := userRepo.UpdateUser(ctx, user)
			Expect(err).NotTo(HaveOccurred())
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

		It("should search users by email or display name with default limit", func() {
			rows := sqlmock.NewRows([]string{"id", "email", "display_name"}).
				AddRow(uuid.New(), "john@example.com", "John Doe")

			mock.ExpectQuery(`SELECT \* FROM "users" WHERE email ILIKE \$1 OR display_name ILIKE \$2 OR username ILIKE \$3 LIMIT \$4`).
				WithArgs("%john%", "%john%", "%john%", 10).
				WillReturnRows(rows)

			users, err := userRepo.SearchUsers(ctx, "john", 0)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(users)).To(Equal(1))
			Expect(users[0].Email).To(Equal("john@example.com"))
		})
	})
})
