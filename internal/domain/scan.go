package domain

import "time"

type ScanStatus string

const (
	ScanPending     ScanStatus = "pending"
	ScanRunning     ScanStatus = "running"
	ScanPassed      ScanStatus = "passed"
	ScanFailed      ScanStatus = "failed"
	ScanUnsupported ScanStatus = "unsupported"
)

type FindingSeverity string

const (
	SeverityUnknown  FindingSeverity = "unknown"
	SeverityLow      FindingSeverity = "low"
	SeverityMedium   FindingSeverity = "medium"
	SeverityHigh     FindingSeverity = "high"
	SeverityCritical FindingSeverity = "critical"
)

type ScanFinding struct {
	ID           string          `json:"id"`
	Severity     FindingSeverity `json:"severity"`
	Title        string          `json:"title"`
	Package      string          `json:"package,omitempty"`
	FixedVersion string          `json:"fixedVersion,omitempty"`
}
type ScanReport struct {
	ID          string         `json:"id"`
	Repository  RepositoryName `json:"repository"`
	Manifest    Digest         `json:"manifest"`
	Status      ScanStatus     `json:"status"`
	Findings    []ScanFinding  `json:"findings"`
	Scanner     string         `json:"scanner"`
	StartedAt   time.Time      `json:"startedAt"`
	CompletedAt *time.Time     `json:"completedAt,omitempty"`
	Error       string         `json:"error,omitempty"`
}

func (r ScanReport) Blocks(threshold FindingSeverity) bool {
	rank := map[FindingSeverity]int{SeverityUnknown: 0, SeverityLow: 1, SeverityMedium: 2, SeverityHigh: 3, SeverityCritical: 4}
	for _, f := range r.Findings {
		if rank[f.Severity] >= rank[threshold] {
			return true
		}
	}
	return false
}
