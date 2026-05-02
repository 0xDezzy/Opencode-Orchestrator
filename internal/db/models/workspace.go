package models

import "time"

type Workspace struct {
	ID                                   uint   `gorm:"primaryKey"`
	IssueID                              string `gorm:"uniqueIndex"`
	IssueIdentifier                      string `gorm:"index"`
	Path, BranchName, BaseBranch, Status string
	Dirty                                bool
	CreatedAt                            time.Time
	UpdatedAt                            time.Time
}
