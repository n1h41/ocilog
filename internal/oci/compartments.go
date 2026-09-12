package oci

import (
	"context"
	"fmt"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/identity"
)

// Compartment is a trimmed view of a compartment suitable for display.
type Compartment struct {
	ID     string
	Name   string
	Parent string
	State  string
}

// ListCompartments returns all compartments in the tenancy, prefixed with the
// root (tenancy) compartment itself.
func (c *Client) ListCompartments(ctx context.Context, tenancyID string) ([]Compartment, error) {
	req := identity.ListCompartmentsRequest{
		CompartmentId:          common.String(tenancyID),
		CompartmentIdInSubtree: common.Bool(true),
		AccessLevel:            identity.ListCompartmentsAccessLevelAccessible,
		Limit:                  common.Int(1000),
	}
	resp, err := c.identity.ListCompartments(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("list compartments: %w", err)
	}

	out := make([]Compartment, 0, len(resp.Items)+1)
	out = append(out, Compartment{ID: tenancyID, Name: "(tenancy) root", State: "ACTIVE"})
	for _, comp := range resp.Items {
		out = append(out, Compartment{
			ID:     deref(comp.Id),
			Name:   deref(comp.Name),
			Parent: deref(comp.CompartmentId),
			State:  string(comp.LifecycleState),
		})
	}
	return out, nil
}
