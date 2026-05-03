package db

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"issue-orchestrator/internal/db/models"
)

type Repository struct{ db *gorm.DB }

func NewRepository(g *gorm.DB) *Repository { return &Repository{db: g} }
func (r *Repository) DB() *gorm.DB         { return r.db }
func (r *Repository) CreateRun(ctx context.Context, run *models.Run) error {
	return r.db.WithContext(ctx).Create(run).Error
}
func (r *Repository) UpdateRunState(ctx context.Context, id, state string) error {
	return r.db.WithContext(ctx).Model(&models.Run{}).Where("id = ?", id).Updates(map[string]any{"state": state, "last_heartbeat_at": time.Now()}).Error
}
func (r *Repository) FinishRun(ctx context.Context, id, state, errText string) error {
	now := time.Now()
	vals := map[string]any{"state": state, "finished_at": &now}
	if errText != "" {
		vals["error"] = &errText
	}
	return r.db.WithContext(ctx).Model(&models.Run{}).Where("id = ?", id).Updates(vals).Error
}
func (r *Repository) AppendEvent(ctx context.Context, e *models.Event) error {
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now()
	}
	return r.db.WithContext(ctx).Create(e).Error
}
func (r *Repository) AcquireIssueLock(ctx context.Context, issueID, runID string, ttl time.Duration) (bool, error) {
	now := time.Now()
	l := models.Lock{IssueID: issueID, RunID: runID, ExpiresAt: now.Add(ttl)}
	err := r.db.WithContext(ctx).Create(&l).Error
	if err == nil {
		return true, nil
	}
	var old models.Lock
	if e := r.db.WithContext(ctx).Where("issue_id = ? AND expires_at < ?", issueID, now).First(&old).Error; e == nil {
		old.RunID = runID
		old.ExpiresAt = now.Add(ttl)
		return true, r.db.WithContext(ctx).Save(&old).Error
	}
	return false, nil
}
func (r *Repository) ReleaseIssueLock(ctx context.Context, issueID, runID string) error {
	return r.db.WithContext(ctx).Where("issue_id = ? AND run_id = ?", issueID, runID).Delete(&models.Lock{}).Error
}
func (r *Repository) ReleaseIssueLocks(ctx context.Context, issueID string) error {
	return r.db.WithContext(ctx).Where("issue_id = ?", issueID).Delete(&models.Lock{}).Error
}
func (r *Repository) CleanupExpiredLocks(ctx context.Context) error {
	return r.db.WithContext(ctx).Where("expires_at < ?", time.Now()).Delete(&models.Lock{}).Error
}
func (r *Repository) FindActiveRunByIssue(ctx context.Context, issueID string) (*models.Run, error) {
	var run models.Run
	err := r.db.WithContext(ctx).Where("issue_id = ? AND state IN ?", issueID, []string{"claimed", "preparing", "running_agent", "validating", "retry_queued"}).First(&run).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &run, err
}
func (r *Repository) ListActiveRuns(ctx context.Context) ([]models.Run, error) {
	var runs []models.Run
	err := r.db.WithContext(ctx).Where("state IN ?", []string{"claimed", "preparing", "running_agent", "validating", "retry_queued"}).Order("updated_at desc").Find(&runs).Error
	return runs, err
}
func (r *Repository) ListRecentRuns(ctx context.Context, limit int) ([]models.Run, error) {
	var runs []models.Run
	err := r.db.WithContext(ctx).Order("updated_at desc").Limit(limit).Find(&runs).Error
	return runs, err
}
func (r *Repository) ListEventsByRun(ctx context.Context, runID string, limit int) ([]models.Event, error) {
	var ev []models.Event
	q := r.db.WithContext(ctx).Where("run_id = ?", runID).Order("created_at desc")
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&ev).Error
	return ev, err
}
func (r *Repository) ListLocks(ctx context.Context) ([]models.Lock, error) {
	var l []models.Lock
	err := r.db.WithContext(ctx).Order("expires_at asc").Find(&l).Error
	return l, err
}
func (r *Repository) ListWorkspaces(ctx context.Context) ([]models.Workspace, error) {
	var w []models.Workspace
	err := r.db.WithContext(ctx).Order("updated_at desc").Find(&w).Error
	return w, err
}
func (r *Repository) FindWorkspaceByIssue(ctx context.Context, issueID string) (*models.Workspace, error) {
	var w models.Workspace
	err := r.db.WithContext(ctx).Where("issue_id = ?", issueID).First(&w).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &w, err
}
func (r *Repository) UpdateWorkspaceStatus(ctx context.Context, issueID, status string, dirty bool) error {
	return r.db.WithContext(ctx).Model(&models.Workspace{}).Where("issue_id = ?", issueID).Updates(map[string]any{"status": status, "dirty": dirty}).Error
}
func (r *Repository) ListIssueSnapshots(ctx context.Context) ([]models.IssueSnapshot, error) {
	var snapshots []models.IssueSnapshot
	err := r.db.WithContext(ctx).Order("fetched_at desc").Find(&snapshots).Error
	return snapshots, err
}
func (r *Repository) UpsertIssueSnapshot(ctx context.Context, s *models.IssueSnapshot) error {
	var old models.IssueSnapshot
	err := r.db.WithContext(ctx).Where("issue_id = ?", s.IssueID).First(&old).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return r.db.WithContext(ctx).Create(s).Error
	}
	if err != nil {
		return err
	}
	s.ID = old.ID
	return r.db.WithContext(ctx).Save(s).Error
}
func (r *Repository) UpdateIssueSnapshotState(ctx context.Context, issueID, state string) error {
	return r.db.WithContext(ctx).Model(&models.IssueSnapshot{}).Where("issue_id = ?", issueID).Updates(map[string]any{"state": state, "fetched_at": time.Now()}).Error
}
func (r *Repository) MarkIssueSnapshotRemoved(ctx context.Context, issueID, state string) error {
	return r.db.WithContext(ctx).Model(&models.IssueSnapshot{}).Where("issue_id = ?", issueID).Updates(map[string]any{"state": state, "fetched_at": time.Now()}).Error
}
func (r *Repository) UpsertWorkspace(ctx context.Context, w *models.Workspace) error {
	var old models.Workspace
	err := r.db.WithContext(ctx).Where("issue_id = ?", w.IssueID).First(&old).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return r.db.WithContext(ctx).Create(w).Error
	}
	if err != nil {
		return err
	}
	w.ID = old.ID
	return r.db.WithContext(ctx).Save(w).Error
}

type RuntimeSnapshotData struct {
	Runs       []models.Run
	Issues     []models.IssueSnapshot
	Workspaces []models.Workspace
	Locks      []models.Lock
}

func (r *Repository) RuntimeSnapshot(ctx context.Context) (*RuntimeSnapshotData, error) {
	d := &RuntimeSnapshotData{}
	if err := r.db.WithContext(ctx).Order("updated_at desc").Limit(100).Find(&d.Runs).Error; err != nil {
		return nil, err
	}
	if err := r.db.WithContext(ctx).Order("fetched_at desc").Limit(100).Find(&d.Issues).Error; err != nil {
		return nil, err
	}
	d.Workspaces, _ = r.ListWorkspaces(ctx)
	d.Locks, _ = r.ListLocks(ctx)
	return d, nil
}
