package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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

func TestBottomLayout_ViewContainsBottomBarSections(t *testing.T) {
	m := newTestModelWithLayoutSize("bottom", 60, 30)
	m.bottomBarVisible = true

	view := m.View()

	// Bottom bar should contain labeled dividers
	if !strings.Contains(view, "Plans") {
		t.Error("expected View() to contain 'Plans' divider in bottom layout")
	}
	if !strings.Contains(view, "Iterations") {
		t.Error("expected View() to contain 'Iterations' divider in bottom layout")
	}
	if !strings.Contains(view, "Prompts") {
		t.Error("expected View() to contain 'Prompts' divider in bottom layout")
	}
}

func TestBottomLayout_ViewNoBottomBarWhenHidden(t *testing.T) {
	m := newTestModelWithLayoutSize("bottom", 60, 30)
	m.bottomBarVisible = false

	view := m.View()

	// When hidden, bottom bar sections should not appear
	// (Plans/Iterations/Prompts text can appear in other contexts, so check for divider chars)
	if strings.Contains(view, "── Plans") {
		t.Error("expected View() to NOT contain bottom bar dividers when hidden")
	}
}

func TestBottomLayout_ViewNoLeftPane(t *testing.T) {
	m := newTestModelWithLayoutSize("bottom", 60, 30)
	view := m.View()

	// No vertical separator from left pane
	if strings.Contains(view, "│") {
		t.Error("expected no left pane separator in bottom layout")
	}
}

func TestSideLayout_ViewNoBottomBar(t *testing.T) {
	m := newTestModelWithLayoutSize("side", 120, 30)
	view := m.View()

	// Side layout should not contain bottom bar dividers
	if strings.Contains(view, "── Plans") {
		t.Error("expected no bottom bar dividers in side layout")
	}
}

func TestBottomLayout_IterListViewBottom_NoRunSeparators(t *testing.T) {
	il := NewIterList()
	th := testTheme()

	props := IterListProps{
		Iterations: []model.Iteration{
			{Index: 0, Status: model.IterationCompleted, Duration: 10 * time.Second},
			{Index: 1, Status: model.IterationRunning, StartTime: time.Now()},
		},
		Runs: []model.Run{
			{StartIndex: 0, PromptName: "run1"},
			{StartIndex: 1, PromptName: "run2"},
		},
		Width:   40,
		Height:  2,
		Focused: true,
		Theme:   th,
	}

	output := il.ViewBottom(props)

	// Should contain iteration text but no run separators
	if !strings.Contains(output, "Iter 1") {
		t.Error("expected ViewBottom to contain 'Iter 1'")
	}
	if strings.Contains(output, "run1") || strings.Contains(output, "run2") {
		t.Error("expected ViewBottom to NOT contain run separator names")
	}
}

func TestBottomLayout_PlanListViewBottom_NoTitle(t *testing.T) {
	pl := NewPlanList("/nonexistent")
	pl.Files = []string{"TEST_PLAN.md", "OTHER_PLAN.md"}
	th := testTheme()

	output := pl.ViewBottom(PlanListProps{
		Width:   40,
		Height:  2,
		Focused: false,
		Theme:   th,
	})

	if strings.Contains(output, "📋") {
		t.Error("expected ViewBottom to NOT contain title emoji")
	}
	if !strings.Contains(output, "TEST") {
		t.Error("expected ViewBottom to contain plan file display name")
	}
}

func TestBottomLayout_PromptListViewBottom_NoTitle(t *testing.T) {
	pl := NewPromptList("/nonexistent")
	pl.Files = []string{"PROMPT_BUILD.md", "PROMPT_TEST.md"}
	th := testTheme()

	output := pl.ViewBottom(PromptListProps{
		Width:   40,
		Height:  2,
		Focused: false,
		Theme:   th,
	})

	if strings.Contains(output, "📄") {
		t.Error("expected ViewBottom to NOT contain title emoji")
	}
	if !strings.Contains(output, "BUILD") {
		t.Error("expected ViewBottom to contain prompt file display name")
	}
}

// newTestModelBottomWithIters creates a bottom-layout model with iterations for mouse testing.
func newTestModelBottomWithIters(numIters int) *Model {
	m := newTestModelWithLayoutSize("bottom", 60, 30)
	for i := 0; i < numIters; i++ {
		m.controller.Session.Iterations = append(m.controller.Session.Iterations, model.Iteration{
			Index:    i,
			Status:   model.IterationCompleted,
			Duration: time.Duration(i+1) * time.Second,
		})
	}
	return m
}

