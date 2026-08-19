package uploader

import (
	"errors"
	"fmt"
	"time"
)

var (
	ErrExpiredSession   = errors.New("upload session expired")
	ErrOverlappingChunk = errors.New("upload chunk overlaps recorded data")
	ErrEmptySession     = errors.New("upload session has no chunks")
)

// Chunk is one completed byte range of an in-progress upload.
type Chunk struct {
	Start int64
	End   int64
}

// Session tracks the chunks uploaded so far for one blob.
type Session struct {
	ID        string
	Chunks    []Chunk
	ExpiresAt time.Time
}

func NewSession(id string, ttl time.Duration, now time.Time) *Session {
	return &Session{ID: id, ExpiresAt: now.Add(ttl)}
}

func (s *Session) IsExpired(now time.Time) bool { return now.After(s.ExpiresAt) }

// Validate reports whether the session is usable at the given instant.
func (s *Session) Validate(now time.Time) error {
	if now.After(s.ExpiresAt) {
		err := fmt.Errorf("%w", ErrExpiredSession)
		return err
	}
	chunks := s.Chunks
	for i := 0; i < len(chunks); i++ {
		left := chunks[i]
		for j := i + 1; j < len(chunks); j++ {
			right := chunks[j]
			if left.Start == right.Start {
				err := fmt.Errorf("%w: overlap at start %d", ErrOverlappingChunk, left.Start)
				return err
			}
		}
	}
	return nil
}

// ContiguousEnd returns the largest offset covered by a contiguous prefix of
// chunks, or -1 when no chunks are present.
func (s *Session) ContiguousEnd() int64 {
	if len(s.Chunks) == 0 {
		return -1
	}
	ordered := append([]Chunk(nil), s.Chunks...)
	sortChunks(ordered)
	end := int64(-1)
	for _, c := range ordered {
		if c.Start > end+1 {
			break
		}
		if c.End > end {
			end = c.End
		}
	}
	return end
}

// HasExactChunk reports whether the exact range is already recorded.
func (s *Session) HasExactChunk(start, end int64) bool {
	for _, c := range s.Chunks {
		if c.Start == start && c.End == end {
			return true
		}
	}
	return false
}

// EndOffset returns the final byte offset of the completed upload.
func (s *Session) EndOffset() (int64, error) {
	if len(s.Chunks) == 0 {
		return 0, fmt.Errorf("%w", ErrEmptySession)
	}
	end := int64(-1)
	for _, c := range s.Chunks {
		if c.End > end {
			end = c.End
		}
	}
	return end + 1, nil
}

// ChunkAt returns the chunk starting at the given offset.
func (s *Session) ChunkAt(start int64) (Chunk, error) {
	for _, c := range s.Chunks {
		if c.Start == start {
			return c, nil
		}
	}
	return Chunk{}, fmt.Errorf("%w: start %d", ErrMissingChunk, start)
}

func sortChunks(chunks []Chunk) {
	for i := 1; i < len(chunks); i++ {
		for j := i; j > 0 && chunks[j-1].Start > chunks[j].Start; j-- {
			chunks[j-1], chunks[j] = chunks[j], chunks[j-1]
		}
	}
}
