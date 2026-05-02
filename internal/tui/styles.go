package tui

import (
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
)

var (
	accent       = lipgloss.Color("5")
	accentStrong = lipgloss.Color("13")
	muted        = lipgloss.Color("8")
	border       = lipgloss.Color("8")
	success      = lipgloss.Color("2")
	warning      = lipgloss.Color("3")
	danger       = lipgloss.Color("1")
	info         = lipgloss.Color("4")

	appStyle = lipgloss.NewStyle().Padding(1, 2)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accentStrong).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(border).
			Padding(0, 1)

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(border).
			Padding(0, 1)

	panelTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(accent)
	footerStyle     = lipgloss.NewStyle().Foreground(muted)
	errStyle        = lipgloss.NewStyle().Foreground(danger).Bold(true)
	badgeStyle      = lipgloss.NewStyle().Bold(true).Foreground(accentStrong)
	mutedStyle      = lipgloss.NewStyle().Foreground(muted)
	successStyle    = lipgloss.NewStyle().Foreground(success).Bold(true)
	warningStyle    = lipgloss.NewStyle().Foreground(warning).Bold(true)
	dangerStyle     = lipgloss.NewStyle().Foreground(danger).Bold(true)
	infoStyle       = lipgloss.NewStyle().Foreground(info).Bold(true)
)

func styledTable(t table.Model) table.Model {
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(border).
		BorderBottom(true).
		Bold(true).
		Foreground(accentStrong)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("0")).
		Background(accent).
		Bold(true)
	t.SetStyles(s)
	return t
}
