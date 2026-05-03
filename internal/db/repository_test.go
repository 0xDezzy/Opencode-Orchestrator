package db

import (
	"context"
	"testing"
	"time"

	"issue-orchestrator/internal/db/models"
)

func TestRepositoryLockAndRuntime(t *testing.T) {
	g, err := Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	if err := Migrate(g); err != nil {
		t.Fatal(err)
	}
	r := NewRepository(g)
	ok, err := r.AcquireIssueLock(context.Background(), "i1", "r1", time.Minute)
	if err != nil || !ok {
		t.Fatalf("lock: %v %v", ok, err)
	}
	ok, err = r.AcquireIssueLock(context.Background(), "i1", "r2", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("double lock acquired")
	}
	if err := r.CreateRun(context.Background(), &models.Run{ID: "r1", IssueID: "i1", State: "claimed"}); err != nil {
		t.Fatal(err)
	}
	s, err := r.RuntimeSnapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Runs) != 1 {
		t.Fatal("run missing from snapshot")
	}
}

func TestRuntimeSnapshotHidesRemovedIssueSnapshots(t *testing.T) {
	g, err := Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	if err := Migrate(g); err != nil {
		t.Fatal(err)
	}
	r := NewRepository(g)
	ctx := context.Background()
	for _, snapshot := range []models.IssueSnapshot{
		{IssueID: "i1", Identifier: "DEZ-1", State: "Ready for Agent", FetchedAt: time.Now()},
		{IssueID: "i2", Identifier: "DEZ-2", State: "deleted", FetchedAt: time.Now()},
		{IssueID: "i3", Identifier: "DEZ-3", State: "archived", FetchedAt: time.Now()},
	} {
		snapshot := snapshot
		if err := r.UpsertIssueSnapshot(ctx, &snapshot); err != nil {
			t.Fatal(err)
		}
	}

	snapshot, err := r.RuntimeSnapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Issues) != 1 || snapshot.Issues[0].Identifier != "DEZ-1" {
		t.Fatalf("visible issues = %+v, want only DEZ-1", snapshot.Issues)
	}
}

func TestPruneRemovedIssueDeletesSnapshotLocksAndRemovedWorkspace(t *testing.T) {
	g, err := Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	if err := Migrate(g); err != nil {
		t.Fatal(err)
	}
	r := NewRepository(g)
	ctx := context.Background()
	if err := r.UpsertIssueSnapshot(ctx, &models.IssueSnapshot{IssueID: "i1", Identifier: "DEZ-1", State: "deleted", FetchedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	if err := r.UpsertWorkspace(ctx, &models.Workspace{IssueID: "i1", IssueIdentifier: "DEZ-1", Status: "removed"}); err != nil {
		t.Fatal(err)
	}
	if ok, err := r.AcquireIssueLock(ctx, "i1", "run-1", time.Hour); err != nil || !ok {
		t.Fatalf("acquire lock ok=%v err=%v", ok, err)
	}

	if err := r.PruneRemovedIssue(ctx, "i1"); err != nil {
		t.Fatal(err)
	}
	all, err := r.ListIssueSnapshots(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 0 {
		t.Fatalf("issue snapshots = %d, want 0", len(all))
	}
	workspaces, err := r.ListWorkspaces(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(workspaces) != 0 {
		t.Fatalf("workspaces = %d, want 0", len(workspaces))
	}
	locks, err := r.ListLocks(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(locks) != 0 {
		t.Fatalf("locks = %d, want 0", len(locks))
	}
}
