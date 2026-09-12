package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"n1h41/fw-oci/internal/oci"
)

// tab represents the active screen of the TUI.
type tab int

const (
	tabCompartments tab = iota
	tabLogGroups
	tabLogs
	tabSearch
)

// Model is the root Bubble Tea model.
type Model struct {
	client    *oci.Client
	tenancyID string

	tab    tab
	ready  bool
	width  int
	height int
	err    error

	compartments list.Model
	logGroups    list.Model
	logs         list.Model

	query     textarea.Model
	results   viewport.Model
	searching bool
	searched  bool
	searchErr error

	selectedCompartmentID   string
	selectedCompartmentName string
	selectedLogGroupID      string
	selectedLogGroupName    string

	// selected tracks the multi-selected OCIDs across all three lists. It is
	// the source of truth for the preloaded search scope; list items mirror
	// it so the delegate can render a selection marker.
	selected map[string]bool
}

// New creates the root model bound to an OCI logging client. tenancyID is the
// root compartment; initialCompartment pre-selects a compartment when set.
func New(client *oci.Client, tenancyID, initialCompartment string) *Model {
	cs := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	cs.Title = "Compartments"
	cs.SetFilteringEnabled(true)

	lg := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	lg.Title = "Log Groups"
	lg.SetFilteringEnabled(true)

	ls := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	ls.Title = "Logs"
	ls.SetFilteringEnabled(true)

	q := textarea.New()
	q.Placeholder = "Search query (OCI Logging Query Language)..."
	q.SetHeight(3)
	q.ShowLineNumbers = false
	q.Focus()

	r := viewport.New(0, 0)

	compartment := initialCompartment
	if compartment == "" {
		compartment = tenancyID
	}

	return &Model{
		client:                  client,
		tenancyID:               tenancyID,
		compartments:            cs,
		logGroups:               lg,
		logs:                    ls,
		query:                   q,
		results:                 r,
		selectedCompartmentID:   compartment,
		selectedCompartmentName: compartment,
		selected:                map[string]bool{},
	}
}

func (m *Model) Init() tea.Cmd {
	return m.loadCompartments
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.ready = true
		m.width = msg.Width
		m.height = msg.Height
		m.resize()

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}

		// While editing a list filter, hand the key to the focused list so its
		// filter input handles backspace/enter/tab/esc rather than navigating.
		if m.filterEditing() {
			break
		}

		// On the search tab only navigation keys are intercepted; everything
		// else (letters, digits, space, ...) reaches the query textarea.
		if m.tab == tabSearch {
			switch msg.String() {
			case "esc":
				return m, m.up()
			case "enter":
				return m, m.down()
			}
			break
		}

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

		case "enter":
			return m, m.down()

		case " ":
			return m, m.toggleSelection()

		case "s":
			return m, m.preloadSearch()

		case "backspace":
			return m, m.up()

		case "esc":
			// Let a list clear an applied filter instead of navigating up.
			if l := m.activeList(); l != nil && l.FilterState() == list.FilterApplied {
				break
			}
			return m, m.up()
		}

	case compartmentsMsg:
		m.err = msg.err
		if msg.err == nil {
			items := make([]list.Item, 0, len(msg.compartments))
			for _, c := range msg.compartments {
				items = append(items, compartmentItem{id: c.ID, name: c.Name, state: c.State, selected: m.selected[c.ID]})
			}
			m.compartments.SetItems(items)
		}

	case logGroupsMsg:
		m.err = msg.err
		if msg.err == nil {
			items := make([]list.Item, 0, len(msg.groups))
			for _, g := range msg.groups {
				items = append(items, logGroupItem{id: g.ID, name: g.Name, selected: m.selected[g.ID]})
			}
			m.logGroups.SetItems(items)
		}

	case logsMsg:
		m.err = msg.err
		if msg.err == nil {
			items := make([]list.Item, 0, len(msg.logs))
			for _, l := range msg.logs {
				items = append(items, logItem{id: l.ID, name: l.Name, logType: l.LogType, enabled: l.Enabled, selected: m.selected[l.ID]})
			}
			m.logs.SetItems(items)
		}

	case searchMsg:
		m.searching = false
		m.searchErr = msg.err
		if msg.err == nil {
			m.searched = true
			m.results.SetContent(msg.content)
		}
	}

	switch m.tab {
	case tabCompartments:
		var cmd tea.Cmd
		m.compartments, cmd = m.compartments.Update(msg)
		cmds = append(cmds, cmd)
	case tabLogGroups:
		var cmd tea.Cmd
		m.logGroups, cmd = m.logGroups.Update(msg)
		cmds = append(cmds, cmd)
	case tabLogs:
		var cmd tea.Cmd
		m.logs, cmd = m.logs.Update(msg)
		cmds = append(cmds, cmd)
	case tabSearch:
		var qc, rc tea.Cmd
		m.query, qc = m.query.Update(msg)
		m.results, rc = m.results.Update(msg)
		cmds = append(cmds, qc, rc)
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) resize() {
	contentHeight := m.height - 7
	m.compartments.SetSize(m.width-4, contentHeight)
	m.logGroups.SetSize(m.width-4, contentHeight)
	m.logs.SetSize(m.width-4, contentHeight)
	m.query.SetWidth(m.width - 4)
	m.results.Width = m.width - 4
	m.results.Height = contentHeight - 5
}

