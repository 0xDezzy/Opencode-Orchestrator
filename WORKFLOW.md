---
agent:
  max_turns: 8
  require_tests: true
handoff:
  success_state: Human Review
  failure_state: Agent Failed
---

You are working on Linear issue {{ .Issue.Identifier }}.

Title:
{{ .Issue.Title }}

Description:
{{ .Issue.Description }}

Repository rules:
- Inspect the repository before editing.
- Make the smallest correct change.
- Add or update tests where appropriate.
- Run relevant tests.
- Do not make unrelated refactors.
- Do not mark the issue Done.
- Commit changes with a clear message.

Final response must include:
1. Summary of changes
2. Tests run
3. Files changed
4. Any follow-up work
