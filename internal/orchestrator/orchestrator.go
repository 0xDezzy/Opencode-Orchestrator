package orchestrator

import (
	"context"

	"issue-orchestrator/internal/common/config"
)

type Orchestrator struct {
	Scheduler *Scheduler
	Config    *config.Config
}

func (o *Orchestrator) Run(ctx context.Context, once bool) error {
	if once {
		return o.Scheduler.Tick(ctx)
	}
	return o.Scheduler.Run(ctx)
}
