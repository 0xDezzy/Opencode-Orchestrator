# Opencode Orchestrator

<p>
  <strong>Opencode Orchestrator</strong> is a Go command-line application for coordinating Linear issues, isolated worktrees, and agent execution through <a href="https://github.com/anomalyco/opencode">opencode</a>.
</p>

<p>
  It is designed to claim eligible issues, run automated agent workflows, persist run state, and hand work back for human review.
</p>

## Overview

Opencode Orchestrator connects a few moving pieces that are common in agent-assisted development workflows:

- Linear issue discovery and status updates
- Per-issue Git worktrees for isolated changes
- Configurable workflow prompts from Markdown files
- OpenCode-based agent runs
- SQLite-backed run history and locks
- Terminal UI and local HTTP status endpoints

## Getting Started

### Prerequisites

- Go 1.26 or newer
- Git
- A Linear API key, when using Linear-backed issue tracking
- <a href="https://github.com/anomalyco/opencode">opencode</a>, available on your `PATH`

### Build

```sh
go build -o orchestrator ./cmd/orchestrator
```

### Run a Single Issue

```sh
./orchestrator run --issue ISSUE-123
```

### Start the Daemon

```sh
./orchestrator daemon
```

### Check Status

```sh
./orchestrator status
```

## Configuration

Configuration can be supplied with `--config` and workflow instructions can be supplied with `--workflow`.

```sh
./orchestrator daemon --config ./configs/config.yaml --workflow ./WORKFLOW.md
```

The default workflow file is Markdown with front matter for agent and handoff behavior. See `WORKFLOW.example.md` for a starting point.

## Workflow

The default workflow asks the agent to:

- Inspect the repository before editing
- Make the smallest correct change
- Add or update tests where appropriate
- Run relevant tests
- Avoid unrelated refactors
- Commit changes with a clear message
- Leave the issue in a human-review state rather than marking it done

## Development

Run the test suite with:

```sh
go test ./...
```

Common project areas:

- `internal/cli`: command definitions and dependency wiring
- `internal/issues/linear`: Linear integration
- `internal/agent/opencode`: OpenCode agent runner
- `internal/git`: branch and worktree helpers
- `internal/db`: SQLite persistence and migrations
- `internal/tui`: terminal interface

## Attribution

This project was built utilizing <a href="https://github.com/anomalyco/opencode">opencode</a>.

Please note, this project was vibe coded for the most part. I wanted to mess around with things and I wanted to see how far I could go with it. There will most likely be bugs. I'm still working through things and I just wanted to get this to work for me.
