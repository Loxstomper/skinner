package main

import (
	"testing"

	"github.com/loxstomper/skinner/internal/config"
)

func defaultCfg() config.Config {
	return config.DefaultConfig()
}

func TestParseArgs_NoArgs_IdleMode(t *testing.T) {
	mode, promptFile, maxIter, _, exitOnComplete, err := parseArgsFromSlice([]string{}, defaultCfg())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode != "idle" {
		t.Errorf("expected mode %q, got %q", "idle", mode)
	}
	if promptFile != "" {
		t.Errorf("expected empty promptFile, got %q", promptFile)
	}
	if maxIter != 0 {
		t.Errorf("expected maxIterations 0, got %d", maxIter)
	}
	if exitOnComplete {
		t.Error("expected exitOnComplete false")
	}
}

func TestParseArgs_BuildMode(t *testing.T) {
	mode, promptFile, maxIter, _, _, err := parseArgsFromSlice([]string{"build"}, defaultCfg())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode != "build" {
		t.Errorf("expected mode %q, got %q", "build", mode)
	}
	if promptFile != "PROMPT_BUILD.md" {
		t.Errorf("expected promptFile %q, got %q", "PROMPT_BUILD.md", promptFile)
	}
	if maxIter != 0 {
		t.Errorf("expected maxIterations 0 (unlimited), got %d", maxIter)
	}
}

func TestParseArgs_PlanModeWithCount(t *testing.T) {
	mode, promptFile, maxIter, _, _, err := parseArgsFromSlice([]string{"plan", "5"}, defaultCfg())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode != "plan" {
		t.Errorf("expected mode %q, got %q", "plan", mode)
	}
	if promptFile != "PROMPT_PLAN.md" {
		t.Errorf("expected promptFile %q, got %q", "PROMPT_PLAN.md", promptFile)
	}
	if maxIter != 5 {
		t.Errorf("expected maxIterations 5, got %d", maxIter)
	}
}

func TestParseArgs_BuildWithCount(t *testing.T) {
	mode, _, maxIter, _, _, err := parseArgsFromSlice([]string{"build", "20"}, defaultCfg())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode != "build" {
		t.Errorf("expected mode %q, got %q", "build", mode)
	}
	if maxIter != 20 {
		t.Errorf("expected maxIterations 20, got %d", maxIter)
	}
}

func TestParseArgs_ExitWithModeAndCount(t *testing.T) {
	_, _, maxIter, _, exitOnComplete, err := parseArgsFromSlice([]string{"--exit", "build", "10"}, defaultCfg())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exitOnComplete {
		t.Error("expected exitOnComplete true")
	}
	if maxIter != 10 {
		t.Errorf("expected maxIterations 10, got %d", maxIter)
	}
}

func TestParseArgs_ExitWithoutMode_Error(t *testing.T) {
	_, _, _, _, _, err := parseArgsFromSlice([]string{"--exit"}, defaultCfg())
	if err == nil {
		t.Fatal("expected error for --exit without mode")
	}
}

func TestParseArgs_ExitWithModeNoCount_Error(t *testing.T) {
	_, _, _, _, _, err := parseArgsFromSlice([]string{"--exit", "build"}, defaultCfg())
	if err == nil {
		t.Fatal("expected error for --exit without count")
	}
}

func TestParseArgs_ExitWithCountNoMode_Error(t *testing.T) {
	_, _, _, _, _, err := parseArgsFromSlice([]string{"--exit", "10"}, defaultCfg())
	if err == nil {
		t.Fatal("expected error for --exit without mode")
	}
}

func TestParseArgs_ThemeFlag(t *testing.T) {
	_, _, _, th, _, err := parseArgsFromSlice([]string{"--theme=monokai"}, defaultCfg())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Monokai foreground is #f8f8f2
	if th.Foreground != "#f8f8f2" {
		t.Errorf("expected monokai foreground %q, got %q", "#f8f8f2", th.Foreground)
	}
}

func TestParseArgs_UnknownTheme_Error(t *testing.T) {
	_, _, _, _, _, err := parseArgsFromSlice([]string{"--theme=nonexistent"}, defaultCfg())
	if err == nil {
		t.Fatal("expected error for unknown theme")
	}
}

func TestParseArgs_DefaultTheme(t *testing.T) {
	_, _, _, th, _, err := parseArgsFromSlice([]string{}, defaultCfg())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Default solarized-dark foreground is #839496
	if th.Foreground != "#839496" {
		t.Errorf("expected solarized-dark foreground %q, got %q", "#839496", th.Foreground)
	}
}

func TestParseArgs_AllFlagsCombined(t *testing.T) {
	mode, promptFile, maxIter, th, exitOnComplete, err := parseArgsFromSlice(
		[]string{"--theme=monokai", "--exit", "plan", "15"}, defaultCfg(),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode != "plan" {
		t.Errorf("expected mode %q, got %q", "plan", mode)
	}
	if promptFile != "PROMPT_PLAN.md" {
		t.Errorf("expected promptFile %q, got %q", "PROMPT_PLAN.md", promptFile)
	}
	if maxIter != 15 {
		t.Errorf("expected maxIterations 15, got %d", maxIter)
	}
	if th.Foreground != "#f8f8f2" {
		t.Errorf("expected monokai foreground %q, got %q", "#f8f8f2", th.Foreground)
	}
	if !exitOnComplete {
		t.Error("expected exitOnComplete true")
	}
}
