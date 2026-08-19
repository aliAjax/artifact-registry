package memory

import (
	"context"
	"sort"
	"strings"
	"time"

	"artifact-registry/internal/domain"
)

func (c *Catalog) PutTag(_ context.Context, repo domain.RepositoryName, v *domain.Tag) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.tags[repo] == nil {
		c.tags[repo] = map[string]*domain.Tag{}
	}
	c.tags[repo][v.Name] = copyTag(v)
	return nil
}
func (c *Catalog) GetTag(_ context.Context, repo domain.RepositoryName, name string) (*domain.Tag, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v := c.tags[repo][name]
	if v == nil {
		return nil, domain.ErrTagNotFound
	}
	return copyTag(v), nil
}
func (c *Catalog) ListTags(_ context.Context, repo domain.RepositoryName, p domain.PageRequest) (domain.Page[*domain.Tag], error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	p = p.Normalized()
	names := make([]string, 0, len(c.tags[repo]))
	for n := range c.tags[repo] {
		if n > p.Last {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	out := domain.Page[*domain.Tag]{}
	for _, n := range names {
		if len(out.Items) >= p.Limit {
			out.Next = n
			break
		}
		out.Items = append(out.Items, copyTag(c.tags[repo][n]))
	}
	return out, nil
}
func (c *Catalog) DeleteTag(_ context.Context, repo domain.RepositoryName, name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.tags[repo][name]; !ok {
		return domain.ErrTagNotFound
	}
	delete(c.tags[repo], name)
	return nil
}
func (c *Catalog) CreateUpload(_ context.Context, v *domain.UploadSession) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.uploads[v.ID]; ok {
		return domain.ConflictError{Resource: "upload", Detail: "already exists"}
	}
	c.uploads[v.ID] = copyUpload(v)
	return nil
}
func (c *Catalog) GetUpload(_ context.Context, id string) (*domain.UploadSession, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v := c.uploads[id]
	if v == nil {
		return nil, domain.ErrUploadNotFound
	}
	return copyUpload(v), nil
}
func (c *Catalog) UpdateUpload(_ context.Context, v *domain.UploadSession, version int64) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	old := c.uploads[v.ID]
	if old == nil {
		return domain.ErrUploadNotFound
	}
	if old.Version != version {
		return domain.ErrPreconditionFailed
	}
	c.uploads[v.ID] = copyUpload(v)
	return nil
}
func (c *Catalog) DeleteUpload(ctx context.Context, id string) error {
	c.mu.Lock()
	v := c.uploads[id]
	if v == nil {
		c.mu.Unlock()
		return domain.ErrUploadNotFound
	}
	delete(c.uploads, id)
	c.mu.Unlock()
	return c.object.Delete(ctx, v.TempKey)
}
func (c *Catalog) ListExpiredUploads(_ context.Context, now time.Time) ([]*domain.UploadSession, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := []*domain.UploadSession{}
	for _, v := range c.uploads {
		if v.IsExpired(now) {
			out = append(out, copyUpload(v))
		}
	}
	return out, nil
}
func (c *Catalog) GetQuota(_ context.Context, tenant string) (*domain.Quota, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v := c.quotas[tenant]
	if v == nil {
		return nil, domain.ErrManifestNotFound
	}
	x := *v
	return &x, nil
}
func (c *Catalog) PutQuota(_ context.Context, v *domain.Quota) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	x := *v
	c.quotas[v.Tenant] = &x
	return nil
}
func (c *Catalog) UpdateQuota(_ context.Context, v *domain.Quota, version int64) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	old := c.quotas[v.Tenant]
	if old == nil {
		return domain.ErrManifestNotFound
	}
	if old.Version != version {
		return domain.ErrPreconditionFailed
	}
	x := *v
	c.quotas[v.Tenant] = &x
	return nil
}
func (c *Catalog) Append(_ context.Context, v domain.AuditEntry) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.audits = append(c.audits, v)
	return nil
}
func (c *Catalog) ListAudit(_ context.Context, resource string, p domain.PageRequest) (domain.Page[domain.AuditEntry], error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	p = p.Normalized()
	out := domain.Page[domain.AuditEntry]{}
	for _, v := range c.audits {
		if resource != "" && !strings.HasPrefix(v.Resource, resource) {
			continue
		}
		out.Items = append(out.Items, v)
		if len(out.Items) == p.Limit {
			break
		}
	}
	return out, nil
}
func (c *Catalog) CreateScan(_ context.Context, v domain.ScanReport) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.scans[v.ID]; ok {
		return domain.ConflictError{Resource: "scan", Detail: "exists"}
	}
	c.scans[v.ID] = v
	return nil
}
func (c *Catalog) GetScan(_ context.Context, id string) (domain.ScanReport, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.scans[id]
	if !ok {
		return domain.ScanReport{}, domain.ErrManifestNotFound
	}
	return v, nil
}
func (c *Catalog) ListScans(_ context.Context, repo domain.RepositoryName, p domain.PageRequest) (domain.Page[domain.ScanReport], error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	p = p.Normalized()
	out := domain.Page[domain.ScanReport]{}
	for _, v := range c.scans {
		if repo != "" && v.Repository != repo {
			continue
		}
		out.Items = append(out.Items, v)
	}
	sort.Slice(out.Items, func(i, j int) bool { return out.Items[i].StartedAt.After(out.Items[j].StartedAt) })
	if len(out.Items) > p.Limit {
		out.Next = out.Items[p.Limit].ID
		out.Items = out.Items[:p.Limit]
	}
	return out, nil
}
func (c *Catalog) UpdateScan(_ context.Context, v domain.ScanReport) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.scans[v.ID]; !ok {
		return domain.ErrManifestNotFound
	}
	c.scans[v.ID] = v
	return nil
}
func (c *Catalog) CreateReplication(_ context.Context, v domain.ReplicationTask) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.replications[v.ID] = v
	return nil
}
func (c *Catalog) GetReplication(_ context.Context, id string) (domain.ReplicationTask, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.replications[id]
	if !ok {
		return domain.ReplicationTask{}, domain.ErrManifestNotFound
	}
	return v, nil
}
func (c *Catalog) ListReplications(_ context.Context, p domain.PageRequest) (domain.Page[domain.ReplicationTask], error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	p = p.Normalized()
	out := domain.Page[domain.ReplicationTask]{}
	for _, v := range c.replications {
		out.Items = append(out.Items, v)
	}
	sort.Slice(out.Items, func(i, j int) bool { return out.Items[i].CreatedAt.After(out.Items[j].CreatedAt) })
	if len(out.Items) > p.Limit {
		out.Next = out.Items[p.Limit].ID
		out.Items = out.Items[:p.Limit]
	}
	return out, nil
}
func (c *Catalog) UpdateReplication(_ context.Context, v domain.ReplicationTask, version int64) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	old, ok := c.replications[v.ID]
	if !ok {
		return domain.ErrManifestNotFound
	}
	if old.Version != version {
		return domain.ErrPreconditionFailed
	}
	c.replications[v.ID] = v
	return nil
}
func (c *Catalog) PutPolicy(_ context.Context, v domain.RetentionPolicy) error {
	if err := v.Validate(); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.policies[v.ID] = v
	return nil
}
func (c *Catalog) ListPolicies(_ context.Context) ([]domain.RetentionPolicy, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]domain.RetentionPolicy, 0, len(c.policies))
	for _, v := range c.policies {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}
func (c *Catalog) DeletePolicy(_ context.Context, id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.policies[id]; !ok {
		return domain.ErrManifestNotFound
	}
	delete(c.policies, id)
	return nil
}
