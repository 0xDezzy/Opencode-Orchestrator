package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"issue-orchestrator/internal/app"
	"issue-orchestrator/internal/common/config"
	"issue-orchestrator/internal/db/models"
	"issue-orchestrator/internal/issues"
)

func TestReconcilerDryRunDoesNotMutateRemovedIssue(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()
	local := models.IssueSnapshot{IssueID: "issue-1", Identifier: "DEZ-11", Title: "Missing issue", URL: "https://linear.app/issue/DEZ-11", State: "Human Review", FetchedAt: time.Now().Add(-2 * time.Hour)}
	if err := repo.UpsertIssueSnapshot(ctx, &local); err != nil {
		t.Fatal(err)
	}
	if err := repo.UpsertWorkspace(ctx, &models.Workspace{IssueID: local.IssueID, IssueIdentifier: local.Identifier, BranchName: "agent/dez-11", Status: "active"}); err != nil {
		t.Fatal(err)
	}
	if ok, err := repo.AcquireIssueLock(ctx, local.IssueID, "run-1", time.Hour); err != nil || !ok {
		t.Fatalf("acquire lock ok=%v err=%v", ok, err)
	}

	cfg := config.Defaults()
	reconciler := NewReconciler(&cfg, repo, &fakeTracker{issuesByID: map[string]issues.Issue{}}, app.NewBus(), logrus.New())
	summary := reconciler.Reconcile(ctx, ReconcileOptions{DryRun: true})

	if summary.IssuesRemoved != 1 {
		t.Fatalf("removed issues = %d, want 1", summary.IssuesRemoved)
	}
	if summary.LocksReleased != 1 {
		t.Fatalf("locks released = %d, want 1", summary.LocksReleased)
	}
	if summary.WorktreesRemoved != 1 {
		t.Fatalf("worktrees removed = %d, want 1", summary.WorktreesRemoved)
	}
	snapshot, err := repo.RuntimeSnapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got := snapshot.Issues[0].State; got != "Human Review" {
		t.Fatalf("snapshot state = %q, want Human Review", got)
	}
	if len(snapshot.Locks) != 1 {
		t.Fatalf("locks = %d, want 1", len(snapshot.Locks))
	}
	if got := snapshot.Workspaces[0].Status; got != "active" {
		t.Fatalf("workspace status = %q, want active", got)
	}
}

func TestReconcilerTargetsSingleIssueByIdentifier(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()
	for _, local := range []models.IssueSnapshot{
		{IssueID: "issue-1", Identifier: "DEZ-11", Title: "Old one", State: "Human Review", FetchedAt: time.Now().Add(-2 * time.Hour)},
		{IssueID: "issue-2", Identifier: "DEZ-12", Title: "Old two", State: "Human Review", FetchedAt: time.Now().Add(-2 * time.Hour)},
	} {
		local := local
		if err := repo.UpsertIssueSnapshot(ctx, &local); err != nil {
			t.Fatal(err)
		}
	}

	cfg := config.Defaults()
	tracker := &fakeTracker{issuesByID: map[string]issues.Issue{
		"issue-1": {ID: "issue-1", Identifier: "DEZ-11", Title: "New one", State: "Done"},
		"issue-2": {ID: "issue-2", Identifier: "DEZ-12", Title: "New two", State: "Done"},
	}}
	reconciler := NewReconciler(&cfg, repo, tracker, app.NewBus(), logrus.New())
	summary := reconciler.Reconcile(ctx, ReconcileOptions{Issue: "DEZ-11"})

	if summary.IssuesRefreshed != 1 {
		t.Fatalf("refreshed issues = %d, want 1", summary.IssuesRefreshed)
	}
	if tracker.issueFetches != 1 {
		t.Fatalf("issue fetches = %d, want 1", tracker.issueFetches)
	}
	snapshot, err := repo.RuntimeSnapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	states := map[string]string{}
	for _, issue := range snapshot.Issues {
		states[issue.Identifier] = issue.State
	}
	if states["DEZ-11"] != "Done" {
		t.Fatalf("DEZ-11 state = %q, want Done", states["DEZ-11"])
	}
	if states["DEZ-12"] != "Human Review" {
		t.Fatalf("DEZ-12 state = %q, want Human Review", states["DEZ-12"])
	}
}

func TestReconcilerMarksMissingWorkspacePathRemoved(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()
	local := models.IssueSnapshot{IssueID: "issue-1", Identifier: "DEZ-11", Title: "Done issue", State: "Done", FetchedAt: time.Now()}
	if err := repo.UpsertIssueSnapshot(ctx, &local); err != nil {
		t.Fatal(err)
	}
	if err := repo.UpsertWorkspace(ctx, &models.Workspace{IssueID: local.IssueID, IssueIdentifier: local.Identifier, Path: "/path/that/does/not/exist", BranchName: "agent/dez-11", Status: "active"}); err != nil {
		t.Fatal(err)
	}

	cfg := config.Defaults()
	reconciler := NewReconciler(&cfg, repo, &fakeTracker{issuesByID: map[string]issues.Issue{}}, app.NewBus(), logrus.New())
	summary := reconciler.Reconcile(ctx, ReconcileOptions{})

	if summary.WorktreesRemoved != 1 {
		t.Fatalf("worktrees removed = %d, want 1", summary.WorktreesRemoved)
	}
	workspaces, err := repo.ListWorkspaces(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got := workspaces[0].Status; got != "removed" {
		t.Fatalf("workspace status = %q, want removed", got)
	}
}
