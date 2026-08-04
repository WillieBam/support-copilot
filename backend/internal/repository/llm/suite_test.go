package llm_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestLLMRepository(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "LLM Repository Suite")
}
