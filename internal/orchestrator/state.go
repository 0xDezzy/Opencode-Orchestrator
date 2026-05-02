package orchestrator

type RunState string

const (
	RunStateClaimed      RunState = "claimed"
	RunStatePreparing    RunState = "preparing"
	RunStateRunningAgent RunState = "running_agent"
	RunStateValidating   RunState = "validating"
	RunStateSucceeded    RunState = "succeeded"
	RunStateRetryQueued  RunState = "retry_queued"
	RunStateFailed       RunState = "failed"
	RunStateCanceled     RunState = "canceled"
)

func CanTransition(from, to RunState) bool {
	if from == to {
		return true
	}
	switch from {
	case RunStateClaimed:
		return to == RunStatePreparing || to == RunStateCanceled
	case RunStatePreparing:
		return to == RunStateRunningAgent || to == RunStateFailed || to == RunStateCanceled
	case RunStateRunningAgent:
		return to == RunStateValidating || to == RunStateFailed || to == RunStateCanceled
	case RunStateValidating:
		return to == RunStateSucceeded || to == RunStateFailed
	}
	return false
}
