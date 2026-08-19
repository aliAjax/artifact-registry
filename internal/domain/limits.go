package domain

import "fmt"

type Limits struct {
	MaxManifestBytes int64 `json:"maxManifestBytes"`
	MaxBlobBytes     int64 `json:"maxBlobBytes"`
	MaxChunkBytes    int64 `json:"maxChunkBytes"`
	MaxLayers        int   `json:"maxLayers"`
	MaxRepositories  int   `json:"maxRepositories"`
	MaxTags          int   `json:"maxTags"`
}

func DefaultLimits() Limits {
	return Limits{MaxManifestBytes: 16 << 20, MaxBlobBytes: 10 << 30, MaxChunkBytes: 64 << 20, MaxLayers: 128, MaxRepositories: 1000, MaxTags: 100000}
}
func (l Limits) Validate() error {
	if l.MaxManifestBytes <= 0 {
		return fmt.Errorf("max manifest bytes must be positive")
	}
	if l.MaxBlobBytes < l.MaxManifestBytes {
		return fmt.Errorf("max blob bytes below manifest limit")
	}
	if l.MaxChunkBytes <= 0 || l.MaxChunkBytes > l.MaxBlobBytes {
		return fmt.Errorf("invalid chunk limit")
	}
	if l.MaxLayers <= 0 || l.MaxTags <= 0 {
		return fmt.Errorf("entity limits must be positive")
	}
	return nil
}
func (l Limits) WithinManifest(size int64) bool { return size >= 0 && size <= l.MaxManifestBytes }
func (l Limits) WithinBlob(size int64) bool     { return size >= 0 && size <= l.MaxBlobBytes }
func (l Limits) WithinChunk(size int64) bool    { return size >= 0 && size <= l.MaxChunkBytes }
func (l Limits) Normalize() Limits {
	d := DefaultLimits()
	if l.MaxManifestBytes <= 0 {
		l.MaxManifestBytes = d.MaxManifestBytes
	}
	if l.MaxBlobBytes <= 0 {
		l.MaxBlobBytes = d.MaxBlobBytes
	}
	if l.MaxChunkBytes <= 0 {
		l.MaxChunkBytes = d.MaxChunkBytes
	}
	if l.MaxLayers <= 0 {
		l.MaxLayers = d.MaxLayers
	}
	if l.MaxRepositories <= 0 {
		l.MaxRepositories = d.MaxRepositories
	}
	if l.MaxTags <= 0 {
		l.MaxTags = d.MaxTags
	}
	return l
}
func (l Limits) DescriptorBudget(count int) error {
	if count < 0 || count > l.MaxLayers {
		return fmt.Errorf("descriptor count %d exceeds %d", count, l.MaxLayers)
	}
	return nil
}
func (l Limits) RepositoryBudget(count int) error {
	if count < 0 || count > l.MaxRepositories {
		return fmt.Errorf("repository count %d exceeds %d", count, l.MaxRepositories)
	}
	return nil
}
func (l Limits) TagBudget(count int) error {
	if count < 0 || count > l.MaxTags {
		return fmt.Errorf("tag count %d exceeds %d", count, l.MaxTags)
	}
	return nil
}
func (l Limits) BlobBudget(bytes, used int64) error {
	if bytes < 0 || used < 0 {
		return fmt.Errorf("negative storage budget")
	}
	if bytes > l.MaxBlobBytes-used {
		return ErrQuotaExceeded
	}
	return nil
}
func (l Limits) ChunkCount(size int64) int {
	if size <= 0 {
		return 0
	}
	n := size / l.MaxChunkBytes
	if size%l.MaxChunkBytes != 0 {
		n++
	}
	return int(n)
}
func (l Limits) UploadWindow(size, offset int64) error {
	if offset < 0 || size < offset {
		return ErrInvalidRange
	}
	if size-offset > l.MaxChunkBytes {
		return ErrUploadOutOfOrder
	}
	return nil
}
func (l Limits) ValidateUpload(size int64) error {
	if !l.WithinBlob(size) {
		return ErrQuotaExceeded
	}
	if size > 0 && l.ChunkCount(size) <= 0 {
		return ErrInvalidRange
	}
	return nil
}
func (l Limits) Valid() bool { return l.Validate() == nil }
func (l Limits) Summary() string {
	return fmt.Sprintf("manifest=%d blob=%d chunk=%d", l.MaxManifestBytes, l.MaxBlobBytes, l.MaxChunkBytes)
}
