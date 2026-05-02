package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"

	"issue-orchestrator/internal/app"
)

func rowsFor(v viewName, s app.RuntimeSnapshot) []table.Row {
	rows := []table.Row{}
	switch v {
	case viewRuns:
		for _, r := range s.Runs {
			rows = append(rows, table.Row{r.ID, r.Issue, r.State, fmt.Sprint(r.Attempt), r.Agent, r.Branch, r.Changed, r.Duration, r.Updated})
		}
	case viewIssues:
		for _, i := range s.Issues {
			rows = append(rows, table.Row{i.Issue, i.Title, i.LinearState, i.Labels, i.Assignee, i.Updated, i.Eligible})
		}
	case viewAgents:
		for _, a := range s.Agents {
			rows = append(rows, table.Row{a.RunID, a.Issue, a.Agent, a.SessionID, a.Status, a.LastEvent, a.Runtime, a.Stall})
		}
	case viewWorkspaces:
		for _, w := range s.Workspaces {
			rows = append(rows, table.Row{w.Issue, w.Branch, w.Path, w.Status, w.Dirty, w.Created, w.Updated})
		}
	case viewLocks:
		for _, l := range s.Locks {
			rows = append(rows, table.Row{l.Issue, l.RunID, l.Expires, l.Age})
		}
	}
	return rows
}

func (m *Model) refreshTable() {
	rows := rowsFor(m.view, m.snapshot)
	if m.filter != "" {
		needle := strings.ToLower(m.filter)
		filtered := rows[:0]
		for _, row := range rows {
			if strings.Contains(strings.ToLower(strings.Join(row, " ")), needle) {
				filtered = append(filtered, row)
			}
		}
		rows = filtered
	}
	cursor := m.table.Cursor()
	height := m.tableHeight()
	if height < 2 {
		height = 10
	}
	m.table = styledTable(table.New(
		table.WithColumns(fitColumns(columnsFor(m.view), m.width-8)),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(height),
	))
	if len(rows) > 0 {
		if cursor > len(rows)-1 {
			cursor = len(rows) - 1
		}
		m.table.SetCursor(cursor)
	}
}

func fitColumns(cols []table.Column, width int) []table.Column {
	if width <= 0 || len(cols) == 0 {
		return cols
	}
	out := make([]table.Column, len(cols))
	copy(out, cols)
	total := 0
	for _, c := range out {
		total += c.Width
	}
	separators := len(out) * 2
	budget := width - separators
	if budget <= 0 || total <= budget {
		return out
	}
	for i := range out {
		w := out[i].Width * budget / total
		if w < 6 {
			w = 6
		}
		out[i].Width = w
	}
	return out
}

func columnsFor(v viewName) []table.Column {
	switch v {
	case viewIssues:
		return []table.Column{{"Issue", 12}, {"Title", 30}, {"Linear State", 16}, {"Labels", 18}, {"Assignee", 14}, {"Updated", 20}, {"Eligible", 8}}
	case viewAgents:
		return []table.Column{{"Run ID", 10}, {"Issue", 12}, {"Agent", 10}, {"Session ID", 16}, {"Status", 12}, {"Last Event", 24}, {"Runtime", 10}, {"Stall", 10}}
	case viewWorkspaces:
		return []table.Column{{"Issue", 12}, {"Branch", 24}, {"Path", 40}, {"Status", 12}, {"Dirty", 8}, {"Created", 20}, {"Updated", 20}}
	case viewLocks:
		return []table.Column{{"Issue", 24}, {"Run ID", 12}, {"Expires", 22}, {"Age", 12}}
	default:
		return []table.Column{{"Run ID", 10}, {"Issue", 12}, {"State", 16}, {"Attempt", 7}, {"Agent", 10}, {"Branch", 24}, {"Changed", 8}, {"Duration", 10}, {"Updated", 20}}
	}
}
