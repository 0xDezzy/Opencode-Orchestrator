package tui

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"issue-orchestrator/internal/app"
)

type ctrl struct{}

func (ctrl) Snapshot(context.Context) (app.RuntimeSnapshot, error) { return app.RuntimeSnapshot{}, nil }
func (ctrl) RequestSchedulerTick(context.Context) error            { return nil }
func (ctrl) RequestReconcile(context.Context) error                { return nil }
func (ctrl) Shutdown(context.Context) error                        { return nil }

type reconcileCtrl struct{ reconciles int }

func (*reconcileCtrl) Snapshot(context.Context) (app.RuntimeSnapshot, error) {
	return app.RuntimeSnapshot{}, nil
}
func (*reconcileCtrl) RequestSchedulerTick(context.Context) error { return nil }
func (c *reconcileCtrl) RequestReconcile(context.Context) error {
	c.reconciles++
	return nil
}
func (*reconcileCtrl) Shutdown(context.Context) error { return nil }

func TestUpdateViewSwitchAndLogs(t *testing.T) {
	m := New(context.Background(), ctrl{}, nil, nil, 10)
	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	got := mm.(Model)
	if got.view != viewIssues {
		t.Fatal("view did not switch")
	}
	mm, _ = got.Update(LogMsg{})
	got = mm.(Model)
	if len(got.logLines) != 1 {
		t.Fatal("log not appended")
	}
}

func TestCtrlRRequestsReconcile(t *testing.T) {
	c := &reconcileCtrl{}
	m := New(context.Background(), c, nil, nil, 10)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	if cmd == nil {
		t.Fatal("ctrl+r did not return a command")
	}
	if msg := cmd(); msg.(ErrorMsg).Err != nil {
		t.Fatalf("reconcile command returned error: %v", msg.(ErrorMsg).Err)
	}
	if c.reconciles != 1 {
		t.Fatalf("reconciles = %d, want 1", c.reconciles)
	}
}

func TestRefreshTableSwitchesColumnCountsWithoutPanic(t *testing.T) {
	m := New(context.Background(), ctrl{}, nil, nil, 10)
	m.width = 120
	m.snapshot = app.RuntimeSnapshot{
		Runs:   []app.RunView{{ID: "run-1", Issue: "STR-1", State: "running", Attempt: 1}},
		Issues: []app.IssueView{{Issue: "STR-1", Title: "Test issue", LinearState: "Backlog", Labels: "agent", Eligible: "yes"}},
	}
	m.refreshTable()
	m.view = viewIssues

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("refreshTable panicked after switching column counts: %v", r)
		}
	}()
	m.refreshTable()
	_ = m.table.View()
}

func TestDetailsToggleKeepsTableVisible(t *testing.T) {
	m := New(context.Background(), ctrl{}, nil, nil, 10)
	m.width = 120
	m.height = 50
	m.resize()
	tableHeight := m.table.Height()
	m.details = true
	m.resize()
	if m.table.Height() != tableHeight {
		t.Fatalf("details changed table height: got %d want %d", m.table.Height(), tableHeight)
	}
	if m.table.Height() < 6 {
		t.Fatalf("details made table too small: got %d", m.table.Height())
	}
}

func TestDetailsNavigationKeepsIssueTableVisible(t *testing.T) {
	m := New(context.Background(), ctrl{}, nil, nil, 10)
	m.width = 120
	m.height = 50
	m.view = viewIssues
	m.details = true
	m.snapshot = app.RuntimeSnapshot{Issues: []app.IssueView{
		{Issue: "STR-1", Title: "First issue", LinearState: "Backlog", Labels: "agent", Eligible: "yes"},
		{Issue: "STR-2", Title: "Second issue", LinearState: "Backlog", Labels: "agent", Eligible: "yes"},
	}}
	m.resize()
	m.refreshTable()

	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	got := mm.(Model)
	if got.table.Cursor() != 1 {
		t.Fatalf("cursor did not move: got %d", got.table.Cursor())
	}
	if len(got.table.Rows()) != 2 {
		t.Fatalf("rows disappeared after details navigation: got %d", len(got.table.Rows()))
	}
	if got.table.View() == "" {
		t.Fatal("table view is blank after details navigation")
	}
}

