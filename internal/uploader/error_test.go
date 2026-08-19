package uploader

import (
	"bytes"
	"errors"
	"testing"
	"time"
)

func TestWriterAddDuplicateChunk(t *testing.T) {
	w := NewWriter()
	if err := w.Add(0, bytes.NewReader([]byte("x"))); err != nil {
		t.Fatal(err)
	}
	err := w.Add(0, bytes.NewReader([]byte("y")))
	if !errors.Is(err, ErrDuplicateChunk) {
		t.Fatalf("expected ErrDuplicateChunk, got %v", err)
	}
}

func TestWriterAddNilReader(t *testing.T) {
	w := NewWriter()
	err := w.Add(0, nil)
	if !errors.Is(err, ErrNilReader) {
		t.Fatalf("expected ErrNilReader, got %v", err)
	}
}

func TestSessionValidateExpired(t *testing.T) {
	now := time.Now()
	s := NewSession("s1", time.Minute, now)
	err := s.Validate(now.Add(time.Hour))
	if !errors.Is(err, ErrExpiredSession) {
		t.Fatalf("expected ErrExpiredSession, got %v", err)
	}
}

func TestSessionValidateOverlap(t *testing.T) {
	now := time.Now()
	s := NewSession("s1", time.Minute, now)
	s.Chunks = []Chunk{{Start: 0, End: 9}, {Start: 0, End: 9}}
	err := s.Validate(now)
	if !errors.Is(err, ErrOverlappingChunk) {
		t.Fatalf("expected ErrOverlappingChunk, got %v", err)
	}
}

func TestWriterPartReaderMissing(t *testing.T) {
	w := NewWriter()
	_, err := w.PartReader(3)
	if !errors.Is(err, ErrMissingChunk) {
		t.Fatalf("expected ErrMissingChunk, got %v", err)
	}
}

func TestSessionEndOffsetEmpty(t *testing.T) {
	s := NewSession("s1", time.Minute, time.Now())
	_, err := s.EndOffset()
	if !errors.Is(err, ErrEmptySession) {
		t.Fatalf("expected ErrEmptySession, got %v", err)
	}
}

func TestWriterRemoveMissing(t *testing.T) {
	w := NewWriter()
	if err := w.Remove(4); !errors.Is(err, ErrMissingChunk) {
		t.Fatalf("expected ErrMissingChunk, got %v", err)
	}
}

func TestSessionChunkAtMissing(t *testing.T) {
	s := NewSession("s1", time.Minute, time.Now())
	_, err := s.ChunkAt(4)
	if !errors.Is(err, ErrMissingChunk) {
		t.Fatalf("expected ErrMissingChunk, got %v", err)
	}
}
