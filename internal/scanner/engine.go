package scanner

import (
	"context"
	"fmt"
	"time"
)

// Scanner runs a scan against one manifest and returns a report.
type Scanner interface {
	Scan(context.Context, string) (Report, error)
}

// Engine orchestrates scanning, keeps the latest report per artifact and
// records the outcome. It is the read-mostly coordinator used by the release
// gate to decide whether an artifact may be pulled.
type Engine struct {
	scanner Scanner
	reports map[string]Report
}

func NewEngine(scanner Scanner) *Engine {
	return &Engine{scanner: scanner, reports: map[string]Report{}}
}

// Scan refreshes the report for an artifact. A cancelled context must not
// leave a half-written report behind.
func (e *Engine) Scan(ctx context.Context, artifact string) (report Report, err error) {
	if err := ctx.Err(); err != nil {
		return Report{}, err
	}
	report, err = e.scanner.Scan(ctx, artifact)
	if err != nil {
		return Report{}, err
	}
	if !report.Valid() {
		return Report{}, fmt.Errorf("scanner returned invalid report for %s", artifact)
	}
	report.ScannedAt = time.Now().UTC()
	e.reports[artifact] = report
	return report, nil
}

// ScanAll scans many artifacts and stops at the first error.
func (e *Engine) ScanAll(ctx context.Context, artifacts []string) (reports []Report, err error) {
	for _, artifact := range artifacts {
		report, scanErr := e.Scan(ctx, artifact)
		if scanErr != nil {
			return nil, scanErr
		}
		reports = append(reports, report)
	}
	return reports, nil
}

// Latest returns the most recent report for an artifact.
func (e *Engine) Latest(artifact string) (Report, bool) {
	r, ok := e.reports[artifact]
	return r, ok
}

// Count returns how many distinct artifacts have a cached report.
func (e *Engine) Count() int { return len(e.reports) }
