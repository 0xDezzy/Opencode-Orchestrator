package orchestrator

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"issue-orchestrator/internal/app"
	"issue-orchestrator/internal/common/config"
	"issue-orchestrator/internal/db"
	"issue-orchestrator/internal/db/models"
	gitpkg "issue-orchestrator/internal/git"
	"issue-orchestrator/internal/issues"
)

func TestSchedulerTickReconcilesTerminalSnapshotWithoutDispatch(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()
	local := models.IssueSnapshot{IssueID: "issue-1", Identifier: "DEZ-6", Title: "Terminal issue", URL: "https://linear.app/issue/DEZ-6", State: "Human Review", FetchedAt: time.Now()}
	if err := repo.UpsertIssueSnapshot(ctx, &local); err != nil {
		t.Fatal(err)
	}

	cfg := config.Defaults()
	tracker := &fakeTracker{issuesByID: map[string]issues.Issue{
		"issue-1": {ID: "issue-1", Identifier: "DEZ-6", Title: "Terminal issue", URL: local.URL, State: "Done"},
	}}
	scheduler := NewScheduler(&cfg, repo, tracker, nil, app.NewBus(), logrus.New())

	if err := scheduler.Tick(ctx); err != nil {
		t.Fatal(err)
	}

	snapshot, err := repo.RuntimeSnapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got := snapshot.Issues[0].State; got != "Done" {
		t.Fatalf("snapshot state = %q, want Done", got)
	}
	if tracker.candidateFetches != 1 {
		t.Fatalf("candidate fetches = %d, want 1", tracker.candidateFetches)
	}
	if tracker.issueFetches != 1 {
		t.Fatalf("issue fetches = %d, want 1", tracker.issueFetches)
	}
	if len(snapshot.Runs) != 0 {
		t.Fatalf("runs = %d, want 0", len(snapshot.Runs))
	}
}

func TestSchedulerTickPreservesTerminalWorkspaceWhenConfigured(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()
	local := models.IssueSnapshot{IssueID: "issue-1", Identifier: "DEZ-6", Title: "Terminal issue", URL: "https://linear.app/issue/DEZ-6", State: "Human Review", FetchedAt: time.Now()}
	if err := repo.UpsertIssueSnapshot(ctx, &local); err != nil {
		t.Fatal(err)
	}
	if err := repo.UpsertWorkspace(ctx, &models.Workspace{IssueID: local.IssueID, IssueIdentifier: local.Identifier, Path: t.TempDir(), BranchName: "agent/dez-6", Status: "active"}); err != nil {
		t.Fatal(err)
	}

	cfg := config.Defaults()
	cfg.Workspace.PreserveSuccessful = true
	tracker := &fakeTracker{issuesByID: map[string]issues.Issue{
		"issue-1": {ID: "issue-1", Identifier: "DEZ-6", Title: "Terminal issue", URL: local.URL, State: "Done"},
	}}
	scheduler := NewScheduler(&cfg, repo, tracker, nil, app.NewBus(), logrus.New())

	if err := scheduler.Tick(ctx); err != nil {
		t.Fatal(err)
	}

	workspaces, err := repo.ListWorkspaces(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got := workspaces[0].Status; got != "preserved" {
		t.Fatalf("workspace status = %q, want preserved", got)
	}
	var events []models.Event
	if err := repo.DB().WithContext(ctx).Where("type = ?", "worktree.terminal_preserved").Find(&events).Error; err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("terminal preserved events = %d, want 1", len(events))
	}
}

