package git

import (
	"path/filepath"
	"regexp"
	"strings"
)

var bad = regexp.MustCompile(`[^a-zA-Z0-9._/-]+`)

func SanitizePart(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = bad.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-./_")
	if len(s) > 48 {
		s = s[:48]
	}
	if s == "" {
		s = "issue"
	}
	return s
}
func SafeIssueSlug(identifier, title string) string {
	return SanitizePart(identifier) + "-" + SanitizePart(title)
}
func Inside(parent, child string) bool {
	p, _ := filepath.Abs(parent)
	c, _ := filepath.Abs(child)
	rel, err := filepath.Rel(p, c)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, "../")
}
