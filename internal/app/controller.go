package app

import (
	"context"
	"time"
)

type Controller interface {
	Snapshot(context.Context) (RuntimeSnapshot, error)
	RequestSchedulerTick(context.Context) error
	RequestReconcile(context.Context) error
	Shutdown(context.Context) error
}
type RuntimeSnapshot struct {
	Runs       []RunView
	Issues     []IssueView
	Agents     []AgentView
	Workspaces []WorkspaceView
	Locks      []LockView
	Stats      RuntimeStats
	UpdatedAt  time.Time
}
type RuntimeStats struct {
	ActiveRuns, QueuedRuns, FailedRuns, SucceededRuns, LockedIssues int
	LastSchedulerTick                                               *time.Time
	LastSchedulerError                                              string
}
type RunView struct {
	ID, Issue, URL, State, Agent, SessionID, Branch, Worktree, Changed, Duration, Updated, Error string
	Attempt                                                                                      int
}
type IssueView struct{ Issue, Title, LinearState, Labels, Assignee, Updated, Eligible string }
type AgentView struct{ RunID, Issue, Agent, SessionID, Status, LastEvent, Runtime, Stall string }
type WorkspaceView struct{ Issue, Branch, Path, Status, Dirty, Created, Updated string }
type LockView struct{ Issue, RunID, Expires, Age string }
