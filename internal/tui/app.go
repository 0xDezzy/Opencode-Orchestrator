package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"issue-orchestrator/internal/app"
	"issue-orchestrator/internal/common/logging"
)

func Run(ctx context.Context, c app.Controller, bus app.EventBus, hook *logging.Hook, maxLogs int) error {
	events, unsubscribe := bus.Subscribe(64)
	defer unsubscribe()
	var logs <-chan logging.LogEvent
	if hook != nil {
		logs = hook.Events()
	}
	p := tea.NewProgram(New(ctx, c, logs, events, maxLogs), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
