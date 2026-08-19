package scanner

import (
	"sort"
	"strings"
	"time"
)

// Severity orders scan findings so the most critical item drives the release
// decision. The numeric rank is stable and never changes between releases.
type Severity string

const (
	SeverityUnknown   Severity = "unknown"
	SeverityLow       Severity = "low"
	SeverityMedium    Severity = "medium"
	SeverityHigh      Severity = "high"
	SeverityCritical  Severity = "critical"
)

func (s Severity) Rank() int {
	switch s {
	case SeverityCritical:
		return 4
	case SeverityHigh:
		return 3
	case SeverityMedium:
		return 2
	case SeverityLow:
		return 1
	default:
		return 0
	}
}

// Finding is one vulnerability observed for a single artifact.
type Finding struct {
	ID          string
	Severity    Severity
	Description string
}

// Report summarizes the findings produced by an external scanner for one
// manifest. It is immutable once published.
type Report struct {
	Artifact  string
	Findings  []Finding
	ScannedAt time.Time
}

func (r Report) HighestSeverity() Severity {
	best := SeverityUnknown
	for _, f := range r.Findings {
		if f.Severity.Rank() > best.Rank() {
			best = f.Severity
		}
	}
	return best
}

func (r Report) CountBySeverity(sev Severity) int {
	n := 0
	for _, f := range r.Findings {
		if f.Severity == sev {
			n++
		}
	}
	return n
}

// SortedFindings returns findings ordered from most to least severe. The
// returned slice must be independent from the report so callers can reorder it
// without corrupting the immutable report.
func (r Report) SortedFindings() []Finding {
	out := append([]Finding(nil), r.Findings...)
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Severity.Rank() > out[j].Severity.Rank()
	})
	return out
}

func (r Report) Valid() bool {
	if strings.TrimSpace(r.Artifact) == "" {
		return false
	}
	for _, f := range r.Findings {
		if strings.TrimSpace(f.ID) == "" || strings.TrimSpace(f.Description) == "" {
			return false
		}
	}
	return true
}
