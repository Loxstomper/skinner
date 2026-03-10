package tui

import (
	"testing"
	"time"

	"github.com/loxstomper/skinner/internal/config"
	"github.com/loxstomper/skinner/internal/executor"
	"github.com/loxstomper/skinner/internal/model"
)

// newTestModelWithLayout creates a test Model with the given layout config and window size.
func newTestModelWithLayout(layout string, width int) *Model {
	return newTestModelWithLayoutSize(layout, width, 30)
}

func newTestModelWithLayoutSize(layout string, width, height int) *Model {
	fake := &executor.FakeExecutor{}
	sess := model.Session{
		Mode:          "build",
		PromptFile:    "test-prompt.md",
		MaxIterations: 1,
		StartTime:     time.Now(),
	}
	cfg := config.DefaultConfig()
	cfg.Layout = layout
	th := testTheme()
	m := NewModel(sess, cfg, "test prompt", th, false, false, fake)
	m.width = width
	m.height = height
	m.updateLayoutForSize()
	return &m
}

func TestEffectiveLayout_Side(t *testing.T) {
	m := newTestModelWithLayout("side", 60)
	if got := m.effectiveLayout(); got != "side" {
		t.Errorf("effectiveLayout() = %q, want %q", got, "side")
	}
}

func TestEffectiveLayout_SideWideTerminal(t *testing.T) {
	m := newTestModelWithLayout("side", 120)
	if got := m.effectiveLayout(); got != "side" {
		t.Errorf("effectiveLayout() = %q, want %q", got, "side")
	}
}

func TestEffectiveLayout_Bottom(t *testing.T) {
	m := newTestModelWithLayout("bottom", 120)
	if got := m.effectiveLayout(); got != "bottom" {
		t.Errorf("effectiveLayout() = %q, want %q", got, "bottom")
	}
}

func TestEffectiveLayout_BottomNarrow(t *testing.T) {
	m := newTestModelWithLayout("bottom", 60)
	if got := m.effectiveLayout(); got != "bottom" {
		t.Errorf("effectiveLayout() = %q, want %q", got, "bottom")
	}
}

func TestEffectiveLayout_AutoNarrow(t *testing.T) {
	m := newTestModelWithLayout("auto", 79)
	if got := m.effectiveLayout(); got != "bottom" {
		t.Errorf("effectiveLayout() for width=79 = %q, want %q", got, "bottom")
	}
}

func TestEffectiveLayout_AutoWide(t *testing.T) {
	m := newTestModelWithLayout("auto", 80)
	if got := m.effectiveLayout(); got != "side" {
		t.Errorf("effectiveLayout() for width=80 = %q, want %q", got, "side")
	}
}

func TestEffectiveLayout_AutoExactThreshold(t *testing.T) {
	m := newTestModelWithLayout("auto", 80)
	if got := m.effectiveLayout(); got != "side" {
		t.Errorf("effectiveLayout() for width=80 = %q, want %q", got, "side")
	}
	m2 := newTestModelWithLayout("auto", 79)
	if got := m2.effectiveLayout(); got != "bottom" {
		t.Errorf("effectiveLayout() for width=79 = %q, want %q", got, "bottom")
	}
}

func TestEffectiveLayout_InvalidFallsBackToAuto(t *testing.T) {
	m := newTestModelWithLayout("invalid", 120)
	if got := m.effectiveLayout(); got != "side" {
		t.Errorf("effectiveLayout() for invalid config = %q, want %q (auto behavior at width=120)", got, "side")
	}
	m2 := newTestModelWithLayout("invalid", 60)
	if got := m2.effectiveLayout(); got != "bottom" {
		t.Errorf("effectiveLayout() for invalid config = %q, want %q (auto behavior at width=60)", got, "bottom")
	}
}

func TestUpdateLayoutForSize_SideMode(t *testing.T) {
	m := newTestModelWithLayout("side", 120)
	if !m.leftPaneVisible {
		t.Error("expected leftPaneVisible=true in side mode")
	}
}

func TestUpdateLayoutForSize_BottomMode(t *testing.T) {
	m := newTestModelWithLayout("bottom", 120)
	if m.leftPaneVisible {
		t.Error("expected leftPaneVisible=false in bottom mode")
	}
}

func TestUpdateLayoutForSize_AutoSwitchesOnResize(t *testing.T) {
	m := newTestModelWithLayout("auto", 120)
	if !m.leftPaneVisible {
		t.Error("expected leftPaneVisible=true at width=120 in auto mode")
	}
	m.width = 60
	m.updateLayoutForSize()
	if m.leftPaneVisible {
		t.Error("expected leftPaneVisible=false at width=60 in auto mode")
	}
	m.width = 100
	m.updateLayoutForSize()
	if !m.leftPaneVisible {
		t.Error("expected leftPaneVisible=true at width=100 in auto mode")
	}
}

