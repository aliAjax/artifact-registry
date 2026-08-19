package scanner

import (
	"context"
	"errors"
	"testing"
)

type failingScanner struct{}

func (failingScanner) Scan(context.Context, string) (Report, error) {
	return Report{}, errors.New("scan failed")
}

func TestScanReturnsError(t *testing.T) {
	e := NewEngine(failingScanner{})
	_, err := e.Scan(context.Background(), "artifact-a")
	if err == nil {
		t.Fatal("expected scan error to be returned")
	}
}

func TestScanAllReturnsError(t *testing.T) {
	e := NewEngine(failingScanner{})
	_, err := e.ScanAll(context.Background(), []string{"artifact-a"})
	if err == nil {
		t.Fatal("expected ScanAll to return the first error")
	}
}

func TestMergeRejectsDifferentArtifacts(t *testing.T) {
	r := Report{Artifact: "a"}
	err := r.Merge(Report{Artifact: "b"})
	if err == nil {
		t.Fatal("expected merge error for different artifacts")
	}
}

func TestCombineRejectsDifferentArtifacts(t *testing.T) {
	r := Report{Artifact: "a"}
	_, err := r.Combine([]Report{{Artifact: "b"}})
	if err == nil {
		t.Fatal("expected combine error for different artifacts")
	}
}

func TestValidateFindingsReturnsError(t *testing.T) {
	r := Report{Artifact: "a", Findings: []Finding{{ID: "", Severity: SeverityLow, Description: "x"}}}
	if err := r.ValidateFindings(); err == nil {
		t.Fatal("expected validation error for empty finding id")
	}
}
