package config

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Action names for configurable keybindings.
const (
	ActionQuit              = "quit"
	ActionHelp              = "help"
	ActionToggleLeftPane    = "toggle_left_pane"
	ActionToggleLineNumbers = "toggle_line_numbers"
	ActionToggleView        = "toggle_view"
	ActionFocusLeft         = "focus_left"
	ActionFocusRight        = "focus_right"
	ActionFocusToggle       = "focus_toggle"
	ActionMoveDown          = "move_down"
	ActionMoveUp            = "move_up"
	ActionJumpTop           = "jump_top"
	ActionJumpBottom        = "jump_bottom"
	ActionExpand            = "expand"
	ActionEscape            = "escape"
)

// KeyBinding represents a single key or key sequence that triggers an action.
type KeyBinding struct {
	// Keys is the sequence of key strings that trigger this binding.
	// Single keys: ["q"], modifier keys: ["ctrl+c"], sequences: ["g", "g"]
	Keys []string
}

// IsSequence returns true if this binding requires multiple key presses.
func (kb KeyBinding) IsSequence() bool {
	return len(kb.Keys) > 1
}

// String returns the key binding as a display string.
func (kb KeyBinding) String() string {
	return strings.Join(kb.Keys, " ")
}

// KeyMap maps action names to their key bindings.
type KeyMap struct {
	Bindings map[string]KeyBinding
}

// DefaultKeyMap returns the default keybindings per the spec.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Bindings: map[string]KeyBinding{
			ActionQuit:              {Keys: []string{"q"}},
			ActionHelp:              {Keys: []string{"?"}},
			ActionToggleLeftPane:    {Keys: []string{"["}},
			ActionToggleLineNumbers: {Keys: []string{"#"}},
			ActionToggleView:        {Keys: []string{"v"}},
			ActionFocusLeft:         {Keys: []string{"h"}},
			ActionFocusRight:        {Keys: []string{"l"}},
			ActionFocusToggle:       {Keys: []string{"tab"}},
			ActionMoveDown:          {Keys: []string{"j"}},
			ActionMoveUp:            {Keys: []string{"k"}},
			ActionJumpTop:           {Keys: []string{"g", "g"}},
			ActionJumpBottom:        {Keys: []string{"G"}},
			ActionExpand:            {Keys: []string{"enter"}},
			ActionEscape:            {Keys: []string{"escape"}},
		},
	}
}

// ParseKeyBinding parses a key binding string like "q", "ctrl+c", or "g g"
// into a KeyBinding.
func ParseKeyBinding(s string) KeyBinding {
	s = strings.TrimSpace(s)
	if s == "" {
		return KeyBinding{}
	}
	parts := strings.Split(s, " ")
	var keys []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			keys = append(keys, p)
		}
	}
	return KeyBinding{Keys: keys}
}

// MatchesKey checks if the given tea.KeyMsg string matches the first (or only)
// key in a single-key binding.
func (km *KeyMap) MatchesKey(action string, keyStr string) bool {
	binding, ok := km.Bindings[action]
	if !ok {
		return false
	}
	if binding.IsSequence() {
		// For sequences, this only matches if keyStr matches the first key.
		// The caller is responsible for tracking sequence state.
		return false
	}
	return binding.Keys[0] == keyStr
}

// SequenceFirstKey returns the first key of a sequence binding, or empty string
// if the action is not a sequence.
func (km *KeyMap) SequenceFirstKey(action string) string {
	binding, ok := km.Bindings[action]
	if !ok || !binding.IsSequence() {
		return ""
	}
	return binding.Keys[0]
}

// SequenceSecondKey returns the second key of a sequence binding, or empty string
// if the action is not a sequence or has fewer than 2 keys.
func (km *KeyMap) SequenceSecondKey(action string) string {
	binding, ok := km.Bindings[action]
	if !ok || len(binding.Keys) < 2 {
		return ""
	}
	return binding.Keys[1]
}

// Resolve takes a tea.KeyMsg and returns the action it matches, accounting for
// the current pending sequence state. It returns the action name and whether
// the pending state should be set/cleared.
//
// Returns:
//   - action: the matched action name, or "" if no match
//   - pendingAction: the action waiting for a second key, or "" to clear
func (km *KeyMap) Resolve(keyStr string, pendingAction string) (action string, newPending string) {
	// If we have a pending sequence, check if the new key completes it.
	if pendingAction != "" {
		binding, ok := km.Bindings[pendingAction]
		if ok && binding.IsSequence() && len(binding.Keys) >= 2 && keyStr == binding.Keys[1] {
			return pendingAction, ""
		}
		// Pending didn't match — clear it and fall through to normal matching.
	}

	// Check for sequence starters first.
	for actionName, binding := range km.Bindings {
		if binding.IsSequence() && binding.Keys[0] == keyStr {
			return "", actionName
		}
	}

	// Check single-key bindings.
	for actionName, binding := range km.Bindings {
		if !binding.IsSequence() && len(binding.Keys) > 0 && binding.Keys[0] == keyStr {
			return actionName, ""
		}
	}

	return "", ""
}

// ActionForDisplay returns the display string for an action's key binding.
func (km *KeyMap) ActionForDisplay(action string) string {
	binding, ok := km.Bindings[action]
	if !ok {
		return ""
	}
	return binding.String()
}

// AllActions returns all action names in display order.
func AllActions() []string {
	return []string{
		ActionMoveDown,
		ActionMoveUp,
		ActionJumpTop,
		ActionJumpBottom,
		ActionFocusToggle,
		ActionFocusLeft,
		ActionFocusRight,
		ActionExpand,
		ActionEscape,
		ActionToggleView,
		ActionToggleLineNumbers,
		ActionToggleLeftPane,
		ActionQuit,
		ActionHelp,
	}
}

// KeyMsgString converts a tea.KeyMsg to a normalized string for matching.
// This handles the Bubble Tea key representation consistently.
func KeyMsgString(msg tea.KeyMsg) string {
	return msg.String()
}

// HasAlternateArrowKey returns true if the given key string has an alternate
// arrow key that should also trigger the same action. Arrow keys are always
// active alongside their letter equivalents per the spec.
func HasAlternateArrowKey(keyStr string) string {
	switch keyStr {
	case "left":
		return "h"
	case "right":
		return "l"
	case "up":
		return "k"
	case "down":
		return "j"
	case "h":
		return "left"
	case "l":
		return "right"
	case "k":
		return "up"
	case "j":
		return "down"
	default:
		return ""
	}
}
