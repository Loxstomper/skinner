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
- **Right pane (timeline)**: maps `scroll + row` to a flat cursor position using `LineToFlatCursor`. Sets cursor if valid. Ignores clicks beyond the last item.
- **Auto-follow**: any click calls `AutoFollow.OnManualMove()` to pause auto-follow.
- **Empty space**: clicks beyond the last row in either pane are ignored (no cursor change).

## Header

- Clicks on the header row (Y == 0) are ignored.
- Scroll events on the header row are ignored.

## Collapsed Groups

- Clicking a collapsed group header selects it (moves cursor to the header) but does not expand it. Expansion is only triggered by the Enter key.
