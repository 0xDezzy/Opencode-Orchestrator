package tui

import (
	"context"
	"errors"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"issue-orchestrator/internal/app"
	"issue-orchestrator/internal/common/logging"
)

type viewName string

const (
	viewRuns       viewName = "runs"
	viewIssues     viewName = "issues"
	viewAgents     viewName = "agents"
	viewWorkspaces viewName = "workspaces"
	viewLocks      viewName = "locks"
)

type Model struct {
	ctx           context.Context
	controller    app.Controller
	logs          <-chan logging.LogEvent
	events        <-chan app.RuntimeEvent
	width, height int
	table         table.Model
	viewport      viewport.Model
	help          help.Model
	keys          keyMap
	snapshot      app.RuntimeSnapshot
	view          viewName
	paused        bool
	logLines      []string
	maxLogs       int
	details       bool
	filter        string
	filtering     bool
	runtimeEvents []app.RuntimeEvent
}

func New(ctx context.Context, c app.Controller, logs <-chan logging.LogEvent, events <-chan app.RuntimeEvent, maxLogs int) Model {
	cols := []table.Column{{Title: "Run ID", Width: 10}, {Title: "Issue", Width: 12}, {Title: "State", Width: 16}, {Title: "Attempt", Width: 7}, {Title: "Agent", Width: 10}, {Title: "Branch", Width: 24}, {Title: "Changed", Width: 8}, {Title: "Duration", Width: 10}, {Title: "Updated", Width: 20}}
	t := styledTable(table.New(table.WithColumns(cols), table.WithFocused(true), table.WithHeight(10)))
	keys := defaultKeys()
	h := help.New()
	h.ShowAll = true
	h.Styles.ShortKey = badgeStyle
	h.Styles.ShortDesc = footerStyle
	h.Styles.FullKey = badgeStyle
	h.Styles.FullDesc = footerStyle
	return Model{ctx: ctx, controller: c, logs: logs, events: events, table: t, viewport: viewport.New(80, 10), help: h, keys: keys, view: viewRuns, maxLogs: maxLogs}
}
func (m Model) Init() tea.Cmd { return tea.Batch(m.snapshotCmd(), m.logCmd(), m.eventCmd()) }
func (m Model) snapshotCmd() tea.Cmd {
	return func() tea.Msg {
		s, err := m.controller.Snapshot(m.ctx)
		if err != nil {
			return ErrorMsg{err}
		}
		return SnapshotMsg{s}
	}
}
func (m Model) logCmd() tea.Cmd {
	return func() tea.Msg {
		if m.logs == nil {
			return nil
		}
		select {
		case ev, ok := <-m.logs:
			if !ok {
				return ErrorMsg{errors.New("log stream closed")}
			}
			return LogMsg{ev}
		case <-m.ctx.Done():
			return nil
		}
	}
}

func (m Model) eventCmd() tea.Cmd {
	return func() tea.Msg {
		if m.events == nil {
			return nil
		}
		select {
		case ev, ok := <-m.events:
			if !ok {
				return nil
			}
			return EventMsg{ev}
		case <-m.ctx.Done():
			return nil
		}
	}
}
