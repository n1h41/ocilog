package search

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"n1h41/fw-oci/internal/history"
	"n1h41/fw-oci/internal/oci"
	"n1h41/fw-oci/internal/tui/state"
	"n1h41/fw-oci/internal/tui/theme"
)

// Focusable fields, cycled with tab/shift+tab.
const (
	focusQuery = iota
	focusPipe
	focusFrom
	focusTo
)

// Result panes, switchable with ctrl+left / ctrl+right.
const (
	paneFormatted = iota
	paneOriginal
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

// copiedMsg reports the outcome of copying to the clipboard.
type copiedMsg struct {
	label string
	err   error
}

// Model renders the search screen: a query textarea, a jq/sed pipeline field,
// from/to date fields, and two side-by-side result panes (formatted output on
// the left, the original JSON on the right).
type Model struct {
	session *state.Session
	query   textarea.Model
	pipe    textarea.Model
	from    textinput.Model
	to      textinput.Model
	focus   int

	results   viewport.Model
	formatted viewport.Model
	pane      int

	content          string
	formattedContent string
	status           string
	searching        bool
	searched         bool
	err              error

	width        int
	history      *history.Store
	showHistory  bool
	historyIndex int
	confirmClear bool
}

func New(session *state.Session) Model {
	q := textarea.New()
	q.Placeholder = "Search query (OCI Logging Query Language)..."
	q.SetHeight(3)
	q.ShowLineNumbers = false
	q.Focus()

	p := textarea.New()
	p.Placeholder = "jq '.[] | .message' | sed 's/foo/bar/g'"
	p.SetHeight(2)
	p.ShowLineNumbers = false

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
	results.KeyMap = resultKeyMap()
	formatted := viewport.New(0, 0)
	formatted.KeyMap = resultKeyMap()

	m := Model{
		session:   session,
		query:     q,
		pipe:      p,
		from:      from,
		to:        to,
		results:   results,
		formatted: formatted,
		history:   history.Load(),
	}
	m.setFormatted("")
	return m
}

// resultKeyMap restricts a viewport to non-typing keys so editing the query,
// pipeline, or dates never scrolls the results.
func resultKeyMap() viewport.KeyMap {
	return viewport.KeyMap{
		PageUp:       key.NewBinding(key.WithKeys("pgup")),
		PageDown:     key.NewBinding(key.WithKeys("pgdown")),
		HalfPageUp:   key.NewBinding(key.WithKeys("ctrl+u")),
		HalfPageDown: key.NewBinding(key.WithKeys("ctrl+d")),
		Up:           key.NewBinding(key.WithKeys("up")),
		Down:         key.NewBinding(key.WithKeys("down")),
		Left:         key.NewBinding(key.WithKeys("left")),
		Right:        key.NewBinding(key.WithKeys("right")),
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m *Model) Resize(width, contentHeight int) {
	m.width = width
	// Reserve the fixed rows above the panes: query (3), blank (1), pipeline
	// label+field (3), blank (1), dates (1), status (1), and the pane header
	// (1) = 11 rows. The panes then get whatever vertical space remains.
	h := contentHeight - 11
	if h < 3 {
		h = 3
	}
	m.results.Height = h
	m.formatted.Height = h
	m.applyWidths()
}

// applyWidths sizes the query, pipeline, and the two result panes to the space
// left of the history sidebar when it is shown.
func (m *Model) applyWidths() {
	w := m.width - 4
	if m.showHistory {
		w -= m.sidebarWidth() + 2
	}
	if w < 20 {
		w = 20
	}
	m.query.SetWidth(w)
	m.pipe.SetWidth(w)

	// Split the remaining width into two panes separated by a small gap. Each
	// pane spends 2 columns on padding and 2 on its border.
	const gap = 2
	inner := (w-gap)/2 - 4
	if inner < 10 {
		inner = 10
	}
	m.results.Width = inner
	m.formatted.Width = inner
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
			m.content = msg.Content
			m.results.SetContent(highlightJSON(msg.Content))
			if p := strings.TrimSpace(m.pipe.Value()); p != "" {
				return m, m.pipeCmd(p, m.content)
			}
			m.setFormatted("")
		}
		return m, nil

	case pipedMsg:
		if msg.err != nil {
			m.status = "pipeline failed"
			m.formattedContent = ""
			m.formatted.SetContent(theme.Error.Render(msg.err.Error()))
		} else {
			m.status = "pipeline applied"
			m.setFormatted(msg.content)
		}
		return m, nil

	case copiedMsg:
		if msg.err != nil {
			m.status = "copy failed: " + msg.err.Error()
		} else {
			m.status = msg.label + " copied to clipboard"
		}
		return m, nil

	case tea.KeyMsg:
		m.status = ""
		if m.showHistory {
			return m.updateHistory(msg)
		}
		switch msg.String() {
		case "ctrl+r":
			m.showHistory = !m.showHistory
			m.historyIndex = 0
			m.applyWidths()
			return m, nil
		case "ctrl+y":
			if m.content == "" {
				m.status = "nothing to copy"
				return m, nil
			}
			return m, copyCmd("results", m.content)
		case "ctrl+o":
			q := strings.TrimSpace(m.query.Value())
			if q == "" {
				m.status = "nothing to copy"
				return m, nil
			}
			return m, copyCmd("query", q)
		case "tab":
			m.focus = (m.focus + 1) % 4
			m.syncFocus()
			return m, nil
		case "shift+tab":
			m.focus = (m.focus + 3) % 4
			m.syncFocus()
			return m, nil
		case "ctrl+left":
			m.pane = paneFormatted
			return m, nil
		case "ctrl+right":
			m.pane = paneOriginal
			return m, nil
		case "ctrl+t":
			m.pane = 1 - m.pane
			return m, nil
		case "esc":
			return m, func() tea.Msg { return state.GoUp{} }
		case "enter":
			if m.focus == focusPipe {
				return m.applyPipeline()
			}
			return m.submit()
		case "pgup":
			m.activeViewport().PageUp()
			return m, nil
		case "pgdown":
			m.activeViewport().PageDown()
			return m, nil
		case "home":
			m.activeViewport().GotoTop()
			return m, nil
		case "end":
			m.activeViewport().GotoBottom()
			return m, nil
		}
	}

	var fc, rc tea.Cmd
	switch m.focus {
	case focusPipe:
		m.pipe, fc = m.pipe.Update(msg)
	case focusFrom:
		m.from, fc = m.from.Update(msg)
	case focusTo:
		m.to, fc = m.to.Update(msg)
	default:
		m.query, fc = m.query.Update(msg)
	}
	if m.pane == paneFormatted {
		m.formatted, rc = m.formatted.Update(msg)
	} else {
		m.results, rc = m.results.Update(msg)
	}
	return m, tea.Batch(fc, rc)
}

