package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"issue-orchestrator/internal/app"
	"issue-orchestrator/internal/common/config"
	"issue-orchestrator/internal/db"
	"issue-orchestrator/internal/db/models"
	"issue-orchestrator/internal/issues"
)

func TestSchedulerTickReconcilesTerminalSnapshotWithoutDispatch(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()
	local := models.IssueSnapshot{IssueID: "issue-1", Identifier: "DEZ-6", Title: "Terminal issue", URL: "https://linear.app/issue/DEZ-6", State: "Human Review", FetchedAt: time.Now()}
	if err := repo.UpsertIssueSnapshot(ctx, &local); err != nil {
		t.Fatal(err)
	}

	cfg := config.Defaults()
	tracker := &fakeTracker{issuesByID: map[string]issues.Issue{
		"issue-1": {ID: "issue-1", Identifier: "DEZ-6", Title: "Terminal issue", URL: local.URL, State: "Done"},
	}}
	scheduler := NewScheduler(&cfg, repo, tracker, nil, app.NewBus(), logrus.New())

	if err := scheduler.Tick(ctx); err != nil {
		t.Fatal(err)
	}

	snapshot, err := repo.RuntimeSnapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got := snapshot.Issues[0].State; got != "Done" {
		t.Fatalf("snapshot state = %q, want Done", got)
	}
	if tracker.candidateFetches != 1 {
		t.Fatalf("candidate fetches = %d, want 1", tracker.candidateFetches)
	}
	if tracker.issueFetches != 1 {
		t.Fatalf("issue fetches = %d, want 1", tracker.issueFetches)
	}
	if len(snapshot.Runs) != 0 {
		t.Fatalf("runs = %d, want 0", len(snapshot.Runs))
	}
}

func TestSchedulerTickPreservesTerminalWorkspaceWhenConfigured(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()
	local := models.IssueSnapshot{IssueID: "issue-1", Identifier: "DEZ-6", Title: "Terminal issue", URL: "https://linear.app/issue/DEZ-6", State: "Human Review", FetchedAt: time.Now()}
	if err := repo.UpsertIssueSnapshot(ctx, &local); err != nil {
		t.Fatal(err)
	}
	if err := repo.UpsertWorkspace(ctx, &models.Workspace{IssueID: local.IssueID, IssueIdentifier: local.Identifier, Path: t.TempDir(), BranchName: "agent/dez-6", Status: "active"}); err != nil {
		t.Fatal(err)
	}

	cfg := config.Defaults()
	cfg.Workspace.PreserveSuccessful = true
	tracker := &fakeTracker{issuesByID: map[string]issues.Issue{
		"issue-1": {ID: "issue-1", Identifier: "DEZ-6", Title: "Terminal issue", URL: local.URL, State: "Done"},
	}}
	scheduler := NewScheduler(&cfg, repo, tracker, nil, app.NewBus(), logrus.New())

	if err := scheduler.Tick(ctx); err != nil {
		t.Fatal(err)
	}

	workspaces, err := repo.ListWorkspaces(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got := workspaces[0].Status; got != "preserved" {
		t.Fatalf("workspace status = %q, want preserved", got)
	}
	var events []models.Event
	if err := repo.DB().WithContext(ctx).Where("type = ?", "worktree.terminal_preserved").Find(&events).Error; err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("terminal preserved events = %d, want 1", len(events))
	}
}

func TestSchedulerTickPreservesDirtyTerminalWorkspace(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()
	local := models.IssueSnapshot{IssueID: "issue-1", Identifier: "DEZ-6", Title: "Terminal issue", URL: "https://linear.app/issue/DEZ-6", State: "Human Review", FetchedAt: time.Now()}
	if err := repo.UpsertIssueSnapshot(ctx, &local); err != nil {
		t.Fatal(err)
	}
	if err := repo.UpsertWorkspace(ctx, &models.Workspace{IssueID: local.IssueID, IssueIdentifier: local.Identifier, Path: t.TempDir(), BranchName: "agent/dez-6", Status: "active", Dirty: true}); err != nil {
		t.Fatal(err)
	}

	cfg := config.Defaults()
	cfg.Workspace.PreserveSuccessful = false
	tracker := &fakeTracker{issuesByID: map[string]issues.Issue{
		"issue-1": {ID: "issue-1", Identifier: "DEZ-6", Title: "Terminal issue", URL: local.URL, State: "Done"},
	}}
	scheduler := NewScheduler(&cfg, repo, tracker, nil, app.NewBus(), logrus.New())

	if err := scheduler.Tick(ctx); err != nil {
		t.Fatal(err)
	}

	workspaces, err := repo.ListWorkspaces(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got := workspaces[0].Status; got != "preserved_dirty" {
		t.Fatalf("workspace status = %q, want preserved_dirty", got)
	}
	if !workspaces[0].Dirty {
		t.Fatal("workspace dirty = false, want true")
	}
}

func newTestRepository(t *testing.T) *db.Repository {
	t.Helper()
	g, err := db.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Migrate(g); err != nil {
		t.Fatal(err)
	}
	return db.NewRepository(g)
}
