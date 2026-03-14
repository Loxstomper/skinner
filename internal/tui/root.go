package tui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/loxstomper/skinner/internal/config"
	"github.com/loxstomper/skinner/internal/executor"
	"github.com/loxstomper/skinner/internal/git"
	"github.com/loxstomper/skinner/internal/model"
	"github.com/loxstomper/skinner/internal/session"
	"github.com/loxstomper/skinner/internal/stats"
	"github.com/loxstomper/skinner/internal/theme"
)

// Bubble Tea messages — thin wrappers around session events

type assistantBatchMsg struct{ session.AssistantBatchEvent }
type toolResultMsg struct{ session.ToolResultEvent }
type usageMsg struct{ session.UsageEvent }
type iterationEndMsg struct{}
type subprocessExitMsg struct{ err error }
type tickMsg time.Time
type promptEditorDoneMsg struct{ err error }
type planEditorDoneMsg struct{ err error }
type planModeDoneMsg struct{ err error }
type gitTickMsg struct{}
type gitRefreshMsg struct{ commits []git.Commit }
type gitTotalStatsMsg struct{ Additions, Deletions int }
type fileExplorerTickMsg struct{}
type fileExplorerRefreshMsg struct {
	roots           []*FileNode
	porcelainOutput string
}
type systemStatsResultMsg struct {
	cpuPct *int
	memPct *int
	// raw CPU sample for next delta
	cpuActive int64
	cpuTotal  int64
}

// Pane focus

type paneID int

const (
	plansPane paneID = iota
	iterationsPane
	promptsPane
	rightPane
)

// rightPaneModeType tracks whether the right pane shows plan content or timeline.
type rightPaneModeType int

const (
	timelineMode rightPaneModeType = iota
	planMode
)

// isLeftPane returns true if the pane is in the left column.
func isLeftPane(p paneID) bool {
	return p == iterationsPane || p == promptsPane || p == plansPane
}

// Model is the Bubble Tea model for the TUI.
type Model struct {
	controller    *session.Controller
	exec          executor.Executor
	config        config.Config
	promptContent string
	theme         theme.Theme

	// Sub-components
	iterList   IterList
	promptList PromptList
	planList   PlanList
	timeline   Timeline

	// Plan view state
	rightPaneMode       rightPaneModeType
	planViewScroll      int
	planViewTotalLines  int
	planScrollPositions map[string]int // per-file scroll persistence

	// Working directory for prompt file scanning
	workDir string

	// UI state
	width, height         int
	focusedPane           paneID
	lastFocusedBottomPane paneID // remembers last-focused bottom bar section for h/← recall
	compactView           bool
	leftPaneVisible       bool
	bottomBarVisible      bool
	lineNumbers           bool

	// KeyMap-driven sequence state machine (replaces hardcoded gPending)
	pendingAction string

	// Event channel for bridging executor events to Bubble Tea
	eventCh chan tea.Msg

	// Exit when done
	exitOnComplete bool

	// Modal state
	activeModal     modalType
	lastCtrlCAt     time.Time
	helpModalScroll int

	// Prompt read modal state
	promptModalFile    string // filename (e.g. "PROMPT_BUILD.md")
	promptModalContent string // cached file content
	promptModalScroll  int    // scroll offset within modal

	// Run modal state
	runModalValue      string // current input value
	runModalLastValue  string // last entered value (pre-fill memory)
	runModalSelected   bool   // whether the pre-filled value is fully selected
	runModalPromptFile string // which prompt file to run

	// Status flash message (transient, clears on next keypress)
	statusFlash string

	// Git view state
	gitViewActive     bool
	gitViewDepth      int // 0=commit list, 1=file list, 2=sub-scroll
	gitCommits        []git.Commit
	gitSelectedCommit int
	gitFiles          []git.FileChange
	gitSelectedFile   int
	gitCommitScroll   int
	gitFileScroll     int
	gitDiffScroll     int
	gitDiffHScroll    int
	gitSessionStart   time.Time
	gitParsedDiff     []Hunk
	gitCommitSummary  string // cached output from ShowCommit
	gitDiffContent    string // cached raw diff for selected file
	gitAutoFollow     bool   // auto-scroll to top on new commits (paused by manual scroll)

	// Async total stats
	gitTotalAdditions   int
	gitTotalDeletions   int
	gitTotalStatsLoaded bool
	gitTotalStatsCancel context.CancelFunc

	// File explorer state
	fileExplorerActive bool
	fileExplorerDepth  int // 0=tree focused, 1=scrollable preview
	fileExplorerTree   *FileTreeView
	filePreviewScroll  int
	filePreviewHScroll int

	// System stats state
	systemStatsAvailable bool // set to true after first successful read
	systemStatsTick      int  // counts 1-second ticks; fires stats read every 2

	// Render cache for plan view and file preview
	renderCache *RenderCache

	// Quit state
	quitting bool
}

func NewModel(sess model.Session, cfg config.Config, promptContent string, th theme.Theme, compactView bool, exitOnComplete bool, exec executor.Executor) Model {
	sessionPtr := &sess
	ctrl := session.NewController(sessionPtr, cfg, nil)

	// Use current working directory for prompt file scanning
	workDir, _ := os.Getwd()

	return Model{
		controller:            ctrl,
		exec:                  exec,
		config:                cfg,
		promptContent:         promptContent,
		theme:                 th,
		compactView:           compactView,
		leftPaneVisible:       true,
		bottomBarVisible:      true,
		lineNumbers:           cfg.LineNumbers,
		exitOnComplete:        exitOnComplete,
		eventCh:               make(chan tea.Msg, 100),
		focusedPane:           iterationsPane,
		lastFocusedBottomPane: iterationsPane,
		iterList:              NewIterList(),
		promptList:            NewPromptList(workDir),
		planList:              NewPlanList(workDir),
		timeline:              NewTimeline(),
		workDir:               workDir,
		planScrollPositions:   make(map[string]int),
		renderCache:           &RenderCache{},
		gitSessionStart:       time.Now(),
	}
}

// Session returns a pointer to the controller's session for read access.
func (m *Model) Session() *model.Session {
	return m.controller.Session
}

