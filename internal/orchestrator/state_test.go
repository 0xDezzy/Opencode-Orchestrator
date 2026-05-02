package orchestrator

import "testing"

func TestRunStateTransitions(t *testing.T) {
	if !CanTransition(RunStateClaimed, RunStatePreparing) {
		t.Fatal("claimed should prepare")
	}
	if CanTransition(RunStateSucceeded, RunStateRunningAgent) {
		t.Fatal("terminal transition allowed")
	}
}
