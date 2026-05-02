package models

import "time"

type Run struct {
	ID                                  string `gorm:"primaryKey" json:"id"`
	IssueID                             string `gorm:"index" json:"issue_id"`
	IssueIdentifier                     string `gorm:"index" json:"issue_identifier"`
	IssueURL                            string `json:"issue_url"`
	State                               string `gorm:"index" json:"state"`
	Attempt                             int    `json:"attempt"`
	WorktreePath, BranchName, AgentName string
	AgentSessionID                      *string
	StartedAt                           time.Time
	FinishedAt                          *time.Time
	LastHeartbeatAt                     *time.Time
	Error                               *string
	Summary                             *string
	ChangedFilesJSON                    *string
	CreatedAt                           time.Time
	UpdatedAt                           time.Time
}
