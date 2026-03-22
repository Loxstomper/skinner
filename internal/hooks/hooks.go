package hooks

import (
	"fmt"
	"strconv"

	"github.com/loxstomper/skinner/internal/config"
)

// HookContext carries per-invocation state for building environment variables.
type HookContext struct {
	Iteration     int    // 1-based iteration index (0 means not set)
	IterationExit *int   // exit code of Claude subprocess (nil if not applicable)
	PromptFile    string // path to prompt file (empty if not applicable)
	MaxIterations int    // max iteration count (0 means unlimited)
	RunIndex      int    // 0-based run index
}

// Runner executes hook commands with the appropriate environment.
type Runner struct {
	Config  config.HooksConfig
	WorkDir string
}

// NewRunner creates a Runner with the given hooks configuration and working directory.
func NewRunner(cfg config.HooksConfig, workDir string) *Runner {
	return &Runner{
		Config:  cfg,
		WorkDir: workDir,
	}
}

// CommandFor returns the shell command string for the given hook name,
// or empty string if not configured.
func (r *Runner) CommandFor(hookName string) string {
	switch hookName {
	case "pre-iteration":
		return r.Config.PreIteration
	case "on-iteration-end":
		return r.Config.OnIterationEnd
	case "on-error":
		return r.Config.OnError
	case "on-idle":
		return r.Config.OnIdle
	default:
		return ""
	}
}

// BuildEnv constructs the environment variable slice for a hook invocation.
// Variables follow the spec: SKINNER_HOOK, SKINNER_ITERATION, SKINNER_PROMPT_FILE,
// SKINNER_MAX_ITERATIONS, SKINNER_RUN_INDEX. Conditionally includes
// SKINNER_ITERATION_EXIT for on-iteration-end and on-error hooks.
func (r *Runner) BuildEnv(hookName string, ctx HookContext) []string {
	env := []string{
		fmt.Sprintf("SKINNER_HOOK=%s", hookName),
	}

	if ctx.Iteration > 0 {
		env = append(env, fmt.Sprintf("SKINNER_ITERATION=%d", ctx.Iteration))
	}

	if ctx.IterationExit != nil {
		env = append(env, fmt.Sprintf("SKINNER_ITERATION_EXIT=%d", *ctx.IterationExit))
	}

	if ctx.PromptFile != "" {
		env = append(env, fmt.Sprintf("SKINNER_PROMPT_FILE=%s", ctx.PromptFile))
	}

	maxIter := "unlimited"
	if ctx.MaxIterations > 0 {
		maxIter = strconv.Itoa(ctx.MaxIterations)
	}
	env = append(env, fmt.Sprintf("SKINNER_MAX_ITERATIONS=%s", maxIter))

	env = append(env, fmt.Sprintf("SKINNER_RUN_INDEX=%d", ctx.RunIndex))

	return env
}
