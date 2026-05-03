package orchestrator

import (
	"context"
	"encoding/json"
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

func TestSchedulerTickRefreshesNonTerminalSnapshotProperties(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()
	stale := time.Now().Add(-2 * time.Hour)
	local := models.IssueSnapshot{IssueID: "issue-1", Identifier: "DEZ-6", Title: "Old title", Description: "old", URL: "https://old", State: "Human Review", LabelsJSON: `["old"]`, Priority: "Low", Assignee: "Old", RawJSON: `{"old":true}`, FetchedAt: stale}
	if err := repo.UpsertIssueSnapshot(ctx, &local); err != nil {
		t.Fatal(err)
	}

	cfg := config.Defaults()
	cfg.Scheduler.IssueStaleAfter = time.Hour
	tracker := &fakeTracker{issuesByID: map[string]issues.Issue{
		"issue-1": {ID: "issue-1", Identifier: "DEZ-6A", Title: "New title", Description: "new", URL: "https://new", State: "Human Review", Labels: []string{"agent", "bug"}, Priority: "High", Assignee: "Alice", Raw: map[string]any{"new": true}},
	}}
	scheduler := NewScheduler(&cfg, repo, tracker, nil, app.NewBus(), logrus.New())

	if err := scheduler.Tick(ctx); err != nil {
		t.Fatal(err)
	}

	snapshot, err := repo.RuntimeSnapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	got := snapshot.Issues[0]
	if got.Identifier != "DEZ-6A" || got.Title != "New title" || got.Description != "new" || got.URL != "https://new" || got.State != "Human Review" || got.Priority != "High" || got.Assignee != "Alice" {
		t.Fatalf("snapshot was not fully refreshed: %+v", got)
	}
	var labels []string
	if err := json.Unmarshal([]byte(got.LabelsJSON), &labels); err != nil {
		t.Fatal(err)
	}
	if strings.Join(labels, ",") != "agent,bug" {
		t.Fatalf("labels = %q, want agent,bug", strings.Join(labels, ","))
	}
	if !strings.Contains(got.RawJSON, `"new":true`) {
		t.Fatalf("raw json = %q, want refreshed raw payload", got.RawJSON)
	}
	if !got.FetchedAt.After(stale) {
		t.Fatalf("fetched_at = %v, want after %v", got.FetchedAt, stale)
	}
}

func TestSchedulerTickMarksMissingIssueRemovedAndReleasesLock(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()
	local := models.IssueSnapshot{IssueID: "issue-1", Identifier: "DEZ-6", Title: "Missing issue", URL: "https://linear.app/issue/DEZ-6", State: "Human Review", FetchedAt: time.Now().Add(-2 * time.Hour)}
	if err := repo.UpsertIssueSnapshot(ctx, &local); err != nil {
		t.Fatal(err)
	}
	if err := repo.UpsertWorkspace(ctx, &models.Workspace{IssueID: local.IssueID, IssueIdentifier: local.Identifier, BranchName: "agent/dez-6", Status: "active"}); err != nil {
		t.Fatal(err)
	}
	if ok, err := repo.AcquireIssueLock(ctx, local.IssueID, "run-1", time.Hour); err != nil || !ok {
		t.Fatalf("acquire lock ok=%v err=%v", ok, err)
	}

	cfg := config.Defaults()
	cfg.Scheduler.IssueStaleAfter = time.Hour
	tracker := &fakeTracker{issuesByID: map[string]issues.Issue{}}
	scheduler := NewScheduler(&cfg, repo, tracker, nil, app.NewBus(), logrus.New())

	if err := scheduler.Tick(ctx); err != nil {
		t.Fatal(err)
	}

	snapshot, err := repo.RuntimeSnapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	allIssues, err := repo.ListIssueSnapshots(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got := allIssues[0].State; got != "deleted" {
		t.Fatalf("snapshot state = %q, want deleted", got)
	}
	if len(snapshot.Issues) != 0 {
		t.Fatalf("runtime issues = %d, want deleted issue hidden", len(snapshot.Issues))
	}
	if len(snapshot.Locks) != 0 {
		t.Fatalf("locks = %d, want 0", len(snapshot.Locks))
	}
	if got := snapshot.Workspaces[0].Status; got != "removed" {
		t.Fatalf("workspace status = %q, want removed", got)
	}
	var events []models.Event
	if err := repo.DB().WithContext(ctx).Where("type = ?", "linear.issue_removed").Find(&events).Error; err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("linear.issue_removed events = %d, want 1", len(events))
	}
}

func TestSchedulerTickSkipsFullReconcileUntilCadenceElapses(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()
	local := models.IssueSnapshot{IssueID: "issue-1", Identifier: "DEZ-6", Title: "Issue", URL: "https://linear.app/issue/DEZ-6", State: "Human Review", FetchedAt: time.Now().Add(-2 * time.Hour)}
	if err := repo.UpsertIssueSnapshot(ctx, &local); err != nil {
		t.Fatal(err)
	}

	cfg := config.Defaults()
	cfg.Scheduler.IssueStaleAfter = time.Hour
	cfg.Scheduler.ReconcileInterval = time.Hour
	tracker := &fakeTracker{issuesByID: map[string]issues.Issue{
		"issue-1": {ID: "issue-1", Identifier: "DEZ-6", Title: "Issue", URL: local.URL, State: "Human Review"},
	}}
	scheduler := NewScheduler(&cfg, repo, tracker, nil, app.NewBus(), logrus.New())

	if err := scheduler.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	if err := scheduler.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	if tracker.issueFetches != 1 {
		t.Fatalf("issue fetches = %d, want 1", tracker.issueFetches)
	}
}

func TestSchedulerTickDoesNotDispatchRemovedCandidate(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()
	issue := issues.Issue{ID: "issue-removed", Identifier: "DEZ-11", Title: "Removed", URL: "https://linear.app/issue/DEZ-11", State: "deleted"}

	cfg := config.Defaults()
	tracker := &fakeTracker{candidates: []issues.Issue{issue}}
	scheduler := NewScheduler(&cfg, repo, tracker, nil, app.NewBus(), logrus.New())

	if err := scheduler.Tick(ctx); err != nil {
		t.Fatal(err)
	}

	snapshot, err := repo.RuntimeSnapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Runs) != 0 {
		t.Fatalf("runs = %d, want 0", len(snapshot.Runs))
	}
	if len(snapshot.Workspaces) != 0 {
		t.Fatalf("workspaces = %d, want 0", len(snapshot.Workspaces))
	}
	allIssues, err := repo.ListIssueSnapshots(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got := allIssues[0].State; got != "deleted" {
		t.Fatalf("snapshot state = %q, want deleted", got)
	}
	if len(snapshot.Issues) != 0 {
		t.Fatalf("runtime issues = %d, want deleted issue hidden", len(snapshot.Issues))
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

func TestSchedulerTickSkipsActiveCandidateAndDispatchesLaterCandidate(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()
	gitRepo := initTestGitRepo(t)
	activeIssue := issues.Issue{ID: "issue-active", Identifier: "DEZ-12A", Title: "Already running", URL: "https://linear.app/issue/DEZ-12A", State: "Ready for Agent"}
	eligibleIssue := issues.Issue{ID: "issue-eligible", Identifier: "DEZ-12B", Title: "Can run", URL: "https://linear.app/issue/DEZ-12B", State: "Ready for Agent"}
	if err := repo.CreateRun(ctx, &models.Run{ID: "run-active", IssueID: activeIssue.ID, IssueIdentifier: activeIssue.Identifier, State: string(RunStateClaimed), Attempt: 1}); err != nil {
		t.Fatal(err)
	}

	cfg := config.Defaults()
	cfg.Scheduler.MaxConcurrentRuns = 2
	cfg.Workspace.RepoPath = gitRepo
	cfg.Workspace.Root = t.TempDir()
	cfg.Workspace.BaseBranch = "main"
	tracker := &fakeTracker{candidates: []issues.Issue{activeIssue, eligibleIssue}}
	scheduler := NewScheduler(&cfg, repo, tracker, nil, app.NewBus(), logrus.New())

	if err := scheduler.Tick(ctx); err != nil {
		t.Fatal(err)
	}

	activeRuns, err := repo.ListActiveRuns(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(activeRuns) != 2 {
		t.Fatalf("active runs = %d, want 2", len(activeRuns))
	}
	if !hasActiveRunForIssue(activeRuns, eligibleIssue.ID) {
		t.Fatalf("eligible issue %q was not dispatched", eligibleIssue.ID)
	}
	if tracker.lastOptions.Limit != cfg.Scheduler.MaxConcurrentRuns {
		t.Fatalf("candidate fetch limit = %d, want %d", tracker.lastOptions.Limit, cfg.Scheduler.MaxConcurrentRuns)
	}
}

func TestSchedulerTickExistingActiveRunsReduceAvailableSlots(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()
	gitRepo := initTestGitRepo(t)
	for _, run := range []models.Run{
		{ID: "run-active-1", IssueID: "issue-active-1", IssueIdentifier: "DEZ-12A", State: string(RunStateClaimed), Attempt: 1},
		{ID: "run-active-2", IssueID: "issue-active-2", IssueIdentifier: "DEZ-12B", State: string(RunStateRunningAgent), Attempt: 1},
	} {
		run := run
		if err := repo.CreateRun(ctx, &run); err != nil {
			t.Fatal(err)
		}
	}
	candidates := []issues.Issue{
		{ID: "issue-new-1", Identifier: "DEZ-12C", Title: "New one", URL: "https://linear.app/issue/DEZ-12C", State: "Ready for Agent"},
		{ID: "issue-new-2", Identifier: "DEZ-12D", Title: "New two", URL: "https://linear.app/issue/DEZ-12D", State: "Ready for Agent"},
		{ID: "issue-new-3", Identifier: "DEZ-12E", Title: "New three", URL: "https://linear.app/issue/DEZ-12E", State: "Ready for Agent"},
	}

	cfg := config.Defaults()
	cfg.Scheduler.MaxConcurrentRuns = 3
	cfg.Workspace.RepoPath = gitRepo
	cfg.Workspace.Root = t.TempDir()
	cfg.Workspace.BaseBranch = "main"
	tracker := &fakeTracker{candidates: candidates}
	scheduler := NewScheduler(&cfg, repo, tracker, nil, app.NewBus(), logrus.New())

	if err := scheduler.Tick(ctx); err != nil {
		t.Fatal(err)
	}

	activeRuns, err := repo.ListActiveRuns(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(activeRuns) != cfg.Scheduler.MaxConcurrentRuns {
		t.Fatalf("active runs = %d, want %d", len(activeRuns), cfg.Scheduler.MaxConcurrentRuns)
	}
	created := 0
	for _, issue := range candidates {
		if hasActiveRunForIssue(activeRuns, issue.ID) {
			created++
		}
	}
	if created != 1 {
		t.Fatalf("new active runs = %d, want 1", created)
	}
}

func TestSchedulerTickDoesNotDispatchMoreThanMaxConcurrentRuns(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()
	gitRepo := initTestGitRepo(t)
	candidates := []issues.Issue{
		{ID: "issue-new-1", Identifier: "DEZ-12A", Title: "New one", URL: "https://linear.app/issue/DEZ-12A", State: "Ready for Agent"},
		{ID: "issue-new-2", Identifier: "DEZ-12B", Title: "New two", URL: "https://linear.app/issue/DEZ-12B", State: "Ready for Agent"},
		{ID: "issue-new-3", Identifier: "DEZ-12C", Title: "New three", URL: "https://linear.app/issue/DEZ-12C", State: "Ready for Agent"},
	}

	cfg := config.Defaults()
	cfg.Scheduler.MaxConcurrentRuns = 2
	cfg.Workspace.RepoPath = gitRepo
	cfg.Workspace.Root = t.TempDir()
	cfg.Workspace.BaseBranch = "main"
	tracker := &fakeTracker{candidates: candidates}
	scheduler := NewScheduler(&cfg, repo, tracker, nil, app.NewBus(), logrus.New())

	if err := scheduler.Tick(ctx); err != nil {
		t.Fatal(err)
	}

	activeRuns, err := repo.ListActiveRuns(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(activeRuns) != cfg.Scheduler.MaxConcurrentRuns {
		t.Fatalf("active runs = %d, want %d", len(activeRuns), cfg.Scheduler.MaxConcurrentRuns)
	}
}

func hasActiveRunForIssue(runs []models.Run, issueID string) bool {
	for _, run := range runs {
		if run.IssueID == issueID {
			return true
		}
	}
	return false
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
