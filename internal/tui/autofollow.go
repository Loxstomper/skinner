package tui

// AutoFollow is a small state machine that tracks whether a scrollable view
// should automatically follow new content. Both the iteration list (left pane)
// and the message timeline (right pane) use an instance.
//
// Rules:
//   - Starts in following mode.
//   - Any manual cursor movement pauses following, unless the cursor is at the end.
//   - Jumping to the end (G / End) resumes following.
//   - New items arriving do not resume following.
type AutoFollow struct {
	following bool
}

// NewAutoFollow returns an AutoFollow that starts in following mode.
func NewAutoFollow() AutoFollow {
	return AutoFollow{following: true}
}

// OnManualMove should be called after any cursor/scroll movement. If the cursor
// ended up at the end position, following continues; otherwise it pauses.
func (af *AutoFollow) OnManualMove(atEnd bool) {
	af.following = atEnd
}

// OnNewItem is called when new content arrives. It does not resume following.
func (af *AutoFollow) OnNewItem() {
	// intentional no-op
}

// JumpToEnd resumes following mode.
func (af *AutoFollow) JumpToEnd() {
	af.following = true
}

// Following reports whether the view is in auto-follow mode.
func (af AutoFollow) Following() bool {
	return af.following
}
