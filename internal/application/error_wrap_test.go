package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"artifact-registry/internal/domain"
	"artifact-registry/internal/memory"
)

func newTestCatalog2(t *testing.T) (*memory.Catalog, *memory.LocalObjectStore) {
	t.Helper()
	object, err := memory.NewLocalObjectStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return memory.NewCatalog(object), object
}

func newTestRegistry2(t *testing.T) (*RegistryService, *memory.Catalog) {
	t.Helper()
	c, object := newTestCatalog2(t)
	s := &RegistryService{Repositories: c, Blobs: c, Manifests: c, Tags: c, Uploads: c, Quotas: c, Audits: c, ObjectStore: object, Events: NoopPublisher{}, Clock: SystemClock{}, UploadTTL: time.Hour, MaxBlobSize: 1 << 30, DefaultQuota: 1 << 30}
	return s, c
}

func TestGetManifestWrapsNotFound(t *testing.T) {
	s, _ := newTestRegistry2(t)
	_, err := s.GetManifest(context.Background(), "repo-a", "sha256:1111111111111111111111111111111111111111111111111111111111111111")
	if !errors.Is(err, domain.ErrManifestNotFound) {
		t.Fatalf("expected ErrManifestNotFound, got %v", err)
	}
}

func TestGetManifestWrapsBlocked(t *testing.T) {
	s, c := newTestRegistry2(t)
	raw := []byte(`{"schemaVersion":2}`)
	m, _ := domain.NewManifest(raw, "application/vnd.oci.image.manifest.v1+json", time.Now())
	_ = c.Create(context.Background(), &domain.Repository{Name: domain.RepositoryName("repo-a"), Tenant: "t", Version: 1})
	m.State = domain.LifecycleBlocked
	_ = c.PutManifest(context.Background(), domain.RepositoryName("repo-a"), m)
	_, err := s.GetManifest(context.Background(), "repo-a", m.Digest.String())
	if !errors.Is(err, domain.ErrManifestBlocked) {
		t.Fatalf("expected ErrManifestBlocked, got %v", err)
	}
}

func TestCreateRepositoryWrapsInvalidName(t *testing.T) {
	s, _ := newTestRegistry2(t)
	_, err := s.CreateRepository(context.Background(), CreateRepositoryCommand{Name: "bad!"})
	if !errors.Is(err, domain.ErrInvalidRepository) {
		t.Fatalf("expected ErrInvalidRepositoryName, got %v", err)
	}
}
