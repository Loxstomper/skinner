package executor

import (
	"context"
	"time"

	"github.com/loxstomper/skinner/internal/session"
)

// FakeExecutor is a test double that emits pre-loaded events.
type FakeExecutor struct {
	Events []session.Event // canned events to emit
	Delay  time.Duration   // optional delay between events
	Prompt string          // records the prompt from the last Start call
}

// Start sends all canned events to the channel and closes it.
// If Delay is set, it pauses between each event. The context can cancel
// delivery early.
func (f *FakeExecutor) Start(ctx context.Context, prompt string) (<-chan session.Event, error) {
	f.Prompt = prompt
	ch := make(chan session.Event, len(f.Events))

	if f.Delay == 0 {
		for _, e := range f.Events {
			ch <- e
		}
		close(ch)
		return ch, nil
	}

	// With delay, run in a goroutine so Start returns immediately.
	go func() {
		defer close(ch)
		for _, e := range f.Events {
			select {
			case <-ctx.Done():
				return
			case <-time.After(f.Delay):
				ch <- e
			}
		}
	}()

	return ch, nil
}

// Kill is a no-op for the fake executor.
func (f *FakeExecutor) Kill() error {
	return nil
}
