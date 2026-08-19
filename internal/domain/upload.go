package domain

import (
	"crypto/rand"
	"encoding/hex"
	"sort"
	"time"
)

type ByteRange struct {
	Start int64 `json:"start"`
	End   int64 `json:"end"`
}

func NewByteRange(start, end int64) (ByteRange, error) {
	if start < 0 || end < start {
		return ByteRange{}, ErrInvalidRange
	}
	return ByteRange{start, end}, nil
}
func (r ByteRange) Length() int64 { return r.End - r.Start + 1 }
func (r ByteRange) Contains(other ByteRange) bool {
	return r.Start <= other.Start && r.End >= other.End
}
func (r ByteRange) Overlaps(other ByteRange) bool {
	return r.Start <= other.End && other.Start <= r.End
}
func (r ByteRange) Adjacent(other ByteRange) bool {
	return r.End+1 == other.Start || other.End+1 == r.Start
}
func (r ByteRange) Merge(other ByteRange) (ByteRange, error) {
	if !r.Overlaps(other) && !r.Adjacent(other) {
		return ByteRange{}, ErrInvalidRange
	}
	a, b := r, other
	if b.Start < a.Start {
		a, b = b, a
	}
	if b.End > a.End {
		a.End = b.End
	}
	return a, nil
}

type Chunk struct {
	Range      ByteRange `json:"range"`
	Digest     Digest    `json:"digest"`
	ReceivedAt time.Time `json:"receivedAt"`
}
type UploadState string

const (
	UploadOpen       UploadState = "open"
	UploadFinalizing UploadState = "finalizing"
	UploadCompleted  UploadState = "completed"
	UploadAborted    UploadState = "aborted"
	UploadExpired    UploadState = "expired"
)

type UploadSession struct {
	ID            string         `json:"id"`
	Repository    RepositoryName `json:"repository"`
	Tenant        string         `json:"tenant"`
	State         UploadState    `json:"state"`
	Chunks        []Chunk        `json:"chunks"`
	ExpectedSize  *int64         `json:"expectedSize,omitempty"`
	ReservedBytes int64          `json:"reservedBytes"`
	CreatedAt     time.Time      `json:"createdAt"`
	ExpiresAt     time.Time      `json:"expiresAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
	Version       int64          `json:"version"`
	Token         string         `json:"token"`
	TempKey       string         `json:"-"`
}

func NewUploadSession(repo RepositoryName, tenant string, expected *int64, ttl time.Duration, now time.Time) (*UploadSession, error) {
	if expected != nil && *expected < 0 {
		return nil, ValidationError{"expectedSize", "negative"}
	}
	id, err := randomToken(16)
	if err != nil {
		return nil, err
	}
	tok, err := randomToken(24)
	if err != nil {
		return nil, err
	}
	return &UploadSession{ID: id, Repository: repo, Tenant: tenant, State: UploadOpen, ExpectedSize: expected, CreatedAt: now, ExpiresAt: now.Add(ttl), UpdatedAt: now, Version: 1, Token: tok, TempKey: "uploads/" + id}, nil
}
func (s *UploadSession) IsExpired(now time.Time) bool { return now.After(s.ExpiresAt) }
func (s *UploadSession) ContiguousEnd() int64 {
	if len(s.Chunks) == 0 {
		return -1
	}
	ranges := s.ranges()
	end := int64(-1)
	for _, r := range ranges {
		if r.Start > end+1 {
			break
		}
		if r.End > end {
			end = r.End
		}
	}
	return end
}
func (s *UploadSession) ranges() []ByteRange {
	v := make([]ByteRange, len(s.Chunks))
	for i, c := range s.Chunks {
		v[i] = c.Range
	}
	sort.Slice(v, func(i, j int) bool { return v[i].Start < v[j].Start })
	return v
}
func (s *UploadSession) HasExactChunk(r ByteRange, d Digest) bool {
	for _, c := range s.Chunks {
		if c.Range == r && c.Digest == d {
			return true
		}
	}
	return false
}
func (s *UploadSession) RecordChunk(r ByteRange, d Digest, now time.Time) error {
	if s.State != UploadOpen {
		return ErrUploadConflict
	}
	if s.IsExpired(now) {
		s.State = UploadExpired
		return ErrUploadExpired
	}
	if s.ExpectedSize != nil && r.End >= *s.ExpectedSize {
		return ValidationError{"range", "exceeds expected size"}
	}
	for _, c := range s.Chunks {
		if c.Range.Overlaps(r) {
			if c.Range == r && c.Digest == d {
				return nil
			}
			return ErrUploadConflict
		}
	}
	s.Chunks = append(s.Chunks, Chunk{Range: r, Digest: d, ReceivedAt: now})
	s.ReservedBytes += r.Length()
	s.UpdatedAt = now
	s.Version++
	return nil
}
func (s *UploadSession) CanFinalize(size int64, now time.Time) error {
	if s.State != UploadOpen {
		return ErrUploadConflict
	}
	if s.IsExpired(now) {
		return ErrUploadExpired
	}
	if size < 0 {
		return ValidationError{"size", "negative"}
	}
	if s.ExpectedSize != nil && size != *s.ExpectedSize {
		return ValidationError{"size", "does not match expected size"}
	}
	if size == 0 && len(s.Chunks) == 0 {
		return nil
	}
	if s.ContiguousEnd() != size-1 {
		return ErrUploadOutOfOrder
	}
	return nil
}
func (s *UploadSession) Finalize(now time.Time) error {
	if s.State != UploadOpen {
		return ErrUploadConflict
	}
	s.State = UploadFinalizing
	s.UpdatedAt = now
	s.Version++
	return nil
}
func (s *UploadSession) Complete(now time.Time) {
	s.State = UploadCompleted
	s.UpdatedAt = now
	s.Version++
}
func (s *UploadSession) Abort(now time.Time) {
	if s.State == UploadCompleted {
		return
	}
	s.State = UploadAborted
	s.UpdatedAt = now
	s.Version++
}
func randomToken(bytes int) (string, error) {
	b := make([]byte, bytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
