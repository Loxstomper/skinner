package hooks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"

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

// PreIterationResult holds the parsed output of a pre-iteration hook.
type PreIterationResult struct {
	Prompt string // replacement prompt (empty = use prompt file)
	Title  string // optional header text for the timeline pane
	Done   bool   // true = stop the loop
}

// RunPre executes the pre-iteration hook and parses its JSON output.
// Returns (PreIterationResult{}, nil) if the hook is not configured or
// stdout is empty/invalid JSON (iteration proceeds normally per spec).
// Returns error only on non-zero exit or timeout.
func (r *Runner) RunPre(ctx context.Context, hookCtx HookContext) (PreIterationResult, error) {
	command := r.CommandFor("pre-iteration")
	if command == "" {
		return PreIterationResult{}, nil
	}

	timeout := time.Duration(r.Config.Timeout.TimeoutFor("pre-iteration")) * time.Second
	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "sh", "-c", command)
	cmd.Dir = r.WorkDir
	cmd.Env = append(os.Environ(), r.BuildEnv("pre-iteration", hookCtx)...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return PreIterationResult{}, fmt.Errorf("pre-iteration hook failed: %s: %w", stderr.String(), err)
	}

	// Empty stdout = no effect, proceed normally
	out := bytes.TrimSpace(stdout.Bytes())
	if len(out) == 0 {
		return PreIterationResult{}, nil
	}

	// Parse JSON output
	var parsed struct {
		Prompt string `json:"prompt"`
		Title  string `json:"title"`
		Done   *bool  `json:"done"`
	}
	if err := json.Unmarshal(out, &parsed); err != nil {
		// Invalid JSON = no effect per spec
		return PreIterationResult{}, nil
	}

	// done takes precedence over prompt and title
	if parsed.Done != nil && *parsed.Done {
		return PreIterationResult{Done: true}, nil
	}

	result := PreIterationResult{
		Prompt: parsed.Prompt,
		Title:  parsed.Title,
	}

	if result.Prompt != "" || result.Title != "" {
		return result, nil
	}

	// Valid JSON but no recognized keys = no effect
	return PreIterationResult{}, nil
}

// RunEvent launches a fire-and-forget on-* hook in a goroutine.
// Returns immediately. Errors, timeouts, and output are silently ignored.
// No-op if the hook is not configured.
func (r *Runner) RunEvent(hookName string, hookCtx HookContext) {
	command := r.CommandFor(hookName)
	if command == "" {
		return
	}

	timeout := time.Duration(r.Config.Timeout.TimeoutFor(hookName)) * time.Second
	env := r.BuildEnv(hookName, hookCtx)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		cmd := exec.CommandContext(ctx, "sh", "-c", command)
		cmd.Dir = r.WorkDir
		cmd.Env = append(os.Environ(), env...)
		_ = cmd.Run()
	}()
}
