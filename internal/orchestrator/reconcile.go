package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"issue-orchestrator/internal/app"
	"issue-orchestrator/internal/db/models"
	"issue-orchestrator/internal/git"
	"issue-orchestrator/internal/issues"
)

func activeRunState(state string) bool {
	switch RunState(state) {
	case RunStateClaimed, RunStatePreparing, RunStateRunningAgent, RunStateValidating, RunStateRetryQueued:
		return true
	default:
		return false
	}
}

func runNeedsHeartbeat(run models.Run) bool {
	return activeRunState(run.State) && run.LastHeartbeatAt == nil
}

func (s *Scheduler) reconcileClaimedRuns(ctx context.Context) {
	cutoff := time.Now().Add(-staleClaimedRunAge(s.cfg.Scheduler.PollInterval))
	var runs []models.Run
	if err := s.repo.DB().WithContext(ctx).Where("state = ? AND updated_at < ?", string(RunStateClaimed), cutoff).Find(&runs).Error; err != nil {
		return
	}
	for _, run := range runs {
		msg := "claimed run never reached worker; marking stale so it can be rescheduled"
		_ = s.repo.FinishRun(ctx, run.ID, string(RunStateFailed), msg)
		_ = releaseRunLock(ctx, s.repo, run.IssueID, run.ID)
		_ = s.repo.AppendEvent(ctx, &models.Event{RunID: run.ID, IssueID: run.IssueID, Type: "run.stale_claimed", Source: "scheduler", Message: msg})
		s.bus.Publish(app.RuntimeEvent{Type: "run.stale_claimed", RunID: run.ID, IssueID: run.IssueID, Payload: msg})
	}
}

func staleClaimedRunAge(poll time.Duration) time.Duration {
	threshold := 2 * poll
	if threshold < time.Minute {
		return time.Minute
	}
	return threshold
}

func (s *Scheduler) reconcileTerminalIssues(ctx context.Context, skip map[string]struct{}) {
	if s.tracker == nil || s.repo == nil || s.cfg == nil {
		return
	}
	snapshots, err := s.repo.ListIssueSnapshots(ctx)
	if err != nil {
		return
	}
	for _, snapshot := range snapshots {
		if s.isTerminalState(snapshot.State) {
			s.reconcileTerminalWorkspace(ctx, snapshot.IssueID, snapshot.Identifier, snapshot.State)
		}
	}
	if !s.fullReconcileDue(time.Now()) {
		return
	}
	s.lastFullReconcile = time.Now()
	limit := s.cfg.Scheduler.ReconcileBatchLimit
	fetched := 0
	for _, snapshot := range snapshots {
		if _, ok := skip[snapshot.IssueID]; ok {
			continue
		}
		if isRemovedIssueState(snapshot.State) {
			continue
		}
		if limit > 0 && fetched >= limit {
			break
		}
		fetched++
		issue, err := s.tracker.FetchIssue(ctx, snapshot.IssueID)
		if err != nil {
			if issues.IsIssueNotFound(err) {
				s.markIssueRemoved(ctx, snapshot, "deleted")
				continue
			}
			s.appendSchedulerEvent(ctx, snapshot.IssueID, "linear.reconcile_failed", "failed to reconcile Linear issue", map[string]any{"error": err.Error()})
			continue
		}
		if issue == nil {
			s.markIssueRemoved(ctx, snapshot, "deleted")
			continue
		}
		if err := s.repo.UpsertIssueSnapshot(ctx, snapshotFromIssue(*issue)); err != nil {
			s.appendSchedulerEvent(ctx, snapshot.IssueID, "linear.reconcile_failed", "failed to update issue snapshot", map[string]any{"error": err.Error()})
			continue
		}
		s.appendSchedulerEvent(ctx, issue.ID, "linear.issue_reconciled", "reconciled Linear issue", map[string]any{"state": issue.State})
		if s.isTerminalState(issue.State) {
			s.reconcileTerminalWorkspace(ctx, issue.ID, issue.Identifier, issue.State)
		}
	}
}

func (s *Scheduler) fullReconcileDue(now time.Time) bool {
	if s.lastFullReconcile.IsZero() {
		return true
	}
	interval := s.cfg.Scheduler.ReconcileInterval
	if interval <= 0 {
		return true
	}
	return !now.Before(s.lastFullReconcile.Add(interval))
}

func (s *Scheduler) markIssueRemoved(ctx context.Context, snapshot models.IssueSnapshot, state string) {
	if err := s.repo.MarkIssueSnapshotRemoved(ctx, snapshot.IssueID, state); err != nil {
		s.appendSchedulerEvent(ctx, snapshot.IssueID, "linear.reconcile_failed", "failed to mark removed Linear issue", map[string]any{"error": err.Error()})
		return
	}
	_ = s.repo.ReleaseIssueLocks(ctx, snapshot.IssueID)
	s.reconcileRemovedWorkspace(ctx, snapshot.IssueID, snapshot.Identifier, state)
	s.appendSchedulerEvent(ctx, snapshot.IssueID, "linear.issue_removed", "marked Linear issue removed", map[string]any{"state": state})
}

