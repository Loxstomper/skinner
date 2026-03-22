package hooks

import (
	"testing"

	"github.com/loxstomper/skinner/internal/config"
)

func TestNewRunner(t *testing.T) {
	cfg := config.HooksConfig{
		PreIteration: "./check.sh",
	}
	r := NewRunner(cfg, "/tmp/work")

	if r.WorkDir != "/tmp/work" {
		t.Errorf("expected WorkDir=%q, got %q", "/tmp/work", r.WorkDir)
	}
	if r.Config.PreIteration != "./check.sh" {
		t.Errorf("expected PreIteration=%q, got %q", "./check.sh", r.Config.PreIteration)
	}
}

func TestRunner_CommandFor(t *testing.T) {
	cfg := config.HooksConfig{
		PreIteration:   "./pre.sh",
		OnIterationEnd: "./end.sh",
		OnError:        "./error.sh",
		OnIdle:         "./idle.sh",
	}
	r := NewRunner(cfg, "/tmp")

	tests := []struct {
		hookName string
		want     string
	}{
		{"pre-iteration", "./pre.sh"},
		{"on-iteration-end", "./end.sh"},
		{"on-error", "./error.sh"},
		{"on-idle", "./idle.sh"},
		{"unknown", ""},
	}
	for _, tt := range tests {
		t.Run(tt.hookName, func(t *testing.T) {
			got := r.CommandFor(tt.hookName)
			if got != tt.want {
				t.Errorf("CommandFor(%q)=%q, want %q", tt.hookName, got, tt.want)
			}
		})
	}
}

func TestRunner_BuildEnv_PreIteration(t *testing.T) {
	r := NewRunner(config.HooksConfig{}, "/tmp")
	ctx := HookContext{
		Iteration:     3,
		PromptFile:    "PROMPT.md",
		MaxIterations: 10,
		RunIndex:      1,
	}

	env := r.BuildEnv("pre-iteration", ctx)
	envMap := envToMap(env)

	assertEnv(t, envMap, "SKINNER_HOOK", "pre-iteration")
	assertEnv(t, envMap, "SKINNER_ITERATION", "3")
	assertEnv(t, envMap, "SKINNER_PROMPT_FILE", "PROMPT.md")
	assertEnv(t, envMap, "SKINNER_MAX_ITERATIONS", "10")
	assertEnv(t, envMap, "SKINNER_RUN_INDEX", "1")

	if _, ok := envMap["SKINNER_ITERATION_EXIT"]; ok {
		t.Error("SKINNER_ITERATION_EXIT should not be set for pre-iteration")
	}
}

func TestRunner_BuildEnv_OnIterationEnd(t *testing.T) {
	r := NewRunner(config.HooksConfig{}, "/tmp")
	exitCode := 1
	ctx := HookContext{
		Iteration:     5,
		IterationExit: &exitCode,
		PromptFile:    "PROMPT.md",
		MaxIterations: 20,
		RunIndex:      0,
	}

	env := r.BuildEnv("on-iteration-end", ctx)
	envMap := envToMap(env)

	assertEnv(t, envMap, "SKINNER_HOOK", "on-iteration-end")
	assertEnv(t, envMap, "SKINNER_ITERATION", "5")
	assertEnv(t, envMap, "SKINNER_ITERATION_EXIT", "1")
	assertEnv(t, envMap, "SKINNER_MAX_ITERATIONS", "20")
}

func TestRunner_BuildEnv_UnlimitedIterations(t *testing.T) {
	r := NewRunner(config.HooksConfig{}, "/tmp")
	ctx := HookContext{
		Iteration:     1,
		MaxIterations: 0, // unlimited
	}

	env := r.BuildEnv("pre-iteration", ctx)
	envMap := envToMap(env)

	assertEnv(t, envMap, "SKINNER_MAX_ITERATIONS", "unlimited")
}

func TestRunner_BuildEnv_IdleNoIteration(t *testing.T) {
	r := NewRunner(config.HooksConfig{}, "/tmp")
	ctx := HookContext{
		Iteration: 0, // no iterations have run
	}

	env := r.BuildEnv("on-idle", ctx)
	envMap := envToMap(env)

	assertEnv(t, envMap, "SKINNER_HOOK", "on-idle")
	if _, ok := envMap["SKINNER_ITERATION"]; ok {
		t.Error("SKINNER_ITERATION should not be set when Iteration=0")
	}
}

func TestRunner_BuildEnv_NoPromptFile(t *testing.T) {
	r := NewRunner(config.HooksConfig{}, "/tmp")
	ctx := HookContext{Iteration: 1}

	env := r.BuildEnv("on-idle", ctx)
	envMap := envToMap(env)

	if _, ok := envMap["SKINNER_PROMPT_FILE"]; ok {
		t.Error("SKINNER_PROMPT_FILE should not be set when empty")
	}
}

// envToMap converts a slice of "KEY=VALUE" strings to a map.
func envToMap(env []string) map[string]string {
	m := make(map[string]string, len(env))
	for _, e := range env {
		for i := 0; i < len(e); i++ {
			if e[i] == '=' {
				m[e[:i]] = e[i+1:]
				break
			}
		}
	}
	return m
}

func assertEnv(t *testing.T, envMap map[string]string, key, want string) {
	t.Helper()
	got, ok := envMap[key]
	if !ok {
		t.Errorf("expected %s to be set", key)
		return
	}
	if got != want {
		t.Errorf("%s=%q, want %q", key, got, want)
	}
}
