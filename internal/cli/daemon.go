package cli

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	"issue-orchestrator/internal/app"
	"issue-orchestrator/internal/orchestrator"
	"issue-orchestrator/internal/server"
	"issue-orchestrator/internal/tui"
)

func daemonCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "daemon", RunE: func(cmd *cobra.Command, args []string) error {
		tuiMode := !f.noTUI && (f.tui || (!f.once && isatty.IsTerminal(os.Stdout.Fd())))
		c, wf, repo, tracker, runner, bus, log, err := initDeps(tuiMode)
		if err != nil {
			return err
		}
		ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
		defer cancel()
		worker := orchestrator.NewWorker(c, wf, repo, tracker, runner, bus, log)
		sched := orchestrator.NewScheduler(c, repo, tracker, worker, bus, log)
		ctrl := app.New(repo, bus)
		ctrl.SetTick(sched.Tick)
		var srv *server.Server
		if c.Server.Enabled {
			srv = server.New(c.Server.Address, repo)
			go func() {
				if err := srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
					log.WithError(err).Error("status server failed")
				}
			}()
		}
		ctrl.SetShutdown(func(ctx context.Context) error {
			cancel()
			if srv != nil {
				return srv.Shutdown(ctx)
			}
			return nil
		})
		if f.once {
			sched.SetSynchronousWorkers(true)
			return sched.Tick(ctx)
		}
		go func() { _ = sched.Run(ctx) }()
		if tuiMode {
			return tui.Run(ctx, ctrl, bus, runtimeLogHook, c.TUI.MaxLogLines)
		}
		<-ctx.Done()
		if srv != nil {
			_ = srv.Shutdown(context.Background())
		}
		return nil
	}}
	cmd.Flags().BoolVar(&f.once, "once", false, "run one scheduler tick")
	cmd.Flags().BoolVar(&f.tui, "tui", false, "force TUI")
	cmd.Flags().BoolVar(&f.noTUI, "no-tui", false, "disable TUI")
	cmd.Flags().BoolVar(&f.jsonLogs, "json-logs", false, "json logs")
	cmd.Flags().BoolVar(&f.plainLogs, "plain-logs", false, "plain logs")
	cmd.Flags().StringVar(&f.pollInterval, "poll-interval", "", "poll interval")
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if f.jsonLogs && f.plainLogs {
			return fmt.Errorf("choose one log format")
		}
		if f.pollInterval != "" {
			if _, err := time.ParseDuration(f.pollInterval); err != nil {
				return err
			}
		}
		return nil
	}
	return cmd
}
