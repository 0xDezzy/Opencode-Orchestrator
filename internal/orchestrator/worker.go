package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"issue-orchestrator/internal/agent"
	"issue-orchestrator/internal/app"
	"issue-orchestrator/internal/common/config"
	"issue-orchestrator/internal/db"
	"issue-orchestrator/internal/db/models"
	"issue-orchestrator/internal/issues"
	"issue-orchestrator/internal/llm"
)

type Worker struct {
	cfg     *config.Config
	wf      *config.Workflow
	repo    *db.Repository
	tracker issues.Tracker
	runner  agent.Runner
	bus     app.EventBus
	log     *logrus.Logger
}

func NewWorker(cfg *config.Config, wf *config.Workflow, repo *db.Repository, tracker issues.Tracker, runner agent.Runner, bus app.EventBus, log *logrus.Logger) *Worker {
	return &Worker{cfg: cfg, wf: wf, repo: repo, tracker: tracker, runner: runner, bus: bus, log: log}
}

func (w *Worker) Run(ctx context.Context, run *models.Run, issue issues.Issue) {
	if run.StartedAt.IsZero() {
		run.StartedAt = time.Now()
	}
	startUpdates := map[string]any{"started_at": run.StartedAt}
	if run.WorktreePath != "" {
		startUpdates["worktree_path"] = run.WorktreePath
	}
	if run.BranchName != "" {
		startUpdates["branch_name"] = run.BranchName
	}
	_ = w.repo.DB().WithContext(ctx).Model(&models.Run{}).Where("id = ?", run.ID).Updates(startUpdates).Error
	_ = w.repo.UpdateRunState(ctx, run.ID, string(RunStatePreparing))
	if w.cfg != nil && w.cfg.Handoff.UpdateLinear && w.cfg.Handoff.TransitionOnStart && w.cfg.Linear.RunningState != "" {
		w.transitionIssue(ctx, run, issue.ID, w.cfg.Linear.RunningState)
	}
	if w.cfg != nil && w.cfg.Handoff.UpdateLinear && w.cfg.Handoff.CommentOnStart {
		w.commentIssue(ctx, run, issue.ID, "start", formatStartComment(run, issue))
	}

	_ = w.repo.UpdateRunState(ctx, run.ID, string(RunStateRunningAgent))
	promptData := llm.PromptData{Issue: issue, RunID: run.ID, Attempt: run.Attempt}
	promptData.Workspace.Path = run.WorktreePath
	promptData.Workspace.BranchName = run.BranchName
	if w.cfg != nil {
		promptData.Workspace.BaseBranch = w.cfg.Workspace.BaseBranch
	}
	prompt, err := llm.Render(w.wf, promptData)
	if err != nil {
		w.fail(ctx, run, issue, err)
		return
	}
	result, err := w.runner.RunIssue(ctx, agent.RunIssueRequest{RunID: run.ID, Issue: issue, WorktreeDir: run.WorktreePath, BranchName: run.BranchName, Prompt: prompt, Attempt: run.Attempt, EmitEvent: w.emitAgentEvent(run, issue.ID)})
	if err != nil {
		w.fail(ctx, run, issue, err)
		return
	}

	updates := map[string]any{"state": string(RunStateSucceeded), "finished_at": time.Now()}
	if result != nil {
		if result.SessionID != "" {
			updates["agent_session_id"] = result.SessionID
		}
		if result.Summary != "" {
			updates["summary"] = result.Summary
		}
		if len(result.ChangedFiles) > 0 {
			if b, err := json.Marshal(result.ChangedFiles); err == nil {
				s := string(b)
				updates["changed_files_json"] = &s
			}
		}
	}
	_ = w.repo.DB().WithContext(ctx).Model(&models.Run{}).Where("id = ?", run.ID).Updates(updates).Error
	if w.cfg != nil && w.cfg.Handoff.UpdateLinear && w.cfg.Handoff.TransitionOnSuccess && w.cfg.Linear.ReviewState != "" {
		w.transitionIssue(ctx, run, issue.ID, w.cfg.Linear.ReviewState)
	}
	if w.cfg != nil && w.cfg.Handoff.UpdateLinear && w.cfg.Handoff.CommentOnSuccess {
		w.commentIssue(ctx, run, issue.ID, "success", formatSuccessComment(run, issue, result))
	}
}