func (m *Model) Init() tea.Cmd {
	if m.controller.Session.Mode == "idle" {
		// Idle mode: no auto-start, just tick for prompt file scanning
		return tickCmd()
	}

	// Non-idle mode: create the first Run and start iterating
	promptName := strings.ToUpper(m.controller.Session.Mode)
	m.controller.StartRun(
		promptName,
		m.controller.Session.PromptFile,
		m.controller.Session.MaxIterations,
	)

	return tea.Batch(
		m.spawnIteration(),
		tickCmd(),
	)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateLayoutForSize()
		return m, nil

	case tickMsg:
		// Rescan prompt and plan files on each tick (1s interval)
		m.promptList.ScanFiles(m.workDir)
		m.planList.ScanFiles(m.workDir)

		// Fire system stats read every 2 ticks (2-second interval)
		m.systemStatsTick++
		if m.systemStatsTick%2 == 0 {
			sess := m.controller.Session
			return m, tea.Batch(tickCmd(), systemStatsReadCmd(sess.PrevCPUActive, sess.PrevCPUTotal))
		}
		return m, tickCmd()

	case gitTickMsg:
		if !m.gitViewActive {
			return m, nil
		}
		return m, gitRefreshCmd()

	case gitRefreshMsg:
		if !m.gitViewActive {
			return m, nil
		}
		if msg.commits != nil {
			m.mergeGitCommits(msg.commits)
		}
		return m, gitTickCmd()

	case gitTotalStatsMsg:
		if !m.gitViewActive {
			return m, nil
		}
		m.gitTotalAdditions = msg.Additions
		m.gitTotalDeletions = msg.Deletions
		m.gitTotalStatsLoaded = true
		return m, nil

	case fileExplorerTickMsg:
		if !m.fileExplorerActive {
			return m, nil
		}
		return m, fileExplorerRefreshCmd(m.workDir)

	case fileExplorerRefreshMsg:
		if !m.fileExplorerActive {
			return m, nil
		}
		// Defer refresh while searching to avoid disrupting the search results
		if m.fileExplorerTree != nil && m.fileExplorerTree.IsSearching() {
			return m, fileExplorerTickCmd()
		}
		if msg.roots != nil {
			m.mergeFileExplorerTree(msg.roots, msg.porcelainOutput)
		}
		return m, fileExplorerTickCmd()

	case systemStatsResultMsg:
		sess := m.controller.Session
		if msg.cpuActive != 0 || msg.cpuTotal != 0 {
			sess.PrevCPUActive = msg.cpuActive
			sess.PrevCPUTotal = msg.cpuTotal
			m.systemStatsAvailable = true
		}
		if m.systemStatsAvailable {
			sess.CPUPercent = msg.cpuPct
			sess.MemPercent = msg.memPct
		}
		return m, nil

	case fileExplorerEditorDoneMsg:
		// Editor exited — file preview will re-render on next View()
		return m, nil

	case assistantBatchMsg:
		m.controller.ProcessAssistantBatch(msg.Events)
		if idx := m.controller.RunningIterationIdx(); idx >= 0 {
			if m.iterList.SelectedIndex() == idx {
				iter := &m.controller.Session.Iterations[idx]
				m.timeline.OnNewItems(m.timelineProps(iter.Items))
			}
		}
		return m, waitForEvent(m.eventCh)

	case usageMsg:
		m.controller.ProcessUsage(msg.UsageEvent)
		return m, waitForEvent(m.eventCh)

	case toolResultMsg:
		group := m.controller.ProcessToolResult(msg.ToolResultEvent)
		if group != nil && group.Status() != model.ToolCallRunning && !group.ManualToggle {
			idx := m.controller.RunningIterationIdx()
			if idx >= 0 {
				iter := &m.controller.Session.Iterations[idx]
				for i, item := range iter.Items {
					if item == group {
						cursorOnGroup := m.isCursorOnGroup(iter.Items, i)
						if m.iterList.SelectedIndex() != idx || !cursorOnGroup {
							group.Expanded = false
						}
						break
					}
				}
			}
		}
		return m, waitForEvent(m.eventCh)

	case iterationEndMsg:
		return m, waitForEvent(m.eventCh)

	case subprocessExitMsg:
		m.controller.CompleteIteration(msg.err)

		if !m.quitting && m.controller.ShouldStartNext() {
			return m, m.spawnIteration()
		}
		if m.exitOnComplete {
			m.quitting = true
			return m, tea.Quit
		}
		return m, nil

	case promptEditorDoneMsg:
		// Editor exited; rescan prompt files in case the file was modified
		m.promptList.ScanFiles(m.workDir)
		return m, nil

	case planEditorDoneMsg:
		// Editor exited; rescan plan files and re-render plan content
		m.planList.ScanFiles(m.workDir)
		m.focusedPane = rightPane
		m.rightPaneMode = planMode
		return m, nil

	case planModeDoneMsg:
		// Plan mode CLI exited; rescan plan files
		m.planList.ScanFiles(m.workDir)
		if msg.err != nil {
			m.statusFlash = fmt.Sprintf("plan command failed (exit %v)", msg.err)
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case tea.MouseMsg:
		return m.handleMouse(msg)
	}

	return m, nil
}