// activeViewport returns the viewport of the pane that scroll keys target.
func (m *Model) activeViewport() *viewport.Model {
	if m.pane == paneFormatted {
		return &m.formatted
	}
	return &m.results
}

// syncFocus moves input focus to the currently selected field.
func (m *Model) syncFocus() {
	if m.focus == focusQuery {
		m.query.Focus()
	} else {
		m.query.Blur()
	}
	if m.focus == focusPipe {
		m.pipe.Focus()
	} else {
		m.pipe.Blur()
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
	if err := m.history.Add(history.Entry{
		Query: q,
		From:  strings.TrimSpace(m.from.Value()),
		To:    strings.TrimSpace(m.to.Value()),
	}); err != nil {
		m.status = "history save failed: " + err.Error()
	}
	return m, m.searchCmd(q, from, to)
}

// updateHistory handles keys while the history popup is open.
func (m Model) updateHistory(msg tea.KeyMsg) (Model, tea.Cmd) {
	entries := m.history.Entries()
	key := msg.String()
	if key != "ctrl+x" {
		m.confirmClear = false
	}
	switch key {
	case "ctrl+x":
		if len(entries) == 0 {
			break
		}
		if !m.confirmClear {
			m.confirmClear = true
			break
		}
		m.confirmClear = false
		if err := m.history.Clear(); err != nil {
			m.status = "history clear failed: " + err.Error()
		} else {
			m.historyIndex = 0
			m.status = "history cleared"
		}
	case "esc", "ctrl+r":
		m.showHistory = false
		m.applyWidths()
	case "up", "ctrl+p":
		if m.historyIndex > 0 {
			m.historyIndex--
		}
	case "down", "ctrl+n":
		if m.historyIndex < len(entries)-1 {
			m.historyIndex++
		}
	case "home":
		m.historyIndex = 0
	case "end":
		if len(entries) > 0 {
			m.historyIndex = len(entries) - 1
		}
	case "enter":
		if len(entries) > 0 {
			e := entries[m.historyIndex]
			m.query.SetValue(e.Query)
			m.query.CursorEnd()
			if e.From != "" {
				m.from.SetValue(e.From)
			}
			if e.To != "" {
				m.to.SetValue(e.To)
			}
			m.focus = focusQuery
			m.syncFocus()
			m.status = "loaded query from history"
		}
		m.showHistory = false
		m.applyWidths()
	}
	return m, nil
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

// copyCmd writes content to the system clipboard without blocking the UI.
func copyCmd(label, content string) tea.Cmd {
	return func() tea.Msg {
		return copiedMsg{label: label, err: clipboard.WriteAll(content)}
	}
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

// applyPipeline runs the typed shell pipeline against the current JSON result.
func (m Model) applyPipeline() (Model, tea.Cmd) {
	pipeline := strings.TrimSpace(m.pipe.Value())
	if pipeline == "" {
		m.status = "pipeline is empty"
		m.setFormatted("")
		return m, nil
	}
	if m.content == "" {
		m.status = "run a search first"
		return m, nil
	}
	m.status = "applying pipeline..."
	return m, m.pipeCmd(pipeline, m.content)
}

// setFormatted stores raw pipeline output and renders it into the left pane,
// highlighting it only when it is valid JSON.
func (m *Model) setFormatted(raw string) {
	m.formattedContent = raw
	switch {
	case strings.TrimSpace(raw) == "":
		m.formatted.SetContent(theme.Help.Render("enter a pipeline above and press enter to format"))
	case json.Valid([]byte(raw)):
		m.formatted.SetContent(highlightJSON(raw))
	default:
		m.formatted.SetContent(raw)
	}
}

// panesView renders the formatted output and the original JSON side by side.
func (m Model) panesView() string {
	left := m.paneBox("formatted", m.formatted.View(), m.pane == paneFormatted)
	right := m.paneBox("original", m.results.View(), m.pane == paneOriginal)
	return lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", right)
}

// paneBox frames a viewport with a title, highlighting the active pane.
func (m Model) paneBox(title, view string, active bool) string {
	style := theme.Pane
	if active {
		style = theme.PaneActive
	}
	header := theme.SidebarTitle.Render(title)
	return style.Width(m.results.Width + 2).Height(m.results.Height + 1).Render(header + "\n" + view)
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
		body = m.panesView()
	}

	pipe := lipgloss.JoinVertical(lipgloss.Left,
		theme.Help.Render("pipeline (jq/sed, enter to apply):"),
		m.pipe.View(),
	)

	dates := lipgloss.JoinHorizontal(lipgloss.Left,
		theme.Help.Render("from: "), m.from.View(),
		"   ",
		theme.Help.Render("to: "), m.to.View(),
	)

	status := ""
	if m.status != "" {
		status = theme.Help.Render(m.status)
	}
	if m.showHistory {
		body = padLines(body, m.results.Height+1)
	}
	main := lipgloss.JoinVertical(lipgloss.Left, m.query.View(), "", pipe, "", dates, status, body)

	if !m.showHistory {
		return main
	}
	sidebar := m.historySidebar(lipgloss.Height(main))
	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, "  ", main)
}