func (w *Worker) fail(ctx context.Context, run *models.Run, issue issues.Issue, err error) {
	_ = w.repo.FinishRun(ctx, run.ID, string(RunStateFailed), err.Error())
	if w.cfg != nil && w.cfg.Handoff.UpdateLinear && w.cfg.Handoff.TransitionOnFailure && w.cfg.Linear.FailedState != "" {
		w.transitionIssue(ctx, run, issue.ID, w.cfg.Linear.FailedState)
	}
	if w.cfg != nil && w.cfg.Handoff.UpdateLinear && w.cfg.Handoff.CommentOnFailure {
		w.commentIssue(ctx, run, issue.ID, "failure", formatFailureComment(run, issue, err, w.cfg))
	}
}

func (w *Worker) commentIssue(ctx context.Context, run *models.Run, issueID, phase, body string) {
	if w.tracker == nil || body == "" {
		return
	}
	if err := w.tracker.Comment(ctx, issueID, body); err != nil {
		if w.log != nil {
			w.log.WithError(err).WithField("issue_id", issueID).Warn("linear comment failed")
		}
		w.appendEvent(ctx, run.ID, issueID, "linear.comment_failed", "linear", "failed to comment on issue", map[string]any{"phase": phase})
		return
	}
	w.appendEvent(ctx, run.ID, issueID, "linear.comment", "linear", "commented on issue", map[string]any{"phase": phase})
}

func formatStartComment(run *models.Run, issue issues.Issue) string {
	var b strings.Builder
	fmt.Fprintf(&b, "**Agent run Started**\n\nIssue: %s - %s\nRun: %s\nAttempt: %d", issue.Identifier, issue.Title, run.ID, run.Attempt)
	if run.BranchName != "" {
		fmt.Fprintf(&b, "\nBranch: %s", run.BranchName)
	}
	if run.WorktreePath != "" {
		fmt.Fprintf(&b, "\nWorktree: %s", run.WorktreePath)
	}
	return b.String()
}

func formatSuccessComment(run *models.Run, issue issues.Issue, result *agent.RunIssueResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "**Agent run Succeeded**\n\nIssue: %s - %s\nRun: %s\nAttempt: %d", issue.Identifier, issue.Title, run.ID, run.Attempt)
	if result == nil {
		return b.String()
	}
	if result.SessionID != "" {
		fmt.Fprintf(&b, "\nSession: %s", result.SessionID)
	}
	if result.Summary != "" {
		fmt.Fprintf(&b, "\n\nSummary:\n%s", result.Summary)
	}
	if len(result.ChangedFiles) > 0 {
		b.WriteString("\n\nChanged files:")
		for _, file := range result.ChangedFiles {
			fmt.Fprintf(&b, "\n- %s", file)
		}
	}
	return b.String()
}

func formatFailureComment(run *models.Run, issue issues.Issue, err error, cfg *config.Config) string {
	var b strings.Builder
	fmt.Fprintf(&b, "**Agent run Failed**\n\nIssue: %s - %s\nRun: %s\nAttempt: %d", issue.Identifier, issue.Title, run.ID, run.Attempt)
	if err != nil {
		reason := err.Error()
		if cfg != nil && cfg.Linear.APIKey != "" {
			reason = strings.ReplaceAll(reason, cfg.Linear.APIKey, "[redacted]")
		}
		fmt.Fprintf(&b, "\n\nReason:\n%s", reason)
	}
	return b.String()
}

func (w *Worker) transitionIssue(ctx context.Context, run *models.Run, issueID, state string) {
	if w.tracker == nil || state == "" {
		return
	}
	if err := w.tracker.Transition(ctx, issueID, state); err != nil {
		if w.log != nil {
			w.log.WithError(err).WithField("issue_id", issueID).Warn("linear transition failed")
		}
		return
	}
	_ = w.repo.UpdateIssueSnapshotState(ctx, issueID, state)
	w.appendEvent(ctx, run.ID, issueID, "linear.transition", "linear", "transitioned issue", map[string]any{"state": state})
}

func (w *Worker) emitAgentEvent(run *models.Run, issueID string) func(context.Context, agent.Event) error {
	return func(ctx context.Context, ev agent.Event) error {
		payload := ev.Payload
		return w.appendEvent(ctx, run.ID, issueID, ev.Type, ev.Source, ev.Message, payload)
	}
}

func (w *Worker) appendEvent(ctx context.Context, runID, issueID, typ, source, message string, payload any) error {
	var raw string
	if payload != nil {
		if b, err := json.Marshal(payload); err == nil {
			raw = string(b)
		}
	}
	err := w.repo.AppendEvent(ctx, &models.Event{RunID: runID, IssueID: issueID, Type: typ, Source: source, Message: message, PayloadJSON: raw})
	if w.bus != nil {
		w.bus.Publish(app.RuntimeEvent{Type: typ, RunID: runID, IssueID: issueID, Payload: payload})
	}
	return err
}
