package search

import (
	"strings"

	"github.com/charmbracelet/x/ansi"

	"n1h41/ocilog/internal/tui/theme"
)

// sidebarWidth returns the total width of the history sidebar, scaled to the
// terminal but kept within a readable range.
func (m Model) sidebarWidth() int {
	w := min(max(m.width/3, 24), 40)
	return w
}

// historySidebar renders the list of previous queries as a left side panel of
// the given total height.
func (m Model) historySidebar(height int) string {
	entries := m.history.Entries()

	inner := m.sidebarWidth() - 4
	contentH := max(height-2, 1)

	// Reserve a line each for the title, the spacer, and the footer.
	maxVisible := max(contentH-3, 1)

	lines := []string{theme.SidebarTitle.Render("history")}
	if len(entries) == 0 {
		lines = append(lines, theme.Help.Render("no previous searches"))
	} else {
		start := 0
		if m.historyIndex >= maxVisible {
			start = m.historyIndex - maxVisible + 1
		}
		end := min(start+maxVisible, len(entries))
		for i := start; i < end; i++ {
			e := entries[i]
			line := e.Query
			if e.Pipe != "" {
				line += "  | " + e.Pipe
			}
			if e.From != "" || e.To != "" {
				line += "  [" + e.From + " → " + e.To + "]"
			}
			line = ansi.Truncate(line, inner, "…")
			if i == m.historyIndex {
				if w := ansi.StringWidth(line); w < inner {
					line += strings.Repeat(" ", inner-w)
				}
				line = theme.Selected.Render(line)
			}
			lines = append(lines, line)
		}
	}

	footer := "enter:load  ctrl+x:clear"
	if m.confirmClear {
		footer = "ctrl+x again to clear"
	}
	lines = append(lines, "", theme.Help.Render(footer))

	// lipgloss Width includes padding but not the border, so add it back.
	return theme.Sidebar.Width(inner + 2).Height(contentH).Render(strings.Join(lines, "\n"))
}

// padLines pads s with blank lines until it is at least height lines tall.
func padLines(s string, height int) string {
	if height <= 0 {
		return s
	}
	lines := strings.Split(s, "\n")
	for len(lines) < height {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}
