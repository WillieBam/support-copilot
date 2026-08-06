package classifier_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/WillieBam/support_copilot/backend/internal/classifier"
	"github.com/WillieBam/support_copilot/backend/types"
)

var _ = Describe("IntentClassifier", func() {
	var c *classifier.IntentClassifier

	BeforeEach(func() {
		c = classifier.NewIntentClassifier()
	})

	DescribeTable("should return IntentConversational for social / acknowledgement prompts",
		func(prompt string) {
			Expect(c.Classify(prompt)).To(Equal(classifier.IntentConversational))
		},
		// Acknowledgements
		Entry("ok", "ok"),
		Entry("ok byebye", "ok byebye"),
		Entry("okay", "okay"),
		Entry("alright", "alright"),
		Entry("got it", "got it"),
		Entry("sure", "sure"),
		Entry("fine", "fine"),
		Entry("noted", "noted"),
		// Sign-offs
		Entry("bye", "bye"),
		Entry("byebye", "byebye"),
		Entry("goodbye", "goodbye"),
		Entry("see you", "see you"),
		Entry("later", "later"),
		// Gratitude
		Entry("thanks", "thanks"),
		Entry("thank you", "thank you"),
		Entry("Thank You!", "Thank You!"),
		Entry("ty", "ty"),
		Entry("cheers", "cheers"),
		// Greetings — standard
		Entry("hi", "hi"),
		Entry("hello", "hello"),
		Entry("hey", "hey"),
		Entry("good morning", "good morning"),
		// Greetings — informal variants
		Entry("halo", "halo"),
		Entry("halo are you ok", "halo are you ok"),
		Entry("hei", "hei"),
		Entry("yo", "yo"),
		// Wellness small-talk
		Entry("are you ok", "are you ok"),
		Entry("are you ok?", "are you ok?"),
		Entry("how are you?", "how are you?"),
		Entry("you good?", "you good?"),
		// Yes/No
		Entry("yes", "yes"),
		Entry("no", "no"),
		Entry("yep", "yep"),
		Entry("nah", "nah"),
		// Completion
		Entry("done", "done"),
		Entry("stop", "stop"),
		Entry("that's all", "that's all"),
		Entry("finished", "finished"),
		// Short-message heuristic (no UUID, no task keyword, ≤80 chars)
		Entry("what's up", "what's up"),
		Entry("cool", "cool"),
	)

	DescribeTable("should return IntentTask for task-oriented prompts",
		func(prompt string) {
			Expect(c.Classify(prompt)).To(Equal(classifier.IntentTask))
		},
		Entry("validate alert uuid",
			"validate alert 550e8400-e29b-41d4-a716-446655440000"),
		Entry("check alert",
			"check alert 123e4567-e89b-12d3-a456-426614174000"),
		Entry("what is the system status",
			"what is the current system status?"),
		Entry("is the service healthy",
			"is the auth-service healthy right now?"),
		Entry("alert id provided",
			"please validate 550e8400-e29b-41d4-a716-446655440000"),
		Entry("ok followed by long uuid content is task",
			"ok, now validate alert 550e8400-e29b-41d4-a716-446655440000"),
		Entry("contains task keyword error",
			"the service is throwing an error"),
		Entry("contains task keyword monitor",
			"please monitor the cpu usage"),
		Entry("short list runbooks",
			"list runbooks"),
	)

	Context("ClassifyWithHistory", func() {
		It("should return IntentTask when user replies affirmative in a task conversation", func() {
			history := []types.HistoryMessage{
				{Role: "user", Content: "update runbook for payment gateway"},
				{Role: "assistant", Content: "I am ready to update the runbook. Should I proceed?"},
			}
			Expect(c.ClassifyWithHistory("yes that is correct", history)).To(Equal(classifier.IntentTask))
			Expect(c.ClassifyWithHistory("ok, I will wait for you", history)).To(Equal(classifier.IntentTask))
			Expect(c.ClassifyWithHistory("go ahead", history)).To(Equal(classifier.IntentTask))
		})

		It("should return IntentConversational when user says goodbye at the end of a conversation", func() {
			history := []types.HistoryMessage{
				{Role: "user", Content: "update runbook for payment gateway"},
				{Role: "assistant", Content: "Done! I updated the runbook."},
			}
			Expect(c.ClassifyWithHistory("thanks mate", history)).To(Equal(classifier.IntentConversational))
			Expect(c.ClassifyWithHistory("bye", history)).To(Equal(classifier.IntentConversational))
		})
	})
})

var _ = Describe("LooksLikeEmbeddedToolCall", func() {
	DescribeTable("should detect hallucinated JSON tool-call content",
		func(content string, expected bool) {
			Expect(classifier.LooksLikeEmbeddedToolCall(content)).To(Equal(expected))
		},
		Entry("greet tool call", `{"name": "greet", "parameters": {"message": "I"}}`, true),
		Entry("function key variant", `{"function": "validate_alert", "arguments": {"alert_id": "abc"}}`, true),
		Entry("plain text", "Hello! How can I help you?", false),
		Entry("empty string", "", false),
		Entry("normal JSON that is not a tool call", `{"key": "value"}`, false),
		Entry("partial match no parameters key", `{"name": "greet"}`, false),
	)
})

var _ = Describe("ParseEmbeddedToolCall", func() {
	It("should parse embedded tool call with name and parameters", func() {
		content := `{"name": "validate_alert", "parameters": {"alert_id": "550e8400-e29b-41d4-a716-446655440000"}}`
		toolCall, err := classifier.ParseEmbeddedToolCall(content)
		Expect(err).NotTo(HaveOccurred())
		Expect(toolCall).NotTo(BeNil())
		Expect(toolCall.Function.Name).To(Equal("validate_alert"))
	})

	It("should parse embedded tool call with function and arguments", func() {
		content := `{"function": "search_runbooks", "arguments": {"query": "redis"}}`
		toolCall, err := classifier.ParseEmbeddedToolCall(content)
		Expect(err).NotTo(HaveOccurred())
		Expect(toolCall).NotTo(BeNil())
		Expect(toolCall.Function.Name).To(Equal("search_runbooks"))
	})

	It("should fail when content does not contain JSON braces", func() {
		_, err := classifier.ParseEmbeddedToolCall("plain text with no json")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("no json object found"))
	})

	It("should fail when embedded tool call has no tool name", func() {
		_, err := classifier.ParseEmbeddedToolCall(`{"parameters": {"key": "val"}}`)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("embedded tool call missing tool name"))
	})
})

