package opencode

import (
	"encoding/json"
	"strings"
	"testing"

	"issue-orchestrator/internal/common/config"
)

func TestOpenCodeConfigContentIncludesModels(t *testing.T) {
	got := openCodeConfigContent(config.OpenCodeConfig{
		Model:      "anthropic/claude-sonnet-4-5",
		SmallModel: "anthropic/claude-haiku-4-5",
	})
	if got == "" {
		t.Fatal("expected config content")
	}
	var payload map[string]string
	if err := json.Unmarshal([]byte(got), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["model"] != "anthropic/claude-sonnet-4-5" {
		t.Fatalf("model not encoded: %q", payload["model"])
	}
	if payload["small_model"] != "anthropic/claude-haiku-4-5" {
		t.Fatalf("small_model not encoded: %q", payload["small_model"])
	}
}

func TestOpenCodeEnvReplacesInlineConfig(t *testing.T) {
	got := openCodeEnv([]string{"A=B", "OPENCODE_CONFIG_CONTENT={}", "C=D"}, config.OpenCodeConfig{Model: "openai/gpt-5"})
	var count int
	for _, kv := range got {
		if strings.HasPrefix(kv, "OPENCODE_CONFIG_CONTENT=") {
			count++
			if !strings.Contains(kv, "openai/gpt-5") {
				t.Fatalf("model missing from env: %s", kv)
			}
		}
	}
	if count != 1 {
		t.Fatalf("expected one OPENCODE_CONFIG_CONTENT entry, got %d in %#v", count, got)
	}
}

func TestOpenCodeEnvSetsServerPassword(t *testing.T) {
	got := openCodeEnv([]string{"A=B", "OPENCODE_SERVER_PASSWORD=old", "C=D"}, config.OpenCodeConfig{ServerPassword: "secret"})
	joined := strings.Join(got, "\n")
	if strings.Contains(joined, "OPENCODE_SERVER_PASSWORD=old") {
		t.Fatalf("env kept old server password: %#v", got)
	}
	if !strings.Contains(joined, "OPENCODE_SERVER_PASSWORD=secret") {
		t.Fatalf("env missing server password: %#v", got)
	}
}
