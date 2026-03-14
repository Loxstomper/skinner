package tui

import (
	"os"
	"time"
)

// RenderCache is a single-slot cache for expensive file rendering operations.
// It eliminates per-frame re-rendering of markdown (glamour) and source code
// content that hasn't changed. Since only one file preview or plan view is
// visible at a time, a single slot is sufficient.
type RenderCache struct {
	path    string
	modTime time.Time
	width   int
	lines   []string
}

// Get returns the cached rendered lines if the cache is valid for the given
// path and width. It calls os.Stat to verify the file's modification time
// hasn't changed. Returns (lines, true) on hit, (nil, false) on miss.
func (c *RenderCache) Get(path string, width int) ([]string, bool) {
	if c == nil {
		return nil, false
	}
	if c.path != path || c.width != width {
		return nil, false
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, false
	}
	if !info.ModTime().Equal(c.modTime) {
		return nil, false
	}
	return c.lines, true
}

// Set stores rendered lines in the cache along with the cache key fields
// (path, modTime, width). Called after a cache miss produces new content.
func (c *RenderCache) Set(path string, modTime time.Time, width int, lines []string) {
	if c == nil {
		return
	}
	c.path = path
	c.modTime = modTime
	c.width = width
	c.lines = lines
}
