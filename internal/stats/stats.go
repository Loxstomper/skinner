package stats

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// CPUSample holds raw values from /proc/stat for computing deltas.
type CPUSample struct {
	Active int64
	Total  int64
}

// ReadCPUSample reads the aggregate CPU line from /proc/stat and returns
// active and total jiffies. Returns an error if the file cannot be read or parsed.
func ReadCPUSample() (CPUSample, error) {
	return ReadCPUSampleFrom("/proc/stat")
}

// ReadCPUSampleFrom reads CPU stats from a given file path (for testing).
func ReadCPUSampleFrom(path string) (CPUSample, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return CPUSample{}, err
	}
	return ParseCPUSample(string(data))
}

// ParseCPUSample parses the content of /proc/stat and extracts aggregate CPU values.
// Expected first line: cpu  user nice system idle iowait irq softirq steal [guest guest_nice]
func ParseCPUSample(content string) (CPUSample, error) {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 8 || fields[0] != "cpu" {
			continue
		}
		// Parse: user nice system idle iowait irq softirq steal
		vals := make([]int64, 0, 8)
		for _, f := range fields[1:] {
			v, err := strconv.ParseInt(f, 10, 64)
			if err != nil {
				break
			}
			vals = append(vals, v)
		}
		if len(vals) < 7 {
			return CPUSample{}, fmt.Errorf("stats: too few fields in cpu line: %d", len(vals))
		}
		// user=0 nice=1 system=2 idle=3 iowait=4 irq=5 softirq=6 steal=7(optional)
		user, nice, system, idle, iowait, irq, softirq := vals[0], vals[1], vals[2], vals[3], vals[4], vals[5], vals[6]
		var steal int64
		if len(vals) > 7 {
			steal = vals[7]
		}
		active := user + nice + system + irq + softirq + steal
		total := active + idle + iowait
		return CPUSample{Active: active, Total: total}, nil
	}
	return CPUSample{}, fmt.Errorf("stats: no aggregate cpu line found")
}

// CPUPercent computes CPU utilization percentage from two samples.
// Returns nil if the delta is zero (e.g. first sample).
func CPUPercent(prev, cur CPUSample) *int {
	deltaTotal := cur.Total - prev.Total
	if deltaTotal <= 0 {
		return nil
	}
	deltaActive := cur.Active - prev.Active
	pct := int((deltaActive * 100) / deltaTotal)
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	return &pct
}

// ReadMemPercent reads /proc/meminfo and returns memory utilization percentage.
// Returns nil if the file cannot be read or required fields are missing.
func ReadMemPercent() *int {
	return ReadMemPercentFrom("/proc/meminfo")
}

// ReadMemPercentFrom reads memory stats from a given file path (for testing).
func ReadMemPercentFrom(path string) *int {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return ParseMemPercent(string(data))
}

// ParseMemPercent parses /proc/meminfo content and computes memory utilization.
// Memory% = ((MemTotal - MemAvailable) / MemTotal) * 100
func ParseMemPercent(content string) *int {
	var memTotal, memAvailable int64
	var foundTotal, foundAvailable bool

	for _, line := range strings.Split(content, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		switch fields[0] {
		case "MemTotal:":
			v, err := strconv.ParseInt(fields[1], 10, 64)
			if err == nil {
				memTotal = v
				foundTotal = true
			}
		case "MemAvailable:":
			v, err := strconv.ParseInt(fields[1], 10, 64)
			if err == nil {
				memAvailable = v
				foundAvailable = true
			}
		}
		if foundTotal && foundAvailable {
			break
		}
	}

	if !foundTotal || !foundAvailable || memTotal <= 0 {
		return nil
	}

	pct := int(((memTotal - memAvailable) * 100) / memTotal)
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	return &pct
}
