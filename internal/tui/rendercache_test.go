package tui

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRenderCacheMissOnEmpty(t *testing.T) {
	c := &RenderCache{}
	lines, ok := c.Get("/some/path", 80)
	if ok {
		t.Fatal("expected cache miss on empty cache")
	}
	if lines != nil {
		t.Fatal("expected nil lines on miss")
	}
}

func TestRenderCacheHitAfterSet(t *testing.T) {
	c := &RenderCache{}

	// Create a temp file so os.Stat works
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	expected := []string{"rendered line 1", "rendered line 2"}
	c.Set(path, info.ModTime(), 80, expected)

	lines, ok := c.Get(path, 80)
	if !ok {
		t.Fatal("expected cache hit")
	}
	if len(lines) != len(expected) {
		t.Fatalf("expected %d lines, got %d", len(expected), len(lines))
	}
	for i, line := range lines {
		if line != expected[i] {
			t.Fatalf("line %d: expected %q, got %q", i, expected[i], line)
		}
	}
}

func TestRenderCacheMissOnPathChange(t *testing.T) {
	c := &RenderCache{}

	dir := t.TempDir()
	path1 := filepath.Join(dir, "file1.md")
	path2 := filepath.Join(dir, "file2.md")
	if err := os.WriteFile(path1, []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path2, []byte("b"), 0o644); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path1)
	if err != nil {
		t.Fatal(err)
	}

	c.Set(path1, info.ModTime(), 80, []string{"line"})

	_, ok := c.Get(path2, 80)
	if ok {
		t.Fatal("expected cache miss when path changes")
	}
}

func TestRenderCacheMissOnWidthChange(t *testing.T) {
	c := &RenderCache{}

	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	c.Set(path, info.ModTime(), 80, []string{"line"})

	_, ok := c.Get(path, 120)
	if ok {
		t.Fatal("expected cache miss when width changes")
	}
}

func TestRenderCacheMissOnModTimeChange(t *testing.T) {
	c := &RenderCache{}

	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	if err := os.WriteFile(path, []byte("v1"), 0o644); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	c.Set(path, info.ModTime(), 80, []string{"old"})

	// Modify the file to change its mod time
	// Ensure mod time actually changes (some filesystems have 1s granularity)
	time.Sleep(10 * time.Millisecond)
	newTime := time.Now().Add(time.Second)
	if err := os.Chtimes(path, newTime, newTime); err != nil {
		t.Fatal(err)
	}

	_, ok := c.Get(path, 80)
	if ok {
		t.Fatal("expected cache miss when file modtime changes")
	}
}

func TestRenderCacheMissOnDeletedFile(t *testing.T) {
	c := &RenderCache{}

	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	c.Set(path, info.ModTime(), 80, []string{"line"})

	// Delete the file
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}

	_, ok := c.Get(path, 80)
	if ok {
		t.Fatal("expected cache miss when file is deleted")
	}
}

func TestRenderCacheNilSafe(t *testing.T) {
	var c *RenderCache

	// Get on nil should return miss, not panic
	lines, ok := c.Get("/path", 80)
	if ok {
		t.Fatal("expected miss on nil cache")
	}
	if lines != nil {
		t.Fatal("expected nil lines on nil cache")
	}

	// Set on nil should not panic
	c.Set("/path", time.Now(), 80, []string{"line"})
}
