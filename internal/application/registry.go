package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
	"time"

	"artifact-registry/internal/domain"
)

type RegistryService struct {
	Repositories RepositoryStore
	Blobs        BlobStore
	Manifests    ManifestStore
	Tags         TagStore
	Uploads      UploadStore
	Quotas       QuotaStore
	Audits       AuditStore
	ObjectStore  ObjectStore
	Events       EventPublisher
	Clock        Clock
	UploadTTL    time.Duration
	MaxBlobSize  int64
	DefaultQuota int64
}
type CreateRepositoryCommand struct {
	Name          string `json:"name"`
	Tenant        string `json:"tenant"`
	ImmutableTags bool   `json:"immutableTags"`
	Quota         int64  `json:"quota"`
}

func (s *RegistryService) CreateRepository(ctx context.Context, cmd CreateRepositoryCommand) (*domain.Repository, error) {
	name, err := domain.ParseRepositoryName(cmd.Name)
	if err != nil {
		return nil, err
	}
	now := s.now()
	repo, err := domain.NewRepository(name, cmd.Tenant, cmd.ImmutableTags, now)
	if err != nil {
		return nil, err
	}
	if err = s.Repositories.Create(ctx, repo); err != nil {
		return nil, err
	}
	limit := cmd.Quota
	if limit <= 0 {
		limit = s.DefaultQuota
	}
	if err = s.Quotas.PutQuota(ctx, &domain.Quota{Tenant: cmd.Tenant, Limit: limit, UpdatedAt: now, Version: 1}); err != nil {
		return nil, err
	}
	s.audit(ctx, "repository.create", repo.Name.String(), map[string]string{"tenant": cmd.Tenant})
	return repo, nil
}
func (s *RegistryService) ListRepositories(ctx context.Context, prefix string, page domain.PageRequest) (domain.Page[*domain.Repository], error) {
	return s.Repositories.List(ctx, prefix, page)
}
func (s *RegistryService) SetTagImmutability(ctx context.Context, repoName string, value bool, ifMatch int64) (*domain.Repository, error) {
	repo, err := s.repository(ctx, repoName)
	if err != nil {
		return nil, err
	}
	if ifMatch > 0 && repo.Version != ifMatch {
		return nil, domain.ErrPreconditionFailed
	}
	old := repo.Version
	repo.SetImmutableTags(value, s.now())
	if err = s.Repositories.Update(ctx, repo, old); err != nil {
		return nil, err
	}
	s.audit(ctx, "repository.immutability", repo.Name.String(), map[string]string{"enabled": fmt.Sprint(value)})
	return repo, nil
}
func (s *RegistryService) StartUpload(ctx context.Context, repoName string, expected *int64) (*domain.UploadSession, error) {
	repo, err := s.repository(ctx, repoName)
	if err != nil {
		return nil, err
	}
	if expected != nil && *expected > s.MaxBlobSize {
		return nil, domain.ValidationError{Field: "expectedSize", Reason: "exceeds max blob size"}
	}
	session, err := domain.NewUploadSession(repo.Name, repo.Tenant, expected, s.UploadTTL, s.now())
	if err != nil {
		return nil, err
	}
	if expected != nil {
		if err = s.reserve(ctx, repo.Tenant, *expected); err != nil {
			return nil, err
		}
	}
	if err = s.Uploads.CreateUpload(ctx, session); err != nil {
		if expected != nil {
			s.release(ctx, repo.Tenant, *expected)
		}
		return nil, err
	}
	s.audit(ctx, "upload.start", repo.Name.String()+"/uploads/"+session.ID, nil)
	return session, nil
}
func (s *RegistryService) UploadChunk(ctx context.Context, repoName, id, token string, rng domain.ByteRange, body io.Reader) (*domain.UploadSession, error) {
	session, err := s.Uploads.GetUpload(ctx, id)
	if err != nil {
		return nil, err
	}
	if session.Repository.String() != repoName || session.Token != token {
		return nil, domain.ErrUploadNotFound
	}
	if rng.Length() > s.MaxBlobSize {
		return nil, domain.ValidationError{Field: "chunk", Reason: "exceeds maximum size"}
	}
	limited := io.LimitReader(body, rng.Length()+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) != rng.Length() {
		return nil, domain.ValidationError{Field: "Content-Range", Reason: "body length does not match range"}
	}
	digest := domain.DigestBytes(data)
	if session.HasExactChunk(rng, digest) {
		return session, nil
	}
	if _, err = s.ObjectStore.WriteAt(ctx, session.TempKey, rng.Start, bytes.NewReader(data)); err != nil {
		return nil, err
	}
	old := session.Version
	if err = session.RecordChunk(rng, digest, s.now()); err != nil {
		return nil, err
	}
	if err = s.Uploads.UpdateUpload(ctx, session, old); err != nil {
		return nil, err
	}
	s.audit(ctx, "upload.chunk", repoName+"/uploads/"+id, map[string]string{"range": fmt.Sprintf("%d-%d", rng.Start, rng.End)})
	return session, nil
}
func (s *RegistryService) CompleteUpload(ctx context.Context, repoName, id, token, digestText, mediaType string) (*domain.Blob, error) {
	session, err := s.Uploads.GetUpload(ctx, id)
	if err != nil {
		return nil, err
	}
	if session.Repository.String() != repoName || session.Token != token {
		return nil, domain.ErrUploadNotFound
	}
	expected, err := domain.ParseDigest(digestText)
	if err != nil {
		return nil, err
	}
	size, err := s.ObjectStore.Size(ctx, session.TempKey)
	if err != nil {
		return nil, err
	}
	if err = session.CanFinalize(size, s.now()); err != nil {
		return nil, err
	}
	old := session.Version
	if err = session.Finalize(s.now()); err != nil {
		return nil, err
	}
	if err = s.Uploads.UpdateUpload(ctx, session, old); err != nil {
		return nil, err
	}
	stream, _, err := s.ObjectStore.Open(ctx, session.TempKey, domain.ByteRange{Start: 0, End: size - 1})
	if err != nil && size != 0 {
		return nil, err
	}
	var actual domain.Digest
	if size == 0 {
		actual = domain.DigestBytes(nil)
	} else {
		actual, _, err = domain.DigestReader(stream)
		stream.Close()
		if err != nil {
			return nil, err
		}
	}
	if actual != expected {
		session.Abort(s.now())
		_ = s.Uploads.UpdateUpload(ctx, session, session.Version-1)
		s.releaseReservation(ctx, session)
		return nil, fmt.Errorf("%w: %s", domain.ErrDigestMismatch, actual)
	}
	key := blobKey(expected)
	blob, err := domain.NewBlob(expected, size, defaultMediaType(mediaType), key, s.now())
	if err != nil {
		return nil, err
	}
	exists, err := s.ObjectStore.Exists(ctx, key)
	if err != nil {
		return nil, err
	}
	if !exists {
		if err = s.ObjectStore.Move(ctx, session.TempKey, key); err != nil {
			return nil, err
		}
	}
	if size > 0 {
		r, _, err := s.ObjectStore.Open(ctx, key, domain.ByteRange{Start: 0, End: size - 1})
		if err != nil {
			return nil, err
		}
		err = s.Blobs.PutBlob(ctx, blob, r)
		r.Close()
		if err != nil {
			return nil, err
		}
	} else {
		if err = s.Blobs.PutBlob(ctx, blob, strings.NewReader("")); err != nil {
			return nil, err
		}
	}
	session.Complete(s.now())
	_ = s.Uploads.UpdateUpload(ctx, session, session.Version-1)
	s.commitReservation(ctx, session, size)
	s.audit(ctx, "upload.complete", repoName+"/blobs/"+expected.String(), map[string]string{"size": fmt.Sprint(size)})
	return blob, nil
}
func (s *RegistryService) MountBlob(ctx context.Context, target, source, digestText string) (*domain.Blob, error) {
	targetRepo, err := s.repository(ctx, target)
	if err != nil {
		return nil, err
	}
	sourceRepo, err := s.repository(ctx, source)
	if err != nil {
		return nil, err
	}
	if sourceRepo.Tenant != targetRepo.Tenant {
		return nil, domain.ConflictError{Resource: "mount", Detail: "cross-tenant mount denied"}
	}
	d, err := domain.ParseDigest(digestText)
	if err != nil {
		return nil, err
	}
	b, err := s.Blobs.GetBlob(ctx, d)
	if err != nil {
		return nil, err
	}
	s.audit(ctx, "blob.mount", target+"/blobs/"+d.String(), map[string]string{"source": source})
	return b, nil
}
func (s *RegistryService) GetBlob(ctx context.Context, repoName, digest string, rng *domain.ByteRange) (io.ReadCloser, *domain.Blob, error) {
	if _, err := s.repository(ctx, repoName); err != nil {
		return nil, nil, err
	}
	d, err := domain.ParseDigest(digest)
	if err != nil {
		return nil, nil, err
	}
	b, err := s.Blobs.GetBlob(ctx, d)
	if err != nil {
		return nil, nil, err
	}
	rangeToRead := domain.ByteRange{Start: 0, End: b.Size - 1}
	if b.Size == 0 {
		return io.NopCloser(strings.NewReader("")), b, nil
	}
	if rng != nil {
		rangeToRead = *rng
	}
	return s.Blobs.OpenBlob(ctx, d, rangeToRead)
}
func (s *RegistryService) PutManifest(ctx context.Context, repoName, reference, mediaType string, raw []byte) (*domain.Manifest, error) {
	repo, err := s.repository(ctx, repoName)
	if err != nil {
		return nil, err
	}
	if int64(len(raw)) > 16<<20 {
		return nil, domain.ValidationError{Field: "manifest", Reason: "larger than 16 MiB"}
	}
	m, err := domain.NewManifest(raw, mediaType, s.now())
	if err != nil {
		return nil, err
	}
	for _, d := range m.References() {
		if _, err = s.Blobs.GetBlob(ctx, d); err != nil {
			return nil, fmt.Errorf("manifest reference %s: %w", d, err)
		}
	}
	if err = s.Manifests.PutManifest(ctx, repo.Name, m); err != nil {
		return nil, err
	}
	if parsed, parseErr := domain.ParseDigest(reference); parseErr == nil && parsed != m.Digest {
		return nil, domain.ErrDigestMismatch
	}
	if _, parseErr := domain.ParseDigest(reference); parseErr != nil {
		old, findErr := s.Tags.GetTag(ctx, repo.Name, reference)
		if findErr == nil {
			if err = old.Move(m.Digest, s.now()); err != nil {
				return nil, err
			}
			if err = s.Tags.PutTag(ctx, repo.Name, old); err != nil {
				return nil, err
			}
		} else {
			tag, err := domain.NewTag(reference, m.Digest, repo.ImmutableTags, s.now())
			if err != nil {
				return nil, err
			}
			if err = s.Tags.PutTag(ctx, repo.Name, tag); err != nil {
				return nil, err
			}
		}
	}
	s.audit(ctx, "manifest.put", repoName+"/manifests/"+m.Digest.String(), map[string]string{"reference": reference})
	if s.Events != nil {
		_ = s.Events.Publish(ctx, "manifest.pushed", m.Descriptor())
	}
	return m, nil
}
func (s *RegistryService) GetManifest(ctx context.Context, repo, reference string) (*domain.Manifest, error) {
	r, err := s.repository(ctx, repo)
	if err != nil {
		return nil, err
	}
	m, err := s.Manifests.GetManifest(ctx, r.Name, reference)
	if err != nil {
		return nil, err
	}
	if !m.State.AllowsPull() {
		return nil, domain.ErrManifestBlocked
	}
	return m, nil
}
func (s *RegistryService) ListTags(ctx context.Context, repo string, page domain.PageRequest) (domain.Page[*domain.Tag], error) {
	r, err := s.repository(ctx, repo)
	if err != nil {
		return domain.Page[*domain.Tag]{}, err
	}
	return s.Tags.ListTags(ctx, r.Name, page)
}
func (s *RegistryService) Referrers(ctx context.Context, repo, digest, artifactType string) ([]*domain.Manifest, error) {
	r, err := s.repository(ctx, repo)
	if err != nil {
		return nil, err
	}
	d, err := domain.ParseDigest(digest)
	if err != nil {
		return nil, err
	}
	return s.Manifests.FindReferrers(ctx, r.Name, d, artifactType)
}
func (s *RegistryService) DeleteManifest(ctx context.Context, repo, digest string) error {
	r, err := s.repository(ctx, repo)
	if err != nil {
		return err
	}
	d, err := domain.ParseDigest(digest)
	if err != nil {
		return err
	}
	if err = s.Manifests.DeleteManifest(ctx, r.Name, d); err == nil {
		s.audit(ctx, "manifest.delete", repo+"/manifests/"+digest, nil)
	}
	return err
}
func (s *RegistryService) AbortUpload(ctx context.Context, repo, id, token string) error {
	u, err := s.Uploads.GetUpload(ctx, id)
	if err != nil {
		return err
	}
	if u.Repository.String() != repo || u.Token != token {
		return domain.ErrUploadNotFound
	}
	u.Abort(s.now())
	s.releaseReservation(ctx, u)
	return s.Uploads.DeleteUpload(ctx, id)
}
func (s *RegistryService) repository(ctx context.Context, name string) (*domain.Repository, error) {
	parsed, err := domain.ParseRepositoryName(name)
	if err != nil {
		return nil, err
	}
	return s.Repositories.Get(ctx, parsed)
}
func (s *RegistryService) now() time.Time {
	if s.Clock != nil {
		return s.Clock.Now().UTC()
	}
	return time.Now().UTC()
}
func (s *RegistryService) reserve(ctx context.Context, tenant string, n int64) error {
	q, err := s.Quotas.GetQuota(ctx, tenant)
	if err != nil {
		return err
	}
	old := q.Version
	if err = q.Reserve(n, s.now()); err != nil {
		return err
	}
	return s.Quotas.UpdateQuota(ctx, q, old)
}
func (s *RegistryService) release(ctx context.Context, tenant string, n int64) {
	q, err := s.Quotas.GetQuota(ctx, tenant)
	if err == nil {
		old := q.Version
		if q.Release(n, s.now()) == nil {
			_ = s.Quotas.UpdateQuota(ctx, q, old)
		}
	}
}
func (s *RegistryService) releaseReservation(ctx context.Context, u *domain.UploadSession) {
	if u.ExpectedSize != nil {
		s.release(ctx, u.Tenant, *u.ExpectedSize)
	}
}
func (s *RegistryService) commitReservation(ctx context.Context, u *domain.UploadSession, actual int64) {
	if u.ExpectedSize == nil {
		return
	}
	q, err := s.Quotas.GetQuota(ctx, u.Tenant)
	if err != nil {
		return
	}
	old := q.Version
	if q.Commit(*u.ExpectedSize, actual, s.now()) == nil {
		_ = s.Quotas.UpdateQuota(ctx, q, old)
	}
}
func (s *RegistryService) audit(ctx context.Context, action, resource string, detail map[string]string) {
	if s.Audits == nil {
		return
	}
	raw := action + "|" + resource + "|" + fmt.Sprint(detail)
	h := sha256.Sum256([]byte(raw))
	_ = s.Audits.Append(ctx, domain.AuditEntry{ID: hex.EncodeToString(h[:8]), At: s.now(), Actor: actor(ctx), Action: action, Resource: resource, Detail: detail, Hash: hex.EncodeToString(h[:])})
}
func actor(ctx context.Context) string {
	if v, ok := ctx.Value(actorKey{}).(string); ok && v != "" {
		return v
	}
	return "anonymous"
}

type actorKey struct{}

func WithActor(ctx context.Context, actor string) context.Context {
	return context.WithValue(ctx, actorKey{}, actor)
}
func blobKey(d domain.Digest) string { return "blobs/sha256/" + d.Hex()[:2] + "/" + d.Hex() }
func defaultMediaType(v string) string {
	if strings.TrimSpace(v) == "" {
		return "application/octet-stream"
	}
	return v
}
