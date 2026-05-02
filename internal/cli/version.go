package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"issue-orchestrator/internal/version"
)

func versionCmd() *cobra.Command {
	return &cobra.Command{Use: "version", Run: func(cmd *cobra.Command, args []string) { fmt.Println(version.String()) }}
}
