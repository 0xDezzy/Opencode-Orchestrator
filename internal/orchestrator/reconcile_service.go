package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"

	"issue-orchestrator/internal/app"
	"issue-orchestrator/internal/common/config"
	"issue-orchestrator/internal/db"
	"issue-orchestrator/internal/db/models"
	"issue-orchestrator/internal/git"
	"issue-orchestrator/internal/issues"
)

type ReconcileOptions struct {
	Issue  string
	DryRun bool
	Force  bool
}

type ReconcileSummary struct {
	IssuesRefreshed    int      `json:"issues_refreshed"`
	IssuesRemoved      int      `json:"issues_removed"`
	WorktreesRemoved   int      `json:"worktrees_removed"`
	WorktreesPreserved int      `json:"worktrees_preserved"`
	LocksReleased      int      `json:"locks_released"`
	Failures           int      `json:"failures"`
	Decisions          []string `json:"decisions,omitempty"`
	Errors             []string `json:"errors,omitempty"`
}

func (s ReconcileSummary) Failed() bool { return s.Failures > 0 }

type Reconciler struct {
	cfg     *config.Config
	repo    *db.Repository
	tracker issues.Tracker
	bus     app.EventBus
	log     *logrus.Logger
}

func NewReconciler(cfg *config.Config, repo *db.Repository, tracker issues.Tracker, bus app.EventBus, log *logrus.Logger) *Reconciler {
	return &Reconciler{cfg: cfg, repo: repo, tracker: tracker, bus: bus, log: log}
}

func (r *Reconciler) Reconcile(ctx context.Context, opts ReconcileOptions) ReconcileSummary {
	summary := ReconcileSummary{}
	if r.repo == nil || r.cfg == nil || r.tracker == nil {
		summary.addFailure("reconciler is not fully configured")
		return summary
	}

	snapshots, err := r.snapshots(ctx, opts.Issue)
	if err != nil {
		summary.addFailure(err.Error())
		return summary
	}
	for _, snapshot := range snapshots {
		r.reconcileSnapshot(ctx, snapshot, opts, &summary)
	}
	return summary
}

func (r *Reconciler) snapshots(ctx context.Context, issue string) ([]models.IssueSnapshot, error) {
	if issue == "" {
		return r.repo.ListIssueSnapshots(ctx)
	}
	snapshot, err := r.repo.FindIssueSnapshot(ctx, issue)
	if err != nil {
		return nil, err
	}
	if snapshot != nil {
		return []models.IssueSnapshot{*snapshot}, nil
	}
	remote, err := r.tracker.FetchIssue(ctx, issue)
	if err != nil {
		return nil, err
	}
	if remote == nil {
		return nil, fmt.Errorf("issue %q is not known locally and was not found in Linear", issue)
	}
	return []models.IssueSnapshot{*snapshotFromIssue(*remote)}, nil
}

func (r *Reconciler) reconcileSnapshot(ctx context.Context, snapshot models.IssueSnapshot, opts ReconcileOptions, summary *ReconcileSummary) {
	if isRemovedIssueState(snapshot.State) {
		r.reconcileRemovedWorkspace(ctx, snapshot.IssueID, snapshot.Identifier, snapshot.State, opts, summary)
		return
	}

	issue, err := r.tracker.FetchIssue(ctx, snapshot.IssueID)
	if err != nil {
		if issues.IsIssueRemoved(err) {
			r.markIssueRemoved(ctx, snapshot, "deleted", opts, summary)
			return
		}
		summary.addFailure(fmt.Sprintf("%s: %v", snapshot.Identifier, err))
		r.emit(ctx, snapshot.IssueID, opts, "linear.reconcile_failed", "failed to reconcile Linear issue", map[string]any{"error": err.Error()})
		return
	}
	if issue == nil {
		r.markIssueRemoved(ctx, snapshot, "deleted", opts, summary)
		return
	}
	if isRemovedIssueState(issue.State) {
		if !opts.DryRun {
			if err := r.repo.UpsertIssueSnapshot(ctx, snapshotFromIssue(*issue)); err != nil {
				summary.addFailure(fmt.Sprintf("%s: %v", snapshot.Identifier, err))
				return
			}
		}
		r.markIssueRemoved(ctx, *snapshotFromIssue(*issue), issue.State, opts, summary)
		return
	}
	if !opts.DryRun {
		if err := r.repo.UpsertIssueSnapshot(ctx, snapshotFromIssue(*issue)); err != nil {
			summary.addFailure(fmt.Sprintf("%s: %v", snapshot.Identifier, err))
			return
		}
	}
	summary.IssuesRefreshed++
	summary.addDecision(fmt.Sprintf("refreshed issue %s", firstNonEmpty(issue.Identifier, snapshot.Identifier, issue.ID)))
	r.emit(ctx, issue.ID, opts, "linear.issue_reconciled", "reconciled Linear issue", map[string]any{"state": issue.State})
	if r.isTerminalState(issue.State) {
		r.reconcileTerminalWorkspace(ctx, issue.ID, issue.Identifier, issue.State, opts, summary)
	}
}

