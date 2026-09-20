package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"

	"n1h41/ocilog/internal/tui/theme"
)

// helpKeyWidth is the column width reserved for the key column of the help
// overlay, wide enough for the longest binding label.
const helpKeyWidth = 19

// helpEntry is a single key/description pair shown in the help overlay.
type helpEntry struct {
	key  string
	desc string
}

var (
	helpKeyStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	helpDescStyle    = theme.Help
	helpSectionStyle = theme.SidebarTitle
)

var (
	helpNavEntries = []helpEntry{
		{"1 / 2 / 3 / 4", "switch tabs"},
		{"tab", "next tab"},
		{"esc / backspace", "go up one level"},
	}

	helpListEntries = []helpEntry{
		{"up / down", "navigate"},
		{"/", "filter"},
		{"space", "select / deselect"},
		{"enter", "open"},
	}

	helpGlobalEntries = []helpEntry{
		{"s", "search selected scope"},
		{"r", "refresh list"},
		{"? / ctrl+h", "toggle this help"},
		{"ctrl+c", "quit"},
	}

	helpSearchEntries = []helpEntry{
		{"tab / shift+tab", "cycle fields"},
		{"enter", "run search / apply pipeline"},
		{"ctrl+left/right", "switch result pane"},
		{"ctrl+t", "toggle result pane"},
		{"ctrl+x", "expand focused pane"},
		{"pgup / pgdn", "scroll results"},
		{"home / end", "top / bottom"},
		{"ctrl+r", "search history"},
		{"ctrl+y", "copy active pane"},
		{"ctrl+o", "copy as oci command"},
	}

	helpHistoryEntries = []helpEntry{
		{"up / down", "navigate"},
		{"enter", "load entry"},
		{"ctrl+x twice", "clear history"},
		{"esc / ctrl+r", "close"},
	}
)

// helpColumn renders a titled, key-aligned group of help entries.
func helpColumn(title string, entries []helpEntry) string {
	lines := []string{helpSectionStyle.Render(title)}
	for _, e := range entries {
		key := helpKeyStyle.Render(fmt.Sprintf("%-*s", helpKeyWidth, e.key))
		lines = append(lines, key+helpDescStyle.Render(e.desc))
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// helpView renders the full key reference as a centered modal.
func (m *Model) helpView() string {
	left := lipgloss.JoinVertical(lipgloss.Left,
		helpColumn("Navigation", helpNavEntries),
		"",
		helpColumn("Lists", helpListEntries),
		"",
		helpColumn("Global", helpGlobalEntries),
	)
	right := lipgloss.JoinVertical(lipgloss.Left,
		helpColumn("Search", helpSearchEntries),
		"",
		helpColumn("History", helpHistoryEntries),
	)

	body := lipgloss.JoinHorizontal(lipgloss.Top, left, "      ", right)
	footer := theme.Help.Render("press ? / esc to close")
	content := lipgloss.JoinVertical(lipgloss.Center,
		helpSectionStyle.Render("ocilog help"),
		"",
		body,
		"",
		footer,
	)
	if m.width > 4 {
		content = theme.Sidebar.Width(m.width - 8).Render(content)
	}
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}
