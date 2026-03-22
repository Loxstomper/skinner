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
