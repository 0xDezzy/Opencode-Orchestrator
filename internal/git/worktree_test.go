package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestCreateOrReuseWorktreeUsesExistingBranchWhenDirectoryWasRemoved(t *testing.T) {
	ctx := context.Background()
	repo := t.TempDir()
	if _, err := run(ctx, repo, "init", "-b", "main"); err != nil {
		t.Fatal(err)
	}
	if _, err := run(ctx, repo, "config", "user.email", "test@example.com"); err != nil {
		t.Fatal(err)
	}
	if _, err := run(ctx, repo, "config", "user.name", "Test User"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := run(ctx, repo, "add", "README.md"); err != nil {
		t.Fatal(err)
	}
	if _, err := run(ctx, repo, "commit", "-m", "initial"); err != nil {
		t.Fatal(err)
	}

	opts := WorktreeOptions{
		RepoPath:        repo,
		WorkspaceRoot:   filepath.Join(repo, ".orchestrator", "worktrees"),
		IssueID:         "issue-1",
		IssueIdentifier: "DEZ-13",
		Title:           "Make TUI table columns fill available terminal width",
		BaseBranch:      "main",
		BranchPrefix:    "agent",
	}
	first, err := CreateOrReuseWorktree(ctx, opts)
	if err != nil {
		t.Fatal(err)
	}
	if err := RemoveWorktree(ctx, repo, first.Path, false); err != nil {
		t.Fatal(err)
	}

	second, err := CreateOrReuseWorktree(ctx, opts)
	if err != nil {
		t.Fatal(err)
	}
	if second.Path != first.Path {
		t.Fatalf("path = %q, want %q", second.Path, first.Path)
	}
	if second.BranchName != first.BranchName {
		t.Fatalf("branch = %q, want %q", second.BranchName, first.BranchName)
	}
}
