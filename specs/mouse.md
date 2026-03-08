# Mouse Support

Mouse support allows scroll-wheel scrolling and click-to-select in both panes.

## Mouse Mode

The TUI enables cell-motion mouse tracking (`tea.WithMouseCellMotion()`). This reports button presses, releases, and scroll-wheel events with cell coordinates.

## Scroll Wheel

- **Target pane**: determined by the X coordinate of the event — left of the separator (column 32) targets the left pane, right targets the right pane.
- **Scroll amount**: 3 lines per wheel tick.
- **Focus switch**: scrolling a pane switches focus to that pane.
- **Auto-follow**: any mouse scroll calls `AutoFollow.OnManualMove()` to pause auto-follow.

## Click

- **Target pane**: same X-coordinate rule as scroll.
- **Button**: only left-click (`MouseButtonLeft` with `MouseActionPress`) is handled; other buttons are ignored.
- **Focus switch**: clicking a pane switches focus to that pane.
- **Row mapping**: subtract the header height (1 line) from `msg.Y` to get the pane-relative row. Add the pane's scroll offset to get the absolute row index.
- **Left pane (iteration list)**: sets cursor to `scroll + row` if within the valid iteration range. Resets the timeline position. Ignores clicks beyond the last iteration.
- **Right pane (timeline)**: maps `scroll + row` to a flat cursor position using `LineToFlatCursor`. Sets cursor if valid. Ignores clicks beyond the last item. If the cursor was already on the clicked row, the click toggles expand/collapse — identical to pressing `enter` (see below).
- **Auto-follow**: any click calls `AutoFollow.OnManualMove()` to pause auto-follow.
- **Empty space**: clicks beyond the last row in either pane are ignored (no cursor change).

## Click-to-Expand (Right Pane)

Clicking a row that is already selected triggers the same action as `enter`:

- **Text block**: toggles expanded/collapsed.
- **Tool call (collapsed)**: expands it.
- **Tool call (expanded, small content)**: collapses it.
- **Tool call (expanded, large content)**: enters sub-scroll mode (see [sub-scroll.md](sub-scroll.md)).
- **Tool call group header**: toggles group expand/collapse.
- **Tool call within expanded group**: behaves like a standalone tool call (above).

This allows touchscreen and mouse-only users to fully navigate and interact with the timeline without a keyboard.

## Header

- Clicks on the header row (Y == 0) are ignored.
- Scroll events on the header row are ignored.

## Sub-Scroll Mode

Clicking works while sub-scroll mode is active:

- **Click inside the expanded content area**: ignored (scroll via `j`/`k` or swipe).
- **Click the summary row of the sub-scrolled item**: exits sub-scroll mode and collapses the item.
- **Click any other row in the timeline**: exits sub-scroll mode and selects the clicked row.

This provides a touch-friendly way to exit sub-scroll without the `escape` key.
