package compartments

import (
	"context"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"n1h41/ocilog/internal/oci"
	"n1h41/ocilog/internal/tui/listutil"
	"n1h41/ocilog/internal/tui/state"
	"n1h41/ocilog/internal/tui/theme"
)

type item struct {
	id       string
	name     string
	state    string
	selected bool
}

func (i item) Title() string {
	if i.selected {
		return "[x] " + i.name
	}
	return "[ ] " + i.name
}

func (i item) Description() string { return i.state + "  " + i.id }
func (i item) FilterValue() string { return i.name + " " + i.id }
func (i item) OCID() string        { return i.id }

func (i item) SelectionKey() string { return i.id }

func (i item) WithSelected(selected bool) list.Item {
	i.selected = selected
	return i
}

// Loaded is the result of an asynchronous compartment load.
type Loaded struct {
	Compartments []oci.Compartment
	Err          error
}

// Model renders the list of compartments.
type Model struct {
	session *state.Session
	list    list.Model
	err     error
}

func New(session *state.Session) Model {
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Compartments"
	l.SetFilteringEnabled(true)
	return Model{session: session, list: l}
}

func (m Model) Init() tea.Cmd { return m.LoadCmd() }

// LoadCmd reloads the compartment list for the tenancy.
func (m Model) LoadCmd() tea.Cmd {
	id := m.session.TenancyID
	return func() tea.Msg { return m.load(id) }
}

func (m Model) load(id string) tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	comps, err := m.session.Client.ListCompartments(ctx, id)
	return Loaded{Compartments: comps, Err: err}
}

func (m *Model) Resize(width, contentHeight int) {
	m.list.SetSize(width-4, contentHeight)
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case Loaded:
		m.err = msg.Err
		if msg.Err == nil {
			items := make([]list.Item, 0, len(msg.Compartments))
			for _, c := range msg.Compartments {
				items = append(items, item{
					id:       c.ID,
					name:     c.Name,
					state:    c.State,
					selected: m.session.IsSelected(c.ID),
				})
			}
			m.list.SetItems(items)
		}
		return m, nil

	case tea.KeyMsg:
		if m.list.FilterState() == list.Filtering {
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)
			return m, cmd
		}
		switch msg.String() {
		case " ":
			return m, listutil.ToggleSelection(&m.list, m.session)
		case "enter":
			if sel, ok := m.list.SelectedItem().(item); ok {
				return m, func() tea.Msg { return state.OpenCompartment{ID: sel.id, Name: sel.name} }
			}
			return m, nil
		case "esc":
			if m.list.FilterState() == list.FilterApplied {
				var cmd tea.Cmd
				m.list, cmd = m.list.Update(msg)
				return m, cmd
			}
			return m, func() tea.Msg { return state.GoUp{} }
		case "backspace":
			return m, func() tea.Msg { return state.GoUp{} }
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	body := m.list.View()
	if m.err != nil {
		return lipgloss.JoinVertical(lipgloss.Left,
			theme.Error.Render(m.err.Error()),
			"",
			body,
		)
	}
	return body
}

// FilterEditing reports whether the list filter input is active.
func (m Model) FilterEditing() bool {
	return m.list.FilterState() == list.Filtering
}
