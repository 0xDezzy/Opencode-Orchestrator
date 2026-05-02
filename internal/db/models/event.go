package models

import "time"

type Event struct {
	ID                           uint   `gorm:"primaryKey" json:"id"`
	RunID                        string `gorm:"index" json:"run_id"`
	IssueID                      string `gorm:"index" json:"issue_id"`
	Type                         string `gorm:"index" json:"type"`
	Source, Message, PayloadJSON string
	CreatedAt                    time.Time `gorm:"index" json:"created_at"`
}
