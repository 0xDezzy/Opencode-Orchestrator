package orchestrator

import (
	"context"
	"time"

	"issue-orchestrator/internal/app"
	"issue-orchestrator/internal/db/models"
)

func activeRunState(state string) bool {
	switch RunState(state) {
	case RunStateClaimed, RunStatePreparing, RunStateRunningAgent, RunStateValidating, RunStateRetryQueued:
		return true
	default:
		return false
	}
}

func runNeedsHeartbeat(run models.Run) bool {
	return activeRunState(run.State) && run.LastHeartbeatAt == nil
}

func (s *Scheduler) reconcileClaimedRuns(ctx context.Context) {
	cutoff := time.Now().Add(-staleClaimedRunAge(s.cfg.Scheduler.PollInterval))
	var runs []models.Run
	if err := s.repo.DB().WithContext(ctx).Where("state = ? AND updated_at < ?", string(RunStateClaimed), cutoff).Find(&runs).Error; err != nil {
		return
	}
	for _, run := range runs {
		msg := "claimed run never reached worker; marking stale so it can be rescheduled"
		_ = s.repo.FinishRun(ctx, run.ID, string(RunStateFailed), msg)
		_ = releaseRunLock(ctx, s.repo, run.IssueID, run.ID)
		_ = s.repo.AppendEvent(ctx, &models.Event{RunID: run.ID, IssueID: run.IssueID, Type: "run.stale_claimed", Source: "scheduler", Message: msg})
		s.bus.Publish(app.RuntimeEvent{Type: "run.stale_claimed", RunID: run.ID, IssueID: run.IssueID, Payload: msg})
	}
}

func staleClaimedRunAge(poll time.Duration) time.Duration {
	threshold := 2 * poll
	if threshold < time.Minute {
		return time.Minute
	}
	return threshold
}
