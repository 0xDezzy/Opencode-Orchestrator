package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	cfgpkg "issue-orchestrator/internal/common/config"
	"issue-orchestrator/internal/db"
)

func dbCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "db"}
	mig := &cobra.Command{Use: "migrate", RunE: func(cmd *cobra.Command, args []string) error {
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
		fmt.Println("database migrated:", c.SQLite.Path)
		return nil
	}}
	reset := &cobra.Command{Use: "reset", RunE: func(cmd *cobra.Command, args []string) error {
		if !f.yes {
			return fmt.Errorf("--yes is required")
		}
		c, err := cfgpkg.Load(f.config)
		if err != nil {
			return err
		}
		_ = os.Remove(c.SQLite.Path)
		g, err := db.Open(c.SQLite.Path)
		if err != nil {
			return err
		}
		return db.Migrate(g)
	}}
	reset.Flags().BoolVar(&f.yes, "yes", false, "confirm reset")
	cmd.AddCommand(mig, reset)
	return cmd
}
