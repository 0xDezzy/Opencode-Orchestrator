package config

import "testing"

func TestValidateRequiresCoreFields(t *testing.T) {
	c := Defaults()
	c.SQLite.Path = ""
	if err := c.Validate(false); err == nil {
		t.Fatal("expected sqlite validation error")
	}
}
func TestParseWorkflowFrontMatter(t *testing.T) {
	w, err := ParseWorkflow("---\nagent:\n  require_tests: true\n---\nHello {{ .Issue.Identifier }}")
	if err != nil {
		t.Fatal(err)
	}
	if w.FrontMatter["agent"] == nil {
		t.Fatal("front matter not parsed")
	}
	if w.Body == "" {
		t.Fatal("body missing")
	}
}
