package listutil

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"n1h41/fw-oci/internal/tui/state"
)

// Item is a list item that carries an OCID and can render a selection marker.
type Item interface {
	list.Item
	OCID() string
	SelectionKey() string
	WithSelected(bool) list.Item
}

// ToggleSelection flips the selection of the item under the cursor, mirroring
// the result into the session. GlobalIndex maps the visible cursor to the item's
// position in the unfiltered Items slice, so SetItem updates in place without
// resetting the cursor.
func ToggleSelection(l *list.Model, s *state.Session) tea.Cmd {
	if l.FilterState() == list.Filtering {
		return nil
	}
	idx := l.GlobalIndex()
	items := l.Items()
	if idx < 0 || idx >= len(items) {
		return nil
	}
	it, ok := items[idx].(Item)
	if !ok {
		return nil
	}
	key := it.SelectionKey()
	sel := !s.IsSelected(key)
	s.SetSelected(key, sel)
	items[idx] = it.WithSelected(sel)
	return l.SetItem(idx, items[idx])
}