const ctrlCForceQuitWindow = 500 * time.Millisecond

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	km := &m.config.KeyMap

	// Clear status flash on any keypress.
	if m.statusFlash != "" {
		m.statusFlash = ""
	}

	// ctrl+c: double within 500ms force-quits, single shows quit modal.
	if key == "ctrl+c" {
		now := time.Now()
		if !m.lastCtrlCAt.IsZero() && now.Sub(m.lastCtrlCAt) < ctrlCForceQuitWindow {
			// Double ctrl+c: force quit immediately.
			m.quitting = true
			_ = m.exec.Kill()
			return m, tea.Quit
		}
		m.lastCtrlCAt = now
		if m.exitOnComplete {
			// --exit flag: bypass quit confirmation modal entirely.
			m.quitting = true
			_ = m.exec.Kill()
			return m, tea.Quit
		}
		m.activeModal = modalQuitConfirm
		return m, nil
	}

	// When a modal is active, route keys to the modal handler.
	if m.activeModal != modalNone {
		return m.handleModalKey(msg)
	}

	// When file explorer is active and in search mode, route raw keys to search handler
	// before action resolution (search needs raw character input).
	if m.fileExplorerActive && m.fileExplorerTree != nil && m.fileExplorerTree.IsSearching() {
		if m.handleFileExplorerSearchKey(key) {
			return m, nil
		}
	}

	// Resolve the key through the KeyMap, handling sequence state.
	action, newPending := km.Resolve(key, m.pendingAction)
	m.pendingAction = newPending

	// If we just set a pending action (sequence started), wait for next key.
	if action == "" && newPending != "" {
		return m, nil
	}

	// If no action matched via KeyMap, check arrow key alternates.
	// Arrow keys are always active alongside their letter equivalents per spec.
	if action == "" {
		if alt := config.HasAlternateArrowKey(key); alt != "" {
			action, _ = km.Resolve(alt, "")
		}
	}

	// Also handle "home" as an alternate for jump_top (always active).
	if action == "" && key == "home" {
		action = config.ActionJumpTop
	}
	// "end" as an alternate for jump_bottom (always active).
	if action == "" && key == "end" {
		action = config.ActionJumpBottom
	}
	// pgup/pgdn — delegate directly to focused component.
	if action == "" && key == "pgup" {
		action = "page_up"
	}
	if action == "" && key == "pgdown" {
		action = "page_down"
	}

	// Intercept digit keys for count buffer when right pane is focused
	// and no action was resolved yet.
	if action == "" && newPending == "" && m.focusedPane == rightPane && !m.timeline.InSubScroll() {
		if len(key) == 1 && key[0] >= '0' && key[0] <= '9' {
			m.timeline.AccumulateDigit(rune(key[0]))
			return m, nil
		}
	}

	// Any resolved action (or unrecognized key) clears the count buffer,
	// except for move_down/move_up which consume it.
	clearCount := true

	// When file explorer is active, route keys to the file explorer handler.
	if m.fileExplorerActive {
		// `/` activates fuzzy search at depth 0 — not a configured action
		if action == "" && key == "/" && m.fileExplorerDepth == 0 {
			action = "search"
		}
		return m.handleFileExplorerKey(action)
	}

	// When git view is active, route keys to the git view handler.
	if m.gitViewActive {
		return m.handleGitViewKey(action)
	}

	// When in sub-scroll mode, only allow specific actions.
	if m.timeline.InSubScroll() {
		m.timeline.ClearCount()
		switch action {
		case config.ActionQuit:
			if m.exitOnComplete {
				m.quitting = true
				_ = m.exec.Kill()
				return m, tea.Quit
			}
			m.activeModal = modalQuitConfirm
		case config.ActionHelp:
			m.activeModal = modalHelp
			m.helpModalScroll = 0
		case config.ActionEscape:
			m.timeline.ExitSubScroll()
		case config.ActionMoveDown, config.ActionMoveUp,
			config.ActionJumpTop, config.ActionJumpBottom,
			config.ActionExpand:
			m.timeline.HandleAction(action, m.currentTimelineProps())
		}
		// All other keys are ignored in sub-scroll mode.
		return m, nil
	}

	switch action {
	case config.ActionQuit:
		m.timeline.ClearCount()
		if m.exitOnComplete {
			m.quitting = true
			_ = m.exec.Kill()
			return m, tea.Quit
		}
		m.activeModal = modalQuitConfirm
		return m, nil

	case config.ActionHelp:
		m.timeline.ClearCount()
		m.activeModal = modalHelp
		m.helpModalScroll = 0
		return m, nil

	case config.ActionToggleLeftPane:
		if m.effectiveLayout() == "bottom" {
			m.bottomBarVisible = !m.bottomBarVisible
			if !m.bottomBarVisible && isLeftPane(m.focusedPane) {
				m.focusedPane = rightPane
			}
		} else {
			m.leftPaneVisible = !m.leftPaneVisible
			if !m.leftPaneVisible && isLeftPane(m.focusedPane) {
				m.focusedPane = rightPane
			}
		}

	case config.ActionFocusToggle:
		if m.effectiveLayout() == "bottom" {
			if !m.bottomBarVisible {
				break
			}
			// Bottom layout cycle: Timeline → Plans → Iterations → Prompts → Timeline
			switch m.focusedPane {
			case rightPane:
				m.focusedPane = plansPane
				m.rightPaneMode = planMode
			case plansPane:
				m.lastFocusedBottomPane = plansPane
				m.focusedPane = iterationsPane
				m.rightPaneMode = timelineMode
			case iterationsPane:
				m.lastFocusedBottomPane = iterationsPane
				m.focusedPane = promptsPane
			case promptsPane:
				m.lastFocusedBottomPane = promptsPane
				m.focusedPane = rightPane
			}
		} else {
			if !m.leftPaneVisible {
				break
			}
			// Side layout cycle: Plans → Iterations → Prompts → Timeline → Plans
			switch m.focusedPane {
			case plansPane:
				m.focusedPane = iterationsPane
				m.rightPaneMode = timelineMode
			case iterationsPane:
				m.focusedPane = promptsPane
			case promptsPane:
				m.focusedPane = rightPane
			case rightPane:
				m.focusedPane = plansPane
				m.rightPaneMode = planMode
			}
		}

	case config.ActionFocusLeft:
		if m.effectiveLayout() == "bottom" {
			// In bottom layout, h/← from main area focuses last-focused bottom bar section
			if m.focusedPane == rightPane && m.bottomBarVisible {
				m.focusedPane = m.lastFocusedBottomPane
			}
		} else if m.leftPaneVisible {
			if m.focusedPane == rightPane {
				// h from right pane: go to plans if in plan mode, iterations if in timeline mode
				if m.rightPaneMode == planMode {
					m.focusedPane = plansPane
				} else {
					m.focusedPane = iterationsPane
				}
			}
		}

	case config.ActionFocusRight:
		if m.effectiveLayout() == "bottom" {
			// In bottom layout, l/→ from bottom bar → main area
			if isLeftPane(m.focusedPane) {
				m.lastFocusedBottomPane = m.focusedPane
				m.focusedPane = rightPane
			}
		} else {
			m.focusedPane = rightPane
		}

	case config.ActionToggleView:
		m.compactView = !m.compactView

	case config.ActionToggleLineNumbers:
		m.lineNumbers = !m.lineNumbers

	case config.ActionRun:
		// r key: open run modal from prompt picker
		if m.focusedPane == promptsPane && m.controller.Phase() != model.PhaseRunning {
			if f := m.promptList.SelectedFile(); f != "" {
				m.openRunModal(f)
			}
		}

	case config.ActionGitView:
		return m, m.enterGitView()

	case config.ActionFileExplorer:
		return m, m.enterFileExplorer()

	case config.ActionPlanMode:
		// p key: launch interactive plan mode CLI (disabled while running)
		if m.controller.Phase() != model.PhaseRunning {
			return m, m.launchPlanMode()
		}

	case config.ActionExpand:
		switch m.focusedPane {
		case plansPane:
			// Enter on plans pane: switch to plan content view
			m.focusedPane = rightPane
			m.rightPaneMode = planMode
		case iterationsPane:
			m.focusedPane = rightPane
			m.rightPaneMode = timelineMode
		case promptsPane:
			if f := m.promptList.SelectedFile(); f != "" {
				content, err := ReadFileContent(m.workDir, f)
				if err == nil {
					m.promptModalFile = f
					m.promptModalContent = content
					m.promptModalScroll = 0
					m.activeModal = modalPromptRead
				}
			}
		default:
			m.timeline.HandleAction("expand", m.currentTimelineProps())
		}

	case config.ActionJumpTop:
		switch m.focusedPane {
		case plansPane:
			m.planList.HandleAction("jump_top", m.planListProps())
		case iterationsPane:
			m.iterList.JumpToTop()
			m.timeline.ResetPosition()
		case promptsPane:
			m.promptList.HandleAction("jump_top", m.promptListProps())
		default:
			if m.rightPaneMode == planMode {
				m.planViewScroll = 0
			} else {
				m.timeline.JumpToTop()
			}
		}

	case config.ActionJumpBottom:
		switch m.focusedPane {
		case plansPane:
			m.planList.HandleAction("jump_bottom", m.planListProps())
		case iterationsPane:
			m.iterList.JumpToBottom(len(m.controller.Session.Iterations), m.iterListHeight(), m.controller.Session.Runs)
			m.timeline.ResetPosition()
		case promptsPane:
			m.promptList.HandleAction("jump_bottom", m.promptListProps())
		default:
			if m.rightPaneMode == planMode {
				m.planViewScroll = ClampPlanScroll(m.planViewTotalLines, m.planViewTotalLines, m.rightPaneHeight())
			} else {
				m.timeline.JumpToBottom(m.currentTimelineProps())
			}
		}

	case config.ActionMoveDown, config.ActionMoveUp:
		switch m.focusedPane {
		case plansPane:
			oldCursor := m.planList.Cursor
			m.planList.HandleAction(action, m.planListProps())
			if m.planList.Cursor != oldCursor {
				// Reset scroll when switching between plans
				m.planViewScroll = 0
			}
		case iterationsPane:
			oldCursor := m.iterList.Cursor
			m.iterList.HandleAction(action, m.iterListProps())
			if m.iterList.Cursor != oldCursor {
				m.timeline.ResetPosition()
			}
		case promptsPane:
			m.promptList.HandleAction(action, m.promptListProps())
		default:
			if m.rightPaneMode == planMode {
				if action == config.ActionMoveDown {
					m.planViewScroll++
				} else {
					m.planViewScroll--
				}
				m.planViewScroll = ClampPlanScroll(m.planViewScroll, m.planViewTotalLines, m.rightPaneHeight())
			} else {
				count := m.timeline.ConsumeCount()
				clearCount = false // ConsumeCount already cleared
				m.timeline.HandleActionWithCount(action, count, m.currentTimelineProps())
			}
		}

	case "page_up", "page_down":
		switch m.focusedPane {
		case plansPane:
			m.planList.HandleAction(action, m.planListProps())
		case iterationsPane:
			oldCursor := m.iterList.Cursor
			m.iterList.HandleAction(action, m.iterListProps())
			if m.iterList.Cursor != oldCursor {
				m.timeline.ResetPosition()
			}
		case promptsPane:
			m.promptList.HandleAction(action, m.promptListProps())
		default:
			if m.rightPaneMode == planMode {
				pageSize := m.rightPaneHeight() - 1
				if action == "page_down" {
					m.planViewScroll += pageSize
				} else {
					m.planViewScroll -= pageSize
				}
				m.planViewScroll = ClampPlanScroll(m.planViewScroll, m.planViewTotalLines, m.rightPaneHeight())
			} else {
				m.timeline.HandleAction(action, m.currentTimelineProps())
			}
		}
	}

	// Edit plan file: launch editor for plan files (from plan list or plan content view)
	if action == config.ActionEditPlan && (m.focusedPane == plansPane || (m.focusedPane == rightPane && m.rightPaneMode == planMode)) {
		if f := m.planList.SelectedFile(); f != "" {
			return m, m.launchPlanEditor(f)
		}
	}

	if clearCount {
		m.timeline.ClearCount()
	}

	return m, nil
}

