package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"artifact-registry/internal/domain"
)

func TestSubmitScanWrapsNotImplemented(t *testing.T) {
	c, _ := newTestCatalog2(t)
	_ = c.Create(context.Background(), &domain.Repository{Name: domain.RepositoryName("repo-a"), Tenant: "t", Version: 1})
	raw := []byte(`{"schemaVersion":2}`)
	m, _ := domain.NewManifest(raw, "application/vnd.oci.image.manifest.v1+json", time.Now())
	m.State = domain.LifecycleAvailable
	_ = c.PutManifest(context.Background(), domain.RepositoryName("repo-a"), m)
	s := &LifecycleService{Manifests: c, Tags: c, Repositories: c, Blobs: c, Policies: c, Scans: c, Replications: c, Scanner: nil, Replicator: StaticReplicator{}, Clock: SystemClock{}, Audits: c}
	_, err := s.SubmitScan(context.Background(), "repo-a", m.Digest.String())
	if !errors.Is(err, domain.ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented, got %v", err)
	}
}
