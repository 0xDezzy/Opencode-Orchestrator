package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	orch "issue-orchestrator/internal/orchestrator"
)

func reconcileCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "reconcile", Short: "Reconcile local issue and worktree state", RunE: func(cmd *cobra.Command, args []string) error {
		c, _, repo, tracker, _, bus, log, err := initDeps(false)
		if err != nil {
			return err
		}
		reconciler := orch.NewReconciler(c, repo, tracker, bus, log)
		summary := reconciler.Reconcile(cmd.Context(), orch.ReconcileOptions{Issue: f.issue, DryRun: f.dryRun, Force: f.force})
		if f.json {
			b, err := json.MarshalIndent(summary, "", "  ")
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(b))
		} else {
			printReconcileSummary(cmd, summary)
		}
		if summary.Failed() {
			return fmt.Errorf("reconcile completed with %d failure(s)", summary.Failures)
		}
		return nil
	}}
	cmd.Flags().StringVar(&f.issue, "issue", "", "Linear issue identifier or ID")
	cmd.Flags().BoolVar(&f.dryRun, "dry-run", false, "preview reconcile changes without mutating local state")
	cmd.Flags().BoolVar(&f.force, "force", false, "allow forced clean worktree removal when git requires it")
	cmd.Flags().BoolVar(&f.json, "json", false, "print structured JSON output")
	return cmd
}

func printReconcileSummary(cmd *cobra.Command, summary orch.ReconcileSummary) {
	lines := []string{
		fmt.Sprintf("issues refreshed: %d", summary.IssuesRefreshed),
		fmt.Sprintf("issues removed/marked: %d", summary.IssuesRemoved),
		fmt.Sprintf("worktrees removed: %d", summary.WorktreesRemoved),
		fmt.Sprintf("worktrees preserved: %d", summary.WorktreesPreserved),
		fmt.Sprintf("locks released: %d", summary.LocksReleased),
		fmt.Sprintf("failures: %d", summary.Failures),
	}
	if len(summary.Decisions) > 0 {
		lines = append(lines, "decisions:")
		for _, decision := range summary.Decisions {
			lines = append(lines, "- "+decision)
		}
	}
	if len(summary.Errors) > 0 {
		lines = append(lines, "errors:")
		for _, err := range summary.Errors {
			lines = append(lines, "- "+err)
		}
	}
	fmt.Fprintln(cmd.OutOrStdout(), strings.Join(lines, "\n"))
}