// launchPlanEditor launches $EDITOR for the given plan file.
func (m *Model) launchPlanEditor(filename string) tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}
	filePath := m.workDir + "/" + filename
	return tea.ExecProcess(exec.Command(editor, filePath), func(err error) tea.Msg {
		return planEditorDoneMsg{err: err}
	})
}

// launchPlanMode launches an interactive CLI session via sh -c with the configured plan command.
func (m *Model) launchPlanMode() tea.Cmd {
	return tea.ExecProcess(exec.Command("sh", "-c", m.config.PlanCommand), func(err error) tea.Msg {
		return planModeDoneMsg{err: err}
	})
}

// handleModalKey routes key presses to the active modal.
func (m *Model) handleModalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if m.activeModal == modalQuitConfirm {
		switch key {
		case "y":
			m.quitting = true
			_ = m.exec.Kill()
			return m, tea.Quit
		case "n", "esc":
			m.activeModal = modalNone
			return m, nil
		}
		// All other keys are ignored in quit modal.
		return m, nil
	}

	if m.activeModal == modalHelp {
		// pgup/pgdn scroll the help modal when content overflows.
		switch key {
		case "pgup":
			m.helpModalScroll -= 5
			if m.helpModalScroll < 0 {
				m.helpModalScroll = 0
			}
			return m, nil
		case "pgdown":
			m.helpModalScroll += 5
			return m, nil
		}
		// Any other key dismisses the help modal.
		m.activeModal = modalNone
		m.helpModalScroll = 0
		return m, nil
	}

	if m.activeModal == modalPromptRead {
		return m.handlePromptModalKey(key)
	}

	if m.activeModal == modalRunConfig {
		return m.handleRunModalKey(key)
	}

	return m, nil
}

// handlePromptModalKey handles key presses in the prompt read modal.
func (m *Model) handlePromptModalKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc":
		m.activeModal = modalNone
		return m, nil
	case "r":
		// Open run modal from prompt read modal
		if m.controller.Phase() != model.PhaseRunning {
			m.openRunModal(m.promptModalFile)
		}
		return m, nil
	case "e":
		// Launch $EDITOR for the prompt file
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vi"
		}
		filePath := m.workDir + "/" + m.promptModalFile
		m.activeModal = modalNone
		c := tea.ExecProcess(exec.Command(editor, filePath), func(err error) tea.Msg {
			return promptEditorDoneMsg{err: err}
		})
		return m, c
	case "j", "down":
		maxScroll := PromptModalMaxScroll(m.promptModalContent, m.height)
		if m.promptModalScroll < maxScroll {
			m.promptModalScroll++
		}
	case "k", "up":
		if m.promptModalScroll > 0 {
			m.promptModalScroll--
		}
	case "pgdown":
		maxScroll := PromptModalMaxScroll(m.promptModalContent, m.height)
		m.promptModalScroll += 10
		if m.promptModalScroll > maxScroll {
			m.promptModalScroll = maxScroll
		}
	case "pgup":
		m.promptModalScroll -= 10
		if m.promptModalScroll < 0 {
			m.promptModalScroll = 0
		}
	}
	return m, nil
}

