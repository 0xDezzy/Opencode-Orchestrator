package git

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

type WorktreeOptions struct{ RepoPath, WorkspaceRoot, IssueID, IssueIdentifier, Title, BaseBranch, BranchPrefix string }
type Worktree struct {
	Path, BranchName string
	Reused           bool
}

func WorktreeExists(path string) bool { st, err := os.Stat(path); return err == nil && st.IsDir() }
func CreateOrReuseWorktree(ctx context.Context, o WorktreeOptions) (*Worktree, error) {
	slug := SafeIssueSlug(o.IssueIdentifier, o.Title)
	root, err := filepath.Abs(o.WorkspaceRoot)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(root, slug)
	if !Inside(root, path) {
		return nil, fmt.Errorf("worktree path escapes workspace root")
	}
	branch := BranchName(o.BranchPrefix, o.IssueIdentifier, o.Title)
	if WorktreeExists(path) {
		return &Worktree{Path: path, BranchName: branch, Reused: true}, nil
	}
	if err := os.MkdirAll(root, 0755); err != nil {
		return nil, err
	}
	if err := Fetch(ctx, o.RepoPath, "origin"); err != nil {
		return nil, err
	}
	base := o.BaseBranch
	if hasRemote(ctx, o.RepoPath, "origin") {
		base = "origin/" + o.BaseBranch
	}
	if branchExists(ctx, o.RepoPath, branch) {
		_, err = run(ctx, o.RepoPath, "worktree", "add", path, branch)
	} else {
		_, err = run(ctx, o.RepoPath, "worktree", "add", "-b", branch, path, base)
	}
	if err != nil {
		return nil, err
	}
	return &Worktree{Path: path, BranchName: branch}, nil
}

func branchExists(ctx context.Context, repoPath, branch string) bool {
	_, err := run(ctx, repoPath, "show-ref", "--verify", "--quiet", "refs/heads/"+branch)
	return err == nil
}

func hasRemote(ctx context.Context, repoPath, remote string) bool {
	out, err := run(ctx, repoPath, "remote")
	return err == nil && hasLine(out, remote)
}
