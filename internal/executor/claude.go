package executor

import (
	"bufio"
	"context"
	"io"
	"os/exec"
	"strings"
	"sync"

	"github.com/loxstomper/skinner/internal/parser"
	"github.com/loxstomper/skinner/internal/session"
)

// ClaudeExecutor spawns a Claude CLI subprocess and streams parsed events.
type ClaudeExecutor struct {
	mu  sync.Mutex
	cmd *exec.Cmd
}

// Start spawns `claude -p --dangerously-skip-permissions --output-format=stream-json --verbose`,
// pipes the prompt to stdin, and returns a channel of session events. The channel
// is closed after the subprocess exits and a SubprocessExitEvent has been sent.
func (e *ClaudeExecutor) Start(ctx context.Context, prompt string) (<-chan session.Event, error) {
	cmd := exec.CommandContext(ctx, "claude",
		"-p",
		"--dangerously-skip-permissions",
		"--output-format=stream-json",
		"--verbose",
	)
	cmd.Stdin = strings.NewReader(prompt)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	e.mu.Lock()
	e.cmd = cmd
	e.mu.Unlock()

	ch := make(chan session.Event, 64)

	go func() {
		defer close(ch)
		readEvents(stdout, ch)
		waitErr := cmd.Wait()
		ch <- session.SubprocessExitEvent{Err: waitErr}
	}()

	return ch, nil
}

// Kill terminates the running subprocess, if any.
func (e *ClaudeExecutor) Kill() error {
	e.mu.Lock()
	cmd := e.cmd
	e.mu.Unlock()

	if cmd == nil || cmd.Process == nil {
		return nil
	}
	return cmd.Process.Kill()
}

// readEvents reads stdout line-by-line, parses each line with the parser, and
// converts parser events to session events sent on the channel. ToolUseEvent
// and TextEvent from a single assistant message are grouped into an
// AssistantBatchEvent; UsageEvent, ToolResultEvent, and IterationEndEvent are
// sent individually.
func readEvents(r io.Reader, ch chan<- session.Event) {
	scanner := bufio.NewScanner(r)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		events, err := parser.ParseStreamEvent(line)
		if err != nil {
			continue
		}

		var batch []session.Event
		for _, evt := range events {
			switch e := evt.(type) {
			case parser.UsageEvent:
				ch <- session.UsageEvent{
					Model:                    e.Model,
					InputTokens:              e.InputTokens,
					OutputTokens:             e.OutputTokens,
					CacheReadInputTokens:     e.CacheReadInputTokens,
					CacheCreationInputTokens: e.CacheCreationInputTokens,
				}
			case parser.ToolUseEvent:
				batch = append(batch, session.ToolUseEvent{
					ID:       e.ID,
					Name:     e.Name,
					Summary:  e.Summary,
					LineInfo: e.LineInfo,
					RawInput: e.RawInput,
				})
			case parser.TextEvent:
				batch = append(batch, session.TextEvent{
					Text: e.Text,
				})
			case parser.ToolResultEvent:
				ch <- session.ToolResultEvent{
					ToolUseID: e.ToolUseID,
					IsError:   e.IsError,
					LineInfo:  e.LineInfo,
					Content:   e.Content,
				}
			case parser.IterationEndEvent:
				ch <- session.IterationEndEvent{}
			}
		}
		if len(batch) > 0 {
			ch <- session.AssistantBatchEvent{Events: batch}
		}
	}
}
