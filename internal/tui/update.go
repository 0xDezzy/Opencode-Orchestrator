package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"issue-orchestrator/internal/common/logging"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resize()
		m.refreshTable()
	case tea.KeyMsg:
		if m.filtering {
			switch msg.String() {
			case "enter", "esc":
				m.filtering = false
				m.refreshTable()
			case "backspace", "ctrl+h":
				if len(m.filter) > 0 {
					m.filter = m.filter[:len(m.filter)-1]
					m.refreshTable()
				}
			case "ctrl+u":
				m.filter = ""
				m.refreshTable()
			default:
				if len(msg.Runes) > 0 {
					m.filter += string(msg.Runes)
					m.refreshTable()
				}
			}
			return m, nil
		}
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.ToggleHelp):
			m.help.ShowAll = !m.help.ShowAll
		case key.Matches(msg, m.keys.Runs):
			m.view = viewRuns
			m.refreshTable()
			m.resize()
		case key.Matches(msg, m.keys.Issues):
			m.view = viewIssues
			m.refreshTable()
			m.resize()
		case key.Matches(msg, m.keys.Agents):
			m.view = viewAgents
			m.refreshTable()
			m.resize()
		case key.Matches(msg, m.keys.Workspaces):
			m.view = viewWorkspaces
			m.refreshTable()
			m.resize()
		case key.Matches(msg, m.keys.Locks):
			m.view = viewLocks
			m.refreshTable()
			m.resize()
		case key.Matches(msg, m.keys.Filter):
			m.filtering = true
		case key.Matches(msg, m.keys.Close):
			m.details = false
			m.filtering = false
			m.resize()
		case key.Matches(msg, m.keys.Details):
			m.details = !m.details
			m.resize()
		case key.Matches(msg, m.keys.Top):
			m.table.GotoTop()
		case key.Matches(msg, m.keys.Bottom):
			m.table.GotoBottom()
		case key.Matches(msg, m.keys.ClearLogs):
			m.clearLogs()
		case key.Matches(msg, m.keys.Pause):
			m.paused = !m.paused
		case key.Matches(msg, m.keys.Tick):
			return m, func() tea.Msg { return ErrorMsg{m.controller.RequestSchedulerTick(m.ctx)} }
		}
	case SnapshotMsg:
		m.snapshot = msg.Snapshot
		m.refreshTable()
		m.resize()
		return m, tea.Tick(time.Second, func(t time.Time) tea.Msg { return TickMsg{t} })
	case TickMsg:
		return m, m.snapshotCmd()
	case LogMsg:
		if !m.paused {
			m.appendLog(logging.FormatLogEvent(msg.Event))
		}
		return m, m.logCmd()
	case EventMsg:
		m.runtimeEvents = append(m.runtimeEvents, msg.Event)
		if len(m.runtimeEvents) > 100 {
			m.runtimeEvents = m.runtimeEvents[len(m.runtimeEvents)-100:]
		}
		if m.details {
			m.resize()
		}
		if !m.paused {
			m.appendLog(formatRuntimeEvent(msg.Event))
		}
		return m, tea.Batch(m.eventCmd(), m.snapshotCmd())
	case ErrorMsg:
		if msg.Err != nil {
			m.appendLog(errStyle.Render(msg.Err.Error()))
		}
	}
	m.table, cmd = m.table.Update(msg)
	if msg, ok := msg.(tea.KeyMsg); ok && m.details && m.tableNavigationKey(msg) {
		m.refreshTable()
	}
	return m, cmd
}

func (m Model) tableNavigationKey(msg tea.KeyMsg) bool {
	return key.Matches(msg, m.table.KeyMap.LineUp) ||
		key.Matches(msg, m.table.KeyMap.LineDown) ||
		key.Matches(msg, m.table.KeyMap.PageUp) ||
		key.Matches(msg, m.table.KeyMap.PageDown) ||
		key.Matches(msg, m.table.KeyMap.HalfPageUp) ||
		key.Matches(msg, m.table.KeyMap.HalfPageDown) ||
		key.Matches(msg, m.table.KeyMap.GotoTop) ||
		key.Matches(msg, m.table.KeyMap.GotoBottom)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
