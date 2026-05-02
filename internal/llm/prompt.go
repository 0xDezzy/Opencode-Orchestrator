package llm

import (
	"time"

	"issue-orchestrator/internal/issues"
)

type PromptData struct {
	Issue     issues.Issue
	RunID     string
	Attempt   int
	Workspace struct{ Path, BranchName, BaseBranch string }
	Config    any
	Now       time.Time
}
