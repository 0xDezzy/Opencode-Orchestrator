package opencode

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"issue-orchestrator/internal/agent"
	"issue-orchestrator/internal/common/config"
	"issue-orchestrator/internal/issues"
)

func TestRunIssueUsesOpenCodeServeSessionInWorktree(t *testing.T) {
	t.Parallel()

	worktree := t.TempDir()
	var sawSession bool
	var sawPrompt bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/global/health"):
			w.WriteHeader(http.StatusOK)
		case strings.HasSuffix(r.URL.Path, "/session") || strings.HasSuffix(r.URL.Path, "/session/"):
			if r.Method != http.MethodPost {
				t.Fatalf("session method = %s", r.Method)
			}
			if got := r.URL.Query().Get("directory"); got != worktree {
				t.Fatalf("session directory = %q, want %q", got, worktree)
			}
			sawSession = true
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":        "ses_123",
				"directory": worktree,
				"projectID": "proj_123",
				"title":     "DEZ-1: Test issue",
				"time":      map[string]any{"created": 1, "updated": 1},
			})
		case strings.HasSuffix(r.URL.Path, "/session/ses_123/message") || strings.HasSuffix(r.URL.Path, "/session/ses_123/message/"):
			if r.Method != http.MethodPost {
				t.Fatalf("prompt method = %s", r.Method)
			}
			if got := r.URL.Query().Get("directory"); got != worktree {
				t.Fatalf("prompt directory = %q, want %q", got, worktree)
			}
			var body struct {
				Parts []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"parts"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode prompt body: %v", err)
			}
			if len(body.Parts) != 1 || body.Parts[0].Type != "text" || body.Parts[0].Text != "fix it" {
				t.Fatalf("prompt parts = %#v", body.Parts)
			}
			sawPrompt = true
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"info": map[string]any{
					"id":         "msg_123",
					"role":       "assistant",
					"sessionID":  "ses_123",
					"providerID": "openai",
					"modelID":    "gpt-5.5",
					"time":       map[string]any{"created": 1, "completed": 2},
				},
				"parts": []any{},
			})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	runner := New(config.OpenCodeConfig{
		ServerURL:       server.URL,
		AutoStartServer: false,
		Timeout:         time.Minute,
	})
	result, err := runner.RunIssue(context.Background(), agent.RunIssueRequest{
		Issue:       issues.Issue{Identifier: "DEZ-1", Title: "Test issue"},
		WorktreeDir: worktree,
		Prompt:      "fix it",
		EmitEvent: func(context.Context, agent.Event) error {
			return nil
		},
	})
	if err != nil {
		t.Fatalf("RunIssue returned error: %v", err)
	}
	if result.SessionID != "ses_123" {
		t.Fatalf("SessionID = %q, want ses_123", result.SessionID)
	}
	if !sawSession || !sawPrompt {
		t.Fatalf("sawSession=%v sawPrompt=%v", sawSession, sawPrompt)
	}
}

func TestRunIssueAuthenticatesToPasswordProtectedServer(t *testing.T) {
	t.Parallel()

	worktree := t.TempDir()
	const password = "server-password"
	var authedHealth bool
	var authedSession bool
	var authedPrompt bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, got, ok := r.BasicAuth()
		if !ok || got != password {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		switch {
		case strings.HasSuffix(r.URL.Path, "/global/health"):
			authedHealth = true
			w.WriteHeader(http.StatusOK)
		case strings.HasSuffix(r.URL.Path, "/session") || strings.HasSuffix(r.URL.Path, "/session/"):
			authedSession = true
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":        "ses_auth",
				"directory": worktree,
				"projectID": "proj_auth",
				"title":     "DEZ-2: Auth issue",
				"time":      map[string]any{"created": 1, "updated": 1},
			})
		case strings.HasSuffix(r.URL.Path, "/session/ses_auth/message") || strings.HasSuffix(r.URL.Path, "/session/ses_auth/message/"):
			authedPrompt = true
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"info": map[string]any{
					"id":         "msg_auth",
					"role":       "assistant",
					"sessionID":  "ses_auth",
					"providerID": "openai",
					"modelID":    "gpt-5.5",
					"time":       map[string]any{"created": 1, "completed": 2},
				},
				"parts": []any{},
			})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	runner := New(config.OpenCodeConfig{
		ServerURL:       server.URL,
		ServerPassword:  password,
		AutoStartServer: false,
		Timeout:         time.Minute,
	})
	result, err := runner.RunIssue(context.Background(), agent.RunIssueRequest{
		Issue:       issues.Issue{Identifier: "DEZ-2", Title: "Auth issue"},
		WorktreeDir: worktree,
		Prompt:      "fix it",
		EmitEvent: func(context.Context, agent.Event) error {
			return nil
		},
	})
	if err != nil {
		t.Fatalf("RunIssue returned error: %v", err)
	}
	if result.SessionID != "ses_auth" {
		t.Fatalf("SessionID = %q, want ses_auth", result.SessionID)
	}
	if !authedHealth || !authedSession || !authedPrompt {
		t.Fatalf("authedHealth=%v authedSession=%v authedPrompt=%v", authedHealth, authedSession, authedPrompt)
	}
}

func TestServerPasswordFallsBackToOpenCodeEnvironment(t *testing.T) {
	t.Setenv("OPENCODE_SERVER_PASSWORD", "from-env")

	if got := serverPassword(config.OpenCodeConfig{}); got != "from-env" {
		t.Fatalf("serverPassword = %q, want from-env", got)
	}
}
