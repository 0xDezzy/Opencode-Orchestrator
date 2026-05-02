package models

import "time"

type Lock struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	IssueID   string    `gorm:"uniqueIndex" json:"issue_id"`
	RunID     string    `gorm:"index" json:"run_id"`
	ExpiresAt time.Time `gorm:"index" json:"expires_at"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
