package tui

import (
	"testing"

	"github.com/charmbracelet/bubbles/table"
)

func TestFitColumnsExpandsWideTables(t *testing.T) {
	for _, view := range []viewName{viewRuns, viewIssues, viewAgents, viewWorkspaces, viewLocks} {
		cols := fitColumns(columnsFor(view), 180)
		if got, want := renderedColumnWidth(cols), 180; got != want {
			t.Fatalf("%s rendered width = %d, want %d", view, got, want)
		}
	}
}

func TestFitColumnsExpandsReadableColumns(t *testing.T) {
	for _, tc := range []struct {
		view  viewName
		title string
	}{
		{viewRuns, "Branch"},
		{viewIssues, "Title"},
		{viewAgents, "Last Event"},
		{viewWorkspaces, "Path"},
		{viewLocks, "Expires"},
	} {
		base := columnWidth(columnsFor(tc.view), tc.title)
		wide := columnWidth(fitColumns(columnsFor(tc.view), 180), tc.title)
		if wide <= base {
			t.Fatalf("%s %q width = %d, want greater than base %d", tc.view, tc.title, wide, base)
		}
	}
}

func TestFitColumnsKeepsCompactColumnsCompact(t *testing.T) {
	cols := fitColumns(columnsFor(viewRuns), 180)
	if got, want := columnWidth(cols, "Attempt"), columnWidth(columnsFor(viewRuns), "Attempt"); got != want {
		t.Fatalf("Attempt width = %d, want %d", got, want)
	}
	if got, want := columnWidth(cols, "Changed"), columnWidth(columnsFor(viewRuns), "Changed"); got != want {
		t.Fatalf("Changed width = %d, want %d", got, want)
	}
}

func TestFitColumnsShrinksNarrowTables(t *testing.T) {
	for _, view := range []viewName{viewRuns, viewIssues, viewAgents, viewWorkspaces, viewLocks} {
		cols := fitColumns(columnsFor(view), 72)
		if got, want := renderedColumnWidth(cols), 72; got > want {
			t.Fatalf("%s rendered width = %d, want <= %d", view, got, want)
		}
		for _, col := range cols {
			if col.Width < 1 {
				t.Fatalf("%s %q width = %d, want >= 1", view, col.Title, col.Width)
			}
		}
	}
}

func renderedColumnWidth(cols []table.Column) int {
	total := len(cols) * 2
	for _, col := range cols {
		total += col.Width
	}
	return total
}

func columnWidth(cols []table.Column, title string) int {
	for _, col := range cols {
		if col.Title == title {
			return col.Width
		}
	}
	return 0
}
