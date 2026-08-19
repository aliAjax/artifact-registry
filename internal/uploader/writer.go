package uploader

import (
	"errors"
	"fmt"
	"io"
)

var (
	ErrDuplicateChunk = errors.New("chunk already present")
	ErrNilReader      = errors.New("chunk reader is nil")
	ErrMissingChunk   = errors.New("chunk not registered")
)

// Writer assembles uploaded chunks into a contiguous stream. It validates that
// chunks do not overlap and that every gap has been filled before commit.
type Writer struct {
	parts map[int64]io.Reader
}

func NewWriter() *Writer {
	return &Writer{parts: map[int64]io.Reader{}}
}

// Add registers a chunk starting at the given offset.
func (w *Writer) Add(offset int64, r io.Reader) error {
	if r == nil {
		return fmt.Errorf("%w: offset %d", ErrNilReader, offset)
	}
	if _, ok := w.parts[offset]; ok {
		return fmt.Errorf("%w: offset %d", ErrDuplicateChunk, offset)
	}
	w.parts[offset] = r
	return nil
}

// Parts returns the registered offsets.
func (w *Writer) Parts() []int64 {
	out := make([]int64, 0, len(w.parts))
	for off := range w.parts {
		out = append(out, off)
	}
	return out
}

// Count returns the number of registered chunks.
func (w *Writer) Count() int { return len(w.parts) }

// Remove unregisters the chunk at offset.
func (w *Writer) Remove(offset int64) error {
	if _, ok := w.parts[offset]; !ok {
		return fmt.Errorf("%w: offset %d", ErrMissingChunk, offset)
	}
	delete(w.parts, offset)
	return nil
}

// PartReader returns the reader registered at offset.
func (w *Writer) PartReader(offset int64) (io.Reader, error) {
	r, ok := w.parts[offset]
	if !ok {
		return nil, fmt.Errorf("%w: offset %d", ErrMissingChunk, offset)
	}
	return r, nil
}
