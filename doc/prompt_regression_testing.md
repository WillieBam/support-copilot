# Prompt Regression Testing

> **Tooling:** [PromptFoo](https://promptfoo.dev/docs) · **Location:** [`tests/prompt_regression/`](../tests/prompt_regression/)

---

## 1. Purpose

The Support Copilot's behaviour is entirely governed by its [system prompt](../backend/internal/prompt/system_prompt.md). Every rule encoded there — from alert classification to output formatting — is invisible to unit tests that only cover application code. A seemingly small wording tweak to the system prompt can silently break critical agent behaviours such as:

- Incorrectly linking a false-alarm alert to an incident.
- Hallucinating runbook steps when a tool returns "not found".
- Responding to a greeting by calling a diagnostic tool.

**Prompt regression testing** addresses this gap by running a battery of scenario-based assertions against the live LLM each time the system prompt changes. If any assertion fails, the test suite exits with a non-zero code, blocking the change from merging until the regression is fixed.

### Key goals

| Goal | Why it matters |
|---|---|
| **Catch silent regressions** | Prompt edits can alter agent behaviour without any code change |
| **Enforce documented rules** | Each test maps to a named functional requirement (e.g. `FR-FA-01`) |
| **Enable confident iteration** | Engineers can refine prompts knowing regressions surface immediately |
| **CI gate** | The suite runs automatically on every pull request that touches the system prompt |

---

## 2. How It Works

### 2.1 Tooling — PromptFoo

[PromptFoo](https://promptfoo.dev) is an open-source LLM evaluation framework. It:

1. Sends test inputs to the model using a configurable **provider** (Gemini in this project).
2. Evaluates each model response against **assertions** (string checks, regex, or an LLM-as-judge rubric).
3. Produces a pass/fail result per test case and an overall exit code.

### 2.2 Architecture

```
tests/prompt_regression/
├── promptfooconfig.yaml          # Root config: wires prompt + provider + test files
├── prompts/
│   └── chat_template.json        # Chat format: [system, user] message array
├── cases/
│   ├── 01_off_topic.yaml
│   ├── 02_small_talk.yaml
│   ├── 03_alert_false_alarm.yaml
│   ├── 04_alert_real_alert.yaml
│   ├── 05_no_fabrication.yaml
│   ├── 06_output_format.yaml
│   ├── 07_source_attribution.yaml
│   └── 08_tool_discipline.yaml
└── .env                          # OPENAI_API_KEY (Gemini-compatible key)
```

### 2.3 Provider Configuration

The suite uses the **Gemini API** via its OpenAI-compatible endpoint, so PromptFoo's standard `openai:chat` provider can be used without custom plugins.

```yaml
# From promptfooconfig.yaml
providers:
  - id: openai:chat:gemini-3.5-flash-lite
    config:
      apiBaseUrl: https://generativelanguage.googleapis.com/v1beta/openai
      temperature: 0   # deterministic outputs for regression
```

`temperature: 0` ensures the model output is as deterministic as possible, making assertion failures reproducible.

### 2.4 The System Prompt as a Variable

The [system_prompt.md](../backend/internal/prompt/system_prompt.md) is loaded as a shared variable across all test cases. This means the test suite always runs against the **current** system prompt — there is no need to copy-paste prompt text into test files.

```yaml
# From promptfooconfig.yaml
defaultTest:
  vars:
    system_prompt: file://../../backend/internal/prompt/system_prompt.md
```

### 2.5 Assertion Types Used

| Type | How it works |
|---|---|
| `contains` | Response must include the exact string |
| `not-contains` | Response must NOT include the exact string |
| `contains-any` | Response must include at least one string from a list |
| `not-regex` | Response must NOT match the regex pattern |
| `llm-rubric` | A second LLM call evaluates the response against a natural-language criterion |

`llm-rubric` is particularly important for behavioural rules (e.g. "declines politely", "provides diagnosis") that cannot be expressed as a simple string match.

### 2.6 Running the Tests

#### Prerequisites

```bash
npm install -g promptfoo
```

#### Setup API key

```bash
cd tests/prompt_regression
cp .env.example .env
# Add your Gemini API key as OPENAI_API_KEY in .env
```

#### Run

```bash
cd tests/prompt_regression
promptfoo eval
```

#### View results in browser

```bash
promptfoo view
```

### 2.7 CI Integration

The suite is designed to be a CI gate. Add the following step to your GitHub Actions workflow for any PR that modifies the system prompt:

```yaml
- name: Run prompt regression tests
  env:
    OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
  run: |
    npm install -g promptfoo
    cd tests/prompt_regression
    promptfoo eval --no-cache
```

> PromptFoo exits with code `1` if any assertion fails — the pipeline will block the merge automatically.

---

## 3. Test Cases

The suite is split into **8 files**, each covering a distinct behavioural concern. All test IDs follow the convention `FR-<category>-<number>` (Functional Requirement).

---

### Category 01 — Off-Topic Request Rejection
**File:** [`cases/01_off_topic.yaml`](../tests/prompt_regression/cases/01_off_topic.yaml)

**Rule:** The agent must politely decline all non-IT requests (creative writing, games, jokes) and redirect the user to IT support tasks.

| Test ID | User Input | Key Assertions |
|---|---|---|
| `FR-OT-01` | "Write me a song about server downtime" | Does NOT contain "Here's a song" or 🎵; politely declines and redirects |
| `FR-OT-02` | "Can you write a poem about Kubernetes?" | Declines poem; mentions "IT Support" or "Support Copilot" |
| `FR-OT-03` | "Let's play 20 questions!" | Refuses game; redirects to incidents, alerts, or runbooks |
| `FR-OT-04` | "Tell me a story about a brave DevOps engineer" | Does NOT match `/once upon a time/i`; declines and redirects |
| `FR-OT-05` | "Tell me a funny joke about databases" | Does NOT tell a joke; declines politely and steers back to IT scope |

---

### Category 02 — Small Talk & Greetings
**File:** [`cases/02_small_talk.yaml`](../tests/prompt_regression/cases/02_small_talk.yaml)

**Rule:** Greetings, acknowledgements, and sign-offs must be handled conversationally. No tool calls must be triggered.

| Test ID | User Input | Key Assertions |
|---|---|---|
| `FR-ST-01` | "Hello!" | Short friendly greeting; no tool calls |
| `FR-ST-02` | "Thanks!" | Does NOT contain `validate_alert` or `list_incidents`; warm acknowledgment |
| `FR-ST-03` | "Bye" | Short farewell; no tools called, no questions asked |
| `FR-ST-04` | "Ok got it" | Brief natural reply; no tools invoked |
| `FR-ST-05` | "Alright" | Does NOT contain `get_incident`; short conversational reply |
| `FR-ST-06` | "Yes" | Brief conversational reply; no tool calls without further context |
| `FR-ST-07` | "Alright, thanks for the help today!" | Does NOT contain `list_incidents`, `validate_alert`, or `get_runbook`; friendly closing |

---

### Category 03 — False Alarm Handling (status: 1)
**File:** [`cases/03_alert_false_alarm.yaml`](../tests/prompt_regression/cases/03_alert_false_alarm.yaml)

**Rule:** When `validate_alert` returns `status: 1`, the agent must label it as 🟢 False Alarm, reassure the user, and must **not** link, escalate, or provide remediation.

| Test ID | Scenario | Key Assertions |
|---|---|---|
| `FR-FA-01` | A-001 status:1, confidence 0.95 | Does NOT contain `link_alert_to_incident`; does NOT ask "would you like to link"; contains "False Alarm" |
| `FR-FA-02` | A-002 status:1, all metrics normal | Does NOT contain `kubectl`; does NOT contain remediation step patterns; reassures system is normal |
| `FR-FA-03` | A-003 status:1, anomaly_score 0.05 | Response MUST contain "🟢" |
| `FR-FA-04` | A-004 status:1, no anomaly | Does NOT match `/link (this\|the) alert\|link.*to.*incident/i`; does NOT ask "would you like / shall I / do you want" |

---

### Category 04 — Real Alert Handling (status: 0)
**File:** [`cases/04_alert_real_alert.yaml`](../tests/prompt_regression/cases/04_alert_real_alert.yaml)

**Rule:** When `validate_alert` returns `status: 0`, the agent must label it as 🚨 Real Alert, provide technical diagnosis, and auto-link if an active incident exists — without asking for permission.

| Test ID | Scenario | Key Assertions |
|---|---|---|
| `FR-RA-01` | A-010 status:0, cpu:98, memory:95 | Contains 🚨 or "Real Alert" or "Confirmed Anomaly"; diagnostic context included |
| `FR-RA-02` | A-011 status:0 with active INC-101 | Does NOT match "would you like (me) to link"; does NOT match "shall I link / do you want me to link" |
| `FR-RA-03` | A-020 status:0, cpu:99, latency:8000ms, error_rate:45 | Provides concrete technical diagnosis / remediation commands |
| `FR-RA-04` | A-012 status:0, no open incidents | States alert is unlinked; does NOT fabricate an incident ID |
| `FR-RA-05` | A-013 status:0, risk_level: Critical | Includes "Critical" risk level in the response |

---

### Category 05 — No Fabrication / Anti-Hallucination
**File:** [`cases/05_no_fabrication.yaml`](../tests/prompt_regression/cases/05_no_fabrication.yaml)

**Rule:** When tools return "not found" or errors, the agent must inform the user clearly and suggest next steps. It must never invent alert IDs, incident IDs, metrics, or runbook steps.

| Test ID | Scenario | Key Assertions |
|---|---|---|
| `FR-NF-01` | `validate_alert` returns `alert not found` for A-NOTEXIST-9999 | Does NOT match `/anomaly score.*0\.[0-9]/i` or `/confidence.*0\.[0-9]/i`; suggests checking the ID |
| `FR-NF-02` | `get_incident` returns `incident not found` for INC-9999 | Does NOT contain "INC-9999 was caused by"; does not invent details |
| `FR-NF-03` | `get_runbook` returns `runbook not found` for RB-FAKE-001 | Does NOT invent runbook steps or content; suggests verifying the ID |
| `FR-NF-04` | "Show me runbook RB-FAKE-001" (no tool result) | Does NOT match filler phrases like "I am checking / Please hold on / Let me look / one moment" |
| `FR-NF-05` | MCP-1 connection timed out, asked for anomaly score of A-050 | Contains one of: "timed out / error / failed / MCP-1 / unable / cannot"; does NOT fabricate anomaly score |

---

### Category 06 — Output Format Compliance
**File:** [`cases/06_output_format.yaml`](../tests/prompt_regression/cases/06_output_format.yaml)

**Rule:** Incident and runbook data must always be rendered as structured Markdown. Raw JSON must never be leaked to the user.

| Test ID | Scenario | Key Assertions |
|---|---|---|
| `FR-OF-01` | `get_incident` result for INC-101 | Does NOT match raw JSON pattern `{"id":"INC`; formatted as structured Markdown |
| `FR-OF-02` | `get_incident` result for INC-102 | Contains at least one of: `## Incident Overview`, `## Details`, `## Linked Runbooks` |
| `FR-OF-03` | `get_runbook` result for RB-001 | Does NOT match raw JSON pattern `{"id":"RB`; rendered as clean Markdown |
| `FR-OF-04` | `get_runbook` result for RB-002 | Contains at least one of: RB-002, "DB Failover Runbook", "Active"; formatted layout |

---

### Category 07 — Source Attribution
**File:** [`cases/07_source_attribution.yaml`](../tests/prompt_regression/cases/07_source_attribution.yaml)

**Rule:** Every diagnosis or recommendation must explicitly cite its source — **MCP-1** for alert validation/ML results, **MCP-2** for runbook/knowledge base content.

| Test ID | Scenario | Key Assertions |
|---|---|---|
| `FR-SA-01` | MCP-1 alert validation result for A-020 (anomaly_score: 0.87) | Contains at least one of: "MCP-1", "ML model", "anomaly score", "ml_prediction" |
| `FR-SA-02` | MCP-2 retrieved `payment-api-restart.md` document | Contains "payment-api-restart.md" or "MCP-2"; does not give generic advice |
| `FR-SA-03` | Combined MCP-1 alert + MCP-2 runbook `db-connection-fix.md` | Contains MCP-1 reference AND MCP-2 / document reference |
| `FR-SA-04` | Alert A-022 mentioned, no tool results available | Does NOT fabricate anomaly scores; guides user to run `validate_alert` first |

---

### Category 08 — Tool Call Discipline
**File:** [`cases/08_tool_discipline.yaml`](../tests/prompt_regression/cases/08_tool_discipline.yaml)

**Rule:** Tools must only be called with required parameters provided by the user. The agent must never expose or request `team_id`, never use placeholder IDs, and never call tools on conversational messages.

| Test ID | User Input | Key Assertions |
|---|---|---|
| `FR-TD-01` | "Validate my alert" (no alert_id given) | Does NOT contain `validate_alert` tool call; asks user for the alert ID |
| `FR-TD-02` | "List all incidents for my team" | Does NOT match `/team.?id\|provide.*team\|enter.*team/i`; calls `list_incidents` without requesting team_id |
| `FR-TD-03` | "Get me incident details" (no incident ID given) | Asks user to specify which incident (e.g., INC-101) |
| `FR-TD-04` | "Thanks, that's all I needed!" | Does NOT contain any of: `list_incidents`, `validate_alert`, `get_runbook`, `get_incident` |
| `FR-TD-05` | "Update my runbook to add a new step" (no runbook_id) | Asks for the specific runbook ID before calling `update_runbook` |
| `FR-TD-06` | "Link this alert to an incident" (no incident ID) | Asks for the specific incident ID; does NOT invent or guess an incident surrogate key |

---

## 4. Test Results

> 📸 Test results are captured via `promptfoo view` in the browser. Screenshots below reflect the latest evaluation run.

<!-- Add your screenshots from `promptfoo view` here -->

*(Screenshots to be added after running `promptfoo eval` and `promptfoo view`)*

---

*Last updated: September 2026*
