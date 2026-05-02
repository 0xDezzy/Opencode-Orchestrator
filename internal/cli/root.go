package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

type flags struct {
	config, workflow, logLevel                 string
	tui, noTUI, jsonLogs, plainLogs, once, yes bool
	issue                                      string
	pollInterval                               string
}

var f flags

func Execute() {
	root := rootCmd()
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func rootCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "orchestrator"}
	cmd.PersistentFlags().StringVarP(&f.config, "config", "c", "", "config file")
	cmd.PersistentFlags().StringVarP(&f.workflow, "workflow", "w", "", "workflow file")
	cmd.PersistentFlags().StringVar(&f.logLevel, "log-level", "", "log level")
	cmd.AddCommand(daemonCmd(), runCmd(), statusCmd(), dbCmd(), versionCmd())
	return cmd
}
