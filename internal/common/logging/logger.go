package logging

import (
	"io"
	"log"
	"log/slog"
	"os"

	"github.com/sirupsen/logrus"
)

type Options struct {
	Level, Format string
	TUI           bool
	FilePath      string
	MirrorToFile  bool
	Hook          *Hook
}

func Setup(o Options) (*logrus.Logger, error) {
	if o.TUI {
		// Keep dependency logs from writing below Bubble Tea's renderer.
		log.SetOutput(io.Discard)
		slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	}
	l := logrus.New()
	lvl, err := logrus.ParseLevel(o.Level)
	if err != nil {
		lvl = logrus.InfoLevel
	}
	l.SetLevel(lvl)
	if o.Format == "json" {
		l.SetFormatter(&logrus.JSONFormatter{})
	} else {
		l.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	}
	if o.Hook != nil {
		l.AddHook(o.Hook)
	}
	var outs []io.Writer
	if !o.TUI {
		outs = append(outs, os.Stderr)
	}
	if o.TUI && o.MirrorToFile && o.FilePath != "" {
		f, err := openLogFile(o.FilePath)
		if err != nil {
			return nil, err
		}
		outs = append(outs, f)
	}
	if len(outs) == 0 {
		l.SetOutput(io.Discard)
	} else {
		l.SetOutput(io.MultiWriter(outs...))
	}
	return l, nil
}
