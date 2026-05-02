package models

import "time"

type IssueSnapshot struct {
	ID                                                                      uint   `gorm:"primaryKey"`
	IssueID                                                                 string `gorm:"index"`
	Identifier                                                              string `gorm:"index"`
	Title, Description, URL, State, LabelsJSON, Priority, Assignee, RawJSON string
	FetchedAt                                                               time.Time `gorm:"index"`
	CreatedAt                                                               time.Time
	UpdatedAt                                                               time.Time
}
