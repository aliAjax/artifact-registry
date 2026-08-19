package application

import (
	"context"
	"sort"
	"strings"
	"time"

	"artifact-registry/internal/domain"
)

type CatalogQuery struct {
	Repositories RepositoryStore
	Manifests    ManifestStore
	Tags         TagStore
	Blobs        BlobStore
	Clock        Clock
}
type RepositorySummary struct {
	Name          string    `json:"name"`
	Tenant        string    `json:"tenant"`
	ManifestCount int       `json:"manifestCount"`
	TagCount      int       `json:"tagCount"`
	BlobCount     int       `json:"blobCount"`
	Bytes         int64     `json:"bytes"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

func (q *CatalogQuery) Summarize(ctx context.Context, name string) (RepositorySummary, error) {
	r, err := q.Repositories.Get(ctx, domain.RepositoryName(name))
	if err != nil {
		return RepositorySummary{}, err
	}
	ds, err := q.Manifests.ListManifestDigests(ctx, r.Name)
	if err != nil {
		return RepositorySummary{}, err
	}
	tags, err := q.listTags(ctx, r.Name)
	if err != nil {
		return RepositorySummary{}, err
	}
	summary := RepositorySummary{Name: r.Name.String(), Tenant: r.Tenant, ManifestCount: len(ds), TagCount: len(tags), UpdatedAt: r.UpdatedAt}
	for _, d := range ds {
		m, e := q.Manifests.GetManifest(ctx, r.Name, d.String())
		if e == nil {
			summary.Bytes += m.Size
		}
	}
	blobList, e := q.Blobs.ListBlobs(ctx)
	if e == nil {
		for _, b := range blobList {
			summary.BlobCount++
			summary.Bytes += b.Size
		}
	}
	return summary, nil
}
func (q *CatalogQuery) listTags(ctx context.Context, r domain.RepositoryName) ([]*domain.Tag, error) {
	out := []*domain.Tag{}
	last := ""
	for {
		p, err := q.Tags.ListTags(ctx, r, domain.PageRequest{Limit: 1000, Last: last})
		if err != nil {
			return nil, err
		}
		out = append(out, p.Items...)
		if p.Next == "" {
			return out, nil
		}
		last = p.Next
	}
}
func (q *CatalogQuery) FindByPrefix(ctx context.Context, repo, tagPrefix string, p domain.PageRequest) (domain.Page[*domain.Tag], error) {
	r, err := domain.ParseRepositoryName(repo)
	if err != nil {
		return domain.Page[*domain.Tag]{}, err
	}
	page, err := q.Tags.ListTags(ctx, r, p)
	if err != nil {
		return page, err
	}
	filtered := page.Items[:0]
	for _, t := range page.Items {
		if strings.HasPrefix(t.Name, tagPrefix) {
			filtered = append(filtered, t)
		}
	}
	page.Items = filtered
	return page, nil
}
func (q *CatalogQuery) ManifestHistory(ctx context.Context, repo string) ([]*domain.Manifest, error) {
	r, err := domain.ParseRepositoryName(repo)
	if err != nil {
		return nil, err
	}
	ds, err := q.Manifests.ListManifestDigests(ctx, r)
	if err != nil {
		return nil, err
	}
	out := make([]*domain.Manifest, 0, len(ds))
	for _, d := range ds {
		m, e := q.Manifests.GetManifest(ctx, r, d.String())
		if e == nil {
			out = append(out, m)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}
func (q *CatalogQuery) CalculateStorage(ctx context.Context) (int64, error) {
	blobs, err := q.Blobs.ListBlobs(ctx)
	if err != nil {
		return 0, err
	}
	var total int64
	for _, b := range blobs {
		total += b.Size
	}
	return total, nil
}
func (q *CatalogQuery) HotBlobs(ctx context.Context, limit int) ([]*domain.Blob, error) {
	blobs, err := q.Blobs.ListBlobs(ctx)
	if err != nil {
		return nil, err
	}
	sort.Slice(blobs, func(i, j int) bool { return blobs[i].LastAccessedAt.After(blobs[j].LastAccessedAt) })
	if limit > 0 && len(blobs) > limit {
		blobs = blobs[:limit]
	}
	return blobs, nil
}

type DigestSet struct{ items map[domain.Digest]struct{} }

func NewDigestSet() *DigestSet                { return &DigestSet{items: map[domain.Digest]struct{}{}} }
func (s *DigestSet) Add(d domain.Digest)      { s.items[d] = struct{}{} }
func (s *DigestSet) Has(d domain.Digest) bool { _, ok := s.items[d]; return ok }
func (s *DigestSet) Delete(d domain.Digest)   { delete(s.items, d) }
func (s *DigestSet) Len() int                 { return len(s.items) }
func (s *DigestSet) List() []domain.Digest {
	out := make([]domain.Digest, 0, len(s.items))
	for d := range s.items {
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
