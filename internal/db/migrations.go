package db

import (
	"gorm.io/gorm"

	"issue-orchestrator/internal/db/models"
)

func Migrate(g *gorm.DB) error {
	return g.AutoMigrate(&models.Run{}, &models.Event{}, &models.Lock{}, &models.IssueSnapshot{}, &models.Workspace{})
}
