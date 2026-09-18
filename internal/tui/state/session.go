package state

import (
	"fmt"
	"strings"

	"n1h41/fw-oci/internal/oci"
)

// Session carries the cross-view state shared by all views: the OCI client and
// the current navigation/selection context.
type Session struct {
	Client    *oci.Client
	TenancyID string

	CompartmentID   string
	CompartmentName string
	LogGroupID      string
	LogGroupName    string

	selected      map[string]bool
	selectedOrder []string
}

// New creates a session. When initialCompartment is empty the tenancy itself is
// used as the initial compartment.
func New(client *oci.Client, tenancyID, initialCompartment string) *Session {
	comp := initialCompartment
	if comp == "" {
		comp = tenancyID
	}
	return &Session{
		Client:          client,
		TenancyID:       tenancyID,
		CompartmentID:   comp,
		CompartmentName: comp,
		selected:        map[string]bool{},
	}
}

// IsSelected reports whether the given selection key is currently selected.
func (s *Session) IsSelected(ocid string) bool { return s.selected[ocid] }

// SetSelected marks (or clears) a selection key as selected, preserving order.
func (s *Session) SetSelected(ocid string, selected bool) {
	if selected {
		if !s.selected[ocid] {
			s.selected[ocid] = true
			s.selectedOrder = append(s.selectedOrder, ocid)
		}
		return
	}
	if !s.selected[ocid] {
		return
	}
	delete(s.selected, ocid)
	for i, o := range s.selectedOrder {
		if o == ocid {
			s.selectedOrder = append(s.selectedOrder[:i], s.selectedOrder[i+1:]...)
			break
		}
	}
}

// SelectedCount returns the number of selected items.
func (s *Session) SelectedCount() int { return len(s.selectedOrder) }

// ScopeOCIDs returns the selection keys for the search scope: all selected
// items, or a fallback to the drilled-into log group, then compartment, when
// none selected.
func (s *Session) ScopeOCIDs() []string {
	if len(s.selectedOrder) > 0 {
		return append([]string{}, s.selectedOrder...)
	}
	if s.LogGroupID != "" {
		return []string{s.LogGroupID}
	}
	if s.CompartmentID != "" {
		return []string{s.CompartmentID}
	}
	return nil
}

// BuildScopeQuery assembles a `search "ocid1", "ocid2" | ` prefix from the
// current selection.
func (s *Session) BuildScopeQuery() string {
	ocids := s.ScopeOCIDs()
	if len(ocids) == 0 {
		return ""
	}
	quoted := make([]string, len(ocids))
	for i, id := range ocids {
		quoted[i] = fmt.Sprintf("%q", id)
	}
	return "search " + strings.Join(quoted, ", ") + " | "
}

// Navigation messages emitted by views and handled by the root model.

// GoUp requests navigation up one level.
type GoUp struct{}

// OpenCompartment requests drilling into a compartment.
type OpenCompartment struct {
	ID   string
	Name string
}

// OpenLogGroup requests drilling into a log group.
type OpenLogGroup struct {
	ID   string
	Name string
}
