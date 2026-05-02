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
