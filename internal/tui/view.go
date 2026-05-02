package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	contentWidth := maxInt(40, m.width-4)
	filter := mutedStyle.Render("none")
	if m.filter != "" || m.filtering {
		filter = warningStyle.Render(fmt.Sprintf("%q", m.filter))
		if m.filtering {
			filter = warningStyle.Render(fmt.Sprintf("%q_", m.filter))
		}
	}

	header := headerStyle.Width(contentWidth).Render(lipgloss.JoinHorizontal(
		lipgloss.Top,
		badgeStyle.Render("orchestrator"),
		mutedStyle.Render("  view="),
		infoStyle.Render(string(m.view)),
		mutedStyle.Render("  filter="),
		filter,
		mutedStyle.Render("  active="),
		successStyle.Render(fmt.Sprint(m.snapshot.Stats.ActiveRuns)),
		mutedStyle.Render(" queued="),
		infoStyle.Render(fmt.Sprint(m.snapshot.Stats.QueuedRuns)),
		mutedStyle.Render(" failed="),
		dangerStyle.Render(fmt.Sprint(m.snapshot.Stats.FailedRuns)),
		mutedStyle.Render(" locks="),
		warningStyle.Render(fmt.Sprint(m.snapshot.Stats.LockedIssues)),
	))

	tableTitle := panelTitleStyle.Render(fmt.Sprintf("%s · %d rows", viewLabel(m.view), len(m.table.Rows())))
	tableBody := strings.TrimRight(m.table.View(), "\n")
	if len(m.table.Rows()) == 0 {
		tableBody = mutedStyle.Render("No rows for this view yet.")
	}
	tablePanel := panelStyle.Width(contentWidth).Render(tableTitle + "\n" + tableBody)

	sections := []string{header}
	sections = append(sections, tablePanel)
	if m.details {
		sections = append(sections, m.detailView(contentWidth))
	}
	footer := footerStyle.Render(m.help.View(m.keys))
	logViewport := m.viewport
	if m.height > 0 {
		usedHeight := lipgloss.Height(strings.Join(append(sections, footer), "\n")) + 2
		logViewport.Height = maxInt(1, m.height-usedHeight-4)
	}
	logs := logViewport.View()
	if strings.TrimSpace(logs) == "" {
		logs = mutedStyle.Render("Waiting for scheduler, agent, or log events...")
	}
	logTitle := panelTitleStyle.Render("Event Stream")
	if m.paused {
		logTitle += " " + warningStyle.Render("paused")
	}
	sections = append(sections, panelStyle.Width(contentWidth).Render(logTitle+"\n"+logs))
	sections = append(sections, footer)
	return appStyle.Render(strings.Join(sections, "\n"))
}

func viewLabel(v viewName) string {
	switch v {
	case viewRuns:
		return "Runs"
	case viewIssues:
		return "Issues"
	case viewAgents:
		return "Agents"
	case viewWorkspaces:
		return "Workspaces"
	case viewLocks:
		return "Locks"
	default:
		return string(v)
	}
}