func (s *Scheduler) reconcileRemovedWorkspace(ctx context.Context, issueID, identifier, state string) {
	workspace, err := s.repo.FindWorkspaceByIssue(ctx, issueID)
	if err != nil || workspace == nil || workspace.Status == "removed" || strings.HasPrefix(workspace.Status, "preserved") {
		return
	}
	dirty := workspace.Dirty
	if workspace.Path != "" && git.WorktreeExists(workspace.Path) {
		if hasChanges, err := git.HasChanges(ctx, workspace.Path); err == nil && hasChanges {
			dirty = true
		}
		if unpushed, err := git.HasUnpushedCommits(ctx, workspace.Path); err != nil || unpushed {
			dirty = true
		}
	}
	if dirty {
		_ = s.repo.UpdateWorkspaceStatus(ctx, issueID, "preserved_dirty", true)
		s.emitTerminalWorkspaceEvent(ctx, issueID, identifier, "worktree.removed_preserved", "preserved dirty or unpushed removed issue worktree", state, workspace.Path, true)
		return
	}
	if workspace.Path != "" && git.WorktreeExists(workspace.Path) {
		if err := git.RemoveWorktree(ctx, s.cfg.Workspace.RepoPath, workspace.Path, false); err != nil {
			_ = s.repo.UpdateWorkspaceStatus(ctx, issueID, "preserved_remove_failed", false)
			s.emitTerminalWorkspaceEvent(ctx, issueID, identifier, "worktree.removed_preserved", fmt.Sprintf("preserved removed issue worktree after removal failed: %v", err), state, workspace.Path, false)
			return
		}
	}
	_ = s.repo.UpdateWorkspaceStatus(ctx, issueID, "removed", false)
	s.emitTerminalWorkspaceEvent(ctx, issueID, identifier, "worktree.removed", "removed worktree for removed Linear issue", state, workspace.Path, false)
}

func isRemovedIssueState(state string) bool {
	return strings.EqualFold(state, "deleted") || strings.EqualFold(state, "archived")
}

func (s *Scheduler) reconcileTerminalWorkspace(ctx context.Context, issueID, identifier, state string) {
	if s.repo == nil || s.cfg == nil || !s.isTerminalState(state) {
		return
	}
	workspace, err := s.repo.FindWorkspaceByIssue(ctx, issueID)
	if err != nil || workspace == nil || workspace.Status == "removed" || strings.HasPrefix(workspace.Status, "preserved") {
		return
	}
	active, err := s.repo.FindActiveRunByIssue(ctx, issueID)
	if err != nil || active != nil {
		if workspace.Status != "active_terminal" {
			_ = s.repo.UpdateWorkspaceStatus(ctx, issueID, "active_terminal", workspace.Dirty)
			s.emitTerminalWorkspaceEvent(ctx, issueID, identifier, "worktree.terminal_preserved", "preserved terminal worktree with active run", state, workspace.Path, workspace.Dirty)
		}
		return
	}
	if s.preserveTerminalWorkspace(state) {
		_ = s.repo.UpdateWorkspaceStatus(ctx, issueID, "preserved", workspace.Dirty)
		s.emitTerminalWorkspaceEvent(ctx, issueID, identifier, "worktree.terminal_preserved", "preserved terminal worktree", state, workspace.Path, workspace.Dirty)
		return
	}
	dirty := workspace.Dirty
	if workspace.Path != "" && git.WorktreeExists(workspace.Path) {
		if hasChanges, err := git.HasChanges(ctx, workspace.Path); err == nil && hasChanges {
			dirty = true
		}
		if unpushed, err := git.HasUnpushedCommits(ctx, workspace.Path); err != nil || unpushed {
			dirty = true
		}
	}
	if dirty {
		_ = s.repo.UpdateWorkspaceStatus(ctx, issueID, "preserved_dirty", true)
		s.emitTerminalWorkspaceEvent(ctx, issueID, identifier, "worktree.terminal_preserved", "preserved dirty or unpushed terminal worktree", state, workspace.Path, true)
		return
	}
	if workspace.Path != "" && git.WorktreeExists(workspace.Path) {
		if err := git.RemoveWorktree(ctx, s.cfg.Workspace.RepoPath, workspace.Path, false); err != nil {
			_ = s.repo.UpdateWorkspaceStatus(ctx, issueID, "preserved_remove_failed", false)
			s.emitTerminalWorkspaceEvent(ctx, issueID, identifier, "worktree.terminal_preserved", fmt.Sprintf("preserved terminal worktree after removal failed: %v", err), state, workspace.Path, false)
			return
		}
	}
	_ = s.repo.UpdateWorkspaceStatus(ctx, issueID, "removed", false)
	s.emitTerminalWorkspaceEvent(ctx, issueID, identifier, "worktree.terminal_removed", "removed terminal worktree", state, workspace.Path, false)
}

func (s *Scheduler) isTerminalState(state string) bool {
	if s.cfg == nil {
		return false
	}
	for _, terminal := range s.cfg.Linear.TerminalStates {
		if strings.EqualFold(terminal, state) {
			return true
		}
	}
	return false
}

func (s *Scheduler) preserveTerminalWorkspace(state string) bool {
	if strings.EqualFold(state, "Done") {
		return s.cfg.Workspace.PreserveSuccessful
	}
	return s.cfg.Workspace.PreserveFailed
}

func (s *Scheduler) emitTerminalWorkspaceEvent(ctx context.Context, issueID, identifier, typ, message, state, path string, dirty bool) {
	s.appendSchedulerEvent(ctx, issueID, typ, message, map[string]any{"identifier": identifier, "state": state, "path": path, "dirty": dirty})
	if s.log != nil {
		s.log.WithFields(map[string]any{"issue_id": issueID, "identifier": identifier, "state": state, "path": path, "dirty": dirty}).Info(message)
	}
}

func (s *Scheduler) appendSchedulerEvent(ctx context.Context, issueID, typ, message string, payload any) {
	var raw string
	if payload != nil {
		if b, err := json.Marshal(payload); err == nil {
			raw = string(b)
		}
	}
	_ = s.repo.AppendEvent(ctx, &models.Event{IssueID: issueID, Type: typ, Source: "scheduler", Message: message, PayloadJSON: raw})
	if s.bus != nil {
		s.bus.Publish(app.RuntimeEvent{Type: typ, IssueID: issueID, Payload: payload})
	}
}