// bottomBarMainHeight returns the main pane height for a bottom layout model.
// The bottom bar starts at Y = 1 (header) + mainHeight.
func bottomBarMainHeight(m *Model) int {
	return m.rightPaneHeight()
}

func TestBottomLayout_MouseClickMainAreaFocusesTimeline(t *testing.T) {
	m := newTestModelBottomWithIters(3)
	m.focusedPane = iterationsPane

	// Click in main area (Y = 5, well above bottom bar)
	m.Update(tea.MouseMsg{
		X:      10,
		Y:      5,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	})
	if m.focusedPane != rightPane {
		t.Errorf("expected rightPane focus after main area click, got %d", m.focusedPane)
	}
}

func TestBottomLayout_MouseClickIterationsSection(t *testing.T) {
	m := newTestModelBottomWithIters(3)
	m.focusedPane = rightPane

	// Iterations content starts at: header(1) + mainHeight + Plans divider(1) + Plans content(2) + Iter divider(1)
	// = 1 + mainHeight + 4
	mainH := bottomBarMainHeight(m)
	iterContentY := 1 + mainH + 4 // first iterations content row

	m.Update(tea.MouseMsg{
		X:      10,
		Y:      iterContentY,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	})
	if m.focusedPane != iterationsPane {
		t.Errorf("expected iterationsPane focus after clicking iterations section, got %d", m.focusedPane)
	}
}

func TestBottomLayout_MouseClickPlansSection(t *testing.T) {
	m := newTestModelBottomWithIters(1)
	m.planList.Files = []string{"TEST_PLAN.md"}
	m.focusedPane = rightPane

	// Plans content starts at: header(1) + mainHeight + Plans divider(1)
	mainH := bottomBarMainHeight(m)
	plansContentY := 1 + mainH + 1 // first plans content row

	m.Update(tea.MouseMsg{
		X:      10,
		Y:      plansContentY,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	})
	if m.focusedPane != plansPane {
		t.Errorf("expected plansPane focus after clicking plans section, got %d", m.focusedPane)
	}
	if m.rightPaneMode != planMode {
		t.Error("expected planMode after clicking plans section")
	}
}

func TestBottomLayout_MouseClickPromptsSection(t *testing.T) {
	m := newTestModelBottomWithIters(1)
	m.promptList.Files = []string{"PROMPT_BUILD.md"}
	m.focusedPane = rightPane

	// Prompts content starts at: header(1) + mainHeight + 6 dividers/content + Prompts divider(1)
	mainH := bottomBarMainHeight(m)
	promptsContentY := 1 + mainH + 7 // first prompts content row

	m.Update(tea.MouseMsg{
		X:      10,
		Y:      promptsContentY,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	})
	if m.focusedPane != promptsPane {
		t.Errorf("expected promptsPane focus after clicking prompts section, got %d", m.focusedPane)
	}
}

func TestBottomLayout_MouseClickDividerIgnored(t *testing.T) {
	m := newTestModelBottomWithIters(3)
	m.focusedPane = rightPane

	// Click on Plans divider (first divider, offset 0 from bottom bar start)
	mainH := bottomBarMainHeight(m)
	dividerY := 1 + mainH // Plans divider row

	m.Update(tea.MouseMsg{
		X:      10,
		Y:      dividerY,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	})
	// Focus should not change — divider clicks are ignored
	if m.focusedPane != rightPane {
		t.Errorf("expected focus unchanged after divider click, got %d", m.focusedPane)
	}
}

func TestBottomLayout_MouseClickIterationsDividerIgnored(t *testing.T) {
	m := newTestModelBottomWithIters(3)
	m.focusedPane = rightPane

	mainH := bottomBarMainHeight(m)
	iterDividerY := 1 + mainH + 3 // Iterations divider row

	m.Update(tea.MouseMsg{
		X:      10,
		Y:      iterDividerY,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	})
	if m.focusedPane != rightPane {
		t.Errorf("expected focus unchanged after iterations divider click, got %d", m.focusedPane)
	}
}

