package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	goai "github.com/zendev-sh/goai"
	"github.com/zendev-sh/goai/provider"
	"github.com/zendev-sh/goai/provider/ollama"

	"issue-orchestrator/internal/common/config"
	"issue-orchestrator/internal/issues"
)

type Selector interface {
	SelectTasks(context.Context, SelectionRequest) (SelectionResult, error)
}

type SelectionRequest struct {
	Issues       []issues.Issue `json:"issues"`
	Slots        int            `json:"slots"`
	ActiveStates []string       `json:"active_states"`
	RunningIDs   []string       `json:"running_ids"`
}

type SelectionResult struct {
	Selected  []string `json:"selected"`
	Rationale string   `json:"rationale"`
}

type GoAISelector struct {
	model           provider.LanguageModel
	maxOutputTokens int
	temperature     float64
}

func NewSelector(cfg config.LLMConfig) (Selector, error) {
	if !cfg.SelectTasks {
		return DeterministicSelector{}, nil
	}
	if cfg.Provider != "ollama" {
		return nil, fmt.Errorf("unsupported llm provider %q", cfg.Provider)
	}
	client := &http.Client{Timeout: cfg.Timeout}
	return GoAISelector{model: ollama.Chat(cfg.Model, ollama.WithBaseURL(cfg.BaseURL), ollama.WithHTTPClient(client)), maxOutputTokens: cfg.MaxOutputTokens, temperature: cfg.Temperature}, nil
}

func (s GoAISelector) SelectTasks(ctx context.Context, req SelectionRequest) (SelectionResult, error) {
	if req.Slots <= 0 || len(req.Issues) == 0 {
		return SelectionResult{}, nil
	}
	payload, err := json.Marshal(selectionPayload(req))
	if err != nil {
		return SelectionResult{}, err
	}
	result, err := s.model.DoGenerate(ctx, provider.GenerateParams{
		System:          selectionSystemPrompt,
		Messages:        []provider.Message{goai.UserMessage(string(payload))},
		MaxOutputTokens: s.maxOutputTokens,
		Temperature:     &s.temperature,
	})
	if err != nil {
		return SelectionResult{}, err
	}
	parsed, err := ParseSelection(result.Text)
	if err != nil {
		return SelectionResult{}, err
	}
	return clampSelection(parsed, req), nil
}

type DeterministicSelector struct{}

func (DeterministicSelector) SelectTasks(_ context.Context, req SelectionRequest) (SelectionResult, error) {
	ids := make([]string, 0, len(req.Issues))
	for _, issue := range stableIssues(req.Issues) {
		if len(ids) >= req.Slots {
			break
		}
		ids = append(ids, issue.ID)
	}
	return SelectionResult{Selected: ids, Rationale: "deterministic priority fallback"}, nil
}

func ParseSelection(raw string) (SelectionResult, error) {
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start < 0 || end < start {
		return SelectionResult{}, fmt.Errorf("llm selection did not contain a json object")
	}
	var out SelectionResult
	if err := json.Unmarshal([]byte(raw[start:end+1]), &out); err != nil {
		return SelectionResult{}, err
	}
	return out, nil
}

func clampSelection(result SelectionResult, req SelectionRequest) SelectionResult {
	allowed := map[string]struct{}{}
	for _, issue := range req.Issues {
		allowed[issue.ID] = struct{}{}
	}
	seen := map[string]struct{}{}
	selected := []string{}
	for _, id := range result.Selected {
		if len(selected) >= req.Slots {
			break
		}
		if _, ok := allowed[id]; !ok {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		selected = append(selected, id)
	}
	result.Selected = selected
	return result
}

func selectionPayload(req SelectionRequest) map[string]any {
	items := make([]map[string]any, 0, len(req.Issues))
	for _, issue := range stableIssues(req.Issues) {
		items = append(items, map[string]any{"id": issue.ID, "identifier": issue.Identifier, "title": issue.Title, "state": issue.State, "priority": issue.Priority, "labels": issue.Labels, "assignee": issue.Assignee, "created_at": issue.CreatedAt, "updated_at": issue.UpdatedAt, "url": issue.URL})
	}
	return map[string]any{"slots": req.Slots, "active_states": req.ActiveStates, "running_ids": req.RunningIDs, "issues": items, "required_response": map[string]any{"selected": []string{"issue-id"}, "rationale": "brief reason"}}
}

func stableIssues(in []issues.Issue) []issues.Issue {
	out := append([]issues.Issue(nil), in...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Priority != out[j].Priority {
			if out[i].Priority == "" {
				return false
			}
			if out[j].Priority == "" {
				return true
			}
			return out[i].Priority < out[j].Priority
		}
		if !out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].CreatedAt.Before(out[j].CreatedAt)
		}
		return out[i].Identifier < out[j].Identifier
	})
	return out
}

const selectionSystemPrompt = `You are the issue dispatch policy layer for a Symphony-like coding-agent orchestrator.
Pick which Linear issues should be dispatched now.
Follow these rules:
- Return only JSON with keys selected and rationale.
- selected must contain tracker issue IDs, not human identifiers.
- Select at most the provided slots.
- Prefer eligible active-state issues with high priority, older created_at, clear scope, and no no-agent signal.
- Do not select issues already listed as running.
- Do not invent issue IDs.`
