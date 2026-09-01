# Support Copilot — Prompt Regression Tests

This directory contains a **PromptFoo** regression test suite that guards the behavior defined in [`system_prompt.md`](../../backend/internal/prompt/system_prompt.md).

> Any time you edit the system prompt, run this suite to catch regressions before deploying.

---

## 📁 Structure

```
prompt_regression/
├── promptfooconfig.yaml        # Main config — wires prompt + provider + tests
├── prompts/
│   └── chat_template.json      # Chat format: [system, user] message array
├── cases/
│   ├── 01_off_topic.yaml       # Reject non-IT requests (songs, poems, games)
│   ├── 02_small_talk.yaml      # Greetings & acks — no tool calls
│   ├── 03_alert_false_alarm.yaml # status:1 → no link, no remediation, 🟢
│   ├── 04_alert_real_alert.yaml  # status:0 → auto-link, diagnosis, 🚨
│   ├── 05_no_fabrication.yaml  # Missing records → inform, never hallucinate
│   ├── 06_output_format.yaml   # Always Markdown, never raw JSON
│   ├── 07_source_attribution.yaml # Every recommendation cites MCP-1/MCP-2
│   └── 08_tool_discipline.yaml # Only call tools with required params
└── .env.example                # Copy to .env and fill in your API key
```

---

## 🚀 Quick Start

### 1. Install PromptFoo

```bash
npm install -g promptfoo
```

### 2. Set your API key

```bash
cp .env.example .env
# Edit .env and add your OPENAI_API_KEY
```

### 3. Run the tests

```bash
cd tests/prompt_regression
promptfoo eval
```

### 4. View results in browser

```bash
promptfoo view
```

---

## 🧪 Test Categories

| File | Behaviors Tested | Key Rule |
|------|-----------------|----------|
| `01_off_topic.yaml` | Songs, poems, games, stories | Always decline & redirect |
| `02_small_talk.yaml` | Hello, thanks, bye, ok | No tools on social messages |
| `03_alert_false_alarm.yaml` | status:1 alerts | Never link, never remediate |
| `04_alert_real_alert.yaml` | status:0 alerts | Auto-link, always diagnose |
| `05_no_fabrication.yaml` | Missing records / tool errors | Never hallucinate IDs or metrics |
| `06_output_format.yaml` | Incident & runbook display | Markdown only, no raw JSON |
| `07_source_attribution.yaml` | Recommendations | Always cite MCP-1 or MCP-2 |
| `08_tool_discipline.yaml` | Tool call triggers | Ask for missing params first |

---

## 🔁 CI Integration

Add to your CI pipeline (GitHub Actions example):

```yaml
- name: Run prompt regression tests
  env:
    OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
  run: |
    npm install -g promptfoo
    cd tests/prompt_regression
    promptfoo eval --no-cache
```

> PromptFoo exits with code `1` if any test fails — CI will catch regressions automatically.

---

## ➕ Adding New Tests

1. Create a new YAML file in `cases/` following the naming convention.
2. Add it to the `tests:` list in `promptfooconfig.yaml`.
3. Use `llm-rubric` for behavioral checks, `contains`/`not-contains` for deterministic string checks.
