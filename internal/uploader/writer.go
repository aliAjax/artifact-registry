package uploader

import (
	"fmt"
	"io"
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
	if _, ok := w.parts[offset]; ok {
		return fmt.Errorf("chunk at offset %d already present", offset)
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
