package logging

import (
	"testing"

	"github.com/sirupsen/logrus"
)

func TestHookConvertsLogrusEntry(t *testing.T) {
	h := NewHook(2)
	l := logrus.New()
	l.AddHook(h)
	l.WithField("component", "test").Warn("hello")
	ev := <-h.Events()
	if ev.Component != "test" || ev.Level != "warning" || ev.Message != "hello" {
		t.Fatalf("bad event %+v", ev)
	}
}
