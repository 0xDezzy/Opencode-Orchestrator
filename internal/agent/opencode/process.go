package opencode

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strings"

	"issue-orchestrator/internal/common/config"
)

func command(ctx context.Context, cfg config.OpenCodeConfig, cwd string) *exec.Cmd {
	bin := cfg.Command
	a := append([]string{}, cfg.Args...)
	a = append(a, "--cwd", cwd)
	cmd := exec.CommandContext(ctx, bin, a...)
	cmd.Dir = cwd
	cmd.Env = openCodeEnv(os.Environ(), cfg)
	return cmd
}

func openCodeEnv(base []string, cfg config.OpenCodeConfig) []string {
	content := openCodeConfigContent(cfg)
	if content == "" {
		return base
	}
	env := make([]string, 0, len(base)+1)
	for _, kv := range base {
		if strings.HasPrefix(kv, "OPENCODE_CONFIG_CONTENT=") {
			continue
		}
		env = append(env, kv)
	}
	return append(env, "OPENCODE_CONFIG_CONTENT="+content)
}

func openCodeConfigContent(cfg config.OpenCodeConfig) string {
	if cfg.Model == "" && cfg.SmallModel == "" {
		return ""
	}
	payload := map[string]string{}
	if cfg.Model != "" {
		payload["model"] = cfg.Model
	}
	if cfg.SmallModel != "" {
		payload["small_model"] = cfg.SmallModel
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	return string(b)
}
