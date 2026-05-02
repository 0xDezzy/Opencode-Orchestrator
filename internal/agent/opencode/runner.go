package opencode

import (
	"bufio"
	"context"
	"fmt"
	"time"

	acp "github.com/coder/acp-go-sdk"

	"issue-orchestrator/internal/agent"
	"issue-orchestrator/internal/common/config"
)

type Runner struct{ cfg config.OpenCodeConfig }

func New(cfg config.OpenCodeConfig) *Runner { return &Runner{cfg: cfg} }

func (r *Runner) RunIssue(ctx context.Context, req agent.RunIssueRequest) (*agent.RunIssueResult, error) {
	timeout := r.cfg.Timeout
	if timeout <= 0 {
		timeout = 2 * time.Hour
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := command(ctx, r.cfg, req.WorktreeDir)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start opencode acp: %w", err)
	}
	go func() {
		s := bufio.NewScanner(stderr)
		for s.Scan() {
			_ = req.EmitEvent(ctx, agent.Event{Type: "agent.event", Source: "opencode.stderr", Message: s.Text(), CreatedAt: time.Now()})
		}
	}()
	c := client{events: func(t, m string) {
		_ = req.EmitEvent(ctx, agent.Event{Type: "agent.event", Source: "opencode", Message: m, Payload: t, CreatedAt: time.Now()})
	}}
	conn := acp.NewClientSideConnection(c, stdin, stdout)
	if _, err := conn.Initialize(ctx, acp.InitializeRequest{ProtocolVersion: acp.ProtocolVersionNumber, ClientCapabilities: acp.ClientCapabilities{Fs: acp.FileSystemCapabilities{ReadTextFile: true, WriteTextFile: true}, Terminal: true}}); err != nil {
		_ = cmd.Process.Kill()
		return nil, fmt.Errorf("initialize acp: %w", err)
	}
	sess, err := conn.NewSession(ctx, acp.NewSessionRequest{Cwd: req.WorktreeDir, McpServers: []acp.McpServer{}})
	if err != nil {
		_ = cmd.Process.Kill()
		return nil, fmt.Errorf("create acp session: %w", err)
	}
	_ = req.EmitEvent(ctx, agent.Event{Type: "agent.started", Source: "opencode", Message: "ACP session started", Payload: map[string]any{"session_id": sess.SessionId}, CreatedAt: time.Now()})
	resp, err := conn.Prompt(ctx, acp.PromptRequest{SessionId: sess.SessionId, Prompt: []acp.ContentBlock{acp.TextBlock(req.Prompt)}})
	if err != nil {
		_ = cmd.Process.Kill()
		return nil, fmt.Errorf("prompt acp session: %w", err)
	}
	_ = cmd.Process.Kill()
	_ = cmd.Wait()
	return &agent.RunIssueResult{SessionID: string(sess.SessionId), StopReason: fmt.Sprintf("%v", resp.StopReason), Summary: "OpenCode ACP prompt completed", Raw: resp}, nil
}
