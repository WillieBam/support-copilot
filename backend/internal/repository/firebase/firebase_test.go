package firebase_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/WillieBam/support_copilot/backend/app/config"
	firebaseRepo "github.com/WillieBam/support_copilot/backend/internal/repository/firebase"
)

var _ = Describe("FirebaseRepository", func() {
	Context("Initialization", func() {
		It("should fail to initialize when service account file path is invalid or empty", func() {
			cfg := config.Get()
			cfg.Firebase.ServiceAccountPath = "nonexistent_file_path_12345.json"

			repo, err := firebaseRepo.NewFirebaseRepository(cfg)
			Expect(err).To(HaveOccurred())
			Expect(repo).To(BeNil())
		})

		It("should return an error when the auth client has not been initialized", func() {
			repo := &firebaseRepo.FirebaseRepository{}

			_, err := repo.VerifyIDToken(context.Background(), "token")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("auth client"))
		})
	})
})
