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
)
