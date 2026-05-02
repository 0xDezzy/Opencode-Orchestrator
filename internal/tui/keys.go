package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Quit       key.Binding
	ToggleHelp key.Binding
	Runs       key.Binding
	Issues     key.Binding
	Agents     key.Binding
	Workspaces key.Binding
	Locks      key.Binding
	Details    key.Binding
	Filter     key.Binding
	Close      key.Binding
	Top        key.Binding
	Bottom     key.Binding
	Pause      key.Binding
	ClearLogs  key.Binding
	Tick       key.Binding
}

func defaultKeys() keyMap {
	return keyMap{
		Quit:       key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		ToggleHelp: key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "more")),
		Runs:       key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "runs")),
		Issues:     key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "issues")),
		Agents:     key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "agents")),
		Workspaces: key.NewBinding(key.WithKeys("w"), key.WithHelp("w", "workspaces")),
		Locks:      key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "locks")),
		Details:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "details")),
		Filter:     key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		Close:      key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "close")),
		Top:        key.NewBinding(key.WithKeys("g", "home"), key.WithHelp("g", "top")),
		Bottom:     key.NewBinding(key.WithKeys("G", "end"), key.WithHelp("G", "bottom")),
		Pause:      key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "pause logs")),
		ClearLogs:  key.NewBinding(key.WithKeys("ctrl+l"), key.WithHelp("ctrl+l", "clear logs")),
		Tick:       key.NewBinding(key.WithKeys("R"), key.WithHelp("R", "tick now")),
	}
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Quit, k.ToggleHelp, k.Runs, k.Issues, k.Details, k.Filter, k.Pause}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Quit, k.ToggleHelp, k.Close},
		{k.Runs, k.Issues, k.Agents, k.Workspaces, k.Locks},
		{k.Details, k.Filter, k.Top, k.Bottom},
		{k.Pause, k.ClearLogs, k.Tick},
	}
}
