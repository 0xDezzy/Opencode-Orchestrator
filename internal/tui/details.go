package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
)

func (m *Model) detailView(width int) string {
	row := m.table.SelectedRow()
	if len(row) == 0 {
		return panelStyle.Width(width).Render(mutedStyle.Render("No row selected"))
	}
	cols := columnsFor(m.view)
	lines := make([]string, 0, len(row)+4)
	lines = append(lines, fmt.Sprintf("## %s Details", viewLabel(m.view)), "")
	for i, cell := range row {
		label := fmt.Sprintf("col%d", i+1)
		if i < len(cols) {
			label = cols[i].Title
		}
		lines = append(lines, fmt.Sprintf("- **%s:** %s", label, cell))
	}
	if len(m.runtimeEvents) > 0 {
		last := m.runtimeEvents[len(m.runtimeEvents)-1]
		lines = append(lines, "", fmt.Sprintf("- **Last event:** `%s`", formatRuntimeEvent(last)))
	}
	markdown := strings.Join(lines, "\n")
	rendered := markdown
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStyles(terminalMarkdownStyle()),
		glamour.WithWordWrap(maxInt(24, width-4)),
	)
	if err == nil {
		if out, renderErr := renderer.Render(markdown); renderErr == nil {
			rendered = strings.TrimRight(out, "\n")
		}
	}
	return panelStyle.Width(width).Render(rendered)
}

func terminalMarkdownStyle() ansi.StyleConfig {
	return ansi.StyleConfig{
		Document: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{BlockSuffix: "\n"},
		},
		Heading: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				BlockSuffix: "\n",
				Color:       stringPtr("5"),
				Bold:        boolPtr(true),
			},
		},
		H1: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Prefix: "", Suffix: ""}},
		H2: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Prefix: "", Suffix: ""}},
		H3: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Prefix: "", Suffix: ""}},
		List: ansi.StyleList{
			StyleBlock:  ansi.StyleBlock{Margin: uintPtr(0)},
			LevelIndent: 2,
		},
		Item: ansi.StylePrimitive{BlockPrefix: "• "},
		Strong: ansi.StylePrimitive{
			Bold: boolPtr(true),
		},
		Emph: ansi.StylePrimitive{
			Italic: boolPtr(true),
		},
		Code: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{
			Color: stringPtr("6"),
		}},
		Link: ansi.StylePrimitive{
			Color:     stringPtr("4"),
			Underline: boolPtr(true),
		},
		LinkText: ansi.StylePrimitive{
			Color: stringPtr("4"),
			Bold:  boolPtr(true),
		},
	}
}

func stringPtr(v string) *string { return &v }
func boolPtr(v bool) *bool       { return &v }
func uintPtr(v uint) *uint       { return &v }
