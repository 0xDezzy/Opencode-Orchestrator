package cli

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"issue-orchestrator/internal/agent/opencode"
	"issue-orchestrator/internal/app"
	cfgpkg "issue-orchestrator/internal/common/config"
	"issue-orchestrator/internal/common/logging"
	"issue-orchestrator/internal/db"
	"issue-orchestrator/internal/db/models"
	"issue-orchestrator/internal/issues"
	lin "issue-orchestrator/internal/issues/linear"
	orch "issue-orchestrator/internal/orchestrator"
)

func runCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "run", RunE: func(cmd *cobra.Command, args []string) error {
		if f.issue == "" {
			return fmt.Errorf("--issue is required")
		}
		c, wf, repo, tracker, runner, bus, log, err := initDeps(false)
		if err != nil {
			return err
		}
		issue, err := tracker.FetchIssue(cmd.Context(), f.issue)
		if err != nil {
			return err
		}
		worker := orch.NewWorker(c, wf, repo, tracker, runner, bus, log)
		run := orchRun(issue)
		if err := repo.CreateRun(cmd.Context(), run); err != nil {
			return err
		}
		ok, err := repo.AcquireIssueLock(cmd.Context(), issue.ID, run.ID, c.Scheduler.LockTTL)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("issue is locked")
		}
		worker.Run(cmd.Context(), run, *issue)
		fmt.Println("run finished", run.ID)
		return nil
	}}
	cmd.Flags().StringVar(&f.issue, "issue", "", "Linear issue identifier or ID")
	return cmd
}
func orchRun(i *issues.Issue) *models.Run {
	return &models.Run{ID: uuid.NewString(), IssueID: i.ID, IssueIdentifier: i.Identifier, IssueURL: i.URL, State: string(orch.RunStateClaimed), Attempt: 1, AgentName: "opencode"}
}

var runtimeLogHook *logging.Hook

func initDeps(tuiMode bool) (*cfgpkg.Config, *cfgpkg.Workflow, *db.Repository, issues.Tracker, *opencode.Runner, *app.Bus, *logrus.Logger, error) {
	c, err := cfgpkg.Load(f.config)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, err
	}
	if f.logLevel != "" {
		c.Logging.Level = f.logLevel
	}
	wf, _ := cfgpkg.LoadWorkflow(f.workflow)
	g, err := db.Open(c.SQLite.Path)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, err
	}
	if err := db.Migrate(g); err != nil {
		return nil, nil, nil, nil, nil, nil, nil, err
	}
	hook := logging.NewHook(c.TUI.MaxLogLines)
	runtimeLogHook = hook
	log, err := logging.Setup(logging.Options{Level: c.Logging.Level, Format: c.Logging.Format, TUI: tuiMode, FilePath: c.Logging.FilePath, MirrorToFile: c.Logging.TUIMirrorToFile, Hook: hook})
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, err
	}
	var tracker issues.Tracker = emptyTracker{}
	if c.Linear.APIKey != "" {
		lt, err := lin.New(c.Linear)
		if err != nil {
			return nil, nil, nil, nil, nil, nil, nil, err
		}
		tracker = lt
	}
	bus := app.NewBus()
	return c, wf, db.NewRepository(g), tracker, opencode.New(c.OpenCode), bus, log, nil
}

type emptyTracker struct{}

func (emptyTracker) FetchCandidateIssues(context.Context, issues.FetchOptions) ([]issues.Issue, error) {
	return nil, nil
}
func (emptyTracker) FetchIssue(context.Context, string) (*issues.Issue, error) {
	return nil, fmt.Errorf("linear api key is required")
}
func (emptyTracker) Comment(context.Context, string, string) error    { return nil }
func (emptyTracker) Transition(context.Context, string, string) error { return nil }