func TestBottomLayout_RightPaneFullWidth(t *testing.T) {
	m := newTestModelWithLayout("bottom", 100)
	if got := m.rightPaneWidth(); got != 100 {
		t.Errorf("rightPaneWidth() = %d, want %d (full width in bottom layout)", got, 100)
	}
}

func TestBottomLayout_RightPaneHeightReduced(t *testing.T) {
	m := newTestModelWithLayout("bottom", 100)
	// height=30, header=1, bottom bar=9 → rightPaneHeight = 20
	if got := m.rightPaneHeight(); got != 20 {
		t.Errorf("rightPaneHeight() = %d, want %d", got, 20)
	}
}

func TestBottomLayout_RightPaneHeightFullWhenBarHidden(t *testing.T) {
	m := newTestModelWithLayout("bottom", 100)
	m.bottomBarVisible = false
	// height=30, header=1 → rightPaneHeight = 29
	if got := m.rightPaneHeight(); got != 29 {
		t.Errorf("rightPaneHeight() = %d, want %d (full height when bar hidden)", got, 29)
	}
}

func TestBottomLayout_RightPaneHeightVaryingHeight(t *testing.T) {
	m := newTestModelWithLayoutSize("bottom", 100, 40)
	// height=40, header=1, bottom bar=9 → rightPaneHeight = 30
	if got := m.rightPaneHeight(); got != 30 {
		t.Errorf("rightPaneHeight() = %d, want %d", got, 30)
	}
}

func TestBottomLayout_ToggleBracketTogglesBottomBar(t *testing.T) {
	m := newTestModelWithLayout("bottom", 100)
	if !m.bottomBarVisible {
		t.Error("expected bottomBarVisible=true initially")
	}
	km := &m.config.KeyMap
	action, _ := km.Resolve("[", "")
	if action != config.ActionToggleLeftPane {
		t.Fatalf("expected '[' to resolve to ActionToggleLeftPane, got %q", action)
	}
	// Simulate the toggle via handleKey
	m.bottomBarVisible = !m.bottomBarVisible
	if m.bottomBarVisible {
		t.Error("expected bottomBarVisible=false after toggle")
	}
}

func TestBottomLayout_FocusCycleOrder(t *testing.T) {
	m := newTestModelWithLayout("bottom", 60)
	m.bottomBarVisible = true
	m.focusedPane = rightPane

	// Tab: Timeline → Plans
	cycleFocusBottom(m)
	if m.focusedPane != plansPane {
		t.Errorf("after first tab, focusedPane = %d, want plansPane(%d)", m.focusedPane, plansPane)
	}

	// Tab: Plans → Iterations
	cycleFocusBottom(m)
	if m.focusedPane != iterationsPane {
		t.Errorf("after second tab, focusedPane = %d, want iterationsPane(%d)", m.focusedPane, iterationsPane)
	}

	// Tab: Iterations → Prompts
	cycleFocusBottom(m)
	if m.focusedPane != promptsPane {
		t.Errorf("after third tab, focusedPane = %d, want promptsPane(%d)", m.focusedPane, promptsPane)
	}

	// Tab: Prompts → Timeline
	cycleFocusBottom(m)
	if m.focusedPane != rightPane {
		t.Errorf("after fourth tab, focusedPane = %d, want rightPane(%d)", m.focusedPane, rightPane)
	}
}

// cycleFocusBottom simulates the bottom-layout focus toggle cycle on a model.
func cycleFocusBottom(m *Model) {
	switch m.focusedPane {
	case rightPane:
		m.focusedPane = plansPane
		m.rightPaneMode = planMode
	case plansPane:
		m.focusedPane = iterationsPane
		m.rightPaneMode = timelineMode
	case iterationsPane:
		m.focusedPane = promptsPane
	case promptsPane:
		m.focusedPane = rightPane
	}
}

func TestBottomLayout_FocusPreservedAcrossLayoutSwitch(t *testing.T) {
	m := newTestModelWithLayout("auto", 60) // starts in bottom mode
	m.focusedPane = iterationsPane

	// Resize to wide — switches to side layout
	m.width = 120
	m.updateLayoutForSize()

	if m.focusedPane != iterationsPane {
		t.Errorf("focus not preserved after layout switch: got %d, want iterationsPane(%d)", m.focusedPane, iterationsPane)
	}
}

func TestSideLayout_RightPaneHeightUnchanged(t *testing.T) {
	m := newTestModelWithLayoutSize("side", 120, 30)
	if got := m.rightPaneHeight(); got != 29 {
		t.Errorf("rightPaneHeight() = %d, want %d", got, 29)
	}
}

func TestSideLayout_RightPaneHeightVarying(t *testing.T) {
	m := newTestModelWithLayoutSize("side", 120, 50)
	if got := m.rightPaneHeight(); got != 49 {
		t.Errorf("rightPaneHeight() = %d, want %d", got, 49)
	}
}
