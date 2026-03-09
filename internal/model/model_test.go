package model

import (
	"testing"
	"time"
)

func TestToolCallGroupStatus(t *testing.T) {
	tests := []struct {
		name     string
		children []*ToolCall
		want     ToolCallStatus
	}{
		{
			name: "all running",
			children: []*ToolCall{
				{Status: ToolCallRunning},
				{Status: ToolCallRunning},
			},
			want: ToolCallRunning,
		},
		{
			name: "all done",
			children: []*ToolCall{
				{Status: ToolCallDone},
				{Status: ToolCallDone},
			},
			want: ToolCallDone,
		},
		{
			name: "mixed running and done returns running",
			children: []*ToolCall{
				{Status: ToolCallDone},
				{Status: ToolCallRunning},
			},
			want: ToolCallRunning,
		},
		{
			name: "has error no running returns error",
			children: []*ToolCall{
				{Status: ToolCallDone},
				{Status: ToolCallError},
			},
			want: ToolCallError,
		},
		{
			name: "has error and running returns running",
			children: []*ToolCall{
				{Status: ToolCallError},
				{Status: ToolCallRunning},
			},
			want: ToolCallRunning,
		},
		{
			name: "all error",
			children: []*ToolCall{
				{Status: ToolCallError},
				{Status: ToolCallError},
			},
			want: ToolCallError,
		},
		{
			name: "single running",
			children: []*ToolCall{
				{Status: ToolCallRunning},
			},
			want: ToolCallRunning,
		},
		{
			name: "single done",
			children: []*ToolCall{
				{Status: ToolCallDone},
			},
			want: ToolCallDone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &ToolCallGroup{Children: tt.children}
			if got := g.Status(); got != tt.want {
				t.Errorf("Status() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToolCallGroupGroupDuration(t *testing.T) {
	base := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		children []*ToolCall
		want     time.Duration
	}{
		{
			name: "running group returns zero",
			children: []*ToolCall{
				{Status: ToolCallRunning, StartTime: base},
				{Status: ToolCallDone, StartTime: base, Duration: time.Second},
			},
			want: 0,
		},
		{
			name: "single completed child",
			children: []*ToolCall{
				{Status: ToolCallDone, StartTime: base, Duration: 2 * time.Second},
			},
			want: 2 * time.Second,
		},
		{
			name: "overlapping children uses wallclock span",
			children: []*ToolCall{
				{Status: ToolCallDone, StartTime: base, Duration: 3 * time.Second},
				{Status: ToolCallDone, StartTime: base.Add(1 * time.Second), Duration: 5 * time.Second},
			},
			// earliest = base, latest end = base+1s+5s = base+6s, span = 6s
			want: 6 * time.Second,
		},
		{
			name: "sequential children",
			children: []*ToolCall{
				{Status: ToolCallDone, StartTime: base, Duration: 2 * time.Second},
				{Status: ToolCallDone, StartTime: base.Add(3 * time.Second), Duration: 1 * time.Second},
			},
			// earliest = base, latest end = base+3s+1s = base+4s, span = 4s
			want: 4 * time.Second,
		},
		{
			name: "all error children still computes duration",
			children: []*ToolCall{
				{Status: ToolCallError, StartTime: base, Duration: 500 * time.Millisecond},
				{Status: ToolCallError, StartTime: base.Add(1 * time.Second), Duration: 500 * time.Millisecond},
			},
			want: 1500 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &ToolCallGroup{Children: tt.children}
			if got := g.GroupDuration(); got != tt.want {
				t.Errorf("GroupDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToolCallGroupCompletedCount(t *testing.T) {
	tests := []struct {
		name     string
		children []*ToolCall
		want     int
	}{
		{
			name:     "empty group",
			children: []*ToolCall{},
			want:     0,
		},
		{
			name: "all running",
			children: []*ToolCall{
				{Status: ToolCallRunning},
				{Status: ToolCallRunning},
			},
			want: 0,
		},
		{
			name: "mixed statuses counts non-running",
			children: []*ToolCall{
				{Status: ToolCallDone},
				{Status: ToolCallRunning},
				{Status: ToolCallError},
			},
			want: 2,
		},
		{
			name: "all done",
			children: []*ToolCall{
				{Status: ToolCallDone},
				{Status: ToolCallDone},
				{Status: ToolCallDone},
			},
			want: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &ToolCallGroup{Children: tt.children}
			if got := g.CompletedCount(); got != tt.want {
				t.Errorf("CompletedCount() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestToolCallGroupToolCallCount(t *testing.T) {
	tests := []struct {
		name     string
		children []*ToolCall
		want     int
	}{
		{
			name:     "empty",
			children: []*ToolCall{},
			want:     0,
		},
		{
			name: "three children",
			children: []*ToolCall{
				{Name: "Read"},
				{Name: "Read"},
				{Name: "Read"},
			},
			want: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &ToolCallGroup{Children: tt.children}
			if got := g.ToolCallCount(); got != tt.want {
				t.Errorf("ToolCallCount() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestSessionPhaseConstants(t *testing.T) {
	// Verify phase constants have distinct values
	phases := []SessionPhase{PhaseIdle, PhaseRunning, PhaseFinished}
	seen := make(map[SessionPhase]bool)
	for _, p := range phases {
		if seen[p] {
			t.Errorf("duplicate phase value: %d", p)
		}
		seen[p] = true
	}
}

func TestRunStruct(t *testing.T) {
	run := Run{
		PromptName:    "BUILD",
		PromptFile:    "PROMPT_BUILD.md",
		StartIndex:    5,
		MaxIterations: 10,
	}
	if run.PromptName != "BUILD" {
		t.Errorf("expected PromptName BUILD, got %s", run.PromptName)
	}
	if run.PromptFile != "PROMPT_BUILD.md" {
		t.Errorf("expected PromptFile PROMPT_BUILD.md, got %s", run.PromptFile)
	}
	if run.StartIndex != 5 {
		t.Errorf("expected StartIndex 5, got %d", run.StartIndex)
	}
	if run.MaxIterations != 10 {
		t.Errorf("expected MaxIterations 10, got %d", run.MaxIterations)
	}
}

func TestSessionRunsAndPhase(t *testing.T) {
	sess := Session{}

	// Default phase is Idle (zero value)
	if sess.Phase != PhaseIdle {
		t.Errorf("expected default phase PhaseIdle, got %d", sess.Phase)
	}

	// Add runs
	sess.Runs = append(sess.Runs, Run{
		PromptName: "BUILD",
		PromptFile: "PROMPT_BUILD.md",
		StartIndex: 0,
	})
	if len(sess.Runs) != 1 {
		t.Errorf("expected 1 run, got %d", len(sess.Runs))
	}

	// Phase transitions
	sess.Phase = PhaseRunning
	if sess.Phase != PhaseRunning {
		t.Errorf("expected PhaseRunning, got %d", sess.Phase)
	}

	sess.Phase = PhaseFinished
	if sess.Phase != PhaseFinished {
		t.Errorf("expected PhaseFinished, got %d", sess.Phase)
	}
}

func TestIterationToolCallCount(t *testing.T) {
	tests := []struct {
		name  string
		items []TimelineItem
		want  int
	}{
		{
			name:  "empty iteration",
			items: nil,
			want:  0,
		},
		{
			name: "standalone tool calls only",
			items: []TimelineItem{
				&ToolCall{Name: "Read"},
				&ToolCall{Name: "Edit"},
			},
			want: 2,
		},
		{
			name: "group counts children",
			items: []TimelineItem{
				&ToolCallGroup{Children: []*ToolCall{
					{Name: "Read"},
					{Name: "Read"},
					{Name: "Read"},
				}},
			},
			want: 3,
		},
		{
			name: "mixed standalone and groups",
			items: []TimelineItem{
				&ToolCall{Name: "Bash"},
				&ToolCallGroup{Children: []*ToolCall{
					{Name: "Read"},
					{Name: "Read"},
				}},
				&ToolCall{Name: "Edit"},
				&ToolCallGroup{Children: []*ToolCall{
					{Name: "Write"},
				}},
			},
			want: 5,
		},
		{
			name: "text blocks are not counted",
			items: []TimelineItem{
				&TextBlock{Text: "some text"},
				&ToolCall{Name: "Bash"},
				&TextBlock{Text: "more text"},
			},
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iter := &Iteration{Items: tt.items}
			if got := iter.ToolCallCount(); got != tt.want {
				t.Errorf("ToolCallCount() = %d, want %d", got, tt.want)
			}
		})
	}
}
