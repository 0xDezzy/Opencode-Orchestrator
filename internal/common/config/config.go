package config

import "time"

type Config struct {
	App       AppConfig       `mapstructure:"app" json:"app"`
	Logging   LoggingConfig   `mapstructure:"logging" json:"logging"`
	Server    ServerConfig    `mapstructure:"server" json:"server"`
	TUI       TUIConfig       `mapstructure:"tui" json:"tui"`
	SQLite    SQLiteConfig    `mapstructure:"sqlite" json:"sqlite"`
	Linear    LinearConfig    `mapstructure:"linear" json:"linear"`
	Workspace WorkspaceConfig `mapstructure:"workspace" json:"workspace"`
	Scheduler SchedulerConfig `mapstructure:"scheduler" json:"scheduler"`
	LLM       LLMConfig       `mapstructure:"llm" json:"llm"`
	OpenCode  OpenCodeConfig  `mapstructure:"opencode" json:"opencode"`
	Handoff   HandoffConfig   `mapstructure:"handoff" json:"handoff"`
}
type AppConfig struct{ Name string }
type LoggingConfig struct {
	Level, Format, FilePath string
	TUIMirrorToFile         bool `mapstructure:"tui_mirror_to_file" json:"tui_mirror_to_file"`
}
type ServerConfig struct {
	Enabled bool
	Address string
}
type TUIConfig struct {
	Enabled              bool
	RefreshInterval      time.Duration `mapstructure:"refresh_interval" json:"refresh_interval"`
	MaxLogLines          int           `mapstructure:"max_log_lines" json:"max_log_lines"`
	ShowDebugLogs        bool          `mapstructure:"show_debug_logs" json:"show_debug_logs"`
	Table, LogFormat     string
	PreserveScreenOnExit bool `mapstructure:"preserve_screen_on_exit" json:"preserve_screen_on_exit"`
}
type SQLiteConfig struct{ Path string }
type LinearConfig struct {
	APIKey         string   `mapstructure:"api_key" json:"-"`
	TeamKey        string   `mapstructure:"team_key" json:"team_key"`
	ProjectName    string   `mapstructure:"project_name" json:"project_name"`
	ActiveStates   []string `mapstructure:"active_states" json:"active_states"`
	TerminalStates []string `mapstructure:"terminal_states" json:"terminal_states"`
	RunningState   string   `mapstructure:"running_state" json:"running_state"`
	ReviewState    string   `mapstructure:"review_state" json:"review_state"`
	FailedState    string   `mapstructure:"failed_state" json:"failed_state"`
	Labels         LabelConfig
}
type LabelConfig struct{ Include, Exclude []string }
type WorkspaceConfig struct {
	RepoPath           string `mapstructure:"repo_path"`
	Root               string `mapstructure:"root"`
	BaseBranch         string `mapstructure:"base_branch"`
	BranchPrefix       string `mapstructure:"branch_prefix"`
	PreserveFailed     bool   `mapstructure:"preserve_failed" json:"preserve_failed"`
	PreserveSuccessful bool   `mapstructure:"preserve_successful" json:"preserve_successful"`
}
type SchedulerConfig struct {
	PollInterval        time.Duration `mapstructure:"poll_interval"`
	ReconcileInterval   time.Duration `mapstructure:"reconcile_interval"`
	ReconcileBatchLimit int           `mapstructure:"reconcile_batch_limit"`
	MaxConcurrentRuns   int           `mapstructure:"max_concurrent_runs"`
	MaxAttempts         int           `mapstructure:"max_attempts"`
	LockTTL             time.Duration `mapstructure:"lock_ttl"`
	RetryBackoff        time.Duration `mapstructure:"retry_backoff"`
	IssueStaleAfter     time.Duration `mapstructure:"issue_stale_after"`
}
type LLMConfig struct {
	Provider        string        `mapstructure:"provider" json:"provider"`
	Model           string        `mapstructure:"model" json:"model"`
	BaseURL         string        `mapstructure:"base_url" json:"base_url"`
	Timeout         time.Duration `mapstructure:"timeout" json:"timeout"`
	MaxOutputTokens int           `mapstructure:"max_output_tokens" json:"max_output_tokens"`
	Temperature     float64       `mapstructure:"temperature" json:"temperature"`
	SelectTasks     bool          `mapstructure:"select_tasks" json:"select_tasks"`
}
type OpenCodeConfig struct {
	Command           string        `mapstructure:"command" json:"command"`
	Args              []string      `mapstructure:"args" json:"args"`
	ServerURL         string        `mapstructure:"server_url" json:"server_url"`
	ServerPassword    string        `mapstructure:"server_password" json:"server_password"`
	ServeHost         string        `mapstructure:"serve_host" json:"serve_host"`
	ServePort         int           `mapstructure:"serve_port" json:"serve_port"`
	AutoStartServer   bool          `mapstructure:"auto_start_server" json:"auto_start_server"`
	StartupTimeout    time.Duration `mapstructure:"startup_timeout" json:"startup_timeout"`
	Model             string        `mapstructure:"model" json:"model"`
	SmallModel        string        `mapstructure:"small_model" json:"small_model"`
	Timeout           time.Duration `mapstructure:"timeout"`
	StallTimeout      time.Duration `mapstructure:"stall_timeout"`
	MaxPromptAttempts int           `mapstructure:"max_prompt_attempts"`
	AllowedTools      []string      `mapstructure:"allowed_tools"`
}
type HandoffConfig struct {
	UpdateLinear        bool `mapstructure:"update_linear"`
	CommentOnStart      bool `mapstructure:"comment_on_start"`
	CommentOnSuccess    bool `mapstructure:"comment_on_success"`
	CommentOnFailure    bool `mapstructure:"comment_on_failure"`
	TransitionOnStart   bool `mapstructure:"transition_on_start"`
	TransitionOnSuccess bool `mapstructure:"transition_on_success"`
	TransitionOnFailure bool `mapstructure:"transition_on_failure"`
}

