package firebase_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/WillieBam/support_copilot/backend/app/config"
	firebaseRepo "github.com/WillieBam/support_copilot/backend/internal/repository/firebase"
)

var _ = Describe("FirebaseRepository", func() {
	Context("Initialization & Token Verification", func() {
		It("should fail to initialize when service account file path is invalid or empty", func() {
			cfg := config.Get()
			cfg.Firebase.ServiceAccountPath = "nonexistent_file_path_12345.json"

			repo, err := firebaseRepo.NewFirebaseRepository(cfg)
			Expect(err).To(HaveOccurred())
			Expect(repo).To(BeNil())
		})

		It("should return an error when the auth client has not been initialized or receiver is nil", func() {
			repo := &firebaseRepo.FirebaseRepository{}

			_, err := repo.VerifyIDToken(context.Background(), "token")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("auth client"))

			var nilRepo *firebaseRepo.FirebaseRepository
			_, err = nilRepo.VerifyIDToken(context.Background(), "token")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("auth client"))
		})

		It("should initialize correctly with valid service account JSON and execute VerifyIDToken", func() {
			privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
			Expect(err).NotTo(HaveOccurred())

			pkcs8Bytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
			Expect(err).NotTo(HaveOccurred())

			pemBlock := &pem.Block{
				Type:  "PRIVATE KEY",
				Bytes: pkcs8Bytes,
			}
			pemString := string(pem.EncodeToMemory(pemBlock))

			// Write temporary valid credentials file
			tmpDir, err := os.MkdirTemp("", "firebase_test")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tmpDir)

			credPath := filepath.Join(tmpDir, "service_account.json")
			credJSON := fmt.Sprintf(`{
  "type": "service_account",
  "project_id": "test-project",
  "private_key_id": "1234567890",
  "private_key": %q,
  "client_email": "test@test-project.iam.gserviceaccount.com",
  "client_id": "1234567890"
}`, pemString)

			err = os.WriteFile(credPath, []byte(credJSON), 0600)
			Expect(err).NotTo(HaveOccurred())

			cfg := config.Get()
			cfg.Firebase.ServiceAccountPath = credPath

			repo, err := firebaseRepo.NewFirebaseRepository(cfg)
			Expect(err).NotTo(HaveOccurred())
			Expect(repo).NotTo(BeNil())

			// Execute VerifyIDToken against initialized client
			token, err := repo.VerifyIDToken(context.Background(), "invalid-token-sample")
			Expect(err).To(HaveOccurred())
			Expect(token).To(BeNil())
		})

		It("should verify token successfully when FIREBASE_AUTH_EMULATOR_HOST is configured", func() {
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				fmt.Fprintln(w, `{"users":[{"localId":"user-123","email":"user@test.com"}]}`)
			}))
			defer mockServer.Close()

			os.Setenv("FIREBASE_AUTH_EMULATOR_HOST", mockServer.Listener.Addr().String())
			defer os.Unsetenv("FIREBASE_AUTH_EMULATOR_HOST")

			privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
			Expect(err).NotTo(HaveOccurred())

			pkcs8Bytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
			Expect(err).NotTo(HaveOccurred())

			pemBlock := &pem.Block{
				Type:  "PRIVATE KEY",
				Bytes: pkcs8Bytes,
			}
			pemString := string(pem.EncodeToMemory(pemBlock))

			tmpDir, err := os.MkdirTemp("", "firebase_test")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tmpDir)

			credPath := filepath.Join(tmpDir, "service_account.json")
			credJSON := fmt.Sprintf(`{
  "type": "service_account",
  "project_id": "test-project",
  "private_key_id": "1234567890",
  "private_key": %q,
  "client_email": "test@test-project.iam.gserviceaccount.com",
  "client_id": "1234567890"
}`, pemString)

			err = os.WriteFile(credPath, []byte(credJSON), 0600)
			Expect(err).NotTo(HaveOccurred())

			cfg := config.Get()
			cfg.Firebase.ServiceAccountPath = credPath

			repo, err := firebaseRepo.NewFirebaseRepository(cfg)
			Expect(err).NotTo(HaveOccurred())

			header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
			payload := base64.RawURLEncoding.EncodeToString([]byte(`{"iss":"https://securetoken.google.com/test-project","aud":"test-project","sub":"user-123","user_id":"user-123","auth_time":1700000000,"iat":1700000000,"exp":2500000000}`))
			tokenStr := fmt.Sprintf("%s.%s.sig", header, payload)

			token, err := repo.VerifyIDToken(context.Background(), tokenStr)
			Expect(err).NotTo(HaveOccurred())
			Expect(token).NotTo(BeNil())
			Expect(token.UID).To(Equal("user-123"))
		})
	})
})
