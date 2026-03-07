// Package executor abstracts subprocess spawning behind an interface.
// The real implementation wraps os/exec and the parser. Test code provides a
// fake implementation that emits canned events.
package executor

import (
	"context"

	"github.com/loxstomper/skinner/internal/session"
)

// Executor starts a Claude CLI subprocess and returns a stream of typed events.
type Executor interface {
	Start(ctx context.Context, prompt string) (<-chan session.Event, error)
	Kill() error
}
