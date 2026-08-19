// Package application coordinates domain objects through explicit ports.
package application

import (
	"context"
	"io"
	"time"

	"artifact-registry/internal/domain"
)

type RepositoryStore interface {
	Create(context.Context, *domain.Repository) error
	Get(context.Context, domain.RepositoryName) (*domain.Repository, error)
	List(context.Context, string, domain.PageRequest) (domain.Page[*domain.Repository], error)
	Update(context.Context, *domain.Repository, int64) error
}
type BlobStore interface {
	GetBlob(context.Context, domain.Digest) (*domain.Blob, error)
	PutBlob(context.Context, *domain.Blob, io.Reader) error
	OpenBlob(context.Context, domain.Digest, domain.ByteRange) (io.ReadCloser, *domain.Blob, error)
	DeleteBlob(context.Context, domain.Digest) error
	ListBlobs(context.Context) ([]*domain.Blob, error)
}
type ManifestStore interface {
	PutManifest(context.Context, domain.RepositoryName, *domain.Manifest) error
	GetManifest(context.Context, domain.RepositoryName, string) (*domain.Manifest, error)
	ListManifestDigests(context.Context, domain.RepositoryName) ([]domain.Digest, error)
	ListAllManifests(context.Context) (map[domain.RepositoryName][]*domain.Manifest, error)
	DeleteManifest(context.Context, domain.RepositoryName, domain.Digest) error
	FindReferrers(context.Context, domain.RepositoryName, domain.Digest, string) ([]*domain.Manifest, error)
}
type TagStore interface {
	PutTag(context.Context, domain.RepositoryName, *domain.Tag) error
	GetTag(context.Context, domain.RepositoryName, string) (*domain.Tag, error)
	ListTags(context.Context, domain.RepositoryName, domain.PageRequest) (domain.Page[*domain.Tag], error)
	DeleteTag(context.Context, domain.RepositoryName, string) error
}
type UploadStore interface {
	CreateUpload(context.Context, *domain.UploadSession) error
	GetUpload(context.Context, string) (*domain.UploadSession, error)
	UpdateUpload(context.Context, *domain.UploadSession, int64) error
	DeleteUpload(context.Context, string) error
	ListExpiredUploads(context.Context, time.Time) ([]*domain.UploadSession, error)
}
type QuotaStore interface {
	GetQuota(context.Context, string) (*domain.Quota, error)
	PutQuota(context.Context, *domain.Quota) error
	UpdateQuota(context.Context, *domain.Quota, int64) error
}
type AuditStore interface {
	Append(context.Context, domain.AuditEntry) error
	ListAudit(context.Context, string, domain.PageRequest) (domain.Page[domain.AuditEntry], error)
}
type ScanStore interface {
	CreateScan(context.Context, domain.ScanReport) error
	GetScan(context.Context, string) (domain.ScanReport, error)
	ListScans(context.Context, domain.RepositoryName, domain.PageRequest) (domain.Page[domain.ScanReport], error)
	UpdateScan(context.Context, domain.ScanReport) error
}
type ReplicationStore interface {
	CreateReplication(context.Context, domain.ReplicationTask) error
	GetReplication(context.Context, string) (domain.ReplicationTask, error)
	ListReplications(context.Context, domain.PageRequest) (domain.Page[domain.ReplicationTask], error)
	UpdateReplication(context.Context, domain.ReplicationTask, int64) error
}
type PolicyStore interface {
	PutPolicy(context.Context, domain.RetentionPolicy) error
	ListPolicies(context.Context) ([]domain.RetentionPolicy, error)
	DeletePolicy(context.Context, string) error
}
type Clock interface{ Now() time.Time }
type Scanner interface {
	Submit(context.Context, domain.RepositoryName, *domain.Manifest) (domain.ScanReport, error)
}
type Signer interface {
	Verify(context.Context, domain.RepositoryName, *domain.Manifest) error
}
type Replicator interface {
	Copy(context.Context, domain.ReplicationTask) (int64, error)
}
type ObjectStore interface {
	WriteAt(context.Context, string, int64, io.Reader) (int64, error)
	Open(context.Context, string, domain.ByteRange) (io.ReadCloser, int64, error)
	Size(context.Context, string) (int64, error)
	Move(context.Context, string, string) error
	Delete(context.Context, string) error
	Exists(context.Context, string) (bool, error)
	List(context.Context, string) ([]string, error)
}
type EventPublisher interface {
	Publish(context.Context, string, any) error
}
