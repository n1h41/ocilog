package theme

import "github.com/charmbracelet/lipgloss"

// Shared styles for the fw-oci TUI.
var (
	App = lipgloss.NewStyle().Padding(1)

	Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4")).
		Padding(0, 1)

	Help  = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))
	Error = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5F87"))

	JSONKey     = lipgloss.NewStyle().Foreground(lipgloss.Color("#7DCFFF"))
	JSONString  = lipgloss.NewStyle().Foreground(lipgloss.Color("#9ECE6A"))
	JSONNumber  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF9E64"))
	JSONLiteral = lipgloss.NewStyle().Foreground(lipgloss.Color("#BB9AF7"))
	JSONPunct   = lipgloss.NewStyle().Foreground(lipgloss.Color("#565F89"))

	Sidebar = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7D56F4")).
		Padding(0, 1)

	Pane = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#3B4261")).
		Padding(0, 1)

	PaneActive = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	SidebarTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F4"))

	Selected = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4"))
)
