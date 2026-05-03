package orchestrator

import (
	"context"
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

type fakeRunner struct{}

func (fakeRunner) RunIssue(context.Context, agent.RunIssueRequest) (*agent.RunIssueResult, error) {
	return &agent.RunIssueResult{Summary: "done"}, nil
}

type fakeTracker struct {
	candidates       []issues.Issue
	issuesByID       map[string]issues.Issue
	candidateFetches int
	issueFetches     int
}

func (f *fakeTracker) FetchCandidateIssues(context.Context, issues.FetchOptions) ([]issues.Issue, error) {
	f.candidateFetches++
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
func (*fakeTracker) Comment(context.Context, string, string) error    { return nil }
func (*fakeTracker) Transition(context.Context, string, string) error { return nil }
