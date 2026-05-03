package config

import "fmt"

func (c Config) Validate(requireLinear bool) error {
	if requireLinear && c.Linear.APIKey == "" {
		return fmt.Errorf("linear api key is required")
	}
	if c.Workspace.Root == "" {
		return fmt.Errorf("workspace root is required")
	}
	if c.Workspace.BaseBranch == "" {
		return fmt.Errorf("workspace base branch is required")
	}
	if c.Scheduler.PollInterval <= 0 {
		return fmt.Errorf("scheduler poll interval must be positive")
	}
	if c.Scheduler.ReconcileInterval < 0 {
		return fmt.Errorf("scheduler reconcile interval must not be negative")
	}
	if c.Scheduler.ReconcileBatchLimit < 0 {
		return fmt.Errorf("scheduler reconcile batch limit must not be negative")
	}
	if c.Scheduler.MaxConcurrentRuns < 1 {
		return fmt.Errorf("scheduler max concurrent runs must be at least 1")
	}
	if c.Scheduler.MaxAttempts < 1 {
		return fmt.Errorf("scheduler max attempts must be at least 1")
	}
	if c.SQLite.Path == "" {
		return fmt.Errorf("sqlite path is required")
	}
	if c.OpenCode.Command == "" {
		return fmt.Errorf("opencode command is required")
	}
	if c.OpenCode.ServerURL == "" {
		return fmt.Errorf("opencode server url is required")
	}
	if c.OpenCode.ServeHost == "" {
		return fmt.Errorf("opencode serve host is required")
	}
	if c.OpenCode.ServePort <= 0 {
		return fmt.Errorf("opencode serve port must be positive")
	}
	if c.OpenCode.StartupTimeout <= 0 {
		return fmt.Errorf("opencode startup timeout must be positive")
	}
	if c.TUI.RefreshInterval <= 0 {
		return fmt.Errorf("tui refresh interval must be positive")
	}
	if c.TUI.MaxLogLines <= 0 {
		return fmt.Errorf("tui max log lines must be positive")
	}
	if c.LLM.SelectTasks {
		if c.LLM.Provider != "ollama" {
			return fmt.Errorf("llm provider %q is not supported", c.LLM.Provider)
		}
		if c.LLM.Model == "" {
			return fmt.Errorf("llm model is required")
		}
		if c.LLM.BaseURL == "" {
			return fmt.Errorf("llm base url is required")
		}
		if c.LLM.Timeout <= 0 {
			return fmt.Errorf("llm timeout must be positive")
		}
	}
	return nil
}