// openRunModal opens the run config modal for the given prompt file.
func (m *Model) openRunModal(promptFile string) {
	m.runModalPromptFile = promptFile
	if m.runModalLastValue == "" {
		m.runModalLastValue = "10"
	}
	m.runModalValue = m.runModalLastValue
	m.runModalSelected = true
	m.activeModal = modalRunConfig
}

// handleRunModalKey handles key presses in the run config modal.
func (m *Model) handleRunModalKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc":
		m.activeModal = modalNone
		return m, nil

	case "enter":
		// Empty value is invalid — do nothing
		if m.runModalValue == "" {
			return m, nil
		}
		// Parse the iteration count
		maxIter := 0
		for _, c := range m.runModalValue {
			maxIter = maxIter*10 + int(c-'0')
		}
		// Save for pre-fill next time
		m.runModalLastValue = m.runModalValue

		// Close all modals
		m.activeModal = modalNone

		// Read prompt file fresh from disk
		content, err := ReadFileContent(m.workDir, m.runModalPromptFile)
		if err != nil {
			return m, nil
		}
		m.promptContent = content

		// Extract prompt name from filename (e.g. "PROMPT_BUILD.md" -> "BUILD")
		promptName := promptNameFromFile(m.runModalPromptFile)

		// Start the run
		m.controller.StartRun(promptName, m.runModalPromptFile, maxIter)

		return m, m.spawnIteration()

	case "backspace":
		if m.runModalSelected {
			// Clear the selected value
			m.runModalValue = ""
			m.runModalSelected = false
		} else if len(m.runModalValue) > 0 {
			m.runModalValue = m.runModalValue[:len(m.runModalValue)-1]
		}
		return m, nil

	default:
		// Only accept digit characters
		if len(key) == 1 && key[0] >= '0' && key[0] <= '9' {
			if m.runModalSelected {
				// Replace the selected value
				m.runModalValue = key
				m.runModalSelected = false
			} else {
				m.runModalValue += key
			}
		}
		return m, nil
	}
}

// promptNameFromFile extracts the prompt name from a filename.
// e.g. "PROMPT_BUILD.md" -> "BUILD", "PROMPT_PLAN.md" -> "PLAN"
func promptNameFromFile(filename string) string {
	name := strings.TrimSuffix(filename, ".md")
	name = strings.TrimPrefix(name, "PROMPT_")
	return name
}

const mouseScrollLines = 3

func (m *Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	// Ignore events on the header row
	paneRow := msg.Y - 1
	if paneRow < 0 {
		return m, nil
	}

	// When file explorer is active, handle mouse events
	if m.fileExplorerActive {
		return m.handleFileExplorerMouse(msg, paneRow)
	}

	// When git view is active, handle mouse wheel for scrolling
	if m.gitViewActive {
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			m.gitViewScrollBy(-mouseScrollLines)
		case tea.MouseButtonWheelDown:
			m.gitViewScrollBy(mouseScrollLines)
		}
		return m, nil
	}

	// Determine target pane and whether the event is in the bottom bar
	targetPane := rightPane
	inBottomBar := false
	bottomContentRow := 0 // row within a bottom bar section's content (0 or 1)

	if m.effectiveLayout() == "bottom" {
		mainHeight := m.rightPaneHeight()
		if paneRow >= mainHeight && m.bottomBarVisible {
			// Bottom bar region — map Y offset to section
			bottomOffset := paneRow - mainHeight
			// Structure: [divider, content×2] × 3
			// Plans:      0=divider, 1-2=content
			// Iterations: 3=divider, 4-5=content
			// Prompts:    6=divider, 7-8=content
			switch {
			case bottomOffset >= 1 && bottomOffset <= 2:
				targetPane = plansPane
				bottomContentRow = bottomOffset - 1
				inBottomBar = true
			case bottomOffset >= 4 && bottomOffset <= 5:
				targetPane = iterationsPane
				bottomContentRow = bottomOffset - 4
				inBottomBar = true
			case bottomOffset >= 7 && bottomOffset <= 8:
				targetPane = promptsPane
				bottomContentRow = bottomOffset - 7
				inBottomBar = true
			default:
				// Divider line — ignore
				return m, nil
			}
		}
		// else: paneRow < mainHeight → targetPane stays rightPane
	} else {
		// Side layout: determine target by X coordinate
		leftWidth := m.leftPaneWidth()
		if leftWidth > 0 && msg.X < leftWidth {
			paneHeight := m.rightPaneHeight()
			switch {
			case IsInPlanSection(paneRow):
				targetPane = plansPane
			case IsInPromptSection(paneRow, paneHeight):
				targetPane = promptsPane
			default:
				targetPane = iterationsPane
			}
		}
	}

	switch {
	case msg.Button == tea.MouseButtonWheelUp:
		m.focusedPane = targetPane
		if inBottomBar && isLeftPane(targetPane) {
			m.lastFocusedBottomPane = targetPane
		}
		m.updateRightPaneModeForFocus(targetPane)
		switch targetPane {
		case plansPane:
			m.planList.ScrollBy(-mouseScrollLines)
		case iterationsPane:
			count := len(m.controller.Session.Iterations)
			if inBottomBar {
				// Bottom bar: no run separators, use bottomBarSectionHeight
				m.iterList.ScrollBy(-mouseScrollLines, count, bottomBarSectionHeight, nil)
			} else {
				m.iterList.ScrollBy(-mouseScrollLines, count, m.iterListHeight(), m.controller.Session.Runs)
			}
		case promptsPane:
			m.promptList.ScrollBy(-mouseScrollLines)
		default:
			if m.rightPaneMode == planMode {
				m.planViewScroll -= mouseScrollLines
				m.planViewScroll = ClampPlanScroll(m.planViewScroll, m.planViewTotalLines, m.rightPaneHeight())
			} else {
				m.timeline.ScrollBy(-mouseScrollLines, m.currentTimelineProps())
			}
		}

	case msg.Button == tea.MouseButtonWheelDown:
		m.focusedPane = targetPane
		if inBottomBar && isLeftPane(targetPane) {
			m.lastFocusedBottomPane = targetPane
		}
		m.updateRightPaneModeForFocus(targetPane)
		switch targetPane {
		case plansPane:
			m.planList.ScrollBy(mouseScrollLines)
		case iterationsPane:
			count := len(m.controller.Session.Iterations)
			if inBottomBar {
				m.iterList.ScrollBy(mouseScrollLines, count, bottomBarSectionHeight, nil)
			} else {
				m.iterList.ScrollBy(mouseScrollLines, count, m.iterListHeight(), m.controller.Session.Runs)
			}
		case promptsPane:
			m.promptList.ScrollBy(mouseScrollLines)
		default:
			if m.rightPaneMode == planMode {
				m.planViewScroll += mouseScrollLines
				m.planViewScroll = ClampPlanScroll(m.planViewScroll, m.planViewTotalLines, m.rightPaneHeight())
			} else {
				m.timeline.ScrollBy(mouseScrollLines, m.currentTimelineProps())
			}
		}

	case msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress:
		m.focusedPane = targetPane
		if inBottomBar && isLeftPane(targetPane) {
			m.lastFocusedBottomPane = targetPane
		}
		m.updateRightPaneModeForFocus(targetPane)
		if inBottomBar {
			m.handleBottomBarClick(targetPane, bottomContentRow)
		} else {
			switch targetPane {
			case plansPane:
				planRow := PlanSectionRow(paneRow)
				oldCursor := m.planList.Cursor
				m.planList.ClickRow(planRow)
				if m.planList.Cursor != oldCursor {
					m.planViewScroll = 0
				}
			case iterationsPane:
				// Adjust row for plan section + divider above iterations
				iterRow := paneRow - PlanListTotalHeight() - 1
				count := len(m.controller.Session.Iterations)
				oldCursor := m.iterList.Cursor
				if iterRow >= 0 {
					if m.iterList.ClickRow(iterRow, count, m.iterListHeight(), m.controller.Session.Runs) {
						if m.iterList.Cursor != oldCursor {
							m.timeline.ResetPosition()
						}
					}
				}
			case promptsPane:
				paneHeight := m.rightPaneHeight()
				promptRow := PromptSectionRow(paneRow, paneHeight)
				m.promptList.ClickRow(promptRow)
			default:
				props := m.currentTimelineProps()
				if m.timeline.InSubScroll() {
					m.timeline.ClickRowSubScroll(paneRow, props)
				} else {
					m.timeline.ClickRow(paneRow, props)
				}
			}
		}
	}

	return m, nil
}

