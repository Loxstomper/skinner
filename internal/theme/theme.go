package theme

import "sort"

// Theme defines semantic color roles for the TUI.
// All values are hex color strings (e.g. "#839496").
type Theme struct {
	Foreground    string
	ForegroundDim string
	Highlight     string

	StatusRunning string
	StatusSuccess string
	StatusError   string

	ToolNameRunning string
	ToolNameSuccess string
	ToolNameError   string
	ToolSummary     string

	DurationRunning string
	DurationSuccess string
	DurationError   string

	IterRunning string
	IterSuccess string
	IterError   string

	TextBlock string
}

var themes = map[string]Theme{
	"solarized-dark": {
		Foreground:      "#839496",
		ForegroundDim:   "#586e75",
		Highlight:       "#073642",
		StatusRunning:   "#b58900",
		StatusSuccess:   "#859900",
		StatusError:     "#dc322f",
		ToolNameRunning: "#b58900",
		ToolNameSuccess: "#859900",
		ToolNameError:   "#dc322f",
		ToolSummary:     "#586e75",
		DurationRunning: "#b58900",
		DurationSuccess: "#2aa198",
		DurationError:   "#dc322f",
		IterRunning:     "#b58900",
		IterSuccess:     "#839496",
		IterError:       "#dc322f",
		TextBlock:       "#839496",
	},
	"solarized-light": {
		Foreground:      "#657b83",
		ForegroundDim:   "#93a1a1",
		Highlight:       "#eee8d5",
		StatusRunning:   "#b58900",
		StatusSuccess:   "#859900",
		StatusError:     "#dc322f",
		ToolNameRunning: "#b58900",
		ToolNameSuccess: "#859900",
		ToolNameError:   "#dc322f",
		ToolSummary:     "#93a1a1",
		DurationRunning: "#b58900",
		DurationSuccess: "#2aa198",
		DurationError:   "#dc322f",
		IterRunning:     "#b58900",
		IterSuccess:     "#657b83",
		IterError:       "#dc322f",
		TextBlock:       "#657b83",
	},
	"monokai": {
		Foreground:      "#f8f8f2",
		ForegroundDim:   "#75715e",
		Highlight:       "#49483e",
		StatusRunning:   "#e6db74",
		StatusSuccess:   "#a6e22e",
		StatusError:     "#f92672",
		ToolNameRunning: "#e6db74",
		ToolNameSuccess: "#a6e22e",
		ToolNameError:   "#f92672",
		ToolSummary:     "#75715e",
		DurationRunning: "#e6db74",
		DurationSuccess: "#66d9ef",
		DurationError:   "#f92672",
		IterRunning:     "#e6db74",
		IterSuccess:     "#f8f8f2",
		IterError:       "#f92672",
		TextBlock:       "#f8f8f2",
	},
	"nord": {
		Foreground:      "#d8dee9",
		ForegroundDim:   "#4c566a",
		Highlight:       "#3b4252",
		StatusRunning:   "#ebcb8b",
		StatusSuccess:   "#a3be8c",
		StatusError:     "#bf616a",
		ToolNameRunning: "#ebcb8b",
		ToolNameSuccess: "#a3be8c",
		ToolNameError:   "#bf616a",
		ToolSummary:     "#4c566a",
		DurationRunning: "#ebcb8b",
		DurationSuccess: "#88c0d0",
		DurationError:   "#bf616a",
		IterRunning:     "#ebcb8b",
		IterSuccess:     "#d8dee9",
		IterError:       "#bf616a",
		TextBlock:       "#d8dee9",
	},
}

func LookupTheme(name string) (Theme, bool) {
	t, ok := themes[name]
	return t, ok
}

func ThemeNames() []string {
	names := make([]string, 0, len(themes))
	for name := range themes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
