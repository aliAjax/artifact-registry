package application

import (
	"artifact-registry/internal/domain"
	"context"
	"fmt"
	"time"
)

type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now() }

type NoopPublisher struct{}

func (NoopPublisher) Publish(context.Context, string, any) error { return nil }

type DevelopmentScanner struct{ Clock Clock }

func (s DevelopmentScanner) Submit(_ context.Context, repo domain.RepositoryName, m *domain.Manifest) (domain.ScanReport, error) {
	now := time.Now().UTC()
	if s.Clock != nil {
		now = s.Clock.Now().UTC()
	}
	return domain.ScanReport{ID: fmt.Sprintf("scan-%s", m.Digest.Hex()[:12]), Repository: repo, Manifest: m.Digest, Status: domain.ScanPassed, Scanner: "development-policy", StartedAt: now, CompletedAt: &now}, nil
}

type StaticReplicator struct{}

func (StaticReplicator) Copy(_ context.Context, t domain.ReplicationTask) (int64, error) {
	return t.BytesCopied, nil
}
