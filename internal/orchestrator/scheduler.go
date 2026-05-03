package orchestrator

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"issue-orchestrator/internal/app"
	"issue-orchestrator/internal/common/config"
	"issue-orchestrator/internal/db"
	"issue-orchestrator/internal/db/models"
	"issue-orchestrator/internal/git"
	"issue-orchestrator/internal/issues"
)

type Scheduler struct {
	cfg         *config.Config
	repo        *db.Repository
	tracker     issues.Tracker
	worker      *Worker
	bus         app.EventBus
	log         *logrus.Logger
	synchronous bool
}

func NewScheduler(cfg *config.Config, repo *db.Repository, tracker issues.Tracker, worker *Worker, bus app.EventBus, log *logrus.Logger) *Scheduler {
	return &Scheduler{cfg: cfg, repo: repo, tracker: tracker, worker: worker, bus: bus, log: log}
}

func (s *Scheduler) SetSynchronousWorkers(v bool) { s.synchronous = v }

func (s *Scheduler) Run(ctx context.Context) error {
	interval := 30 * time.Second
	if s.cfg != nil && s.cfg.Scheduler.PollInterval > 0 {
		interval = s.cfg.Scheduler.PollInterval
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if err := s.Tick(ctx); err != nil && s.log != nil {
			s.log.WithError(err).Warn("scheduler tick failed")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (s *Scheduler) Tick(ctx context.Context) error {
	if s.tracker == nil {
		return nil
	}
	opts := issues.FetchOptions{}
	if s.cfg != nil {
		opts = issues.FetchOptions{TeamKey: s.cfg.Linear.TeamKey, ProjectName: s.cfg.Linear.ProjectName, ActiveStates: s.cfg.Linear.ActiveStates, IncludeLabels: s.cfg.Linear.Labels.Include, ExcludeLabels: s.cfg.Linear.Labels.Exclude, Limit: s.cfg.Scheduler.MaxConcurrentRuns}
	}
	candidates, err := s.tracker.FetchCandidateIssues(ctx, opts)
	if err != nil {
		return err
	}
	fetched := make(map[string]struct{}, len(candidates))
	for _, issue := range candidates {
		fetched[issue.ID] = struct{}{}
		if err := s.repo.UpsertIssueSnapshot(ctx, snapshotFromIssue(issue)); err != nil {
			return err
		}
		if s.isTerminalState(issue.State) {
			s.reconcileTerminalWorkspace(ctx, issue.ID, issue.Identifier, issue.State)
			continue
		}
		active, err := s.repo.FindActiveRunByIssue(ctx, issue.ID)
		if err != nil || active != nil {
			return err
		}
		wt, err := git.CreateOrReuseWorktree(ctx, git.WorktreeOptions{RepoPath: s.cfg.Workspace.RepoPath, WorkspaceRoot: s.cfg.Workspace.Root, IssueID: issue.ID, IssueIdentifier: issue.Identifier, Title: issue.Title, BaseBranch: s.cfg.Workspace.BaseBranch, BranchPrefix: s.cfg.Workspace.BranchPrefix})
		if err != nil {
			return err
		}
		dirty := false
		if hasChanges, err := git.HasChanges(ctx, wt.Path); err == nil {
			dirty = hasChanges
		}
		if err := s.repo.UpsertWorkspace(ctx, &models.Workspace{IssueID: issue.ID, IssueIdentifier: issue.Identifier, Path: wt.Path, BranchName: wt.BranchName, BaseBranch: s.cfg.Workspace.BaseBranch, Status: "active", Dirty: dirty}); err != nil {
			return err
		}
		run := &models.Run{ID: uuid.NewString(), IssueID: issue.ID, IssueIdentifier: issue.Identifier, IssueURL: issue.URL, State: string(RunStateClaimed), Attempt: 1, WorktreePath: wt.Path, BranchName: wt.BranchName, AgentName: "opencode"}
		if err := s.repo.CreateRun(ctx, run); err != nil {
			return err
		}
		if s.worker != nil {
			if s.synchronous {
				s.worker.Run(ctx, run, issue)
			} else {
				go s.worker.Run(ctx, run, issue)
			}
		}
	}
	s.reconcileTerminalIssues(ctx, fetched)
	if s.bus != nil {
		s.bus.Publish(app.RuntimeEvent{Type: "scheduler.tick.finished"})
	}
	return nil
}

func snapshotFromIssue(issue issues.Issue) *models.IssueSnapshot {
	labels, _ := json.Marshal(issue.Labels)
	raw, _ := json.Marshal(issue.Raw)
	return &models.IssueSnapshot{IssueID: issue.ID, Identifier: issue.Identifier, Title: issue.Title, Description: issue.Description, URL: issue.URL, State: issue.State, LabelsJSON: string(labels), Priority: issue.Priority, Assignee: issue.Assignee, RawJSON: string(raw), FetchedAt: time.Now()}
}
