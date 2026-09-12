package oci

import (
	"context"
	"fmt"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/loggingsearch"
)

// SearchQuery is a time-bounded search against the Logging Search API.
type SearchQuery struct {
	Query string
	Start time.Time
	End   time.Time
	Limit int
}

// SearchLogs runs a query and returns each result's raw JSON payload.
func (c *Client) SearchLogs(ctx context.Context, q SearchQuery) ([]interface{}, error) {
	if q.Limit <= 0 {
		q.Limit = 100
	}

	req := loggingsearch.SearchLogsRequest{
		SearchLogsDetails: loggingsearch.SearchLogsDetails{
			TimeStart:   &common.SDKTime{Time: q.Start},
			TimeEnd:     &common.SDKTime{Time: q.End},
			SearchQuery: common.String(q.Query),
		},
		Limit: common.Int(q.Limit),
	}

	resp, err := c.search.SearchLogs(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("search logs: %w", err)
	}

	out := make([]interface{}, 0, len(resp.Results))
	for _, r := range resp.Results {
		if r.Data == nil {
			out = append(out, nil)
			continue
		}
		out = append(out, *r.Data)
	}
	return out, nil
}