// handleBottomBarClick handles a left-click on a bottom bar section.
// contentRow is the row within the section's content area (0 or 1).
func (m *Model) handleBottomBarClick(target paneID, contentRow int) {
	switch target {
	case plansPane:
		// ViewBottom has no title row — add 1 to match ClickRow's title offset
		oldCursor := m.planList.Cursor
		m.planList.ClickRow(contentRow + 1)
		if m.planList.Cursor != oldCursor {
			m.planViewScroll = 0
		}
	case iterationsPane:
		// ViewBottom has no run separators — pass nil runs
		count := len(m.controller.Session.Iterations)
		oldCursor := m.iterList.Cursor
		if m.iterList.ClickRow(contentRow, count, bottomBarSectionHeight, nil) {
			if m.iterList.Cursor != oldCursor {
				m.timeline.ResetPosition()
			}
		}
	case promptsPane:
		// ViewBottom has no title row — add 1 to match ClickRow's title offset
		m.promptList.ClickRow(contentRow + 1)
	}
}

func (m *Model) View() string {
	if m.quitting {
		return ""
	}
	if m.width == 0 || m.height == 0 {
		return "Starting..."
	}

	header := RenderHeader(m.headerProps())

	// File explorer: render when active
	if m.fileExplorerActive {
		fileExplorerView := m.renderFileExplorer()
		view := header + "\n" + fileExplorerView

		switch m.activeModal {
		case modalQuitConfirm:
			view = RenderQuitConfirmModal(m.width, m.height, m.theme)
		case modalHelp:
			view = RenderHelpModal(m.width, m.height, m.theme, &m.config.KeyMap, m.helpModalScroll)
		}
		return view
	}

	// Git view: render the git view overlay when active
	if m.gitViewActive {
		gitView := m.renderGitView()
		view := header + "\n" + gitView

		// Modal overlay if active
		switch m.activeModal {
		case modalQuitConfirm:
			view = RenderQuitConfirmModal(m.width, m.height, m.theme)
		case modalHelp:
			view = RenderHelpModal(m.width, m.height, m.theme, &m.config.KeyMap, m.helpModalScroll)
		}
		return view
	}

	paneHeight := m.height - 1 // subtract 1 for the header line
	leftWidth := m.leftPaneWidth()
	rightWidth := m.rightPaneWidth()
	rightHeight := m.rightPaneHeight() // accounts for bottom bar

	// Right pane: plan content view or timeline depending on mode
	var right string
	if m.rightPaneMode == planMode {
		var totalLines int
		right, totalLines = RenderPlanView(PlanViewProps{
			Filename: m.planList.SelectedFile(),
			Dir:      m.workDir,
			Width:    rightWidth,
			Height:   rightHeight,
			Scroll:   m.planViewScroll,
			Focused:  m.focusedPane == rightPane,
			Theme:    m.theme,
			Cache:    m.renderCache,
		})
		m.planViewTotalLines = totalLines
	} else {
		tlProps := TimelineProps{
			Items:       m.selectedItems(),
			Width:       rightWidth,
			Height:      rightHeight,
			Focused:     m.focusedPane == rightPane,
			CompactView: m.compactView,
			LineNumbers: m.lineNumbers,
			Theme:       m.theme,
			WorkDir:     m.workDir,
		}
		m.populateThinkingState(&tlProps)
		right = m.timeline.View(tlProps)
	}

	var panes string
	if leftWidth > 0 {
		iterHeight := m.iterListHeight()
		planHeight := PlanListTotalHeight()
		promptHeight := PromptListTotalHeight()

		planView := m.planList.View(PlanListProps{
			Width:   leftWidth,
			Height:  planHeight,
			Focused: m.focusedPane == plansPane,
			Theme:   m.theme,
		})

		// Horizontal divider between plans and iterations
		divider := lipgloss.NewStyle().
			Foreground(lipgloss.Color(m.theme.ForegroundDim)).
			Width(leftWidth).
			Render(strings.Repeat("─", leftWidth))

		iterView := m.iterList.View(IterListProps{
			Iterations: m.controller.Session.Iterations,
			Runs:       m.controller.Session.Runs,
			Width:      leftWidth,
			Height:     iterHeight,
			Focused:    m.focusedPane == iterationsPane,
			Theme:      m.theme,
		})

		promptView := m.promptList.View(PromptListProps{
			Width:   leftWidth,
			Height:  promptHeight,
			Focused: m.focusedPane == promptsPane,
			Theme:   m.theme,
		})

		left := lipgloss.JoinVertical(lipgloss.Left, planView, divider, iterView, divider, promptView)

		sepLines := make([]string, paneHeight)
		for i := range sepLines {
			sepLines[i] = "│"
		}
		separator := lipgloss.NewStyle().
			Foreground(lipgloss.Color(m.theme.ForegroundDim)).
			Render(strings.Join(sepLines, "\n"))

		panes = lipgloss.JoinHorizontal(lipgloss.Top, left, separator, right)
	} else {
		panes = right
	}

	// Bottom bar: render below the main area in bottom layout
	if m.effectiveLayout() == "bottom" && m.bottomBarVisible {
		bottomBar := m.renderBottomBar(rightWidth)
		panes = lipgloss.JoinVertical(lipgloss.Left, panes, bottomBar)
	}

	view := header + "\n" + panes

	// Render modal overlay if active.
	switch m.activeModal {
	case modalQuitConfirm:
		view = RenderQuitConfirmModal(m.width, m.height, m.theme)
	case modalHelp:
		view = RenderHelpModal(m.width, m.height, m.theme, &m.config.KeyMap, m.helpModalScroll)
	case modalPromptRead:
		view = RenderPromptReadModal(PromptModalProps{
			Filename: m.promptModalFile,
			Content:  m.promptModalContent,
			Scroll:   m.promptModalScroll,
			Width:    m.width,
			Height:   m.height,
			Theme:    m.theme,
			Running:  m.controller.Phase() == model.PhaseRunning,
		})
	case modalRunConfig:
		view = RenderRunModal(m.width, m.height, m.theme, m.runModalValue, m.runModalSelected)
	}

	return view
}

