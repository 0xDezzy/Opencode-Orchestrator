package opencode

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	sdk "github.com/sst/opencode-sdk-go"
	"github.com/sst/opencode-sdk-go/option"

	"issue-orchestrator/internal/agent"
	"issue-orchestrator/internal/common/config"
)

type Runner struct {
	cfg        config.OpenCodeConfig
	httpClient *http.Client

	mu       sync.Mutex
	serveCmd *exec.Cmd
}

func New(cfg config.OpenCodeConfig) *Runner {
	return &Runner{cfg: cfg, httpClient: &http.Client{Timeout: 2 * time.Second}}
}

func (r *Runner) RunIssue(ctx context.Context, req agent.RunIssueRequest) (*agent.RunIssueResult, error) {
	timeout := r.cfg.Timeout
	if timeout <= 0 {
		timeout = 2 * time.Hour
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	baseURL := serverURL(r.cfg)
	if err := r.ensureServer(ctx, baseURL); err != nil {
		return nil, err
	}

	client := sdk.NewClient(clientOptions(baseURL, r.cfg)...)
	session, err := client.Session.New(ctx, sdk.SessionNewParams{
		Directory: sdk.String(req.WorktreeDir),
		Title:     sdk.String(sessionTitle(req)),
	})
	if err != nil {
		return nil, fmt.Errorf("create opencode session: %w", err)
	}
	_ = req.EmitEvent(ctx, agent.Event{Type: "agent.started", Source: "opencode", Message: "OpenCode session started", Payload: map[string]any{"session_id": session.ID}, CreatedAt: time.Now()})

	params := sdk.SessionPromptParams{
		Directory: sdk.String(req.WorktreeDir),
		Parts: sdk.F([]sdk.SessionPromptParamsPartUnion{sdk.TextPartInputParam{
			Type: sdk.F(sdk.TextPartInputTypeText),
			Text: sdk.String(req.Prompt),
		}}),
	}
	if r.cfg.Model != "" {
		providerID, modelID := splitModel(r.cfg.Model)
		params.Model = sdk.F(sdk.SessionPromptParamsModel{ProviderID: sdk.String(providerID), ModelID: sdk.String(modelID)})
	}
	for _, tool := range r.cfg.AllowedTools {
		if params.Tools.Value == nil {
			params.Tools = sdk.F(map[string]bool{})
		}
		params.Tools.Value[tool] = true
	}

	resp, err := client.Session.Prompt(ctx, session.ID, params)
	if err != nil {
		return nil, fmt.Errorf("prompt opencode session: %w", err)
	}
	_ = req.EmitEvent(ctx, agent.Event{Type: "agent.completed", Source: "opencode", Message: "OpenCode prompt completed", Payload: map[string]any{"session_id": session.ID, "message_id": resp.Info.ID}, CreatedAt: time.Now()})

	return &agent.RunIssueResult{SessionID: session.ID, StopReason: resp.Info.ID, Summary: "OpenCode prompt completed", Raw: resp}, nil
}

func (r *Runner) ensureServer(ctx context.Context, baseURL string) error {
	if r.serverHealthy(ctx, baseURL) {
		return nil
	}
	if !r.cfg.AutoStartServer {
		return fmt.Errorf("opencode server is not reachable at %s", baseURL)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.serverHealthy(ctx, baseURL) {
		return nil
	}
	if r.serveCmd == nil || r.serveCmd.Process == nil {
		cmd := serveCommand(ctx, r.cfg)
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("start opencode serve: %w", err)
		}
		r.serveCmd = cmd
	}

	deadline := time.Now().Add(startupTimeout(r.cfg))
	for time.Now().Before(deadline) {
		if r.serverHealthy(ctx, baseURL) {
			return nil
		}
		timer := time.NewTimer(100 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	return fmt.Errorf("opencode serve did not become ready at %s", baseURL)
}

func (r *Runner) serverHealthy(ctx context.Context, baseURL string) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/global/health", nil)
	if err != nil {
		return false
	}
	if password := serverPassword(r.cfg); password != "" {
		req.SetBasicAuth("", password)
	}
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

func clientOptions(baseURL string, cfg config.OpenCodeConfig) []option.RequestOption {
	opts := []option.RequestOption{option.WithBaseURL(baseURL)}
	if password := serverPassword(cfg); password != "" {
		req, err := http.NewRequest(http.MethodGet, baseURL, nil)
		if err == nil {
			req.SetBasicAuth("", password)
			opts = append(opts, option.WithHeader("Authorization", req.Header.Get("Authorization")))
		}
	}
	return opts
}

func serverPassword(cfg config.OpenCodeConfig) string {
	if cfg.ServerPassword != "" {
		return cfg.ServerPassword
	}
	return os.Getenv("OPENCODE_SERVER_PASSWORD")
}

func startupTimeout(cfg config.OpenCodeConfig) time.Duration {
	if cfg.StartupTimeout > 0 {
		return cfg.StartupTimeout
	}
	return 10 * time.Second
}

func sessionTitle(req agent.RunIssueRequest) string {
	if req.Issue.Identifier == "" {
		return req.Issue.Title
	}
	if req.Issue.Title == "" {
		return req.Issue.Identifier
	}
	return req.Issue.Identifier + ": " + req.Issue.Title
}

func splitModel(model string) (string, string) {
	providerID, modelID, ok := strings.Cut(model, "/")
	if !ok || providerID == "" || modelID == "" {
		return "", model
	}
	return providerID, modelID
}
