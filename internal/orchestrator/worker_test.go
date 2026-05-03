package orchestrator

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"issue-orchestrator/internal/agent"
	"issue-orchestrator/internal/app"
	"issue-orchestrator/internal/common/config"
	"issue-orchestrator/internal/db"
	"issue-orchestrator/internal/db/models"
	"issue-orchestrator/internal/issues"
)

func TestWorkerTransitionUpdatesIssueSnapshotState(t *testing.T) {
	g, err := db.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Migrate(g); err != nil {
		t.Fatal(err)
	}
	repo := db.NewRepository(g)
	ctx := context.Background()
	issue := issues.Issue{ID: "issue-1", Identifier: "DEZ-4", Title: "Fix stale state", URL: "https://linear.app/issue/DEZ-4", State: "Todo"}
	if err := repo.UpsertIssueSnapshot(ctx, &models.IssueSnapshot{IssueID: issue.ID, Identifier: issue.Identifier, Title: issue.Title, URL: issue.URL, State: issue.State, FetchedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	run := &models.Run{ID: "run-1", IssueID: issue.ID, IssueIdentifier: issue.Identifier, IssueURL: issue.URL, State: string(RunStateClaimed), Attempt: 1, AgentName: "test"}
	if err := repo.CreateRun(ctx, run); err != nil {
		t.Fatal(err)
	}

	cfg := config.Defaults()
	cfg.Handoff.TransitionOnStart = false
	cfg.Handoff.TransitionOnSuccess = true
	cfg.Linear.ReviewState = "Human Review"
	worker := NewWorker(&cfg, &config.Workflow{Body: "test"}, repo, &fakeTracker{}, fakeRunner{}, app.NewBus(), logrus.New())

	worker.Run(ctx, run, issue)

	snapshot, err := repo.RuntimeSnapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got := snapshot.Issues[0].State; got != cfg.Linear.ReviewState {
		t.Fatalf("snapshot state = %q, want %q", got, cfg.Linear.ReviewState)
	}
}

func TestWorkerCommentsOnStartWhenEnabled(t *testing.T) {
	repo, run, issue := setupWorkerRun(t)
	cfg := config.Defaults()
	cfg.Handoff.TransitionOnStart = false
	cfg.Handoff.TransitionOnSuccess = false
	tracker := &fakeTracker{}
	worker := NewWorker(&cfg, &config.Workflow{Body: "test"}, repo, tracker, fakeRunner{}, app.NewBus(), logrus.New())

	worker.Run(context.Background(), run, issue)

	if len(tracker.comments) == 0 {
		t.Fatal("expected start comment")
	}
	comment := tracker.comments[0]
	if comment.issueID != issue.ID {
		t.Fatalf("comment issue ID = %q, want %q", comment.issueID, issue.ID)
	}
	if !strings.Contains(comment.body, "Started") || !strings.Contains(comment.body, issue.Identifier) || !strings.Contains(comment.body, run.ID) {
		t.Fatalf("start comment body %q missing expected run details", comment.body)
	}
}

func TestWorkerCommentsOnSuccessWhenEnabled(t *testing.T) {
	repo, run, issue := setupWorkerRun(t)
	cfg := config.Defaults()
	cfg.Handoff.CommentOnStart = false
	cfg.Handoff.TransitionOnSuccess = false
	tracker := &fakeTracker{}
	runner := resultRunner{result: &agent.RunIssueResult{SessionID: "session-1", Summary: "implemented fix", ChangedFiles: []string{"internal/orchestrator/worker.go", "internal/orchestrator/worker_test.go"}}}
	worker := NewWorker(&cfg, &config.Workflow{Body: "test"}, repo, tracker, runner, app.NewBus(), logrus.New())

	worker.Run(context.Background(), run, issue)

	if len(tracker.comments) != 1 {
		t.Fatalf("comments = %d, want 1", len(tracker.comments))
	}
	body := tracker.comments[0].body
	for _, want := range []string{"Succeeded", "implemented fix", "session-1", "internal/orchestrator/worker.go", "internal/orchestrator/worker_test.go"} {
		if !strings.Contains(body, want) {
			t.Fatalf("success comment body %q missing %q", body, want)
		}
	}
}

func TestWorkerCommentsOnFailureWhenEnabled(t *testing.T) {
	repo, run, issue := setupWorkerRun(t)
	cfg := config.Defaults()
	cfg.Linear.APIKey = "lin-secret-token"
	cfg.Handoff.CommentOnStart = false
	cfg.Handoff.TransitionOnFailure = false
	tracker := &fakeTracker{}
	worker := NewWorker(&cfg, &config.Workflow{Body: "test"}, repo, tracker, errorRunner{err: errors.New("agent failed with lin-secret-token")}, app.NewBus(), logrus.New())

	worker.Run(context.Background(), run, issue)

	if len(tracker.comments) != 1 {
		t.Fatalf("comments = %d, want 1", len(tracker.comments))
	}
	body := tracker.comments[0].body
	if !strings.Contains(body, "Failed") || !strings.Contains(body, "agent failed") {
		t.Fatalf("failure comment body %q missing failure details", body)
	}
	if strings.Contains(body, cfg.Linear.APIKey) || !strings.Contains(body, "[redacted]") {
		t.Fatalf("failure comment body %q did not redact configured secret", body)
	}
}

func TestWorkerDoesNotCommentWhenLinearCommentsDisabled(t *testing.T) {
	tests := []struct {
		name string
		cfg  func(*config.Config)
	}{
		{name: "update linear disabled", cfg: func(cfg *config.Config) { cfg.Handoff.UpdateLinear = false }},
		{name: "start disabled", cfg: func(cfg *config.Config) { cfg.Handoff.CommentOnStart = false; cfg.Handoff.CommentOnSuccess = false }},
		{name: "success disabled", cfg: func(cfg *config.Config) { cfg.Handoff.CommentOnStart = false; cfg.Handoff.CommentOnSuccess = false }},
		{name: "failure disabled", cfg: func(cfg *config.Config) { cfg.Handoff.CommentOnStart = false; cfg.Handoff.CommentOnFailure = false }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, run, issue := setupWorkerRun(t)
			cfg := config.Defaults()
			cfg.Handoff.TransitionOnStart = false
			cfg.Handoff.TransitionOnSuccess = false
			cfg.Handoff.TransitionOnFailure = false
			tt.cfg(&cfg)
			tracker := &fakeTracker{}
			runner := agent.Runner(fakeRunner{})
			if strings.Contains(tt.name, "failure") {
				runner = errorRunner{err: errors.New("agent failed")}
			}
			worker := NewWorker(&cfg, &config.Workflow{Body: "test"}, repo, tracker, runner, app.NewBus(), logrus.New())

			worker.Run(context.Background(), run, issue)

			if len(tracker.comments) != 0 {
				t.Fatalf("comments = %d, want 0", len(tracker.comments))
			}
		})
	}
}

func setupWorkerRun(t *testing.T) (*db.Repository, *models.Run, issues.Issue) {
	t.Helper()
	g, err := db.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Migrate(g); err != nil {
		t.Fatal(err)
	}
	repo := db.NewRepository(g)
	issue := issues.Issue{ID: "issue-1", Identifier: "DEZ-9", Title: "Restore Linear handoff comments", URL: "https://linear.app/issue/DEZ-9", State: "Todo"}
	run := &models.Run{ID: "run-1", IssueID: issue.ID, IssueIdentifier: issue.Identifier, IssueURL: issue.URL, State: string(RunStateClaimed), Attempt: 1, AgentName: "test", WorktreePath: "/tmp/worktree", BranchName: "agent/dez-9"}
	if err := repo.CreateRun(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	return repo, run, issue
}

type fakeRunner struct{}

func (fakeRunner) RunIssue(context.Context, agent.RunIssueRequest) (*agent.RunIssueResult, error) {
	return &agent.RunIssueResult{Summary: "done"}, nil
}

type resultRunner struct{ result *agent.RunIssueResult }

func (r resultRunner) RunIssue(context.Context, agent.RunIssueRequest) (*agent.RunIssueResult, error) {
	return r.result, nil
}

type errorRunner struct{ err error }

func (r errorRunner) RunIssue(context.Context, agent.RunIssueRequest) (*agent.RunIssueResult, error) {
	return nil, r.err
}

type capturingRunner struct{ req agent.RunIssueRequest }

func (r *capturingRunner) RunIssue(_ context.Context, req agent.RunIssueRequest) (*agent.RunIssueResult, error) {
	r.req = req
	return &agent.RunIssueResult{Summary: "done"}, nil
}

type fakeTracker struct {
	candidates       []issues.Issue
	issuesByID       map[string]issues.Issue
	candidateFetches int
	issueFetches     int
	comments         []fakeComment
	lastOptions      issues.FetchOptions
}

type fakeComment struct {
	issueID string
	body    string
}

func (f *fakeTracker) FetchCandidateIssues(_ context.Context, opts issues.FetchOptions) ([]issues.Issue, error) {
	f.candidateFetches++
	f.lastOptions = opts
	return f.candidates, nil
}

func (f *fakeTracker) FetchIssue(_ context.Context, id string) (*issues.Issue, error) {
	f.issueFetches++
	issue, ok := f.issuesByID[id]
	if !ok {
		return nil, nil
	}
	return &issue, nil
}
func (f *fakeTracker) Comment(_ context.Context, issueID, body string) error {
	f.comments = append(f.comments, fakeComment{issueID: issueID, body: body})
	return nil
}
func (*fakeTracker) Transition(context.Context, string, string) error { return nil }