func (r *Reconciler) markIssueRemoved(ctx context.Context, snapshot models.IssueSnapshot, state string, opts ReconcileOptions, summary *ReconcileSummary) {
	if !opts.DryRun {
		if err := r.repo.MarkIssueSnapshotRemoved(ctx, snapshot.IssueID, state); err != nil {
			summary.addFailure(fmt.Sprintf("%s: %v", snapshot.Identifier, err))
			return
		}
	}
	summary.IssuesRemoved++
	if released := r.releaseLocks(ctx, snapshot.IssueID, opts, summary); released > 0 {
		summary.LocksReleased += released
	}
	r.reconcileRemovedWorkspace(ctx, snapshot.IssueID, snapshot.Identifier, state, opts, summary)
	summary.addDecision(fmt.Sprintf("marked issue %s %s", firstNonEmpty(snapshot.Identifier, snapshot.IssueID), state))
	r.emit(ctx, snapshot.IssueID, opts, "linear.issue_removed", "marked Linear issue removed", map[string]any{"state": state})
}

func (r *Reconciler) releaseLocks(ctx context.Context, issueID string, opts ReconcileOptions, summary *ReconcileSummary) int {
	locks, err := r.repo.ListLocks(ctx)
	if err != nil {
		summary.addFailure(err.Error())
		return 0
	}
	count := 0
	for _, lock := range locks {
		if lock.IssueID == issueID {
			count++
		}
	}
	if count > 0 && !opts.DryRun {
		if err := r.repo.ReleaseIssueLocks(ctx, issueID); err != nil {
			summary.addFailure(err.Error())
			return 0
		}
	}
	return count
}

func (r *Reconciler) reconcileRemovedWorkspace(ctx context.Context, issueID, identifier, state string, opts ReconcileOptions, summary *ReconcileSummary) {
	r.reconcileWorkspace(ctx, issueID, identifier, state, true, opts, summary)
}

func (r *Reconciler) reconcileTerminalWorkspace(ctx context.Context, issueID, identifier, state string, opts ReconcileOptions, summary *ReconcileSummary) {
	if !r.isTerminalState(state) {
		return
	}
	r.reconcileWorkspace(ctx, issueID, identifier, state, false, opts, summary)
}

