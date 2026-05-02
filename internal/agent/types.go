package agent

import (
	"context"
	"time"

	"issue-orchestrator/internal/issues"
)

type Event struct {
	Type, Source, Message string
	Payload               any
	CreatedAt             time.Time
}
type RunIssueRequest struct {
	RunID                           string
	Issue                           issues.Issue
	WorktreeDir, BranchName, Prompt string
	Attempt                         int
	EmitEvent                       func(context.Context, Event) error
}
type RunIssueResult struct {
	SessionID, StopReason string
	ChangedFiles          []string
	Summary               string
	Raw                   any
}
