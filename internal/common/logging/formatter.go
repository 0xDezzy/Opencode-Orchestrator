package logging

import "fmt"

func FormatLogEvent(e LogEvent) string {
	c := e.Component
	if c != "" {
		c = " [" + c + "]"
	}
	return fmt.Sprintf("%s %-5s%s %s", e.Time.Format("15:04:05"), e.Level, c, e.Message)
}