func (r *Reconciler) reconcileWorkspace(ctx context.Context, issueID, identifier, state string, removedIssue bool, opts ReconcileOptions, summary *ReconcileSummary) {
	workspace, err := r.repo.FindWorkspaceByIssue(ctx, issueID)
	if err != nil || workspace == nil || workspace.Status == "removed" || strings.HasPrefix(workspace.Status, "preserved") {
		return
	}
	if !removedIssue {
		active, err := r.repo.FindActiveRunByIssue(ctx, issueID)
		if err != nil {
			summary.addFailure(err.Error())
			return
		}
		if active != nil {
			r.updateWorkspace(ctx, issueID, "active_terminal", workspace.Dirty, opts, summary)
			summary.WorktreesPreserved++
			r.workspaceDecision(ctx, issueID, identifier, opts, "worktree.terminal_preserved", "preserved terminal worktree with active run", state, workspace.Path, workspace.Dirty, summary)
			return
		}
		if r.preserveTerminalWorkspace(state) {
			r.updateWorkspace(ctx, issueID, "preserved", workspace.Dirty, opts, summary)
			summary.WorktreesPreserved++
			r.workspaceDecision(ctx, issueID, identifier, opts, "worktree.terminal_preserved", "preserved terminal worktree", state, workspace.Path, workspace.Dirty, summary)
			return
		}
	}

	dirty := workspace.Dirty
	exists := workspace.Path != "" && git.WorktreeExists(workspace.Path)
	if exists {
		if hasChanges, err := git.HasChanges(ctx, workspace.Path); err == nil && hasChanges {
			dirty = true
		}
		if unpushed, err := git.HasUnpushedCommits(ctx, workspace.Path); err != nil || unpushed {
			dirty = true
		}
	}
	if dirty {
		r.updateWorkspace(ctx, issueID, "preserved_dirty", true, opts, summary)
		summary.WorktreesPreserved++
		msg := "preserved dirty or unpushed terminal worktree"
		typ := "worktree.terminal_preserved"
		if removedIssue {
			msg = "preserved dirty or unpushed removed issue worktree"
			typ = "worktree.removed_preserved"
		}
		r.workspaceDecision(ctx, issueID, identifier, opts, typ, msg, state, workspace.Path, true, summary)
		return
	}
	if exists && !opts.DryRun {
		if err := git.RemoveWorktree(ctx, r.cfg.Workspace.RepoPath, workspace.Path, opts.Force); err != nil {
			r.updateWorkspace(ctx, issueID, "preserved_remove_failed", false, opts, summary)
			summary.WorktreesPreserved++
			msg := fmt.Sprintf("preserved terminal worktree after removal failed: %v", err)
			typ := "worktree.terminal_preserved"
			if removedIssue {
				msg = fmt.Sprintf("preserved removed issue worktree after removal failed: %v", err)
				typ = "worktree.removed_preserved"
			}
			r.workspaceDecision(ctx, issueID, identifier, opts, typ, msg, state, workspace.Path, false, summary)
			return
		}
	}
	r.updateWorkspace(ctx, issueID, "removed", false, opts, summary)
	summary.WorktreesRemoved++
	msg := "removed terminal worktree"
	typ := "worktree.terminal_removed"
	if removedIssue {
		msg = "removed worktree for removed Linear issue"
		typ = "worktree.removed"
	}
	r.workspaceDecision(ctx, issueID, identifier, opts, typ, msg, state, workspace.Path, false, summary)
}

func (r *Reconciler) updateWorkspace(ctx context.Context, issueID, status string, dirty bool, opts ReconcileOptions, summary *ReconcileSummary) {
	if opts.DryRun {
		return
	}
	if err := r.repo.UpdateWorkspaceStatus(ctx, issueID, status, dirty); err != nil {
		summary.addFailure(err.Error())
	}
}

func (r *Reconciler) workspaceDecision(ctx context.Context, issueID, identifier string, opts ReconcileOptions, typ, message, state, path string, dirty bool, summary *ReconcileSummary) {
	summary.addDecision(fmt.Sprintf("%s: %s", firstNonEmpty(identifier, issueID), message))
	r.emit(ctx, issueID, opts, typ, message, map[string]any{"identifier": identifier, "state": state, "path": path, "dirty": dirty})
	if r.log != nil {
		r.log.WithFields(map[string]any{"issue_id": issueID, "identifier": identifier, "state": state, "path": path, "dirty": dirty, "dry_run": opts.DryRun}).Info(message)
	}
}

func (r *Reconciler) emit(ctx context.Context, issueID string, opts ReconcileOptions, typ, message string, payload any) {
	if !opts.DryRun {
		var raw string
		if payload != nil {
			if b, err := json.Marshal(payload); err == nil {
				raw = string(b)
			}
		}
		_ = r.repo.AppendEvent(ctx, &models.Event{IssueID: issueID, Type: typ, Source: "reconcile", Message: message, PayloadJSON: raw})
	}
	if r.bus != nil {
		r.bus.Publish(app.RuntimeEvent{Type: typ, IssueID: issueID, Payload: payload})
	}
}

func (r *Reconciler) isTerminalState(state string) bool {
	for _, terminal := range r.cfg.Linear.TerminalStates {
		if strings.EqualFold(terminal, state) {
			return true
		}
	}
	return false
}

func (r *Reconciler) preserveTerminalWorkspace(state string) bool {
	if strings.EqualFold(state, "Done") {
		return r.cfg.Workspace.PreserveSuccessful
	}
	return r.cfg.Workspace.PreserveFailed
}

func (s *ReconcileSummary) addDecision(decision string) {
	s.Decisions = append(s.Decisions, decision)
}

func (s *ReconcileSummary) addFailure(err string) {
	s.Failures++
	s.Errors = append(s.Errors, err)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
