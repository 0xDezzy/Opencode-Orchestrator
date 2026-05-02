package orchestrator

import (
	"context"
	"time"

	"issue-orchestrator/internal/db"
)

func acquireRunLock(ctx context.Context, repo *db.Repository, issueID, runID string, ttl time.Duration) (bool, error) {
	if ttl <= 0 {
		ttl = 30 * time.Minute
	}
	return repo.AcquireIssueLock(ctx, issueID, runID, ttl)
}

func releaseRunLock(ctx context.Context, repo *db.Repository, issueID, runID string) error {
	return repo.ReleaseIssueLock(ctx, issueID, runID)
}