func (m *Model) View() string {
	if !m.ready {
		return "loading..."
	}

	var body string
	switch m.tab {
	case tabCompartments:
		body = m.compartments.View()
	case tabLogGroups:
		body = m.logGroups.View()
	case tabLogs:
		body = m.logs.View()
	case tabSearch:
		var searchBody string
		switch {
		case m.searching:
			searchBody = "searching..."
		case m.searchErr != nil:
			searchBody = errorStyle.Render("error: " + m.searchErr.Error())
		case !m.searched:
			searchBody = helpStyle.Render("enter a query and press enter")
		case m.results.View() == "":
			searchBody = helpStyle.Render("no results")
		default:
			searchBody = m.results.View()
		}
		body = lipgloss.JoinVertical(lipgloss.Left,
			m.query.View(),
			"",
			searchBody,
		)
	}

	if m.err != nil {
		body = lipgloss.JoinVertical(lipgloss.Left,
			errorStyle.Render(m.err.Error()),
			"",
			body,
		)
	}

	status := lipgloss.JoinHorizontal(lipgloss.Top,
		label("compartment", m.selectedCompartmentName),
		"  ",
		label("log group", m.selectedLogGroupName),
		"  ",
		label("selected", fmt.Sprintf("%d", m.selectedCount())),
	)

	help := helpStyle.Render("1:compartments  2:log groups  3:logs  4:search  enter:open  space:select  s:search  esc:up  r:refresh  tab:next  q:quit")

	return appStyle.Render(lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render(" fw-oci "),
		"",
		status,
		"",
		body,
		"",
		help,
	))
}

func label(name, value string) string {
	return lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Bold(true).Render(name+":"),
		" ",
		value,
	)
}

// down advances into the selection (compartment → log groups → logs, or runs
// the search on the search tab).
func (m *Model) down() tea.Cmd {
	switch m.tab {
	case tabCompartments:
		if sel, ok := m.compartments.SelectedItem().(compartmentItem); ok {
			m.selectedCompartmentID = sel.id
			m.selectedCompartmentName = sel.name
			m.tab = tabLogGroups
			return m.loadLogGroups
		}
	case tabLogGroups:
		if sel, ok := m.logGroups.SelectedItem().(logGroupItem); ok {
			m.selectedLogGroupID = sel.id
			m.selectedLogGroupName = sel.name
			m.tab = tabLogs
			return m.loadLogs
		}
	case tabSearch:
		return m.runSearch
	}
	return nil
}

// up moves back one level.
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

func (m *Model) refresh() tea.Cmd {
	switch m.tab {
	case tabCompartments:
		return m.loadCompartments
	case tabLogGroups:
		return m.loadLogGroups
	case tabLogs:
		return m.loadLogs
	}
	return nil
}

// activeList returns the list shown on the current tab, or nil on the search tab.
func (m *Model) activeList() *list.Model {
	switch m.tab {
	case tabCompartments:
		return &m.compartments
	case tabLogGroups:
		return &m.logGroups
	case tabLogs:
		return &m.logs
	}
	return nil
}

// filterEditing reports whether the active list is currently editing its filter.
func (m *Model) filterEditing() bool {
	l := m.activeList()
	return l != nil && l.FilterState() == list.Filtering
}

// toggleSelection flips the selection of the item under the cursor in the
// active list. The list's GlobalIndex maps the visible cursor to the item's
// position in the unfiltered Items slice, so SetItem updates in place without
// resetting the cursor.
func (m *Model) toggleSelection() tea.Cmd {
	l := m.activeList()
	if l == nil || l.FilterState() == list.Filtering {
		return nil
	}

	idx := l.GlobalIndex()
	items := l.Items()
	if idx < 0 || idx >= len(items) {
		return nil
	}

	var id string
	var selected bool
	switch it := items[idx].(type) {
	case compartmentItem:
		it.selected = !it.selected
		items[idx] = it
		id, selected = it.id, it.selected
	case logGroupItem:
		it.selected = !it.selected
		items[idx] = it
		id, selected = it.id, it.selected
	case logItem:
		it.selected = !it.selected
		items[idx] = it
		id, selected = it.id, it.selected
	default:
		return nil
	}

	m.selected[id] = selected
	return l.SetItem(idx, items[idx])
}

