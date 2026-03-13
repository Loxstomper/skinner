# System Stats

## Overview

The header bar displays system-wide CPU and memory utilization as compact, color-tinted indicators. Stats are always visible — even when idle or between iterations — since they reflect host state, not Claude session state.

## Display Format

```
⚙ 42% ◼ 61%
```

- **`⚙ N%`** — CPU utilization (Unicode U+2699 GEAR).
- **`◼ N%`** — Memory utilization (Unicode U+25FC BLACK MEDIUM SQUARE).

The two indicators are separated by a single space. No labels beyond the icons.

## Header Placement

Positioned on the far right of the header bar, after the iteration indicator. Separated from the iteration indicator by three spaces.

**Idle state**:

```
                                    ⏱ --                            Idle   ⚙ 42% ◼ 61%
```

**Running state**:

```
          ⏱ 14m32s   ↑42.1k ↓8.3k tokens   ctx 62%   ~$1.24   5h: 34%  wk: 12%   Iter 3/10 ⟳   ⚙ 42% ◼ 61%
```

## Color Thresholds

Both CPU and memory percentages are colored by severity using existing theme roles:

| Range  | Color          |
|--------|----------------|
| 0–49%  | `StatusSuccess` (green) |
| 50–79% | `StatusRunning` (yellow/warning) |
| 80–100%| `StatusError`  (red) |

The icon inherits the same color as its percentage value. When no data is available yet (first 2 seconds), display `⚙ --% ◼ --%` in `ForegroundDim`.

## Data Source

Read directly from Linux `/proc` filesystem. No external dependencies.

### CPU — `/proc/stat`

Calculate CPU utilization from the first line of `/proc/stat` (`cpu` aggregate):

```
cpu  user nice system idle iowait irq softirq steal guest guest_nice
```

1. On each poll, read the values and compute:
   - `active = user + nice + system + irq + softirq + steal`
   - `total = active + idle + iowait`
2. CPU% = `(delta_active / delta_total) * 100` between the current and previous sample.
3. First sample shows `--` since a delta requires two readings.

### Memory — `/proc/meminfo`

Parse `MemTotal` and `MemAvailable` from `/proc/meminfo`:

```
Memory% = ((MemTotal - MemAvailable) / MemTotal) * 100
```

This matches the `htop` default — used memory excluding reclaimable buffers/cache.

## Polling

- **Interval**: every 2 seconds.
- **Mechanism**: A Bubble Tea `tickMsg` (separate from existing ticks) fires every 2 seconds, triggering a re-read of `/proc/stat` and `/proc/meminfo`.
- **Startup**: First tick fires immediately on program start. CPU shows `--` until the second tick (needs two samples for delta).
- **Cost**: Reading two small proc files every 2s is negligible.

## Configuration

No configuration options. The feature is always enabled. If `/proc/stat` or `/proc/meminfo` cannot be read (e.g. non-Linux platform), the stats section is hidden entirely rather than showing errors.
