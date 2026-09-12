package oci

import (
	"context"
	"fmt"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/logging"
)

// LogGroup is a trimmed view of a log group suitable for display.
type LogGroup struct {
	ID          string
	Name        string
	Compartment string
}

// Log is a trimmed view of a log object suitable for display.
type Log struct {
	ID       string
	LogGroup string
	Name     string
	LogType  string
	Enabled  bool
}

// ListLogGroups returns all log groups in the given compartment OCID.
func (c *Client) ListLogGroups(ctx context.Context, compartmentID string) ([]LogGroup, error) {
	req := logging.ListLogGroupsRequest{
		CompartmentId: common.String(compartmentID),
		Limit:         common.Int(1000),
	}
	resp, err := c.manage.ListLogGroups(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("list log groups: %w", err)
	}

	groups := make([]LogGroup, 0, len(resp.Items))
	for _, g := range resp.Items {
		groups = append(groups, LogGroup{
			ID:          deref(g.Id),
			Name:        deref(g.DisplayName),
			Compartment: deref(g.CompartmentId),
		})
	}
	return groups, nil
}

// ListLogs returns all logs within the given log group OCID.
func (c *Client) ListLogs(ctx context.Context, logGroupID string) ([]Log, error) {
	req := logging.ListLogsRequest{
		LogGroupId: common.String(logGroupID),
		Limit:      common.Int(1000),
	}
	resp, err := c.manage.ListLogs(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("list logs: %w", err)
	}

	logs := make([]Log, 0, len(resp.Items))
	for _, l := range resp.Items {
		logs = append(logs, Log{
			ID:       deref(l.Id),
			LogGroup: deref(l.LogGroupId),
			Name:     deref(l.DisplayName),
			LogType:  string(l.LogType),
			Enabled:  boolDeref(l.IsEnabled),
		})
	}
	return logs, nil
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func boolDeref(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}
