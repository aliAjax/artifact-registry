package domain

import "time"

type Tag struct {
	Name      string    `json:"name"`
	Digest    Digest    `json:"digest"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Version   int64     `json:"version"`
	Immutable bool      `json:"immutable"`
}

func NewTag(name string, digest Digest, immutable bool, now time.Time) (*Tag, error) {
	n, err := ParseTag(name)
	if err != nil {
		return nil, err
	}
	if digest.Validate() != nil {
		return nil, ErrInvalidDigest
	}
	return &Tag{Name: n, Digest: digest, CreatedAt: now, UpdatedAt: now, Version: 1, Immutable: immutable}, nil
}
func (t *Tag) Move(to Digest, now time.Time) error {
	if t.Immutable && t.Digest != to {
		return ErrTagImmutable
	}
	if to.Validate() != nil {
		return ErrInvalidDigest
	}
	t.Digest = to
	t.UpdatedAt = now
	t.Version++
	return nil
}
