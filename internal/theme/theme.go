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

	DiffAdded           string
	DiffRemoved         string
	DiffAddedBg         string
	DiffRemovedBg       string
	DiffAddedEmphasis   string
	DiffRemovedEmphasis string
	DiffLineNumber      string
	DiffSessionCommit   string
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

		DiffAdded:           "#859900",
		DiffRemoved:         "#dc322f",
		DiffAddedBg:         "#1a3a1a",
		DiffRemovedBg:       "#3a1a1a",
		DiffAddedEmphasis:   "#2d5a2d",
		DiffRemovedEmphasis: "#5a2d2d",
		DiffLineNumber:      "#586e75",
		DiffSessionCommit:   "#268bd2",
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

		DiffAdded:           "#859900",
		DiffRemoved:         "#dc322f",
		DiffAddedBg:         "#e6f2e6",
		DiffRemovedBg:       "#f2e6e6",
		DiffAddedEmphasis:   "#c8e6c8",
		DiffRemovedEmphasis: "#e6c8c8",
		DiffLineNumber:      "#93a1a1",
		DiffSessionCommit:   "#268bd2",
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

		DiffAdded:           "#a6e22e",
		DiffRemoved:         "#f92672",
		DiffAddedBg:         "#2a3a1a",
		DiffRemovedBg:       "#3a1a2a",
		DiffAddedEmphasis:   "#3d5a2d",
		DiffRemovedEmphasis: "#5a2d3d",
		DiffLineNumber:      "#75715e",
		DiffSessionCommit:   "#66d9ef",
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

		DiffAdded:           "#a3be8c",
		DiffRemoved:         "#bf616a",
		DiffAddedBg:         "#2e3440",
		DiffRemovedBg:       "#3b2c2f",
		DiffAddedEmphasis:   "#3a4a3a",
		DiffRemovedEmphasis: "#4a3a3a",
		DiffLineNumber:      "#4c566a",
		DiffSessionCommit:   "#88c0d0",
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
