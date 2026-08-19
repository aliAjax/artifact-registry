package domain

import "time"

type Blob struct {
	Digest         Digest    `json:"digest"`
	Size           int64     `json:"size"`
	MediaType      string    `json:"mediaType"`
	CreatedAt      time.Time `json:"createdAt"`
	LastAccessedAt time.Time `json:"lastAccessedAt"`
	References     int64     `json:"references"`
	StorageKey     string    `json:"-"`
}

func NewBlob(d Digest, size int64, mediaType, storageKey string, now time.Time) (*Blob, error) {
	if d.Validate() != nil {
		return nil, ErrInvalidDigest
	}
	if size < 0 {
		return nil, ValidationError{"size", "must not be negative"}
	}
	if !IsBlobMediaType(mediaType) {
		return nil, ErrInvalidMediaType
	}
	return &Blob{Digest: d, Size: size, MediaType: NormalizeMediaType(mediaType), CreatedAt: now, LastAccessedAt: now, StorageKey: storageKey}, nil
}
func (b *Blob) Retain() { b.References++ }
func (b *Blob) Release() error {
	if b.References == 0 {
		return ConflictError{"blob", "cannot decrement zero references"}
	}
	b.References--
	return nil
}
func (b *Blob) Touch(now time.Time) { b.LastAccessedAt = now }
