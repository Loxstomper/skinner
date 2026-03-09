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

	var promptContent string
	if mode != "idle" {
		content, err := os.ReadFile(promptFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading prompt file %s: %v\n", promptFile, err)
			os.Exit(1)
		}
		promptContent = string(content)
	}

	session := model.Session{
		Mode:          mode,
		PromptFile:    promptFile,
		MaxIterations: maxIterations,
		StartTime:     time.Now(),
	}

	compactView := cfg.ViewMode == "compact"
	exec := &executor.ClaudeExecutor{}
	m := tui.NewModel(session, cfg, promptContent, th, compactView, exitOnComplete, exec)
	p := tea.NewProgram(&m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func parseArgs(cfg config.Config) (mode string, promptFile string, maxIterations int, th theme.Theme, exitOnComplete bool) {
	mode, promptFile, maxIterations, th, exitOnComplete, err := parseArgsFromSlice(os.Args[1:], cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	return
}

func parseArgsFromSlice(args []string, cfg config.Config) (mode string, promptFile string, maxIterations int, th theme.Theme, exitOnComplete bool, err error) {
	mode = "idle"
	promptFile = ""
	maxIterations = 0
	themeName := cfg.ThemeName

	for _, arg := range args {
		if strings.HasPrefix(arg, "--theme=") {
			themeName = strings.TrimPrefix(arg, "--theme=")
			continue
		}
		if arg == "--exit" {
			exitOnComplete = true
			continue
		}
		if arg == "build" {
			mode = "build"
			promptFile = "PROMPT_BUILD.md"
		} else if arg == "plan" {
			mode = "plan"
			promptFile = "PROMPT_PLAN.md"
		} else if n, parseErr := strconv.Atoi(arg); parseErr == nil && n > 0 {
			maxIterations = n
		}
	}

	// --exit requires both a prompt mode and iteration count
	if exitOnComplete && (mode == "idle" || maxIterations == 0) {
		err = fmt.Errorf("--exit requires a prompt mode and iteration count\nUsage: skinner [--theme=<name>] [--exit] <plan|build> <max_iterations>")
		return
	}

	var ok bool
	th, ok = theme.LookupTheme(themeName)
	if !ok {
		var buf strings.Builder
		fmt.Fprintf(&buf, "Unknown theme %q. Available themes:\n", themeName)
		for _, name := range theme.ThemeNames() {
			suffix := ""
			if name == "solarized-dark" {
				suffix = " (default)"
			}
			fmt.Fprintf(&buf, "  %s%s\n", name, suffix)
		}
		err = fmt.Errorf("%s", buf.String())
		return
	}

	return
}