func Defaults() Config {
	return Config{App: AppConfig{"orchestrator"}, Logging: LoggingConfig{Level: "info", Format: "text", FilePath: "./.orchestrator/orchestrator.log", TUIMirrorToFile: true}, Server: ServerConfig{true, "127.0.0.1:8787"}, TUI: TUIConfig{Enabled: true, RefreshInterval: time.Second, MaxLogLines: 1000, Table: "runs", LogFormat: "json"}, SQLite: SQLiteConfig{"./.orchestrator/orchestrator.db"}, Linear: LinearConfig{ActiveStates: []string{"Ready for Agent"}, TerminalStates: []string{"Done", "Canceled", "Duplicate"}, RunningState: "Agent Running", ReviewState: "Human Review", FailedState: "Agent Failed", Labels: LabelConfig{Include: []string{"agent"}, Exclude: []string{"no-agent"}}}, Workspace: WorkspaceConfig{RepoPath: ".", Root: "./.orchestrator/worktrees", BaseBranch: "main", BranchPrefix: "agent", PreserveFailed: true, PreserveSuccessful: true}, Scheduler: SchedulerConfig{PollInterval: 30 * time.Second, ReconcileInterval: 10 * time.Minute, ReconcileBatchLimit: 100, MaxConcurrentRuns: 2, MaxAttempts: 3, LockTTL: 2 * time.Hour, RetryBackoff: 5 * time.Minute, IssueStaleAfter: 24 * time.Hour}, LLM: LLMConfig{Provider: "ollama", Model: "granite4.1:latest", BaseURL: "http://localhost:11434/v1", Timeout: 30 * time.Second, MaxOutputTokens: 1200, Temperature: 0.1, SelectTasks: true}, OpenCode: OpenCodeConfig{Command: "opencode", Args: []string{"serve"}, ServerURL: "http://127.0.0.1:4096", ServeHost: "127.0.0.1", ServePort: 4096, AutoStartServer: true, StartupTimeout: 10 * time.Second, Model: "", SmallModel: "", Timeout: 2 * time.Hour, StallTimeout: 10 * time.Minute, MaxPromptAttempts: 1}, Handoff: HandoffConfig{UpdateLinear: true, CommentOnStart: true, CommentOnSuccess: true, CommentOnFailure: true, TransitionOnStart: true, TransitionOnSuccess: true}}
}
