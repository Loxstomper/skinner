package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultKeyMap_AllActionsPresent(t *testing.T) {
	km := DefaultKeyMap()

	expectedActions := AllActions()
	for _, action := range expectedActions {
		if _, ok := km.Bindings[action]; !ok {
			t.Errorf("expected action %q in default keymap", action)
		}
	}
}

func TestDefaultKeyMap_DefaultBindings(t *testing.T) {
	km := DefaultKeyMap()

	cases := []struct {
		action   string
		expected string
	}{
		{ActionQuit, "q"},
		{ActionHelp, "?"},
		{ActionToggleLeftPane, "["},
		{ActionToggleLineNumbers, "#"},
		{ActionToggleView, "v"},
		{ActionFocusLeft, "h"},
		{ActionFocusRight, "l"},
		{ActionFocusToggle, "tab"},
		{ActionMoveDown, "j"},
		{ActionMoveUp, "k"},
		{ActionJumpTop, "g g"},
		{ActionJumpBottom, "G"},
		{ActionExpand, "enter"},
		{ActionEscape, "escape"},
	}

	for _, tc := range cases {
		got := km.Bindings[tc.action].String()
		if got != tc.expected {
			t.Errorf("action %q: expected binding %q, got %q", tc.action, tc.expected, got)
		}
	}
}

func TestParseKeyBinding_SingleKey(t *testing.T) {
	kb := ParseKeyBinding("q")
	if kb.String() != "q" {
		t.Errorf("expected %q, got %q", "q", kb.String())
	}
	if kb.IsSequence() {
		t.Error("single key should not be a sequence")
	}
}

func TestParseKeyBinding_Modifier(t *testing.T) {
	kb := ParseKeyBinding("ctrl+c")
	if kb.String() != "ctrl+c" {
		t.Errorf("expected %q, got %q", "ctrl+c", kb.String())
	}
	if kb.IsSequence() {
		t.Error("modifier key should not be a sequence")
	}
}

func TestParseKeyBinding_Sequence(t *testing.T) {
	kb := ParseKeyBinding("g g")
	if kb.String() != "g g" {
		t.Errorf("expected %q, got %q", "g g", kb.String())
	}
	if !kb.IsSequence() {
		t.Error("'g g' should be a sequence")
	}
	if len(kb.Keys) != 2 {
		t.Errorf("expected 2 keys, got %d", len(kb.Keys))
	}
}

func TestParseKeyBinding_Empty(t *testing.T) {
	kb := ParseKeyBinding("")
	if len(kb.Keys) != 0 {
		t.Errorf("expected 0 keys for empty string, got %d", len(kb.Keys))
	}
}

func TestKeyMap_Resolve_SingleKey(t *testing.T) {
	km := DefaultKeyMap()

	action, pending := km.Resolve("q", "")
	if action != ActionQuit {
		t.Errorf("expected action %q, got %q", ActionQuit, action)
	}
	if pending != "" {
		t.Errorf("expected no pending, got %q", pending)
	}
}

func TestKeyMap_Resolve_SequenceStart(t *testing.T) {
	km := DefaultKeyMap()

	// First 'g' should start a pending sequence.
	action, pending := km.Resolve("g", "")
	if action != "" {
		t.Errorf("expected no action for first g, got %q", action)
	}
	if pending != ActionJumpTop {
		t.Errorf("expected pending=%q, got %q", ActionJumpTop, pending)
	}
}

func TestKeyMap_Resolve_SequenceComplete(t *testing.T) {
	km := DefaultKeyMap()

	// First g → pending.
	_, pending := km.Resolve("g", "")

	// Second g → completes the sequence.
	action, pending2 := km.Resolve("g", pending)
	if action != ActionJumpTop {
		t.Errorf("expected action %q, got %q", ActionJumpTop, action)
	}
	if pending2 != "" {
		t.Errorf("expected pending cleared, got %q", pending2)
	}
}

func TestKeyMap_Resolve_SequenceAborted(t *testing.T) {
	km := DefaultKeyMap()

	// First g → pending.
	_, pending := km.Resolve("g", "")

	// Non-g key → sequence aborted, new key resolves normally.
	action, pending2 := km.Resolve("q", pending)
	if action != ActionQuit {
		t.Errorf("expected action %q after aborting sequence, got %q", ActionQuit, action)
	}
	if pending2 != "" {
		t.Errorf("expected pending cleared after abort, got %q", pending2)
	}
}

func TestKeyMap_Resolve_UnknownKey(t *testing.T) {
	km := DefaultKeyMap()

	action, pending := km.Resolve("z", "")
	if action != "" {
		t.Errorf("expected no action for unknown key, got %q", action)
	}
	if pending != "" {
		t.Errorf("expected no pending for unknown key, got %q", pending)
	}
}

func TestKeyMap_ActionForDisplay(t *testing.T) {
	km := DefaultKeyMap()

	display := km.ActionForDisplay(ActionJumpTop)
	if display != "g g" {
		t.Errorf("expected display %q, got %q", "g g", display)
	}

	display = km.ActionForDisplay(ActionQuit)
	if display != "q" {
		t.Errorf("expected display %q, got %q", "q", display)
	}

	display = km.ActionForDisplay("nonexistent")
	if display != "" {
		t.Errorf("expected empty display for unknown action, got %q", display)
	}
}

