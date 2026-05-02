package tui

import (
	"fmt"
	"strings"

	"issue-orchestrator/internal/app"
)

func (m *Model) appendLog(line string) {
	m.logLines = append(m.logLines, line)
	if m.maxLogs > 0 && len(m.logLines) > m.maxLogs {
		m.logLines = m.logLines[len(m.logLines)-m.maxLogs:]
	}
	m.viewport.SetContent(strings.Join(m.logLines, "\n"))
	m.viewport.GotoBottom()
}

func (m *Model) clearLogs() {
	m.logLines = nil
	m.viewport.SetContent("")
}

func formatRuntimeEvent(ev app.RuntimeEvent) string {
	typ := ev.Type
	switch {
	case strings.Contains(typ, "failed"), strings.Contains(typ, "error"):
		typ = dangerStyle.Render(typ)
	case strings.Contains(typ, "succeeded"), strings.Contains(typ, "finished"):
		typ = successStyle.Render(typ)
	case strings.Contains(typ, "selection"), strings.Contains(typ, "tick"):
		typ = infoStyle.Render(typ)
	default:
		typ = badgeStyle.Render(typ)
	}
	parts := []string{mutedStyle.Render(ev.CreatedAt.Format("15:04:05")), typ}
	if ev.RunID != "" {
		parts = append(parts, "run="+shortID(ev.RunID))
	}
	if ev.IssueID != "" {
		parts = append(parts, "issue="+ev.IssueID)
	}
	if ev.Payload != nil {
		parts = append(parts, fmt.Sprint(ev.Payload))
	}
	return strings.Join(parts, " ")
}

func shortID(v string) string {
	if len(v) > 8 {
		return v[:8]
	}
	return v
}
