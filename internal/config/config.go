package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type ModelPricing struct {
	Input         float64
	Output        float64
	CacheRead     float64
	CacheCreate   float64
	ContextWindow int
}

type HooksTimeoutConfig struct {
	Default        int  // default timeout in seconds for all hooks
	PreIteration   *int // override for pre-iteration hook
	OnIterationEnd *int // override for on-iteration-end hook
	OnError        *int // override for on-error hook
	OnIdle         *int // override for on-idle hook
}

type HooksConfig struct {
	PreIteration   string // shell command for pre-iteration hook
	OnIterationEnd string // shell command for on-iteration-end hook
	OnError        string // shell command for on-error hook
	OnIdle         string // shell command for on-idle hook
	Timeout        HooksTimeoutConfig
}

type Config struct {
	ViewMode    string // "full" or "compact"
	Layout      string // "side", "bottom", "auto"
	LineNumbers bool   // show relative line numbers in right pane
	ThemeName   string
	KeyMap      KeyMap
	Pricing     map[string]ModelPricing
	PlanCommand string // shell command for plan mode (run via sh -c)
	Hooks       HooksConfig
}

func DefaultConfig() Config {
	return Config{
		ViewMode:    "full",
		Layout:      "auto",
		LineNumbers: true,
		ThemeName:   "solarized-dark",
		KeyMap:      DefaultKeyMap(),
		Pricing:     DefaultPricing(),
		PlanCommand: `claude "study specs/README.md"`,
		Hooks: HooksConfig{
			Timeout: HooksTimeoutConfig{
				Default: 10,
			},
		},
	}
}

func DefaultPricing() map[string]ModelPricing {
	return map[string]ModelPricing{
		"claude-opus-4-6": {
			Input:         0.000005,
			Output:        0.000025,
			CacheRead:     0.0000005,
			CacheCreate:   0.00000625,
			ContextWindow: 200000,
		},
		"claude-sonnet-4-5": {
			Input:         0.000003,
			Output:        0.000015,
			CacheRead:     0.0000003,
			CacheCreate:   0.00000375,
			ContextWindow: 200000,
		},
		"claude-haiku-4-5": {
			Input:         0.000001,
			Output:        0.000005,
			CacheRead:     0.0000001,
			CacheCreate:   0.00000125,
			ContextWindow: 200000,
		},
	}
}

// parseDuration converts a duration string like "30s" or "2m" to seconds.
// Returns the value and true on success, or 0 and false on failure.
func parseDuration(s string) (int, bool) {
	if len(s) < 2 {
		return 0, false
	}
	unit := s[len(s)-1]
	num, err := strconv.Atoi(s[:len(s)-1])
	if err != nil || num < 0 {
		return 0, false
	}
	switch unit {
	case 's':
		return num, true
	case 'm':
		return num * 60, true
	default:
		return 0, false
	}
}

// LoadConfig reads ~/.config/skinner/config.toml and returns a Config
// with defaults for any missing values. If the file does not exist or
// cannot be read, defaults are returned with no error.
func LoadConfig() Config {
	cfg := DefaultConfig()

	home, err := os.UserHomeDir()
	if err != nil {
		return cfg
	}

	path := filepath.Join(home, ".config", "skinner", "config.toml")
	f, err := os.Open(path)
	if err != nil {
		return cfg
	}
	defer func() { _ = f.Close() }()

	section := ""
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Section header
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = line[1 : len(line)-1]
			continue
		}

		// Key = value
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		value = strings.Trim(value, `"'`)

		switch {
		case section == "view":
			if key == "mode" && (value == "full" || value == "compact") {
				cfg.ViewMode = value
			}
			if key == "layout" && (value == "side" || value == "bottom" || value == "auto") {
				cfg.Layout = value
			}
			if key == "line_numbers" {
				cfg.LineNumbers = value == "true"
			}
		case section == "theme":
			if key == "name" && value != "" {
				cfg.ThemeName = value
			}
		case section == "keybindings":
			// Validate that the action name is known before overriding.
			if _, ok := cfg.KeyMap.Bindings[key]; ok {
				cfg.KeyMap.Bindings[key] = ParseKeyBinding(value)
			}
		case section == "plan":
			if key == "command" && value != "" {
				cfg.PlanCommand = value
			}
		case section == "hooks":
			switch key {
			case "pre-iteration":
				cfg.Hooks.PreIteration = value
			case "on-iteration-end":
				cfg.Hooks.OnIterationEnd = value
			case "on-error":
				cfg.Hooks.OnError = value
			case "on-idle":
				cfg.Hooks.OnIdle = value
			}
		case section == "hooks.timeout":
			if secs, ok := parseDuration(value); ok {
				v := secs
				switch key {
				case "pre-iteration":
					cfg.Hooks.Timeout.PreIteration = &v
				case "on-iteration-end":
					cfg.Hooks.Timeout.OnIterationEnd = &v
				case "on-error":
					cfg.Hooks.Timeout.OnError = &v
				case "on-idle":
					cfg.Hooks.Timeout.OnIdle = &v
				}
			}
		case strings.HasPrefix(section, "pricing."):
			modelName := strings.TrimPrefix(section, "pricing.")
			mp := cfg.Pricing[modelName]
			switch key {
			case "context_window":
				if v, err := strconv.Atoi(value); err == nil {
					mp.ContextWindow = v
				}
			default:
				if v, err := strconv.ParseFloat(value, 64); err == nil {
					switch key {
					case "input":
						mp.Input = v
					case "output":
						mp.Output = v
					case "cache_read":
						mp.CacheRead = v
					case "cache_create":
						mp.CacheCreate = v
					}
				}
			}
			cfg.Pricing[modelName] = mp
		}
	}

	return cfg
}
