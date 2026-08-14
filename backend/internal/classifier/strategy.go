package classifier

import (
	"regexp"
	"strings"

	"github.com/WillieBam/support_copilot/backend/internal/interfaces"
	"github.com/WillieBam/support_copilot/backend/types"
)

var uuidPattern = regexp.MustCompile(
	`(?i)[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`,
)

// conversationalPatterns is the ordered list of regex rules used to detect
// conversational prompts.
var conversationalPatterns = []*regexp.Regexp{
	// pattern1: ok, okay, alright, got it, sure, fine, noted
	regexp.MustCompile(`(?i)^(ok(ay)?|alright|got\s+it|sure|fine|noted)(\s+[\w\s'!,.]{0,20})?$`),
	// pattern2: bye, goodbye, see you, cya, ttyl, later
	regexp.MustCompile(`(?i)^(bye(bye)?|goodbye|see\s+you|cya|ttyl|later)(\s+[\w\s'!,.]{0,20})?$`),
	// pattern3: thanks, thank you, ty, cheers, much appreciated
	regexp.MustCompile(`(?i)^(thanks?|thank\s+you|ty|cheers|much\s+appreciated)(\s+[\w\s'!,.]{0,20})?$`),
	// pattern4: hi, hello, hey, halo, hei, yo + optional trailing social phrase
	regexp.MustCompile(`(?i)^(hi+|h[ae]llo+|hey+|halo+|hei|yo)(\s+[\w\s'!,.?]{0,30})?$`),
	// pattern5: Good <time-of-day> greetings
	regexp.MustCompile(`(?i)^good\s+(morning|afternoon|evening|day)(\s+[\w\s'!,.]{0,20})?$`),
	// pattern6: yes/no
	regexp.MustCompile(`(?i)^(yes|no|nope|yep|yeah|nah|yup)(\s+[\w\s'!,.]{0,20})?$`),
	// pattern6: done, stop, quit, exit, finish, that's all
	regexp.MustCompile(`(?i)^(done|stop|quit|exit|finish(ed)?|that'?s?\s+all)(\s+[\w\s'!,.]{0,20})?$`),
	// pattern7: wellness / small-talk questions: "are you ok?", "how are you?", "you good?"
	regexp.MustCompile(`(?i)^(are\s+you\s+ok\??|how\s+are\s+you\??|you\s+good\??|you\s+ok\??)$`),
}

var pureSignOffOrGreetingPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)^(hi+|h[ae]llo+|hey+|halo+|hei|yo)(\s+[\w\s'!,.?]{0,30})?$`),
	regexp.MustCompile(`(?i)^good\s+(morning|afternoon|evening|day)(\s+[\w\s'!,.]{0,20})?$`),
	regexp.MustCompile(`(?i)^(bye(bye)?|goodbye|see\s+you|cya|ttyl|later)(\s+[\w\s'!,.]{0,20})?$`),
	regexp.MustCompile(`(?i)^(thanks?|thank\s+you|ty|cheers|much\s+appreciated)(\s+[\w\s'!,.]{0,20})?$`),
}

// taskKeywords are lowercase substrings whose presence in the prompt or history strongly
// signals a task request that may need tool execution
var taskKeywords = []string{
	"validate", "alert", "incident", "check", "inspect", "anomaly",
	"metric", "status", "monitor", "error", "failure", "cpu", "memory",
	"healthy", "health", "service", "latency", "throughput", "outage",
	"runbook", "update", "create", "deprecate", "link", "search", "find",
	"list", "show", "get",
}

// RegexRuleStrategy implements IClassificationStrategy using regular expressions and heuristics.
type RegexRuleStrategy struct{}

func NewRegexRuleStrategy() interfaces.IClassificationStrategy {
	return &RegexRuleStrategy{}
}

func (s *RegexRuleStrategy) Name() string {
	return "regex_rules"
}

func (s *RegexRuleStrategy) Classify(prompt string, history []types.HistoryMessage) (types.Intent, float64, bool) {
	trimmed := strings.TrimSpace(prompt)

	if len(history) > 0 {
		if isPureSignOffOrGreeting(trimmed) {
			return types.IntentConversational, 1.0, true
		}
		if hasTaskContextInHistory(history) {
			return types.IntentTask, 1.0, true
		}
	}

	// check explicit conversational patterns first
	for _, re := range conversationalPatterns {
		if re.MatchString(trimmed) {
			return types.IntentConversational, 1.0, true
		}
	}

	// short-message heuristic: messages under 80 chars with no UUID and none of the known task trigger keywords
	if len(trimmed) <= 80 && !uuidPattern.MatchString(trimmed) && !containsTaskKeyword(trimmed) {
		return types.IntentConversational, 0.9, true
	}

	return types.IntentTask, 1.0, true
}

func containsTaskKeyword(s string) bool {
	lower := strings.ToLower(s)
	for _, kw := range taskKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

func isPureSignOffOrGreeting(s string) bool {
	for _, re := range pureSignOffOrGreetingPatterns {
		if re.MatchString(s) {
			return true
		}
	}
	return false
}

func hasTaskContextInHistory(history []types.HistoryMessage) bool {
	for _, h := range history {
		if uuidPattern.MatchString(h.Content) || containsTaskKeyword(h.Content) {
			return true
		}
	}
	return false
}
