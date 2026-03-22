package hooks

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

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

func TestRunner_RunPre_NotConfigured(t *testing.T) {
	r := NewRunner(config.HooksConfig{}, t.TempDir())
	result, err := r.RunPre(context.Background(), HookContext{Iteration: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Prompt != "" || result.Done {
		t.Errorf("expected empty result, got %+v", result)
	}
}

func TestRunner_RunPre_Prompt(t *testing.T) {
	r := NewRunner(config.HooksConfig{
		PreIteration: `echo '{"prompt": "fix tests"}'`,
		Timeout:      config.HooksTimeoutConfig{Default: 10},
	}, t.TempDir())

	result, err := r.RunPre(context.Background(), HookContext{Iteration: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Prompt != "fix tests" {
		t.Errorf("expected Prompt=%q, got %q", "fix tests", result.Prompt)
	}
	if result.Done {
		t.Error("expected Done=false")
	}
}

func TestRunner_RunPre_Done(t *testing.T) {
	r := NewRunner(config.HooksConfig{
		PreIteration: `echo '{"done": true}'`,
		Timeout:      config.HooksTimeoutConfig{Default: 10},
	}, t.TempDir())

	result, err := r.RunPre(context.Background(), HookContext{Iteration: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Done {
		t.Error("expected Done=true")
	}
}

func TestRunner_RunPre_DoneTakesPrecedence(t *testing.T) {
	r := NewRunner(config.HooksConfig{
		PreIteration: `echo '{"prompt": "fix tests", "done": true}'`,
		Timeout:      config.HooksTimeoutConfig{Default: 10},
	}, t.TempDir())

	result, err := r.RunPre(context.Background(), HookContext{Iteration: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Done {
		t.Error("expected Done=true when both prompt and done are set")
	}
}

func TestRunner_RunPre_EmptyStdout(t *testing.T) {
	r := NewRunner(config.HooksConfig{
		PreIteration: "true", // exits 0 with no output
		Timeout:      config.HooksTimeoutConfig{Default: 10},
	}, t.TempDir())

	result, err := r.RunPre(context.Background(), HookContext{Iteration: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Prompt != "" || result.Done {
		t.Errorf("expected empty result for empty stdout, got %+v", result)
	}
}

func TestRunner_RunPre_InvalidJSON(t *testing.T) {
	r := NewRunner(config.HooksConfig{
		PreIteration: "echo 'not json'",
		Timeout:      config.HooksTimeoutConfig{Default: 10},
	}, t.TempDir())

	result, err := r.RunPre(context.Background(), HookContext{Iteration: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Prompt != "" || result.Done {
		t.Errorf("expected empty result for invalid JSON, got %+v", result)
	}
}

func TestRunner_RunPre_NonZeroExit(t *testing.T) {
	r := NewRunner(config.HooksConfig{
		PreIteration: "echo 'hook error' >&2; exit 1",
		Timeout:      config.HooksTimeoutConfig{Default: 10},
	}, t.TempDir())

	_, err := r.RunPre(context.Background(), HookContext{Iteration: 1})
	if err == nil {
		t.Fatal("expected error for non-zero exit")
	}
}

func TestRunner_RunPre_UnrecognizedKeys(t *testing.T) {
	r := NewRunner(config.HooksConfig{
		PreIteration: `echo '{"foo": "bar"}'`,
		Timeout:      config.HooksTimeoutConfig{Default: 10},
	}, t.TempDir())

	result, err := r.RunPre(context.Background(), HookContext{Iteration: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Prompt != "" || result.Done {
		t.Errorf("expected empty result for unrecognized keys, got %+v", result)
	}
}

func TestRunner_RunPre_Timeout(t *testing.T) {
	one := 1
	r := NewRunner(config.HooksConfig{
		PreIteration: "sleep 10",
		Timeout:      config.HooksTimeoutConfig{Default: 10, PreIteration: &one},
	}, t.TempDir())

	start := time.Now()
	_, err := r.RunPre(context.Background(), HookContext{Iteration: 1})
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error for timeout")
	}
	if elapsed > 3*time.Second {
		t.Errorf("timeout took too long: %v", elapsed)
	}
}

func TestRunner_RunPre_EnvVars(t *testing.T) {
	tmpDir := t.TempDir()
	outFile := filepath.Join(tmpDir, "env.txt")

	r := NewRunner(config.HooksConfig{
		PreIteration: "env | grep SKINNER > " + outFile,
		Timeout:      config.HooksTimeoutConfig{Default: 10},
	}, tmpDir)

	ctx := HookContext{
		Iteration:     3,
		PromptFile:    "PROMPT.md",
		MaxIterations: 10,
		RunIndex:      1,
	}

	_, err := r.RunPre(context.Background(), ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("failed to read env output: %v", err)
	}
	output := string(data)

	expected := []string{
		"SKINNER_HOOK=pre-iteration",
		"SKINNER_ITERATION=3",
		"SKINNER_PROMPT_FILE=PROMPT.md",
		"SKINNER_MAX_ITERATIONS=10",
		"SKINNER_RUN_INDEX=1",
	}
	for _, exp := range expected {
		if !containsLine(output, exp) {
			t.Errorf("expected env to contain %q, got:\n%s", exp, output)
		}
	}
}

func TestRunner_RunEvent_PostIterationEnvVars(t *testing.T) {
	tmpDir := t.TempDir()
	outFile := filepath.Join(tmpDir, "env.txt")

	r := NewRunner(config.HooksConfig{
		OnIterationEnd: "env | grep SKINNER > " + outFile,
		Timeout:        config.HooksTimeoutConfig{Default: 10},
	}, tmpDir)

	exitCode := 1
	ctx := HookContext{
		Iteration:     2,
		IterationExit: &exitCode,
		PromptFile:    "BUILD.md",
		MaxIterations: 5,
	}

	r.RunEvent("on-iteration-end", ctx)

	// Wait for goroutine
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(outFile); err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("failed to read env output: %v", err)
	}
	output := string(data)

	if !containsLine(output, "SKINNER_ITERATION_EXIT=1") {
		t.Errorf("expected SKINNER_ITERATION_EXIT=1 in env, got:\n%s", output)
	}
	if !containsLine(output, "SKINNER_HOOK=on-iteration-end") {
		t.Errorf("expected SKINNER_HOOK=on-iteration-end in env, got:\n%s", output)
	}
}

func containsLine(s, substr string) bool {
	for _, line := range splitLines(s) {
		if line == substr {
			return true
		}
	}
	return false
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func TestRunner_RunEvent_NotConfigured(t *testing.T) {
	r := NewRunner(config.HooksConfig{}, t.TempDir())
	// Should not panic or block
	r.RunEvent("on-idle", HookContext{})
}

func TestRunner_RunEvent_FireAndForget(t *testing.T) {
	tmpDir := t.TempDir()
	marker := filepath.Join(tmpDir, "hook-ran")

	r := NewRunner(config.HooksConfig{
		OnIdle:  "touch " + marker,
		Timeout: config.HooksTimeoutConfig{Default: 10},
	}, tmpDir)

	r.RunEvent("on-idle", HookContext{})

	// Wait for the goroutine to complete
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(marker); err == nil {
			return // success
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("hook did not run within timeout")
}

func TestRunner_RunEvent_NonZeroExitIgnored(t *testing.T) {
	r := NewRunner(config.HooksConfig{
		OnError: "exit 1",
		Timeout: config.HooksTimeoutConfig{Default: 10},
	}, t.TempDir())

	// Should not panic
	r.RunEvent("on-error", HookContext{Iteration: 1})
	time.Sleep(50 * time.Millisecond) // let goroutine finish
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
