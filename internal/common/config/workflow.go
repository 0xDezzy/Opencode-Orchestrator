package config

import (
	"os"
	"strings"

	"go.yaml.in/yaml/v3"
)

type Workflow struct {
	FrontMatter map[string]any
	Body        string
}

func LoadWorkflow(path string) (*Workflow, error) {
	if path == "" {
		path = "WORKFLOW.md"
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseWorkflow(string(b))
}
func ParseWorkflow(s string) (*Workflow, error) {
	w := &Workflow{FrontMatter: map[string]any{}, Body: s}
	if !strings.HasPrefix(s, "---\n") {
		return w, nil
	}
	rest := s[4:]
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return w, nil
	}
	fm := rest[:idx]
	body := strings.TrimPrefix(rest[idx+4:], "\n")
	if strings.TrimSpace(fm) != "" {
		if err := yaml.Unmarshal([]byte(fm), &w.FrontMatter); err != nil {
			return nil, err
		}
	}
	w.Body = body
	return w, nil
}