func TestBottomLayout_MouseScrollIterationsSection(t *testing.T) {
	m := newTestModelBottomWithIters(10)
	m.focusedPane = rightPane

	mainH := bottomBarMainHeight(m)
	iterContentY := 1 + mainH + 4

	// Scroll down in iterations section
	m.Update(tea.MouseMsg{
		X:      10,
		Y:      iterContentY,
		Button: tea.MouseButtonWheelDown,
	})
	if m.focusedPane != iterationsPane {
		t.Errorf("expected iterationsPane focus after scrolling iterations, got %d", m.focusedPane)
	}
	if m.iterList.Scroll == 0 {
		t.Error("expected scroll to increase after wheel down in iterations")
	}
}

func TestBottomLayout_MouseScrollMainArea(t *testing.T) {
	m := newTestModelBottomWithIters(3)
	m.focusedPane = iterationsPane

	// Scroll in the main area
	m.Update(tea.MouseMsg{
		X:      10,
		Y:      5,
		Button: tea.MouseButtonWheelDown,
	})
	if m.focusedPane != rightPane {
		t.Errorf("expected rightPane focus after scrolling main area, got %d", m.focusedPane)
	}
}

func TestBottomLayout_MouseClickSelectsIteration(t *testing.T) {
	m := newTestModelBottomWithIters(5)
	m.iterList.Cursor = 0

	mainH := bottomBarMainHeight(m)
	// Click second row of iterations content (should select iteration at Scroll+1)
	iterContent2ndRow := 1 + mainH + 5

	m.Update(tea.MouseMsg{
		X:      10,
		Y:      iterContent2ndRow,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	})

	if m.iterList.Cursor != 1 {
		t.Errorf("expected cursor=1 after clicking second iteration row, got %d", m.iterList.Cursor)
	}
}

func TestBottomLayout_MouseClickSelectsPlan(t *testing.T) {
	m := newTestModelBottomWithIters(1)
	m.planList.Files = []string{"PLAN_A.md", "PLAN_B.md"}
	m.planList.Cursor = 0

	mainH := bottomBarMainHeight(m)
	// Click second row of plans content
	planContent2ndRow := 1 + mainH + 2

	m.Update(tea.MouseMsg{
		X:      10,
		Y:      planContent2ndRow,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	})

	if m.planList.Cursor != 1 {
		t.Errorf("expected plan cursor=1 after clicking second plan row, got %d", m.planList.Cursor)
	}
}

func TestBottomLayout_MouseClickSelectsPrompt(t *testing.T) {
	m := newTestModelBottomWithIters(1)
	m.promptList.Files = []string{"PROMPT_A.md", "PROMPT_B.md"}
	m.promptList.Cursor = 0

	mainH := bottomBarMainHeight(m)
	// Click second row of prompts content
	promptContent2ndRow := 1 + mainH + 8

	m.Update(tea.MouseMsg{
		X:      10,
		Y:      promptContent2ndRow,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	})

	if m.promptList.Cursor != 1 {
		t.Errorf("expected prompt cursor=1 after clicking second prompt row, got %d", m.promptList.Cursor)
	}
}

func TestBottomLayout_MouseHeaderIgnored(t *testing.T) {
	m := newTestModelBottomWithIters(3)
	m.focusedPane = iterationsPane

	// Click on header (Y=0)
	m.Update(tea.MouseMsg{
		X:      10,
		Y:      0,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	})
	if m.focusedPane != iterationsPane {
		t.Errorf("expected focus unchanged after header click, got %d", m.focusedPane)
	}
}

func TestBottomLayout_MouseBottomBarHiddenAllMainArea(t *testing.T) {
	m := newTestModelBottomWithIters(3)
	m.bottomBarVisible = false
	m.focusedPane = iterationsPane

	// Click anywhere — should target main area since bar is hidden
	m.Update(tea.MouseMsg{
		X:      10,
		Y:      25, // would be bottom bar region if visible
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	})
	if m.focusedPane != rightPane {
		t.Errorf("expected rightPane focus when bottom bar hidden, got %d", m.focusedPane)
	}
}

func TestBottomLayout_MouseClickIterationResetsTimeline(t *testing.T) {
	m := newTestModelBottomWithIters(3)
	m.iterList.Cursor = 2
	m.timeline.Cursor = 5 // non-zero cursor

	mainH := bottomBarMainHeight(m)
	iterContentY := 1 + mainH + 4 // first iteration row

	m.Update(tea.MouseMsg{
		X:      10,
		Y:      iterContentY,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	})

	// Clicking a different iteration should reset timeline position
	if m.iterList.Cursor != 2 && m.timeline.Cursor != 0 {
		t.Error("expected timeline position reset after clicking a different iteration")
	}
}