func TestSchedulerTickPreservesDirtyTerminalWorkspace(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()
	local := models.IssueSnapshot{IssueID: "issue-1", Identifier: "DEZ-6", Title: "Terminal issue", URL: "https://linear.app/issue/DEZ-6", State: "Human Review", FetchedAt: time.Now()}
	if err := repo.UpsertIssueSnapshot(ctx, &local); err != nil {
		t.Fatal(err)
	}
	if err := repo.UpsertWorkspace(ctx, &models.Workspace{IssueID: local.IssueID, IssueIdentifier: local.Identifier, Path: t.TempDir(), BranchName: "agent/dez-6", Status: "active", Dirty: true}); err != nil {
		t.Fatal(err)
	}

	cfg := config.Defaults()
	cfg.Workspace.PreserveSuccessful = false
	tracker := &fakeTracker{issuesByID: map[string]issues.Issue{
		"issue-1": {ID: "issue-1", Identifier: "DEZ-6", Title: "Terminal issue", URL: local.URL, State: "Done"},
	}}
	scheduler := NewScheduler(&cfg, repo, tracker, nil, app.NewBus(), logrus.New())

	if err := scheduler.Tick(ctx); err != nil {
		t.Fatal(err)
	}

	workspaces, err := repo.ListWorkspaces(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got := workspaces[0].Status; got != "preserved_dirty" {
		t.Fatalf("workspace status = %q, want preserved_dirty", got)
	}
	if !workspaces[0].Dirty {
		t.Fatal("workspace dirty = false, want true")
	}
}

func TestSchedulerTickCreatesWorkspaceAndPersistsRunMetadata(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()
	gitRepo := initTestGitRepo(t)
	workspaceRoot := t.TempDir()
	issue := issues.Issue{ID: "issue-8", Identifier: "DEZ-8", Title: "Restore worktree tracking", URL: "https://linear.app/issue/DEZ-8", State: "Ready for Agent"}

	capture := &capturingRunner{}
	cfg := config.Defaults()
	cfg.Workspace.RepoPath = gitRepo
	cfg.Workspace.Root = workspaceRoot
	cfg.Workspace.BaseBranch = "main"
	cfg.Workspace.BranchPrefix = "agent"
	cfg.Handoff.UpdateLinear = false
	tracker := &fakeTracker{candidates: []issues.Issue{issue}}
	worker := NewWorker(&cfg, &config.Workflow{Body: "test workflow"}, repo, tracker, capture, app.NewBus(), logrus.New())
	scheduler := NewScheduler(&cfg, repo, tracker, worker, app.NewBus(), logrus.New())
	scheduler.SetSynchronousWorkers(true)

	if err := scheduler.Tick(ctx); err != nil {
		t.Fatal(err)
	}

	workspaces, err := repo.ListWorkspaces(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(workspaces) != 1 {
		t.Fatalf("workspaces = %d, want 1", len(workspaces))
	}
	w := workspaces[0]
	wantBranch := gitpkg.BranchName(cfg.Workspace.BranchPrefix, issue.Identifier, issue.Title)
	if w.Path == "" || !strings.HasPrefix(w.Path, workspaceRoot) {
		t.Fatalf("workspace path = %q, want path under %q", w.Path, workspaceRoot)
	}
	if _, err := os.Stat(w.Path); err != nil {
		t.Fatalf("workspace path was not created: %v", err)
	}
	if w.BranchName != wantBranch {
		t.Fatalf("workspace branch = %q, want %q", w.BranchName, wantBranch)
	}
	if w.Status != "active" {
		t.Fatalf("workspace status = %q, want active", w.Status)
	}
	if w.Dirty {
		t.Fatal("workspace dirty = true, want false")
	}

	snapshot, err := repo.RuntimeSnapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Runs) != 1 {
		t.Fatalf("runs = %d, want 1", len(snapshot.Runs))
	}
	run := snapshot.Runs[0]
	if run.StartedAt.IsZero() {
		t.Fatal("run started_at was not persisted")
	}
	if run.WorktreePath != w.Path {
		t.Fatalf("run worktree path = %q, want %q", run.WorktreePath, w.Path)
	}
	if run.BranchName != wantBranch {
		t.Fatalf("run branch = %q, want %q", run.BranchName, wantBranch)
	}
	if capture.req.WorktreeDir != w.Path {
		t.Fatalf("runner worktree dir = %q, want %q", capture.req.WorktreeDir, w.Path)
	}
	if capture.req.BranchName != wantBranch {
		t.Fatalf("runner branch = %q, want %q", capture.req.BranchName, wantBranch)
	}
	if !strings.Contains(capture.req.Prompt, "Worktree: "+w.Path) {
		t.Fatalf("prompt missing worktree path: %q", capture.req.Prompt)
	}
	if !strings.Contains(capture.req.Prompt, "Branch: "+wantBranch) {
		t.Fatalf("prompt missing branch: %q", capture.req.Prompt)
	}
}

func newTestRepository(t *testing.T) *db.Repository {
	t.Helper()
	g, err := db.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Migrate(g); err != nil {
		t.Fatal(err)
	}
	return db.NewRepository(g)
}

func initTestGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init", "-b", "main")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", "README.md")
	runGit(t, dir, "-c", "user.name=Test User", "-c", "user.email=test@example.com", "commit", "-m", "initial")
	return dir
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}