func TestDetailsLayoutKeepsTableHeightAcrossViews(t *testing.T) {
	m := New(context.Background(), ctrl{}, nil, nil, 10)
	m.width = 120
	m.height = 50
	m.details = true
	m.snapshot = layoutSnapshot()
	m.resize()
	m.refreshTable()
	tableHeight := m.table.Height()
	viewportHeight := m.viewport.Height

	for _, tc := range []struct {
		key  rune
		view viewName
	}{
		{'r', viewRuns},
		{'i', viewIssues},
		{'a', viewAgents},
		{'w', viewWorkspaces},
		{'l', viewLocks},
	} {
		mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{tc.key}})
		m = mm.(Model)
		if m.view != tc.view {
			t.Fatalf("view did not switch: got %s want %s", m.view, tc.view)
		}
		if m.table.Height() != tableHeight {
			t.Fatalf("%s changed table height: got %d want %d", tc.view, m.table.Height(), tableHeight)
		}
		if m.viewport.Height > viewportHeight {
			t.Fatalf("%s grew log viewport with details open: got %d max %d", tc.view, m.viewport.Height, viewportHeight)
		}
	}
}

func TestDetailsRuntimeEventShrinksLogWithoutMovingTable(t *testing.T) {
	m := New(context.Background(), ctrl{}, nil, nil, 10)
	m.width = 120
	m.height = 50
	m.details = true
	m.snapshot = layoutSnapshot()
	m.resize()
	m.refreshTable()
	tableHeight := m.table.Height()
	viewportHeight := m.viewport.Height

	mm, _ := m.Update(EventMsg{Event: app.RuntimeEvent{Type: "scheduler.tick.finished", CreatedAt: time.Now()}})
	got := mm.(Model)
	if got.table.Height() != tableHeight {
		t.Fatalf("runtime event changed table height: got %d want %d", got.table.Height(), tableHeight)
	}
	if got.viewport.Height >= viewportHeight {
		t.Fatalf("runtime event did not shrink log viewport for last-event details: got %d want less than %d", got.viewport.Height, viewportHeight)
	}
}

func TestRunsDetailsViewKeepsTableHeaderVisible(t *testing.T) {
	m := New(context.Background(), ctrl{}, nil, nil, 10)
	m.width = 120
	m.height = 50
	m.details = true
	m.snapshot = layoutSnapshot()
	m.resize()
	m.refreshTable()

	view := m.View()
	if !strings.Contains(view, "Run ID") || !strings.Contains(view, "Issue") || !strings.Contains(view, "State") {
		t.Fatalf("runs table header is not visible:\n%s", view)
	}
	if height := lipgloss.Height(view); height > m.height {
		t.Fatalf("runs details view exceeds terminal height: got %d want <= %d", height, m.height)
	}
}

func layoutSnapshot() app.RuntimeSnapshot {
	return app.RuntimeSnapshot{
		Runs:       []app.RunView{{ID: "run-1", Issue: "STR-1", State: "running", Attempt: 1, Agent: "opencode", Branch: "branch", Changed: "0", Duration: "1m", Updated: "now"}},
		Issues:     []app.IssueView{{Issue: "STR-1", Title: "Test issue", LinearState: "Backlog", Labels: "agent", Assignee: "dezzy", Updated: "now", Eligible: "yes"}},
		Agents:     []app.AgentView{{RunID: "run-1", Issue: "STR-1", Agent: "opencode", SessionID: "session", Status: "running", LastEvent: "tick", Runtime: "1m", Stall: "no"}},
		Workspaces: []app.WorkspaceView{{Issue: "STR-1", Branch: "branch", Path: "/tmp/work", Status: "ready", Dirty: "no", Created: "now", Updated: "now"}},
		Locks:      []app.LockView{{Issue: "STR-1", RunID: "run-1", Expires: "soon", Age: "1m"}},
	}
}
