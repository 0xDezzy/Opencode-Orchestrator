package app

import (
	"context"
	"testing"

	"issue-orchestrator/internal/db"
	"issue-orchestrator/internal/db/models"
)

func TestSnapshotMapping(t *testing.T) {
	g, _ := db.Open(t.TempDir() + "/x.db")
	_ = db.Migrate(g)
	r := db.NewRepository(g)
	_ = r.CreateRun(context.Background(), &models.Run{ID: "abcdefghi", IssueIdentifier: "MUN-1", State: "failed"})
	a := New(r, NewBus())
	s, err := a.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if s.Stats.FailedRuns != 1 || s.Runs[0].ID != "abcdefgh" {
		t.Fatalf("bad snapshot %+v", s)
	}
}
