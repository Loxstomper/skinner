package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type ModelPricing struct {
	Input       float64
	Output      float64
	CacheRead   float64
	CacheCreate float64
}

type Config struct {
	ViewMode  string // "full" or "compact"
	ThemeName string
	Pricing   map[string]ModelPricing
}

func DefaultConfig() Config {
	return Config{
		ViewMode:  "full",
		ThemeName: "solarized-dark",
		Pricing:   DefaultPricing(),
	}
}

func DefaultPricing() map[string]ModelPricing {
	return map[string]ModelPricing{
		"claude-opus-4-6": {
			Input:       0.000005,
			Output:      0.000025,
			CacheRead:   0.0000005,
			CacheCreate: 0.00000625,
		},
		"claude-sonnet-4-5": {
			Input:       0.000003,
			Output:      0.000015,
			CacheRead:   0.0000003,
			CacheCreate: 0.00000375,
		},
		"claude-haiku-4-5": {
			Input:       0.000001,
			Output:      0.000005,
			CacheRead:   0.0000001,
			CacheCreate: 0.00000125,
		},
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
	defer f.Close()

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
		case section == "theme":
			if key == "name" && value != "" {
				cfg.ThemeName = value
			}
		case strings.HasPrefix(section, "pricing."):
			modelName := strings.TrimPrefix(section, "pricing.")
			mp := cfg.Pricing[modelName]
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
			cfg.Pricing[modelName] = mp
		}
	}

	return cfg
}
