package linear

import (
	"testing"

	"issue-orchestrator/internal/issues"
)

type fakeState struct{ Name string }
type fakeTeam struct{ Key string }
type fakeLabel struct{ Name string }
type fakeLabels struct{ Nodes []*fakeLabel }
type fakeProject struct{ SlugID, Name, ID string }
type fakeUser struct{ DisplayName string }
type fakeIssue struct {
	ID, Identifier, Title, Description, URL string
	Number                                  int
	State                                   fakeState
	Team                                    fakeTeam
	Labels                                  fakeLabels
	Project                                 fakeProject
	Assignee                                fakeUser
}

func TestMapAnyIssue(t *testing.T) {
	got := mapAnyIssue(fakeIssue{ID: "id", Title: "title", Number: 123, State: fakeState{"Ready"}, Team: fakeTeam{"MUN"}})
	if got.Identifier != "MUN-123" || got.State != "Ready" {
		t.Fatalf("bad map: %+v", got)
	}
}

func TestMapAnyIssueMapsLinearRelations(t *testing.T) {
	got := mapAnyIssue(fakeIssue{
		ID:         "id",
		Identifier: "STR-7",
		Title:      "title",
		State:      fakeState{"Ready for Agent"},
		Team:       fakeTeam{"STR"},
		Labels:     fakeLabels{Nodes: []*fakeLabel{{Name: "agent"}, {Name: "backend"}}},
		Project:    fakeProject{SlugID: "huginn-9428c325653d", Name: "Huginn"},
		Assignee:   fakeUser{DisplayName: "Dezzy"},
	})
	if got.ProjectName != "huginn-9428c325653d" || got.Assignee != "Dezzy" {
		t.Fatalf("relations not mapped: %+v", got)
	}
	if len(got.Labels) != 2 || got.Labels[0] != "agent" || got.Labels[1] != "backend" {
		t.Fatalf("labels not mapped: %+v", got.Labels)
	}
}

func TestEligibleUsesProjectAndLabelsCaseInsensitive(t *testing.T) {
	iss := mapAnyIssue(fakeIssue{ID: "id", Identifier: "STR-7", State: fakeState{"Ready for Agent"}, Team: fakeTeam{"STR"}, Labels: fakeLabels{Nodes: []*fakeLabel{{Name: "Agent"}}}, Project: fakeProject{SlugID: "huginn-9428c325653d"}})
	if !eligible(iss, issues.FetchOptions{TeamKey: "str", ProjectName: "HUGINN-9428C325653D", ActiveStates: []string{"ready for agent"}, IncludeLabels: []string{"agent"}}) {
		t.Fatalf("issue should be eligible: %+v", iss)
	}
	if eligible(iss, issues.FetchOptions{ProjectName: "other"}) {
		t.Fatal("issue should not match a different project")
	}
}

func TestProjectMatchesCompositeLinearSlug(t *testing.T) {
	if !projectMatches("9428c325653d", "huginn-9428c325653d") {
		t.Fatal("expected Linear URL-style project slug to match slug id")
	}
	if !projectMatches("Huginn", "huginn") {
		t.Fatal("expected project name to match case-insensitively")
	}
}
