package llm

import (
	"bytes"
	"fmt"
	"text/template"

	"issue-orchestrator/internal/common/config"
)

func Render(workflow *config.Workflow, data PromptData) (string, error) {
	body := ""
	if workflow != nil {
		body = workflow.Body
	}
	tmpl, err := template.New("workflow").Parse(body)
	if err != nil {
		return "", err
	}
	var b bytes.Buffer
	if err := tmpl.Execute(&b, data); err != nil {
		return "", err
	}
	return fmt.Sprintf("Issue: %s\nTitle: %s\nURL: %s\nWorktree: %s\nBranch: %s\nAttempt: %d\n\n%s\n\nCompletion criteria:\n- Make the smallest correct change.\n- Run relevant tests when possible.\n- Do not reveal secrets or modify unrelated files.\n- Finish with summary, tests, files changed, and follow-up work.\n", data.Issue.Identifier, data.Issue.Title, data.Issue.URL, data.Workspace.Path, data.Workspace.BranchName, data.Attempt, b.String()), nil
}
