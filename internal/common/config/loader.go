package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

func Load(path string) (*Config, error) {
	cfg := Defaults()
	v := viper.New()
	v.SetConfigType("yaml")
	v.SetEnvPrefix("ORCH")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	setDefaults(v, cfg)
	if path != "" {
		v.SetConfigFile(path)
	} else {
		v.SetConfigName("orchestrator")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.SetConfigName("config")
		v.AddConfigPath(".")
		v.AddConfigPath("$HOME/.config/orchestrator")
	}
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok && path != "" {
			return nil, fmt.Errorf("read config: %w", err)
		}
	}
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}
	return &cfg, cfg.Validate(false)
}

func setDefaults(v *viper.Viper, c Config) {
	v.SetDefault("app.name", c.App.Name)
	v.SetDefault("app.environment", c.App.Environment)
	v.SetDefault("logging.level", c.Logging.Level)
	v.SetDefault("logging.format", c.Logging.Format)
	v.SetDefault("logging.file_path", c.Logging.FilePath)
	v.SetDefault("logging.tui_mirror_to_file", c.Logging.TUIMirrorToFile)
	v.SetDefault("server.enabled", c.Server.Enabled)
	v.SetDefault("server.address", c.Server.Address)
	v.SetDefault("tui.enabled", c.TUI.Enabled)
	v.SetDefault("tui.refresh_interval", c.TUI.RefreshInterval)
	v.SetDefault("tui.max_log_lines", c.TUI.MaxLogLines)
	v.SetDefault("sqlite.path", c.SQLite.Path)
	v.SetDefault("workspace.repo_path", c.Workspace.RepoPath)
	v.SetDefault("workspace.root", c.Workspace.Root)
	v.SetDefault("workspace.base_branch", c.Workspace.BaseBranch)
	v.SetDefault("workspace.branch_prefix", c.Workspace.BranchPrefix)
	v.SetDefault("scheduler.poll_interval", c.Scheduler.PollInterval)
	v.SetDefault("scheduler.max_concurrent_runs", c.Scheduler.MaxConcurrentRuns)
	v.SetDefault("scheduler.max_attempts", c.Scheduler.MaxAttempts)
	v.SetDefault("scheduler.lock_ttl", c.Scheduler.LockTTL)
	v.SetDefault("opencode.command", c.OpenCode.Command)
	v.SetDefault("opencode.args", c.OpenCode.Args)
	v.SetDefault("opencode.model", c.OpenCode.Model)
	v.SetDefault("opencode.small_model", c.OpenCode.SmallModel)
	v.SetDefault("opencode.timeout", c.OpenCode.Timeout)
	v.SetDefault("opencode.stall_timeout", c.OpenCode.StallTimeout)
}
