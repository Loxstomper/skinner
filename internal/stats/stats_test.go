package stats

import (
	"testing"
)

func TestParseCPUSample(t *testing.T) {
	// Real-world /proc/stat content (truncated to relevant lines)
	content := `cpu  10132153 290696 3084719 46828483 16683 0 25195 0 0 0
cpu0 1393280 32966 572056 13343292 6130 0 17875 0 0 0
`
	sample, err := ParseCPUSample(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// active = user + nice + system + irq + softirq + steal
	// = 10132153 + 290696 + 3084719 + 0 + 25195 + 0 = 13532763
	wantActive := int64(13532763)
	if sample.Active != wantActive {
		t.Errorf("Active = %d, want %d", sample.Active, wantActive)
	}
	// total = active + idle + iowait = 13532763 + 46828483 + 16683 = 60377929
	wantTotal := int64(60377929)
	if sample.Total != wantTotal {
		t.Errorf("Total = %d, want %d", sample.Total, wantTotal)
	}
}

func TestParseCPUSample_MinimalFields(t *testing.T) {
	// 7 value fields (no steal) — should still work
	content := "cpu  100 0 50 200 10 0 5\n"
	sample, err := ParseCPUSample(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// active = 100+0+50+0+5+0(no steal) = 155
	wantActive := int64(155)
	if sample.Active != wantActive {
		t.Errorf("Active = %d, want %d", sample.Active, wantActive)
	}
	// total = 155 + 200 + 10 = 365
	wantTotal := int64(365)
	if sample.Total != wantTotal {
		t.Errorf("Total = %d, want %d", sample.Total, wantTotal)
	}
}

func TestParseCPUSample_TooFewFields(t *testing.T) {
	content := "cpu  100 0 50\n"
	_, err := ParseCPUSample(content)
	if err == nil {
		t.Fatal("expected error for too few fields")
	}
}

func TestParseCPUSample_NoCPULine(t *testing.T) {
	content := "not a proc stat file\n"
	_, err := ParseCPUSample(content)
	if err == nil {
		t.Fatal("expected error for missing cpu line")
	}
}

func TestCPUPercent(t *testing.T) {
	prev := CPUSample{Active: 100, Total: 1000}
	cur := CPUSample{Active: 150, Total: 1100}
	// delta_active=50, delta_total=100 -> 50%
	pct := CPUPercent(prev, cur)
	if pct == nil {
		t.Fatal("expected non-nil percentage")
	}
	if *pct != 50 {
		t.Errorf("CPUPercent = %d, want 50", *pct)
	}
}

func TestCPUPercent_FirstSample(t *testing.T) {
	// Same sample (no delta) -> nil
	s := CPUSample{Active: 100, Total: 1000}
	pct := CPUPercent(s, s)
	if pct != nil {
		t.Errorf("expected nil for zero delta, got %d", *pct)
	}
}

func TestCPUPercent_ZeroPrevious(t *testing.T) {
	// When prev is zero-value (first sample edge case)
	prev := CPUSample{}
	cur := CPUSample{Active: 50, Total: 100}
	pct := CPUPercent(prev, cur)
	if pct == nil {
		t.Fatal("expected non-nil percentage")
	}
	if *pct != 50 {
		t.Errorf("CPUPercent = %d, want 50", *pct)
	}
}

func TestCPUPercent_HighUtilization(t *testing.T) {
	prev := CPUSample{Active: 100, Total: 1000}
	cur := CPUSample{Active: 1090, Total: 2000}
	// delta_active=990, delta_total=1000 -> 99%
	pct := CPUPercent(prev, cur)
	if pct == nil {
		t.Fatal("expected non-nil percentage")
	}
	if *pct != 99 {
		t.Errorf("CPUPercent = %d, want 99", *pct)
	}
}

func TestParseMemPercent(t *testing.T) {
	content := `MemTotal:       16384000 kB
MemFree:         2000000 kB
MemAvailable:    6553600 kB
Buffers:          500000 kB
Cached:          3000000 kB
`
	pct := ParseMemPercent(content)
	if pct == nil {
		t.Fatal("expected non-nil percentage")
	}
	// (16384000 - 6553600) / 16384000 * 100 = 60%
	if *pct != 60 {
		t.Errorf("MemPercent = %d, want 60", *pct)
	}
}

func TestParseMemPercent_HighUsage(t *testing.T) {
	content := `MemTotal:       10000 kB
MemAvailable:    1000 kB
`
	pct := ParseMemPercent(content)
	if pct == nil {
		t.Fatal("expected non-nil percentage")
	}
	// (10000 - 1000) / 10000 * 100 = 90%
	if *pct != 90 {
		t.Errorf("MemPercent = %d, want 90", *pct)
	}
}

func TestParseMemPercent_MissingFields(t *testing.T) {
	// Only MemTotal, no MemAvailable
	content := "MemTotal:       16384000 kB\n"
	pct := ParseMemPercent(content)
	if pct != nil {
		t.Errorf("expected nil for missing MemAvailable, got %d", *pct)
	}
}

func TestParseMemPercent_EmptyContent(t *testing.T) {
	pct := ParseMemPercent("")
	if pct != nil {
		t.Errorf("expected nil for empty content, got %d", *pct)
	}
}

func TestParseMemPercent_ZeroTotal(t *testing.T) {
	content := `MemTotal:       0 kB
MemAvailable:   0 kB
`
	pct := ParseMemPercent(content)
	if pct != nil {
		t.Errorf("expected nil for zero MemTotal, got %d", *pct)
	}
}
