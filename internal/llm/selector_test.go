package llm

import (
	"reflect"
	"testing"
	"time"

	"issue-orchestrator/internal/issues"
)

func TestParseSelectionExtractsJSONObject(t *testing.T) {
	got, err := ParseSelection("```json\n{\"selected\":[\"a\",\"b\"],\"rationale\":\"best next work\"}\n```")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Selected, []string{"a", "b"}) {
		t.Fatalf("selected = %#v", got.Selected)
	}
	if got.Rationale != "best next work" {
		t.Fatalf("rationale = %q", got.Rationale)
	}
}

func TestClampSelectionDropsInvalidDuplicateAndOverflow(t *testing.T) {
	req := SelectionRequest{Slots: 2, Issues: []issues.Issue{{ID: "a"}, {ID: "b"}}}
	got := clampSelection(SelectionResult{Selected: []string{"missing", "a", "a", "b", "c"}}, req)
	if !reflect.DeepEqual(got.Selected, []string{"a", "b"}) {
		t.Fatalf("selected = %#v", got.Selected)
	}
}

func TestDeterministicSelectorUsesPriorityCreatedIdentifier(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	req := SelectionRequest{Slots: 2, Issues: []issues.Issue{
		{ID: "late", Identifier: "APP-3", Priority: "2", CreatedAt: base.Add(time.Hour)},
		{ID: "high", Identifier: "APP-2", Priority: "1", CreatedAt: base.Add(2 * time.Hour)},
		{ID: "old", Identifier: "APP-1", Priority: "2", CreatedAt: base},
	}}
	got, err := (DeterministicSelector{}).SelectTasks(t.Context(), req)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Selected, []string{"high", "old"}) {
		t.Fatalf("selected = %#v", got.Selected)
	}
}
