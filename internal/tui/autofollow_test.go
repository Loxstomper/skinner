package tui

import "testing"

func TestNewAutoFollow_StartsFollowing(t *testing.T) {
	af := NewAutoFollow()
	if !af.Following() {
		t.Error("NewAutoFollow should start in following mode")
	}
}

func TestOnManualMove_PausesWhenNotAtEnd(t *testing.T) {
	af := NewAutoFollow()
	af.OnManualMove(false)
	if af.Following() {
		t.Error("OnManualMove(false) should pause following")
	}
}

func TestOnManualMove_KeepsFollowingAtEnd(t *testing.T) {
	af := NewAutoFollow()
	af.OnManualMove(true)
	if !af.Following() {
		t.Error("OnManualMove(true) should keep following")
	}
}

func TestOnManualMove_ResumesAtEnd(t *testing.T) {
	af := NewAutoFollow()
	af.OnManualMove(false) // pause
	af.OnManualMove(true)  // arrive at end
	if !af.Following() {
		t.Error("OnManualMove(true) should resume following after pause")
	}
}

func TestJumpToEnd_ResumesFollowing(t *testing.T) {
	af := NewAutoFollow()
	af.OnManualMove(false) // pause
	af.JumpToEnd()
	if !af.Following() {
		t.Error("JumpToEnd should resume following")
	}
}

func TestOnNewItem_DoesNotResumeFollowing(t *testing.T) {
	af := NewAutoFollow()
	af.OnManualMove(false) // pause
	af.OnNewItem()
	if af.Following() {
		t.Error("OnNewItem should not resume following")
	}
}

func TestOnNewItem_DoesNotDisruptFollowing(t *testing.T) {
	af := NewAutoFollow()
	af.OnNewItem()
	if !af.Following() {
		t.Error("OnNewItem should not disrupt active following")
	}
}

func TestSequence_PauseJumpPause(t *testing.T) {
	af := NewAutoFollow()

	af.OnManualMove(false) // pause
	if af.Following() {
		t.Error("step 1: should be paused")
	}

	af.JumpToEnd() // resume
	if !af.Following() {
		t.Error("step 2: should be following after JumpToEnd")
	}

	af.OnManualMove(false) // pause again
	if af.Following() {
		t.Error("step 3: should be paused again")
	}
}
