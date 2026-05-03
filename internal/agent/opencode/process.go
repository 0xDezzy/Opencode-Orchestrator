package opencode

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"issue-orchestrator/internal/common/config"
)

func serveCommand(ctx context.Context, cfg config.OpenCodeConfig) *exec.Cmd {
	cmd := exec.CommandContext(ctx, cfg.Command, serveArgs(cfg)...)
	cmd.Env = openCodeEnv(os.Environ(), cfg)
	return cmd
}

func serveArgs(cfg config.OpenCodeConfig) []string {
	host := cfg.ServeHost
	if host == "" {
		host = "127.0.0.1"
	}
	port := cfg.ServePort
	if port <= 0 {
		port = 4096
	}
	return []string{"serve", "--hostname", host, "--port", strconv.Itoa(port)}
}

func serverURL(cfg config.OpenCodeConfig) string {
	if cfg.ServerURL != "" {
		return strings.TrimRight(cfg.ServerURL, "/")
	}
	host := cfg.ServeHost
	if host == "" {
		host = "127.0.0.1"
	}
	port := cfg.ServePort
	if port <= 0 {
		port = 4096
	}
	return fmt.Sprintf("http://%s:%d", host, port)
}

func openCodeEnv(base []string, cfg config.OpenCodeConfig) []string {
	content := openCodeConfigContent(cfg)
	if content == "" && cfg.ServerPassword == "" {
		return base
	}
	env := make([]string, 0, len(base)+1)
	for _, kv := range base {
		if strings.HasPrefix(kv, "OPENCODE_CONFIG_CONTENT=") {
			continue
		}
		if cfg.ServerPassword != "" && strings.HasPrefix(kv, "OPENCODE_SERVER_PASSWORD=") {
			continue
		}
		env = append(env, kv)
	}
	if content != "" {
		env = append(env, "OPENCODE_CONFIG_CONTENT="+content)
	}
	if cfg.ServerPassword != "" {
		env = append(env, "OPENCODE_SERVER_PASSWORD="+cfg.ServerPassword)
	}
	return env
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