// enterSearch switches to the search tab, preloading the scope from the current
// selection when the query is empty (so it never clobbers a query in progress).
func (m *Model) enterSearch() tea.Cmd {
	m.tab = tabSearch
	if strings.TrimSpace(m.query.Value()) == "" {
		m.query.SetValue(m.buildScopeQuery())
		m.query.CursorEnd()
	}
	return nil
}

// preloadSearch rebuilds the scope from the current selection, overwriting any
// existing query, and jumps to the search tab with the scope pre-filled.
func (m *Model) preloadSearch() tea.Cmd {
	m.tab = tabSearch
	m.query.SetValue(m.buildScopeQuery())
	m.query.CursorEnd()
	return nil
}

// buildScopeQuery assembles a `search "ocid1", "ocid2" | ` scope from the
// selected items, falling back to the drilled-into log group or compartment.
func (m *Model) buildScopeQuery() string {
	ocids := make([]string, 0)
	seen := make(map[string]bool)

	for _, l := range []list.Model{m.compartments, m.logGroups, m.logs} {
		for _, it := range l.Items() {
			var id string
			var selected bool
			switch v := it.(type) {
			case compartmentItem:
				id, selected = v.id, v.selected
			case logGroupItem:
				id, selected = v.id, v.selected
			case logItem:
				id, selected = v.id, v.selected
			}
			if selected && id != "" && !seen[id] {
				seen[id] = true
				ocids = append(ocids, id)
			}
		}
	}

	if len(ocids) == 0 {
		switch {
		case m.selectedLogGroupID != "":
			ocids = append(ocids, m.selectedLogGroupID)
		case m.selectedCompartmentID != "":
			ocids = append(ocids, m.selectedCompartmentID)
		}
	}
	if len(ocids) == 0 {
		return ""
	}

	quoted := make([]string, len(ocids))
	for i, id := range ocids {
		quoted[i] = fmt.Sprintf("%q", id)
	}
	return "search " + strings.Join(quoted, ", ") + " | "
}

// selectedCount returns the number of currently selected items.
func (m *Model) selectedCount() int {
	n := 0
	for _, sel := range m.selected {
		if sel {
			n++
		}
	}
	return n
}

// --- async message types ---

type compartmentsMsg struct {
	compartments []oci.Compartment
	err          error
}

type logGroupsMsg struct {
	groups []oci.LogGroup
	err    error
}

type logsMsg struct {
	logs []oci.Log
	err  error
}

type searchMsg struct {
	content string
	err     error
}

func (m *Model) loadCompartments() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	comps, err := m.client.ListCompartments(ctx, m.tenancyID)
	return compartmentsMsg{compartments: comps, err: err}
}

func (m *Model) loadLogGroups() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	groups, err := m.client.ListLogGroups(ctx, m.selectedCompartmentID)
	return logGroupsMsg{groups: groups, err: err}
}

func (m *Model) loadLogs() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logs, err := m.client.ListLogs(ctx, m.selectedLogGroupID)
	return logsMsg{logs: logs, err: err}
}

func (m *Model) runSearch() tea.Msg {
	q := strings.TrimSpace(m.query.Value())
	if q == "" {
		return searchMsg{err: fmt.Errorf("query is empty")}
	}

	m.searching = true
	m.searchErr = nil

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	results, err := m.client.SearchLogs(ctx, oci.SearchQuery{
		Query: q,
		Start: time.Now().Add(-24 * time.Hour),
		End:   time.Now(),
		Limit: 100,
	})
	if err != nil {
		return searchMsg{err: err}
	}

	var content strings.Builder
	for i, r := range results {
		fmt.Fprintf(&content, "%d. %s\n\n", i+1, r)
	}
	return searchMsg{content: content.String()}
}

// --- list item wrappers ---

type compartmentItem struct {
	id       string
	name     string
	state    string
	selected bool
}

func (i compartmentItem) Title() string {
	if i.selected {
		return "[x] " + i.name
	}
	return "[ ] " + i.name
}
func (i compartmentItem) Description() string { return i.state + "  " + i.id }
func (i compartmentItem) FilterValue() string { return i.name + " " + i.id }

type logGroupItem struct {
	id       string
	name     string
	selected bool
}

func (i logGroupItem) Title() string {
	if i.selected {
		return "[x] " + i.name
	}
	return "[ ] " + i.name
}
func (i logGroupItem) Description() string { return i.id }
func (i logGroupItem) FilterValue() string { return i.name + " " + i.id }

type logItem struct {
	id       string
	name     string
	logType  string
	enabled  bool
	selected bool
}

func (i logItem) Title() string {
	if i.selected {
		return "[x] " + i.name
	}
	return "[ ] " + i.name
}
func (i logItem) Description() string {
	return fmt.Sprintf("%s  enabled=%v  %s", i.logType, i.enabled, i.id)
}
func (i logItem) FilterValue() string { return i.name + " " + i.logType }
