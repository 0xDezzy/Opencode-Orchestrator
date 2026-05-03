package git

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

func run(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err != nil {
		return out.String(), fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, out.String())
	}
	return strings.TrimSpace(out.String()), nil
}
func EnsureRepo(ctx context.Context, repoPath string) error {
	_, err := run(ctx, repoPath, "rev-parse", "--show-toplevel")
	return err
}
func Fetch(ctx context.Context, repoPath, remote string) error {
	if remote == "" {
		remote = "origin"
	}
	remotes, err := run(ctx, repoPath, "remote")
	if err != nil {
		return err
	}
	if !hasLine(remotes, remote) {
		return nil
	}
	_, err = run(ctx, repoPath, "fetch", remote)
	return err
}

func hasLine(lines, want string) bool {
	for _, line := range strings.Split(lines, "\n") {
		if strings.TrimSpace(line) == want {
			return true
		}
	}
	return false
}
func HasChanges(ctx context.Context, worktreePath string) (bool, error) {
	out, err := run(ctx, worktreePath, "status", "--porcelain")
	return strings.TrimSpace(out) != "", err
}
func HasUnpushedCommits(ctx context.Context, worktreePath string) (bool, error) {
	if _, err := run(ctx, worktreePath, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}"); err != nil {
		return true, nil
	}
	out, err := run(ctx, worktreePath, "rev-list", "--count", "@{u}..HEAD")
	if err != nil {
		return true, err
	}
	return strings.TrimSpace(out) != "0", nil
}
func HasUnpushedCommitsFromBase(ctx context.Context, worktreePath, baseBranch string) (bool, error) {
	if _, err := run(ctx, worktreePath, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}"); err == nil {
		out, err := run(ctx, worktreePath, "rev-list", "--count", "@{u}..HEAD")
		if err != nil {
			return true, err
		}
		return strings.TrimSpace(out) != "0", nil
	}
	if baseBranch == "" {
		return true, nil
	}
	baseRef := baseBranch
	if _, err := run(ctx, worktreePath, "rev-parse", "--verify", "origin/"+baseBranch); err == nil {
		baseRef = "origin/" + baseBranch
	}
	out, err := run(ctx, worktreePath, "rev-list", "--count", baseRef+"..HEAD")
	if err != nil {
		return true, err
	}
	return strings.TrimSpace(out) != "0", nil
}
func ChangedFiles(ctx context.Context, worktreePath string) ([]string, error) {
	out, err := run(ctx, worktreePath, "diff", "--name-only")
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	return strings.Split(out, "\n"), nil
}
func CurrentHEAD(ctx context.Context, worktreePath string) (string, error) {
	return run(ctx, worktreePath, "rev-parse", "HEAD")
}
func RemoveWorktree(ctx context.Context, repoPath, worktreePath string, force bool) error {
	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, worktreePath)
	_, err := run(ctx, repoPath, args...)
	return err
}
