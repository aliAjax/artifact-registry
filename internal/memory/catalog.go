// Package memory supplies a concurrency-safe catalog for local development and tests.
package memory

import (
	"context"
	"io"
	"sort"
	"strings"
	"sync"

	"artifact-registry/internal/domain"
)

type Catalog struct {
	mu           sync.RWMutex
	repos        map[domain.RepositoryName]*domain.Repository
	blobs        map[domain.Digest]*domain.Blob
	manifests    map[domain.RepositoryName]map[domain.Digest]*domain.Manifest
	tags         map[domain.RepositoryName]map[string]*domain.Tag
	uploads      map[string]*domain.UploadSession
	quotas       map[string]*domain.Quota
	audits       []domain.AuditEntry
	scans        map[string]domain.ScanReport
	replications map[string]domain.ReplicationTask
	policies     map[string]domain.RetentionPolicy
	object       *LocalObjectStore
}

func NewCatalog(object *LocalObjectStore) *Catalog {
	return &Catalog{repos: map[domain.RepositoryName]*domain.Repository{}, blobs: map[domain.Digest]*domain.Blob{}, manifests: map[domain.RepositoryName]map[domain.Digest]*domain.Manifest{}, tags: map[domain.RepositoryName]map[string]*domain.Tag{}, uploads: map[string]*domain.UploadSession{}, quotas: map[string]*domain.Quota{}, scans: map[string]domain.ScanReport{}, replications: map[string]domain.ReplicationTask{}, policies: map[string]domain.RetentionPolicy{}, object: object}
}
func clone[T any](v T) T                               { return v }
func copyRepo(v *domain.Repository) *domain.Repository { x := *v; return &x }
func copyBlob(v *domain.Blob) *domain.Blob             { x := *v; return &x }
func copyManifest(v *domain.Manifest) *domain.Manifest {
	x := *v
	x.Raw = append([]byte(nil), v.Raw...)
	x.Layers = append([]domain.Descriptor(nil), v.Layers...)
	x.Manifests = append([]domain.Descriptor(nil), v.Manifests...)
	return &x
}
func copyTag(v *domain.Tag) *domain.Tag { x := *v; return &x }
func copyUpload(v *domain.UploadSession) *domain.UploadSession {
	x := *v
	x.Chunks = append([]domain.Chunk(nil), v.Chunks...)
	return &x
}
func (c *Catalog) Create(_ context.Context, v *domain.Repository) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.repos[v.Name]; ok {
		return domain.ConflictError{Resource: "repository", Detail: "already exists"}
	}
	c.repos[v.Name] = copyRepo(v)
	return nil
}
func (c *Catalog) Get(_ context.Context, name domain.RepositoryName) (*domain.Repository, error) {
	v, ok := c.repos[name]
	if !ok {
		return nil, domain.ErrManifestNotFound
	}
	return v, nil
}
func (c *Catalog) List(_ context.Context, prefix string, p domain.PageRequest) (domain.Page[*domain.Repository], error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	p = p.Normalized()
	names := make([]string, 0)
	for n := range c.repos {
		if strings.HasPrefix(n.String(), prefix) && n.String() > p.Last {
			names = append(names, n.String())
		}
	}
	sort.Strings(names)
	out := domain.Page[*domain.Repository]{}
	for _, n := range names {
		if len(out.Items) == p.Limit {
			out.Next = n
			break
		}
		out.Items = append(out.Items, copyRepo(c.repos[domain.RepositoryName(n)]))
	}
	return out, nil
}
func (c *Catalog) Update(_ context.Context, v *domain.Repository, version int64) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	old, ok := c.repos[v.Name]
	if !ok {
		return domain.ErrManifestNotFound
	}
	if old.Version != version {
		return domain.ErrPreconditionFailed
	}
	c.repos[v.Name] = copyRepo(v)
	return nil
}
func (c *Catalog) GetBlob(_ context.Context, d domain.Digest) (*domain.Blob, error) {
	v, ok := c.blobs[d]
	if !ok {
		return nil, domain.ErrBlobNotFound
	}
	return v, nil
}
func (c *Catalog) PutBlob(ctx context.Context, v *domain.Blob, r io.Reader) error {
	c.mu.Lock()
	if old, ok := c.blobs[v.Digest]; ok {
		c.mu.Unlock()
		if old.Size != v.Size {
			return domain.ConflictError{Resource: "blob", Detail: "digest has inconsistent size"}
		}
		return nil
	}
	c.mu.Unlock()
	if err := c.object.Put(ctx, v.StorageKey, r); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.blobs[v.Digest]; !ok {
		c.blobs[v.Digest] = copyBlob(v)
	}
	return nil
}
func (c *Catalog) OpenBlob(ctx context.Context, d domain.Digest, r domain.ByteRange) (io.ReadCloser, *domain.Blob, error) {
	c.mu.RLock()
	v, ok := c.blobs[d]
	if !ok {
		c.mu.RUnlock()
		return nil, nil, domain.ErrBlobNotFound
	}
	b := copyBlob(v)
	c.mu.RUnlock()
	stream, _, err := c.object.Open(ctx, b.StorageKey, r)
	return stream, b, err
}
func (c *Catalog) DeleteBlob(ctx context.Context, d domain.Digest) error {
	c.mu.Lock()
	v, ok := c.blobs[d]
	if !ok {
		c.mu.Unlock()
		return domain.ErrBlobNotFound
	}
	if v.References > 0 {
		c.mu.Unlock()
		return domain.ConflictError{Resource: "blob", Detail: "referenced"}
	}
	delete(c.blobs, d)
	c.mu.Unlock()
	return c.object.Delete(ctx, v.StorageKey)
}
func (c *Catalog) ListBlobs(_ context.Context) ([]*domain.Blob, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]*domain.Blob, 0, len(c.blobs))
	for _, v := range c.blobs {
		out = append(out, copyBlob(v))
	}
	return out, nil
}
func (c *Catalog) PutManifest(_ context.Context, repo domain.RepositoryName, v *domain.Manifest) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.repos[repo]; !ok {
		return domain.ErrManifestNotFound
	}
	if c.manifests[repo] == nil {
		c.manifests[repo] = map[domain.Digest]*domain.Manifest{}
	}
	if _, ok := c.manifests[repo][v.Digest]; ok {
		return nil
	}
	c.manifests[repo][v.Digest] = copyManifest(v)
	return nil
}
func (c *Catalog) GetManifest(_ context.Context, repo domain.RepositoryName, ref string) (*domain.Manifest, error) {
	if t := c.tags[repo][ref]; t != nil {
		ref = t.Digest.String()
	}
	d, err := domain.ParseDigest(ref)
	if err != nil {
		return nil, domain.ErrManifestNotFound
	}
	v := c.manifests[repo][d]
	if v == nil {
		return nil, domain.ErrManifestNotFound
	}
	return copyManifest(v), nil
}
func (c *Catalog) ListManifestDigests(_ context.Context, repo domain.RepositoryName) ([]domain.Digest, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]domain.Digest, 0, len(c.manifests[repo]))
	for d := range c.manifests[repo] {
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out, nil
}
func (c *Catalog) ListAllManifests(_ context.Context) (map[domain.RepositoryName][]*domain.Manifest, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := map[domain.RepositoryName][]*domain.Manifest{}
	for r, entries := range c.manifests {
		for _, m := range entries {
			out[r] = append(out[r], copyManifest(m))
		}
	}
	return out, nil
}
func (c *Catalog) DeleteManifest(_ context.Context, repo domain.RepositoryName, d domain.Digest) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.manifests[repo][d]; !ok {
		return domain.ErrManifestNotFound
	}
	delete(c.manifests[repo], d)
	for n, t := range c.tags[repo] {
		if t.Digest == d {
			delete(c.tags[repo], n)
		}
	}
	return nil
}
func (c *Catalog) FindReferrers(_ context.Context, repo domain.RepositoryName, subject domain.Digest, artifactType string) ([]*domain.Manifest, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := []*domain.Manifest{}
	for _, m := range c.manifests[repo] {
		if m.Subject != nil && m.Subject.Digest == subject && (artifactType == "" || m.ArtifactType == artifactType) {
			out = append(out, copyManifest(m))
		}
	}
	return out, nil
}