// headerProps builds the HeaderProps from current model state.
func (m *Model) headerProps() HeaderProps {
	sess := m.controller.Session
	inputTokens := sess.InputTokens + sess.CacheReadTokens + sess.CacheCreationTokens

	contextPercent := -1
	if m.controller.HasKnownModel() && m.controller.LastModel() != "" {
		if pricing, ok := m.config.Pricing[m.controller.LastModel()]; ok && pricing.ContextWindow > 0 {
			contextPercent = int((sess.LastInputTokens + sess.LastCacheReadTokens) * 100 / int64(pricing.ContextWindow))
		}
	}

	// Determine session status from current state
	var sessionStatus model.IterationStatus
	switch {
	case m.controller.RunningIterationIdx() >= 0:
		sessionStatus = model.IterationRunning
	case len(sess.Iterations) > 0 && sess.Iterations[len(sess.Iterations)-1].Status == model.IterationFailed:
		sessionStatus = model.IterationFailed
	default:
		sessionStatus = model.IterationCompleted
	}

	return HeaderProps{
		Phase:           m.controller.Phase(),
		SessionDuration: m.sessionDuration(sess),
		InputTokens:     inputTokens,
		OutputTokens:    sess.OutputTokens,
		ContextPercent:  contextPercent,
		TotalCost:       sess.TotalCost,
		HasKnownModel:   m.controller.HasKnownModel(),
		RateLimit:       sess.RateLimit,
		IterationCount:  len(sess.Iterations),
		MaxIterations:   sess.MaxIterations,
		SessionStatus:   sessionStatus,
		StatusFlash:     m.statusFlash,
		CPUPercent:      sess.CPUPercent,
		MemPercent:      sess.MemPercent,
		Width:           m.width,
		Theme:           m.theme,
	}
}

// sessionDuration returns the total session duration accounting for pause/resume.
// Running: accumulated time from prior runs + elapsed time in current run.
// Finished/Idle: just the accumulated time (current run already folded in on finish).
func (m *Model) sessionDuration(sess *model.Session) time.Duration {
	if m.controller.Phase() == model.PhaseRunning {
		return sess.AccumulatedDuration + time.Since(sess.StartTime)
	}
	return sess.AccumulatedDuration
}

// iterListHeight returns the height available for the iteration list,
// accounting for the plan section, prompt section, and dividers.
func (m *Model) iterListHeight() int {
	paneHeight := m.rightPaneHeight()
	// Subtract plan section (5 rows) + divider (1 row) + prompt section (5 rows) + divider (1 row)
	h := paneHeight - PlanListTotalHeight() - 1 - PromptListTotalHeight() - 1
	if h < 1 {
		h = 1
	}
	return h
}

// iterListProps builds IterListProps from current state.
func (m *Model) iterListProps() IterListProps {
	return IterListProps{
		Iterations: m.controller.Session.Iterations,
		Runs:       m.controller.Session.Runs,
		Width:      m.leftPaneWidth(),
		Height:     m.iterListHeight(),
		Focused:    m.focusedPane == iterationsPane,
		Theme:      m.theme,
	}
}

// planListProps builds PlanListProps from current state.
func (m *Model) planListProps() PlanListProps {
	return PlanListProps{
		Width:   m.leftPaneWidth(),
		Height:  PlanListTotalHeight(),
		Focused: m.focusedPane == plansPane,
		Theme:   m.theme,
	}
}

// updateRightPaneModeForFocus sets the right pane mode based on the target pane.
func (m *Model) updateRightPaneModeForFocus(target paneID) {
	switch target {
	case plansPane:
		m.rightPaneMode = planMode
	case iterationsPane, promptsPane:
		m.rightPaneMode = timelineMode
	}
}

// promptListProps builds PromptListProps from current state.
func (m *Model) promptListProps() PromptListProps {
	return PromptListProps{
		Width:   m.leftPaneWidth(),
		Height:  PromptListTotalHeight(),
		Focused: m.focusedPane == promptsPane,
		Theme:   m.theme,
	}
}

// currentTimelineProps builds TimelineProps for the currently selected iteration.
func (m *Model) currentTimelineProps() TimelineProps {
	return m.timelineProps(m.selectedItems())
}

// timelineProps builds TimelineProps for a given set of items.
func (m *Model) timelineProps(items []model.TimelineItem) TimelineProps {
	props := TimelineProps{
		Items:       items,
		Width:       m.rightPaneWidth(),
		Height:      m.rightPaneHeight(),
		Focused:     m.focusedPane == rightPane,
		CompactView: m.compactView,
		LineNumbers: m.lineNumbers,
		Theme:       m.theme,
		WorkDir:     m.workDir,
	}
	m.populateThinkingState(&props)
	return props
}

// populateThinkingState sets the thinking fields on TimelineProps if the
// selected iteration is the running one and is currently thinking.
func (m *Model) populateThinkingState(props *TimelineProps) {
	runIdx := m.controller.RunningIterationIdx()
	if runIdx < 0 || m.iterList.SelectedIndex() != runIdx {
		return
	}
	iter := &m.controller.Session.Iterations[runIdx]
	if iter.IsThinking() {
		props.IsThinking = true
		props.ThinkingStartTime = iter.ThinkingStartTime
	}
}

// Subprocess management — delegates to executor

