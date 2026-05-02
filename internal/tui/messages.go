package tui

import (
	"time"

	"issue-orchestrator/internal/app"
	"issue-orchestrator/internal/common/logging"
)

type TickMsg struct{ Time time.Time }
type SnapshotMsg struct{ Snapshot app.RuntimeSnapshot }
type LogMsg struct{ Event logging.LogEvent }
type EventMsg struct{ Event app.RuntimeEvent }
type ErrorMsg struct{ Err error }
