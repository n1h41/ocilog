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

// maxSearchWindow is the maximum time range a single Logging Search call can
// cover. Longer ranges are split into consecutive windows.
const maxSearchWindow = 14 * 24 * time.Hour

// SearchLogs runs a query over the requested time range and returns each
// result's raw JSON payload. Because a single Logging Search call only covers
// 14 days, longer ranges are split into consecutive windows and the results
// combined.
func (c *Client) SearchLogs(ctx context.Context, q SearchQuery) ([]any, error) {
	if q.Limit <= 0 {
		q.Limit = 100
	}

	var out []any
	for start := q.Start; start.Before(q.End); {
		end := start.Add(maxSearchWindow)
		if end.After(q.End) {
			end = q.End
		}

		results, err := c.searchWindow(ctx, q.Query, start, end, q.Limit)
		if err != nil {
			return nil, err
		}
		out = append(out, results...)

		start = end
	}
	return out, nil
}

// searchWindow performs a single Logging Search call over one time window.
func (c *Client) searchWindow(ctx context.Context, query string, start, end time.Time, limit int) ([]any, error) {
	req := loggingsearch.SearchLogsRequest{
		SearchLogsDetails: loggingsearch.SearchLogsDetails{
			TimeStart:   &common.SDKTime{Time: start},
			TimeEnd:     &common.SDKTime{Time: end},
			SearchQuery: new(query),
		},
		Limit: new(limit),
	}

	resp, err := c.search.SearchLogs(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("search logs: %w", err)
	}

	out := make([]any, 0, len(resp.Results))
	for _, r := range resp.Results {
		if r.Data == nil {
			out = append(out, nil)
			continue
		}
		out = append(out, *r.Data)
	}
	return out, nil
}
