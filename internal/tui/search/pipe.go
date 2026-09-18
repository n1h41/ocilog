package search

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// pipeTimeout bounds how long a single pipeline run may take.
const pipeTimeout = 10 * time.Second

// pipedMsg is the result of running the jq/sed pipeline.
type pipedMsg struct {
	content string
	err     error
}

// pipeCmd runs the user-supplied shell pipeline with the raw JSON on stdin and
// returns its stdout. jq and sed are provided by the system, so the pipeline
// supports arbitrary chaining such as `jq '.foo' | sed 's/x/y/g'`.
func (m Model) pipeCmd(pipeline, input string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), pipeTimeout)
		defer cancel()

		cmd := exec.CommandContext(ctx, "sh", "-c", pipeline)
		cmd.Stdin = strings.NewReader(input)

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return pipedMsg{err: fmt.Errorf("pipeline timed out after %s", pipeTimeout)}
			}
			msg := strings.TrimSpace(stderr.String())
			if msg == "" {
				msg = err.Error()
			}
			return pipedMsg{err: errors.New(msg)}
		}
		return pipedMsg{content: stdout.String()}
	}
}
