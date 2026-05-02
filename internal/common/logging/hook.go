package logging

import (
	"encoding/json"
	"time"

	"github.com/sirupsen/logrus"
)

type LogEvent struct {
	Time      time.Time
	Level     string
	Component string
	Message   string
	Fields    map[string]any
	RawJSON   string
}
type Hook struct{ ch chan LogEvent }

func NewHook(buffer int) *Hook         { return &Hook{ch: make(chan LogEvent, buffer)} }
func (h *Hook) Levels() []logrus.Level { return logrus.AllLevels }
func (h *Hook) Fire(e *logrus.Entry) error {
	fields := map[string]any{}
	for k, v := range e.Data {
		fields[k] = v
	}
	raw, _ := json.Marshal(fields)
	ev := LogEvent{Time: e.Time, Level: e.Level.String(), Message: e.Message, Fields: fields, RawJSON: string(raw)}
	if c, ok := fields["component"].(string); ok {
		ev.Component = c
	}
	select {
	case h.ch <- ev:
	default:
		<-h.ch
		h.ch <- ev
	}
	return nil
}
func (h *Hook) Events() <-chan LogEvent { return h.ch }