func (m *Model) spawnIteration() tea.Cmd {
	m.controller.StartIteration()
	m.iterList.OnNewIteration(len(m.controller.Session.Iterations), m.iterListHeight(), m.controller.Session.Runs)
	if m.iterList.AutoFollow.Following() {
		m.timeline.ResetPosition()
	}

	ch := m.eventCh

	eventCh, err := m.exec.Start(context.Background(), m.promptContent)
	if err != nil {
		return func() tea.Msg {
			return subprocessExitMsg{err: err}
		}
	}

	go func() {
		for evt := range eventCh {
			switch e := evt.(type) {
			case session.AssistantBatchEvent:
				ch <- assistantBatchMsg{e}
			case session.UsageEvent:
				ch <- usageMsg{e}
			case session.ToolResultEvent:
				ch <- toolResultMsg{e}
			case session.IterationEndEvent:
				ch <- iterationEndMsg{}
			case session.SubprocessExitEvent:
				ch <- subprocessExitMsg{err: e.Err}
			}
		}
	}()

	return waitForEvent(ch)
}

func waitForEvent(ch <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return <-ch
	}
}

// updateLayoutForSize adjusts pane visibility and focus based on the effective layout.
// Called on every WindowSizeMsg and at init.
func (m *Model) updateLayoutForSize() {
	layout := m.effectiveLayout()
	switch layout {
	case "bottom":
		m.leftPaneVisible = false
		// Don't force focus away from left pane sections — they still exist
		// in bottom layout as bottom bar sections.
	case "side":
		m.leftPaneVisible = true
	}
}

// effectiveLayout returns "side" or "bottom" based on config and current terminal width.
// In "auto" mode, switches to "bottom" when width < 80 columns.
func (m *Model) effectiveLayout() string {
	switch m.config.Layout {
	case "side":
		return "side"
	case "bottom":
		return "bottom"
	default: // "auto"
		if m.width > 0 && m.width < 80 {
			return "bottom"
		}
		return "side"
	}
}

// Helpers

const leftPaneFixedWidth = 32

// BottomBarHeight is the total height of the bottom bar: 3 divider lines + 6 content lines.
const BottomBarHeight = 9

// bottomBarSectionHeight is the number of content rows per section in the bottom bar.
const bottomBarSectionHeight = 2

func (m *Model) leftPaneWidth() int {
	if m.leftPaneVisible {
		return leftPaneFixedWidth
	}
	return 0
}

func (m *Model) rightPaneWidth() int {
	if m.effectiveLayout() == "bottom" {
		return m.width
	}
	lpw := m.leftPaneWidth()
	if lpw == 0 {
		return m.width
	}
	rw := m.width - lpw - 1 // 1 for separator
	if rw < 20 {
		rw = 20
	}
	return rw
}

func (m *Model) rightPaneHeight() int {
	h := m.height - 1 // subtract header
	if h < 1 {
		return 20
	}
	if m.effectiveLayout() == "bottom" && m.bottomBarVisible {
		h -= BottomBarHeight
		if h < 1 {
			h = 1
		}
	}
	return h
}

// renderBottomBar renders the bottom bar with Plans, Iterations, and Prompts sections.
// Each section has a labeled divider and 2 content rows.
func (m *Model) renderBottomBar(width int) string {
	th := m.theme

	plansDivider := renderLabeledDivider("Plans", width, th)
	plansContent := m.planList.ViewBottom(PlanListProps{
		Width:   width,
		Height:  bottomBarSectionHeight,
		Focused: m.focusedPane == plansPane,
		Theme:   th,
	})

	iterDivider := renderLabeledDivider("Iterations", width, th)
	iterContent := m.iterList.ViewBottom(IterListProps{
		Iterations: m.controller.Session.Iterations,
		Runs:       m.controller.Session.Runs,
		Width:      width,
		Height:     bottomBarSectionHeight,
		Focused:    m.focusedPane == iterationsPane,
		Theme:      th,
	})

	promptsDivider := renderLabeledDivider("Prompts", width, th)
	promptsContent := m.promptList.ViewBottom(PromptListProps{
		Width:   width,
		Height:  bottomBarSectionHeight,
		Focused: m.focusedPane == promptsPane,
		Theme:   th,
	})

	return lipgloss.JoinVertical(lipgloss.Left,
		plansDivider, plansContent,
		iterDivider, iterContent,
		promptsDivider, promptsContent,
	)
}

// renderLabeledDivider renders a "── Label ──────────" divider line.
func renderLabeledDivider(label string, width int, th theme.Theme) string {
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(th.ForegroundDim))
	nameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(th.Foreground)).Bold(true)

	styledName := nameStyle.Render(label)
	prefix := "── "
	suffix := " "

	usedWidth := lipgloss.Width(prefix) + lipgloss.Width(styledName) + lipgloss.Width(suffix)
	trailing := width - usedWidth
	if trailing < 0 {
		trailing = 0
	}

	return dimStyle.Render(prefix) + styledName + dimStyle.Render(suffix+strings.Repeat("─", trailing))
}

// selectedItems returns the timeline items for the currently selected iteration.
func (m *Model) selectedItems() []model.TimelineItem {
	idx := m.iterList.SelectedIndex()
	if idx >= len(m.controller.Session.Iterations) {
		return nil
	}
	return m.controller.Session.Iterations[idx].Items
}

// isCursorOnGroup returns true if the timeline cursor is on the group header or any of its children.
func (m *Model) isCursorOnGroup(items []model.TimelineItem, itemIdx int) bool {
	curItemIdx, _ := FlatToItem(items, m.timeline.Cursor)
	return curItemIdx == itemIdx
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func gitTickCmd() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return gitTickMsg{}
	})
}

func systemStatsReadCmd(prevActive, prevTotal int64) tea.Cmd {
	return func() tea.Msg {
		cpuSample, cpuErr := stats.ReadCPUSample()
		memPct := stats.ReadMemPercent()

		var cpuPct *int
		if cpuErr == nil && (prevActive != 0 || prevTotal != 0) {
			prev := stats.CPUSample{Active: prevActive, Total: prevTotal}
			cpuPct = stats.CPUPercent(prev, cpuSample)
		}

		var active, total int64
		if cpuErr == nil {
			active = cpuSample.Active
			total = cpuSample.Total
		}

		return systemStatsResultMsg{
			cpuPct:    cpuPct,
			memPct:    memPct,
			cpuActive: active,
			cpuTotal:  total,
		}
	}
}

func gitRefreshCmd() tea.Cmd {
	return func() tea.Msg {
		commits, err := git.LogCommits(200)
		if err != nil {
			return gitRefreshMsg{commits: nil}
		}
		return gitRefreshMsg{commits: commits}
	}
}

func gitTotalStatsCmd(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		additions, deletions, err := git.TotalStats(ctx)
		if err != nil {
			return gitTotalStatsMsg{Additions: 0, Deletions: 0}
		}
		return gitTotalStatsMsg{Additions: additions, Deletions: deletions}
	}
}
