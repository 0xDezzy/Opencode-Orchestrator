package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	cfgpkg "issue-orchestrator/internal/common/config"
	"issue-orchestrator/internal/db"
)

func statusCmd() *cobra.Command {
	return &cobra.Command{Use: "status", RunE: func(cmd *cobra.Command, args []string) error {
		c, err := cfgpkg.Load(f.config)
		if err != nil {
			return err
		}
		g, err := db.Open(c.SQLite.Path)
		if err != nil {
			return err
		}
		if err := db.Migrate(g); err != nil {
			return err
		}
		repo := db.NewRepository(g)
		active, _ := repo.ListActiveRuns(cmd.Context())
		recent, _ := repo.ListRecentRuns(cmd.Context(), 10)
		locks, _ := repo.ListLocks(cmd.Context())
		fmt.Println("Active runs", len(active))
		fmt.Println("Recent runs", len(recent))
		fmt.Println("Current locks", len(locks))
		fmt.Println("Configured workspace root", c.Workspace.Root)
		fmt.Println("Configured Linear team/project filters", c.Linear.TeamKey, c.Linear.ProjectName)
		return nil
	}}
}