func TestHasAlternateArrowKey(t *testing.T) {
	cases := []struct {
		key      string
		expected string
	}{
		{"left", "h"},
		{"right", "l"},
		{"up", "k"},
		{"down", "j"},
		{"h", "left"},
		{"l", "right"},
		{"k", "up"},
		{"j", "down"},
		{"q", ""},
		{"tab", ""},
	}

	for _, tc := range cases {
		got := HasAlternateArrowKey(tc.key)
		if got != tc.expected {
			t.Errorf("HasAlternateArrowKey(%q): expected %q, got %q", tc.key, tc.expected, got)
		}
	}
}

func TestLoadConfig_KeybindingOverrides(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".config", "skinner")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	configPath := filepath.Join(configDir, "config.toml")
	configContent := `[keybindings]
quit = "x"
move_down = "s"
jump_top = "z z"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	t.Setenv("HOME", tmpDir)
	cfg := LoadConfig()

	// Overridden bindings should use new values.
	if cfg.KeyMap.Bindings[ActionQuit].String() != "x" {
		t.Errorf("expected quit binding %q, got %q", "x", cfg.KeyMap.Bindings[ActionQuit].String())
	}
	if cfg.KeyMap.Bindings[ActionMoveDown].String() != "s" {
		t.Errorf("expected move_down binding %q, got %q", "s", cfg.KeyMap.Bindings[ActionMoveDown].String())
	}
	if cfg.KeyMap.Bindings[ActionJumpTop].String() != "z z" {
		t.Errorf("expected jump_top binding %q, got %q", "z z", cfg.KeyMap.Bindings[ActionJumpTop].String())
	}

	// Non-overridden bindings should keep defaults.
	if cfg.KeyMap.Bindings[ActionMoveUp].String() != "k" {
		t.Errorf("expected default move_up binding %q, got %q", "k", cfg.KeyMap.Bindings[ActionMoveUp].String())
	}
	if cfg.KeyMap.Bindings[ActionHelp].String() != "?" {
		t.Errorf("expected default help binding %q, got %q", "?", cfg.KeyMap.Bindings[ActionHelp].String())
	}
}

func TestLoadConfig_UnknownKeybindingIgnored(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".config", "skinner")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	configPath := filepath.Join(configDir, "config.toml")
	configContent := `[keybindings]
nonexistent_action = "x"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	t.Setenv("HOME", tmpDir)
	cfg := LoadConfig()

	// Unknown action should be ignored — no new binding added.
	if _, ok := cfg.KeyMap.Bindings["nonexistent_action"]; ok {
		t.Error("expected unknown action to be ignored, but it was added to keymap")
	}

	// All defaults should still be present.
	for _, action := range AllActions() {
		if _, ok := cfg.KeyMap.Bindings[action]; !ok {
			t.Errorf("expected default action %q to still be present", action)
		}
	}
}

func TestLoadConfig_LineNumbers(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".config", "skinner")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	configPath := filepath.Join(configDir, "config.toml")
	configContent := `[view]
line_numbers = false
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	t.Setenv("HOME", tmpDir)
	cfg := LoadConfig()

	if cfg.LineNumbers {
		t.Error("expected line_numbers=false from config")
	}
}

func TestLoadConfig_LineNumbersDefault(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfg := LoadConfig()
	if !cfg.LineNumbers {
		t.Error("expected line_numbers=true as default")
	}
}

func TestKeyMap_Resolve_RemappedKeys(t *testing.T) {
	km := DefaultKeyMap()
	// Remap quit from "q" to "x"
	km.Bindings[ActionQuit] = ParseKeyBinding("x")

	// "x" should now trigger quit
	action, _ := km.Resolve("x", "")
	if action != ActionQuit {
		t.Errorf("expected action %q for remapped key, got %q", ActionQuit, action)
	}

	// "q" should no longer trigger quit
	action, _ = km.Resolve("q", "")
	if action == ActionQuit {
		t.Error("old key 'q' should not trigger quit after remapping")
	}
}

func TestKeyMap_Resolve_RemappedSequence(t *testing.T) {
	km := DefaultKeyMap()
	// Remap jump_top from "g g" to "z z"
	km.Bindings[ActionJumpTop] = ParseKeyBinding("z z")

	// "z" should start a pending sequence.
	action, pending := km.Resolve("z", "")
	if action != "" {
		t.Errorf("expected no action for first z, got %q", action)
	}
	if pending != ActionJumpTop {
		t.Errorf("expected pending=%q, got %q", ActionJumpTop, pending)
	}

	// Second "z" should complete.
	action, pending = km.Resolve("z", pending)
	if action != ActionJumpTop {
		t.Errorf("expected action %q, got %q", ActionJumpTop, action)
	}
	if pending != "" {
		t.Errorf("expected pending cleared, got %q", pending)
	}

	// "g" should no longer start a sequence for jump_top.
	_, pending = km.Resolve("g", "")
	if pending == ActionJumpTop {
		t.Error("old key 'g' should not start jump_top sequence after remapping")
	}
}
