package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"issue-orchestrator/internal/db"
)

type App struct {
	Repo     *db.Repository
	Bus      EventBus
	tick     func(context.Context) error
	shutdown func(context.Context) error
}

func New(repo *db.Repository, bus EventBus) *App         { return &App{Repo: repo, Bus: bus} }
func (a *App) SetTick(f func(context.Context) error)     { a.tick = f }
func (a *App) SetShutdown(f func(context.Context) error) { a.shutdown = f }
func (a *App) RequestSchedulerTick(ctx context.Context) error {
	if a.tick == nil {
		return nil
	}
	return a.tick(ctx)
}
func (a *App) Shutdown(ctx context.Context) error {
	if a.shutdown != nil {
		return a.shutdown(ctx)
	}
	return nil
}
func (a *App) Snapshot(ctx context.Context) (RuntimeSnapshot, error) {
	d, err := a.Repo.RuntimeSnapshot(ctx)
	if err != nil {
		return RuntimeSnapshot{}, err
	}
	s := RuntimeSnapshot{UpdatedAt: time.Now()}
	for _, r := range d.Runs {
		rv := RunView{ID: short(r.ID), Issue: r.IssueIdentifier, URL: r.IssueURL, State: r.State, Attempt: r.Attempt, Agent: r.AgentName, Branch: r.BranchName, Worktree: r.WorktreePath, Updated: r.UpdatedAt.Format(time.RFC3339), Duration: duration(r.StartedAt, r.FinishedAt)}
		if r.AgentSessionID != nil {
			rv.SessionID = *r.AgentSessionID
		}
		if r.Error != nil {
			rv.Error = *r.Error
		}
		rv.Changed = changedCount(r.ChangedFilesJSON)
		s.Runs = append(s.Runs, rv)
		s.Agents = append(s.Agents, AgentView{RunID: short(r.ID), Issue: r.IssueIdentifier, Agent: r.AgentName, SessionID: rv.SessionID, Status: r.State, LastEvent: rv.Updated, Runtime: rv.Duration, Stall: stall(r.LastHeartbeatAt, r.UpdatedAt)})
		if strings.Contains(r.State, "running") || r.State == "claimed" || r.State == "preparing" {
			s.Stats.ActiveRuns++
		}
		if r.State == "retry_queued" {
			s.Stats.QueuedRuns++
		}
		if r.State == "failed" {
			s.Stats.FailedRuns++
		}
		if r.State == "succeeded" {
			s.Stats.SucceededRuns++
		}
	}
	for _, i := range d.Issues {
		s.Issues = append(s.Issues, IssueView{Issue: i.Identifier, Title: i.Title, LinearState: i.State, Labels: i.LabelsJSON, Assignee: i.Assignee, Updated: i.FetchedAt.Format(time.RFC3339), Eligible: "yes"})
	}
	for _, w := range d.Workspaces {
		s.Workspaces = append(s.Workspaces, WorkspaceView{Issue: w.IssueIdentifier, Branch: w.BranchName, Path: w.Path, Status: w.Status, Dirty: fmt.Sprint(w.Dirty), Created: w.CreatedAt.Format(time.RFC3339), Updated: w.UpdatedAt.Format(time.RFC3339)})
	}
	for _, l := range d.Locks {
		s.Locks = append(s.Locks, LockView{Issue: l.IssueID, RunID: short(l.RunID), Expires: l.ExpiresAt.Format(time.RFC3339), Age: time.Since(l.CreatedAt).Round(time.Second).String()})
	}
	s.Stats.LockedIssues = len(d.Locks)
	return s, nil
}

func duration(start time.Time, finish *time.Time) string {
	if start.IsZero() {
		return ""
	}
	end := time.Now()
	if finish != nil {
		end = *finish
	}
	return end.Sub(start).Round(time.Second).String()
}

func changedCount(raw *string) string {
	if raw == nil || *raw == "" {
		return "0"
	}
	var files []string
	if err := json.Unmarshal([]byte(*raw), &files); err != nil {
		return "?"
	}
	return fmt.Sprint(len(files))
}

func stall(last *time.Time, updated time.Time) string {
	base := updated
	if last != nil {
		base = *last
	}
	if base.IsZero() {
		return ""
	}
	return time.Since(base).Round(time.Second).String()
}
func short(s string) string {
	if len(s) > 8 {
		return s[:8]
	}
	return s
}
