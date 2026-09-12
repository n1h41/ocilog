package search

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"n1h41/fw-oci/internal/oci"
	"n1h41/fw-oci/internal/tui/state"
	"n1h41/fw-oci/internal/tui/theme"
)

// Result is the result of an asynchronous search.
type Result struct {
	Content string
	Err     error
}

// Model renders the search screen: a query textarea plus results viewport.
type Model struct {
	session   *state.Session
	query     textarea.Model
	results   viewport.Model
	searching bool
	searched  bool
	err       error
}

func New(session *state.Session) Model {
	q := textarea.New()
	q.Placeholder = "Search query (OCI Logging Query Language)..."
	q.SetHeight(3)
	q.ShowLineNumbers = false
	q.Focus()
	return Model{
		session: session,
		query:   q,
		results: viewport.New(0, 0),
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m *Model) Resize(width, contentHeight int) {
	m.query.SetWidth(width - 4)
	m.results.Width = width - 4
	m.results.Height = contentHeight - 5
}

// Preload overwrites the query with the current selection scope.
func (m *Model) Preload() {
	m.query.SetValue(m.session.BuildScopeQuery())
	m.query.CursorEnd()
}

// EnterSearch preloads the scope only when the query is empty.
func (m *Model) EnterSearch() {
	if strings.TrimSpace(m.query.Value()) == "" {
		m.Preload()
	}
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case Result:
		m.searching = false
		m.err = msg.Err
		if msg.Err == nil {
			m.searched = true
			m.results.SetContent(msg.Content)
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return state.GoUp{} }
		case "enter":
			q := strings.TrimSpace(m.query.Value())
			if q == "" {
				return m, func() tea.Msg { return Result{Err: fmt.Errorf("query is empty")} }
			}
			m.searching = true
			m.err = nil
			m.searched = false
			return m, m.searchCmd(q)
		}
	}

	var qc, rc tea.Cmd
	m.query, qc = m.query.Update(msg)
	m.results, rc = m.results.Update(msg)
	return m, tea.Batch(qc, rc)
}

func (m Model) searchCmd(q string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		results, err := m.session.Client.SearchLogs(ctx, oci.SearchQuery{
			Query: q,
			Start: time.Now().Add(-24 * time.Hour),
			End:   time.Now(),
			Limit: 100,
		})
		if err != nil {
			return Result{Err: err}
		}

		var b strings.Builder
		for i, r := range results {
			fmt.Fprintf(&b, "%d. %s\n\n", i+1, r)
		}
		return Result{Content: b.String()}
	}
}

func (m Model) View() string {
	body := ""
	switch {
	case m.searching:
		body = "searching..."
	case m.err != nil:
		body = theme.Error.Render("error: " + m.err.Error())
	case !m.searched:
		body = theme.Help.Render("enter a query and press enter")
	default:
		if r := m.results.View(); r == "" {
			body = theme.Help.Render("no results")
		} else {
			body = r
		}
	}
	return lipgloss.JoinVertical(lipgloss.Left, m.query.View(), "", body)
}
