package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"n1h41/fw-oci/internal/oci"
	"n1h41/fw-oci/internal/tui/compartments"
	"n1h41/fw-oci/internal/tui/loggroups"
	"n1h41/fw-oci/internal/tui/logs"
	"n1h41/fw-oci/internal/tui/search"
	"n1h41/fw-oci/internal/tui/state"
	"n1h41/fw-oci/internal/tui/theme"
)

type tab int

const (
	tabCompartments tab = iota
	tabLogGroups
	tabLogs
	tabSearch
)

// Model is the root Bubble Tea model. It owns the shared session and delegates
// to the active view.
type Model struct {
	session *state.Session
	tab     tab

	compartments compartments.Model
	logGroups    loggroups.Model
	logs         logs.Model
	search       search.Model

	ready  bool
	width  int
	height int
}

func New(client *oci.Client, tenancyID, initialCompartment string) *Model {
	session := state.New(client, tenancyID, initialCompartment)
	return &Model{
		session:      session,
		compartments: compartments.New(session),
		logGroups:    loggroups.New(session),
		logs:         logs.New(session),
		search:       search.New(session),
	}
}

func (m *Model) Init() tea.Cmd {
	return m.compartments.LoadCmd()
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.ready = true
		m.width = msg.Width
		m.height = msg.Height
		m.resize()
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}
		if m.filterEditing() {
			return m, m.delegate(msg)
		}
		if m.tab != tabSearch {
			switch msg.String() {
			case "1":
				m.tab = tabCompartments
				return m, nil
			case "2":
				m.tab = tabLogGroups
				return m, nil
			case "3":
				m.tab = tabLogs
				return m, nil
			case "4":
				return m, m.enterSearch()
			case "tab":
				next := (m.tab + 1) % 4
				if next == tabSearch {
					return m, m.enterSearch()
				}
				m.tab = next
				return m, nil
			case "r":
				return m, m.refresh()
			case "s":
				return m, m.preloadSearch()
			}
		}
		return m, m.delegate(msg)

	case state.GoUp:
		return m, m.up()

	case state.OpenCompartment:
		m.session.CompartmentID = msg.ID
		m.session.CompartmentName = msg.Name
		m.tab = tabLogGroups
		return m, m.logGroups.LoadCmd()

	case state.OpenLogGroup:
		m.session.LogGroupID = msg.ID
		m.session.LogGroupName = msg.Name
		m.tab = tabLogs
		return m, m.logs.LoadCmd()

	default:
		return m, m.route(msg)
	}
}

// route sends load/result messages to the view that owns them (independent of
// the active tab), falling back to the active view for everything else.
func (m *Model) route(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case compartments.Loaded:
		var cmd tea.Cmd
		m.compartments, cmd = m.compartments.Update(msg)
		return cmd
	case loggroups.Loaded:
		var cmd tea.Cmd
		m.logGroups, cmd = m.logGroups.Update(msg)
		return cmd
	case logs.Loaded:
		var cmd tea.Cmd
		m.logs, cmd = m.logs.Update(msg)
		return cmd
	case search.Result:
		var cmd tea.Cmd
		m.search, cmd = m.search.Update(msg)
		return cmd
	}
	return m.delegate(msg)
}

// delegate forwards a message to the active view.
func (m *Model) delegate(msg tea.Msg) tea.Cmd {
	switch m.tab {
	case tabCompartments:
		var cmd tea.Cmd
		m.compartments, cmd = m.compartments.Update(msg)
		return cmd
	case tabLogGroups:
		var cmd tea.Cmd
		m.logGroups, cmd = m.logGroups.Update(msg)
		return cmd
	case tabLogs:
		var cmd tea.Cmd
		m.logs, cmd = m.logs.Update(msg)
		return cmd
	case tabSearch:
		var cmd tea.Cmd
		m.search, cmd = m.search.Update(msg)
		return cmd
	}
	return nil
}

func (m *Model) resize() {
	contentHeight := m.height - 7
	m.compartments.Resize(m.width, contentHeight)
	m.logGroups.Resize(m.width, contentHeight)
	m.logs.Resize(m.width, contentHeight)
	m.search.Resize(m.width, contentHeight)
}

func (m *Model) enterSearch() tea.Cmd {
	m.search.EnterSearch()
	m.tab = tabSearch
	return nil
}

func (m *Model) preloadSearch() tea.Cmd {
	m.search.Preload()
	m.tab = tabSearch
	return nil
}

func (m *Model) refresh() tea.Cmd {
	switch m.tab {
	case tabCompartments:
		return m.compartments.LoadCmd()
	case tabLogGroups:
		return m.logGroups.LoadCmd()
	case tabLogs:
		return m.logs.LoadCmd()
	}
	return nil
}

func (m *Model) up() tea.Cmd {
	switch m.tab {
	case tabLogGroups:
		m.tab = tabCompartments
	case tabLogs:
		m.tab = tabLogGroups
	case tabSearch:
		m.tab = tabLogs
	}
	return nil
}

func (m *Model) filterEditing() bool {
	switch m.tab {
	case tabCompartments:
		return m.compartments.FilterEditing()
	case tabLogGroups:
		return m.logGroups.FilterEditing()
	case tabLogs:
		return m.logs.FilterEditing()
	}
	return false
}

func (m *Model) View() string {
	if !m.ready {
		return "loading..."
	}

	status := label("compartment", m.session.CompartmentName) +
		"  " + label("log group", m.session.LogGroupName) +
		"  " + label("selected", fmt.Sprintf("%d", m.session.SelectedCount()))

	help := theme.Help.Render("1:compartments  2:log groups  3:logs  4:search  enter:open  space:select  s:search  esc:up  r:refresh  tab:next  q:quit")

	return theme.App.Render(lipgloss.JoinVertical(lipgloss.Left,
		theme.Title.Render(" fw-oci "),
		"",
		status,
		"",
		m.activeView(),
		"",
		help,
	))
}

func (m *Model) activeView() string {
	switch m.tab {
	case tabCompartments:
		return m.compartments.View()
	case tabLogGroups:
		return m.logGroups.View()
	case tabLogs:
		return m.logs.View()
	case tabSearch:
		return m.search.View()
	}
	return ""
}

func label(name, value string) string {
	return lipgloss.NewStyle().Bold(true).Render(name+": ") + value
}
