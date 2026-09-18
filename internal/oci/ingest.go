package oci

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/loggingingestion"
)

// IngestLogs emits a batch of plain-text log lines to a specific log OCID.
func (c *Client) IngestLogs(ctx context.Context, logID, source, logType string, lines []string) error {
	entries := make([]loggingingestion.LogEntry, 0, len(lines))
	for _, line := range lines {
		entries = append(entries, loggingingestion.LogEntry{
			Data: new(line),
			Id:   new(uuid.NewString()),
			Time: &common.SDKTime{Time: time.Now()},
		})
	}

	req := loggingingestion.PutLogsRequest{
		LogId: new(logID),
		PutLogsDetails: loggingingestion.PutLogsDetails{
			Specversion: new("1.0"),
			LogEntryBatches: []loggingingestion.LogEntryBatch{
				{
					Entries:             entries,
					Source:              new(source),
					Type:                new(logType),
					Defaultlogentrytime: &common.SDKTime{Time: time.Now()},
				},
			},
		},
	}

	if _, err := c.ingest.PutLogs(ctx, req); err != nil {
		return fmt.Errorf("put logs: %w", err)
	}
	return nil
}
