package uploader

import "time"

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

func sortChunks(chunks []Chunk) {
	for i := 1; i < len(chunks); i++ {
		for j := i; j > 0 && chunks[j-1].Start > chunks[j].Start; j-- {
			chunks[j-1], chunks[j] = chunks[j], chunks[j-1]
		}
	}
}
