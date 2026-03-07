package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/loxstomper/skinner/internal/config"
	"github.com/loxstomper/skinner/internal/executor"
	"github.com/loxstomper/skinner/internal/model"
	"github.com/loxstomper/skinner/internal/theme"
	"github.com/loxstomper/skinner/internal/tui"
)

func main() {
	cfg := config.LoadConfig()
	mode, promptFile, maxIterations, th, exitOnComplete := parseArgs(cfg)

	content, err := os.ReadFile(promptFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading prompt file %s: %v\n", promptFile, err)
		os.Exit(1)
	}

	session := model.Session{
		Mode:          mode,
		PromptFile:    promptFile,
		MaxIterations: maxIterations,
		StartTime:     time.Now(),
	}

	compactView := cfg.ViewMode == "compact"
	exec := &executor.ClaudeExecutor{}
	m := tui.NewModel(session, cfg, string(content), th, compactView, exitOnComplete, exec)
	p := tea.NewProgram(&m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func parseArgs(cfg config.Config) (mode string, promptFile string, maxIterations int, th theme.Theme, exitOnComplete bool) {
	mode = "build"
	promptFile = "PROMPT_BUILD.md"
	maxIterations = 0
	themeName := cfg.ThemeName

	args := os.Args[1:]

	for _, arg := range args {
		if strings.HasPrefix(arg, "--theme=") {
			themeName = strings.TrimPrefix(arg, "--theme=")
			continue
		}
		if arg == "--exit" {
			exitOnComplete = true
			continue
		}
		if arg == "plan" {
			mode = "plan"
			promptFile = "PROMPT_PLAN.md"
		} else if n, err := strconv.Atoi(arg); err == nil && n > 0 {
			maxIterations = n
		}
	}

	var ok bool
	th, ok = theme.LookupTheme(themeName)
	if !ok {
		fmt.Fprintf(os.Stderr, "Unknown theme %q. Available themes:\n", themeName)
		for _, name := range theme.ThemeNames() {
			suffix := ""
			if name == "solarized-dark" {
				suffix = " (default)"
			}
			fmt.Fprintf(os.Stderr, "  %s%s\n", name, suffix)
		}
		os.Exit(1)
	}

	return
}
