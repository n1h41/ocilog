package search

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"n1h41/fw-oci/internal/oci"
	"n1h41/fw-oci/internal/tui/state"
	"n1h41/fw-oci/internal/tui/theme"
)

// Focusable fields, cycled with tab/shift+tab.
const (
	focusQuery = iota
	focusFrom
	focusTo
)

// maxSearchSpan is the OCI Logging limit on the searchable time range.
const maxSearchSpan = 180 * 24 * time.Hour

// dateLayout is used to render the default from/to values.
const dateLayout = "2006-01-02 15:04"

// dateLayouts are accepted when parsing the from/to fields, most specific first.
var dateLayouts = []string{"2006-01-02 15:04:05", "2006-01-02 15:04", "2006-01-02"}

// Result is the result of an asynchronous search.
type Result struct {
	Content string
	Err     error
}

// Model renders the search screen: a query textarea, from/to date fields, and
// a results viewport.
type Model struct {
	session   *state.Session
	query     textarea.Model
	from      textinput.Model
	to        textinput.Model
	focus     int
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

	now := time.Now()
	from := textinput.New()
	from.Placeholder = "YYYY-MM-DD HH:MM"
	from.SetValue(now.Add(-24 * time.Hour).Format(dateLayout))
	from.Width = 20

	to := textinput.New()
	to.Placeholder = "YYYY-MM-DD HH:MM"
	to.SetValue(now.Format(dateLayout))
	to.Width = 20

	results := viewport.New(0, 0)
	// Restrict the viewport to non-typing keys so editing the query or dates
	// never scrolls the results.
	results.KeyMap = viewport.KeyMap{
		PageUp:       key.NewBinding(key.WithKeys("pgup")),
		PageDown:     key.NewBinding(key.WithKeys("pgdown")),
		HalfPageUp:   key.NewBinding(key.WithKeys("ctrl+u")),
		HalfPageDown: key.NewBinding(key.WithKeys("ctrl+d")),
		Up:           key.NewBinding(key.WithKeys("up")),
		Down:         key.NewBinding(key.WithKeys("down")),
		Left:         key.NewBinding(key.WithKeys("left")),
		Right:        key.NewBinding(key.WithKeys("right")),
	}

	return Model{
		session: session,
		query:   q,
		from:    from,
		to:      to,
		results: results,
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m *Model) Resize(width, contentHeight int) {
	m.query.SetWidth(width - 4)
	m.results.Width = width - 4
	m.results.Height = contentHeight - 7
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
		case "tab":
			m.focus = (m.focus + 1) % 3
			m.syncFocus()
			return m, nil
		case "shift+tab":
			m.focus = (m.focus + 2) % 3
			m.syncFocus()
			return m, nil
		case "esc":
			return m, func() tea.Msg { return state.GoUp{} }
		case "enter":
			return m.submit()
		case "pgup":
			m.results.PageUp()
			return m, nil
		case "pgdown":
			m.results.PageDown()
			return m, nil
		case "home":
			m.results.GotoTop()
			return m, nil
		case "end":
			m.results.GotoBottom()
			return m, nil
		}
	}

	var fc, rc tea.Cmd
	switch m.focus {
	case focusFrom:
		m.from, fc = m.from.Update(msg)
	case focusTo:
		m.to, fc = m.to.Update(msg)
	default:
		m.query, fc = m.query.Update(msg)
	}
	m.results, rc = m.results.Update(msg)
	return m, tea.Batch(fc, rc)
}

// syncFocus moves input focus to the currently selected field.
func (m *Model) syncFocus() {
	if m.focus == focusQuery {
		m.query.Focus()
	} else {
		m.query.Blur()
	}
	if m.focus == focusFrom {
		m.from.Focus()
	} else {
		m.from.Blur()
	}
	if m.focus == focusTo {
		m.to.Focus()
	} else {
		m.to.Blur()
	}
}

// submit validates the query and date range, then launches the search.
func (m Model) submit() (Model, tea.Cmd) {
	q := strings.TrimSpace(m.query.Value())
	if q == "" {
		m.err = fmt.Errorf("query is empty")
		return m, nil
	}
	from, err := parseDate(m.from.Value(), false)
	if err != nil {
		m.err = err
		return m, nil
	}
	to, err := parseDate(m.to.Value(), true)
	if err != nil {
		m.err = err
		return m, nil
	}
	if to.Before(from) {
		m.err = fmt.Errorf("'to' must be after 'from'")
		return m, nil
	}
	if to.Sub(from) > maxSearchSpan {
		m.err = fmt.Errorf("date range exceeds the OCI limit of 180 days")
		return m, nil
	}

	m.searching = true
	m.err = nil
	m.searched = false
	return m, m.searchCmd(q, from, to)
}

// parseDate accepts the layouts in dateLayouts. When endOfDay is true, a
// date-only value is expanded to the last instant of that day.
func parseDate(s string, endOfDay bool) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("date is empty")
	}
	for _, layout := range dateLayouts {
		t, err := time.ParseInLocation(layout, s, time.Local)
		if err != nil {
			continue
		}
		if endOfDay && layout == "2006-01-02" {
			t = t.Add(24*time.Hour - time.Nanosecond)
		}
		return t, nil
	}
	return time.Time{}, fmt.Errorf("invalid date %q, use YYYY-MM-DD or YYYY-MM-DD HH:MM", s)
}

func (m Model) searchCmd(q string, from, to time.Time) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		results, err := m.session.Client.SearchLogs(ctx, oci.SearchQuery{
			Query: q,
			Start: from,
			End:   to,
			Limit: 100,
		})
		if err != nil {
			return Result{Err: err}
		}

		content, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return Result{Err: fmt.Errorf("format results: %w", err)}
		}
		return Result{Content: string(content)}
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

	dates := lipgloss.JoinHorizontal(lipgloss.Left,
		theme.Help.Render("from: "), m.from.View(),
		"   ",
		theme.Help.Render("to: "), m.to.View(),
	)
	return lipgloss.JoinVertical(lipgloss.Left, m.query.View(), dates, "", body)
}
