package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultPricing(t *testing.T) {
	pricing := DefaultPricing()

	// Check that all three models are present
	expectedModels := []string{"claude-opus-4-6", "claude-sonnet-4-5", "claude-haiku-4-5"}
	if len(pricing) != len(expectedModels) {
		t.Fatalf("expected %d models, got %d", len(expectedModels), len(pricing))
	}

	// Verify each model has ContextWindow set to 200000
	for _, model := range expectedModels {
		mp, ok := pricing[model]
		if !ok {
			t.Errorf("expected model %q to be present in pricing", model)
			continue
		}
		if mp.ContextWindow != 200000 {
			t.Errorf("model %q: expected ContextWindow=200000, got %d", model, mp.ContextWindow)
		}
	}
}

func TestLoadConfig_ContextWindowFromTOML(t *testing.T) {
	// Create a temporary directory to act as HOME
	tmpDir := t.TempDir()

	// Create the config directory structure
	configDir := filepath.Join(tmpDir, ".config", "skinner")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	// Write a test config file with context_window values
	configPath := filepath.Join(configDir, "config.toml")
	configContent := `[view]
mode = "compact"

[theme]
name = "test-theme"

[pricing.claude-opus-4-6]
input = 0.000010
output = 0.000050
context_window = 300000

[pricing.claude-sonnet-4-5]
context_window = 250000

[pricing.custom-model]
input = 0.000001
output = 0.000005
context_window = 150000
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Override HOME environment variable for this test
	t.Setenv("HOME", tmpDir)

	// Load the config
	cfg := LoadConfig()

	// Test that context_window was parsed correctly for existing model
	if opus, ok := cfg.Pricing["claude-opus-4-6"]; ok {
		if opus.ContextWindow != 300000 {
			t.Errorf("claude-opus-4-6: expected ContextWindow=300000, got %d", opus.ContextWindow)
		}
		// Verify other fields were also parsed
		if opus.Input != 0.000010 {
			t.Errorf("claude-opus-4-6: expected Input=0.000010, got %f", opus.Input)
		}
		if opus.Output != 0.000050 {
			t.Errorf("claude-opus-4-6: expected Output=0.000050, got %f", opus.Output)
		}
	} else {
		t.Error("expected claude-opus-4-6 to be present in pricing")
	}

	// Test partial override (only context_window specified)
	if sonnet, ok := cfg.Pricing["claude-sonnet-4-5"]; ok {
		if sonnet.ContextWindow != 250000 {
			t.Errorf("claude-sonnet-4-5: expected ContextWindow=250000, got %d", sonnet.ContextWindow)
		}
		// Default values should still be present for unspecified fields
		if sonnet.Input != 0.000003 {
			t.Errorf("claude-sonnet-4-5: expected default Input=0.000003, got %f", sonnet.Input)
		}
	} else {
		t.Error("expected claude-sonnet-4-5 to be present in pricing")
	}

	// Test new custom model
	if custom, ok := cfg.Pricing["custom-model"]; ok {
		if custom.ContextWindow != 150000 {
			t.Errorf("custom-model: expected ContextWindow=150000, got %d", custom.ContextWindow)
		}
		if custom.Input != 0.000001 {
			t.Errorf("custom-model: expected Input=0.000001, got %f", custom.Input)
		}
	} else {
		t.Error("expected custom-model to be present in pricing")
	}

	// Test that haiku still has default context_window (not overridden in config)
	if haiku, ok := cfg.Pricing["claude-haiku-4-5"]; ok {
		if haiku.ContextWindow != 200000 {
			t.Errorf("claude-haiku-4-5: expected default ContextWindow=200000, got %d", haiku.ContextWindow)
		}
	} else {
		t.Error("expected claude-haiku-4-5 to be present in pricing")
	}
}

func TestDefaultConfig_Layout(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Layout != "auto" {
		t.Errorf("expected default Layout=%q, got %q", "auto", cfg.Layout)
	}
}

func TestLoadConfig_LayoutValues(t *testing.T) {
	for _, val := range []string{"side", "bottom", "auto"} {
		t.Run(val, func(t *testing.T) {
			tmpDir := t.TempDir()
			configDir := filepath.Join(tmpDir, ".config", "skinner")
			if err := os.MkdirAll(configDir, 0755); err != nil {
				t.Fatalf("failed to create config dir: %v", err)
			}

			configContent := "[view]\nlayout = \"" + val + "\"\n"
			if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(configContent), 0644); err != nil {
				t.Fatalf("failed to write config file: %v", err)
			}

			t.Setenv("HOME", tmpDir)
			cfg := LoadConfig()

			if cfg.Layout != val {
				t.Errorf("expected Layout=%q, got %q", val, cfg.Layout)
			}
		})
	}
}

func TestLoadConfig_LayoutInvalidFallsBackToDefault(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".config", "skinner")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	configContent := "[view]\nlayout = \"invalid\"\n"
	if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	t.Setenv("HOME", tmpDir)
	cfg := LoadConfig()

	if cfg.Layout != "auto" {
		t.Errorf("expected Layout=%q for invalid value, got %q", "auto", cfg.Layout)
	}
}

func TestDefaultConfig_PlanCommand(t *testing.T) {
	cfg := DefaultConfig()
	expected := `claude "study specs/README.md"`
	if cfg.PlanCommand != expected {
		t.Errorf("expected default PlanCommand=%q, got %q", expected, cfg.PlanCommand)
	}
}

func TestLoadConfig_PlanCommandOverride(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".config", "skinner")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	configContent := `[plan]
command = "claude --verbose"
`
	if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	t.Setenv("HOME", tmpDir)
	cfg := LoadConfig()

	expected := "claude --verbose"
	if cfg.PlanCommand != expected {
		t.Errorf("expected PlanCommand=%q, got %q", expected, cfg.PlanCommand)
	}
}

func TestDefaultConfig_HooksConfig(t *testing.T) {
	cfg := DefaultConfig()

	// Hook commands should be empty by default
	if cfg.Hooks.PreIteration != "" {
		t.Errorf("expected empty PreIteration, got %q", cfg.Hooks.PreIteration)
	}
	if cfg.Hooks.OnIterationEnd != "" {
		t.Errorf("expected empty OnIterationEnd, got %q", cfg.Hooks.OnIterationEnd)
	}
	if cfg.Hooks.OnError != "" {
		t.Errorf("expected empty OnError, got %q", cfg.Hooks.OnError)
	}
	if cfg.Hooks.OnIdle != "" {
		t.Errorf("expected empty OnIdle, got %q", cfg.Hooks.OnIdle)
	}

	// Default timeout should be 10 seconds
	if cfg.Hooks.Timeout.Default != 10 {
		t.Errorf("expected Timeout.Default=10, got %d", cfg.Hooks.Timeout.Default)
	}

	// Per-hook timeout overrides should be nil
	if cfg.Hooks.Timeout.PreIteration != nil {
		t.Errorf("expected Timeout.PreIteration=nil, got %v", cfg.Hooks.Timeout.PreIteration)
	}
	if cfg.Hooks.Timeout.OnIterationEnd != nil {
		t.Errorf("expected Timeout.OnIterationEnd=nil, got %v", cfg.Hooks.Timeout.OnIterationEnd)
	}
	if cfg.Hooks.Timeout.OnError != nil {
		t.Errorf("expected Timeout.OnError=nil, got %v", cfg.Hooks.Timeout.OnError)
	}
	if cfg.Hooks.Timeout.OnIdle != nil {
		t.Errorf("expected Timeout.OnIdle=nil, got %v", cfg.Hooks.Timeout.OnIdle)
	}
}

func TestLoadConfig_HooksFromTOML(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".config", "skinner")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	configContent := `[hooks]
pre-iteration = "./scripts/check-ready.sh"
on-iteration-end = "echo done >> /tmp/log"
on-error = "notify-send Skinner failed"
on-idle = "./scripts/cleanup.sh"

[hooks.timeout]
pre-iteration = "60s"
on-error = "5s"
on-idle = "2m"
`
	if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	t.Setenv("HOME", tmpDir)
	cfg := LoadConfig()

	// Hook commands
	if cfg.Hooks.PreIteration != "./scripts/check-ready.sh" {
		t.Errorf("expected PreIteration=%q, got %q", "./scripts/check-ready.sh", cfg.Hooks.PreIteration)
	}
	if cfg.Hooks.OnIterationEnd != "echo done >> /tmp/log" {
		t.Errorf("expected OnIterationEnd=%q, got %q", "echo done >> /tmp/log", cfg.Hooks.OnIterationEnd)
	}
	if cfg.Hooks.OnError != "notify-send Skinner failed" {
		t.Errorf("expected OnError=%q, got %q", "notify-send Skinner failed", cfg.Hooks.OnError)
	}
	if cfg.Hooks.OnIdle != "./scripts/cleanup.sh" {
		t.Errorf("expected OnIdle=%q, got %q", "./scripts/cleanup.sh", cfg.Hooks.OnIdle)
	}

	// Timeout overrides
	if cfg.Hooks.Timeout.PreIteration == nil || *cfg.Hooks.Timeout.PreIteration != 60 {
		t.Errorf("expected Timeout.PreIteration=60, got %v", cfg.Hooks.Timeout.PreIteration)
	}
	if cfg.Hooks.Timeout.OnError == nil || *cfg.Hooks.Timeout.OnError != 5 {
		t.Errorf("expected Timeout.OnError=5, got %v", cfg.Hooks.Timeout.OnError)
	}
	if cfg.Hooks.Timeout.OnIdle == nil || *cfg.Hooks.Timeout.OnIdle != 120 {
		t.Errorf("expected Timeout.OnIdle=120 (2m), got %v", cfg.Hooks.Timeout.OnIdle)
	}
	// on-iteration-end not set in config, should be nil
	if cfg.Hooks.Timeout.OnIterationEnd != nil {
		t.Errorf("expected Timeout.OnIterationEnd=nil, got %v", cfg.Hooks.Timeout.OnIterationEnd)
	}
	// Default timeout should be unchanged
	if cfg.Hooks.Timeout.Default != 10 {
		t.Errorf("expected Timeout.Default=10, got %d", cfg.Hooks.Timeout.Default)
	}
}

func TestLoadConfig_HooksUnknownKeysIgnored(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".config", "skinner")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	configContent := `[hooks]
pre-iteration = "./scripts/check.sh"
unknown-hook = "should be ignored"

[hooks.timeout]
unknown-hook = "30s"
`
	if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	t.Setenv("HOME", tmpDir)
	cfg := LoadConfig()

	// Known hook should be parsed
	if cfg.Hooks.PreIteration != "./scripts/check.sh" {
		t.Errorf("expected PreIteration=%q, got %q", "./scripts/check.sh", cfg.Hooks.PreIteration)
	}
	// Unknown hooks silently ignored — no error, other fields unchanged
	if cfg.Hooks.OnError != "" {
		t.Errorf("expected empty OnError, got %q", cfg.Hooks.OnError)
	}
}

func TestLoadConfig_HooksPartial(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".config", "skinner")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	configContent := `[hooks]
pre-iteration = "./scripts/check.sh"
`
	if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	t.Setenv("HOME", tmpDir)
	cfg := LoadConfig()

	if cfg.Hooks.PreIteration != "./scripts/check.sh" {
		t.Errorf("expected PreIteration=%q, got %q", "./scripts/check.sh", cfg.Hooks.PreIteration)
	}
	if cfg.Hooks.OnIterationEnd != "" {
		t.Errorf("expected empty OnIterationEnd, got %q", cfg.Hooks.OnIterationEnd)
	}
	if cfg.Hooks.OnError != "" {
		t.Errorf("expected empty OnError, got %q", cfg.Hooks.OnError)
	}
	if cfg.Hooks.OnIdle != "" {
		t.Errorf("expected empty OnIdle, got %q", cfg.Hooks.OnIdle)
	}
}

func TestHooksTimeoutConfig_TimeoutFor(t *testing.T) {
	five := 5
	sixty := 60

	tests := []struct {
		name     string
		config   HooksTimeoutConfig
		hookName string
		want     int
	}{
		{"pre-iteration default", HooksTimeoutConfig{Default: 10}, "pre-iteration", 30},
		{"on-iteration-end default", HooksTimeoutConfig{Default: 10}, "on-iteration-end", 10},
		{"on-error default", HooksTimeoutConfig{Default: 10}, "on-error", 10},
		{"on-idle default", HooksTimeoutConfig{Default: 10}, "on-idle", 10},
		{"unknown hook", HooksTimeoutConfig{Default: 10}, "unknown", 10},
		{"pre-iteration override", HooksTimeoutConfig{Default: 10, PreIteration: &sixty}, "pre-iteration", 60},
		{"on-error override", HooksTimeoutConfig{Default: 10, OnError: &five}, "on-error", 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.TimeoutFor(tt.hookName)
			if got != tt.want {
				t.Errorf("TimeoutFor(%q)=%d, want %d", tt.hookName, got, tt.want)
			}
		})
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input   string
		wantSec int
		wantOK  bool
	}{
		{"30s", 30, true},
		{"2m", 120, true},
		{"5s", 5, true},
		{"0s", 0, true},
		{"", 0, false},
		{"s", 0, false},
		{"30", 0, false},
		{"abc", 0, false},
		{"30x", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, ok := parseDuration(tt.input)
			if ok != tt.wantOK {
				t.Errorf("parseDuration(%q): ok=%v, want %v", tt.input, ok, tt.wantOK)
			}
			if got != tt.wantSec {
				t.Errorf("parseDuration(%q)=%d, want %d", tt.input, got, tt.wantSec)
			}
		})
	}
}

func TestLoadConfig_NoConfigFile(t *testing.T) {
	// Create a temporary directory with no config file
	tmpDir := t.TempDir()

	t.Setenv("HOME", tmpDir)

	// Load config should return defaults
	cfg := LoadConfig()

	// Verify defaults are returned
	if cfg.ViewMode != "full" {
		t.Errorf("expected default ViewMode=full, got %s", cfg.ViewMode)
	}

	// Verify default pricing with 200000 context window
	if opus, ok := cfg.Pricing["claude-opus-4-6"]; ok {
		if opus.ContextWindow != 200000 {
			t.Errorf("expected default ContextWindow=200000, got %d", opus.ContextWindow)
		}
	} else {
		t.Error("expected default pricing to include claude-opus-4-6")
	}
}
