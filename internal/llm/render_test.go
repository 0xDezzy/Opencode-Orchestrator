package llm

import (
	"strings"
	"testing"

	"issue-orchestrator/internal/common/config"
	"issue-orchestrator/internal/issues"
)

func TestRenderIncludesIssueWorkspaceAndCriteria(t *testing.T) {
	d := PromptData{Issue: issues.Issue{Identifier: "MUN-123", Title: "Fix"}, RunID: "run", Attempt: 2}
	d.Workspace.Path = "/tmp/w"
	d.Workspace.BranchName = "agent/mun-123"
	p, err := Render(&config.Workflow{Body: "Body {{ .Issue.Identifier }}"}, d)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"MUN-123", "/tmp/w", "agent/mun-123", "Completion criteria", "Push the worktree branch to origin using the provided branch name."} {
		if !strings.Contains(p, want) {
			t.Fatalf("prompt missing %s", want)
		}
	}
}