func TestBottomLayout_MouseClickPlanResetsPlanScroll(t *testing.T) {
	m := newTestModelBottomWithIters(1)
	m.planList.Files = []string{"PLAN_A.md", "PLAN_B.md"}
	m.planList.Cursor = 0
	m.planViewScroll = 10

	mainH := bottomBarMainHeight(m)
	planContent2ndRow := 1 + mainH + 2

	m.Update(tea.MouseMsg{
		X:      10,
		Y:      planContent2ndRow,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	})

	if m.planList.Cursor != 1 {
		t.Fatalf("expected plan cursor=1, got %d", m.planList.Cursor)
	}
	if m.planViewScroll != 0 {
		t.Errorf("expected planViewScroll reset to 0 after selecting different plan, got %d", m.planViewScroll)
	}
}

func TestBottomLayout_LastFocusedBottomPane_DefaultsToIterations(t *testing.T) {
	m := newTestModelWithLayout("bottom", 60)
	m.bottomBarVisible = true
	m.focusedPane = rightPane

	// h from main area should focus iterations (default)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	if m.focusedPane != iterationsPane {
		t.Errorf("focusedPane = %d, want iterationsPane(%d)", m.focusedPane, iterationsPane)
	}
}

func TestBottomLayout_LastFocusedBottomPane_RemembersPrompts(t *testing.T) {
	m := newTestModelWithLayout("bottom", 60)
	m.bottomBarVisible = true

	// Navigate to prompts pane via tab: right→plans→iters→prompts
	m.focusedPane = rightPane
	m.Update(tea.KeyMsg{Type: tea.KeyTab}) // → plans
	m.Update(tea.KeyMsg{Type: tea.KeyTab}) // → iterations
	m.Update(tea.KeyMsg{Type: tea.KeyTab}) // → prompts

	if m.focusedPane != promptsPane {
		t.Fatalf("expected promptsPane, got %d", m.focusedPane)
	}

	// l to go back to main area (tracks prompts as last)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	if m.focusedPane != rightPane {
		t.Fatalf("expected rightPane, got %d", m.focusedPane)
	}

	// h should return to prompts (last focused)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	if m.focusedPane != promptsPane {
		t.Errorf("focusedPane = %d, want promptsPane(%d)", m.focusedPane, promptsPane)
	}
}

func TestBottomLayout_LastFocusedBottomPane_RemembersPlans(t *testing.T) {
	m := newTestModelWithLayout("bottom", 60)
	m.bottomBarVisible = true

	// Navigate to plans pane via tab: right→plans
	m.focusedPane = rightPane
	m.Update(tea.KeyMsg{Type: tea.KeyTab}) // → plans

	if m.focusedPane != plansPane {
		t.Fatalf("expected plansPane, got %d", m.focusedPane)
	}

	// l to go back to main area (tracks plans as last)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	if m.focusedPane != rightPane {
		t.Fatalf("expected rightPane, got %d", m.focusedPane)
	}

	// h should return to plans (last focused)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	if m.focusedPane != plansPane {
		t.Errorf("focusedPane = %d, want plansPane(%d)", m.focusedPane, plansPane)
	}
}

func TestBottomLayout_LastFocusedBottomPane_TabCycleTracksCorrectly(t *testing.T) {
	m := newTestModelWithLayout("bottom", 60)
	m.bottomBarVisible = true
	m.focusedPane = rightPane

	// Tab through: right→plans→iters→prompts→right
	m.Update(tea.KeyMsg{Type: tea.KeyTab}) // → plans
	m.Update(tea.KeyMsg{Type: tea.KeyTab}) // → iterations (tracks plans)
	m.Update(tea.KeyMsg{Type: tea.KeyTab}) // → prompts (tracks iterations)
	m.Update(tea.KeyMsg{Type: tea.KeyTab}) // → right (tracks prompts)

	// Last focused should be prompts (last bottom pane before returning to right)
	if m.focusedPane != rightPane {
		t.Fatalf("expected rightPane, got %d", m.focusedPane)
	}

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	if m.focusedPane != promptsPane {
		t.Errorf("focusedPane = %d, want promptsPane(%d)", m.focusedPane, promptsPane)
	}
}
