package memory

import (
	"artifact-registry/internal/domain"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type LocalObjectStore struct{ root string }

func NewLocalObjectStore(root string) (*LocalObjectStore, error) {
	if err := os.MkdirAll(root, 0750); err != nil {
		return nil, err
	}
	return &LocalObjectStore{root: root}, nil
}
func (s *LocalObjectStore) path(key string) (string, error) {
	key = strings.TrimPrefix(filepath.Clean(key), string(filepath.Separator))
	if key == "." || strings.HasPrefix(key, "..") {
		return "", domain.ValidationError{Field: "storage key", Reason: "path traversal"}
	}
	return filepath.Join(s.root, key), nil
}
func (s *LocalObjectStore) Put(ctx context.Context, key string, r io.Reader) error {
	p, err := s.path(key)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(p), 0750); err != nil {
		return err
	}
	tmp := p + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err = io.Copy(f, &contextReader{ctx: ctx, r: r}); err != nil {
		os.Remove(tmp)
		return err
	}
	if err = f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, p)
}
func (s *LocalObjectStore) WriteAt(ctx context.Context, key string, offset int64, r io.Reader) (int64, error) {
	p, err := s.path(key)
	if err != nil {
		return 0, err
	}
	if err = os.MkdirAll(filepath.Dir(p), 0750); err != nil {
		return 0, err
	}
	f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY, 0640)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	if _, err = f.Seek(offset, io.SeekStart); err != nil {
		return 0, err
	}
	return io.Copy(f, &contextReader{ctx: ctx, r: r})
}
func (s *LocalObjectStore) Open(ctx context.Context, key string, r domain.ByteRange) (io.ReadCloser, int64, error) {
	p, err := s.path(key)
	if err != nil {
		return nil, 0, err
	}
	f, err := os.Open(p)
	if err != nil {
		return nil, 0, err
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, 0, err
	}
	size := info.Size()
	if r.Start < 0 || r.End >= size || r.End < r.Start {
		f.Close()
		return nil, size, domain.ErrInvalidRange
	}
	if _, err = f.Seek(r.Start, io.SeekStart); err != nil {
		f.Close()
		return nil, size, err
	}
	return &limitedReadCloser{Reader: io.LimitReader(f, r.Length()), Closer: f}, size, nil
}
func (s *LocalObjectStore) Size(_ context.Context, key string) (int64, error) {
	p, err := s.path(key)
	if err != nil {
		return 0, err
	}
	i, err := os.Stat(p)
	if err != nil {
		return 0, err
	}
	return i.Size(), nil
}
func (s *LocalObjectStore) Move(_ context.Context, from, to string) error {
	a, err := s.path(from)
	if err != nil {
		return err
	}
	b, err := s.path(to)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(b), 0750); err != nil {
		return err
	}
	return os.Rename(a, b)
}
func (s *LocalObjectStore) Delete(_ context.Context, key string) error {
	p, err := s.path(key)
	if err != nil {
		return err
	}
	if err = os.Remove(p); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
func (s *LocalObjectStore) Exists(_ context.Context, key string) (bool, error) {
	p, err := s.path(key)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(p)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}
func (s *LocalObjectStore) List(_ context.Context, prefix string) ([]string, error) {
	base, err := s.path(prefix)
	if err != nil {
		return nil, err
	}
	out := []string{}
	err = filepath.Walk(base, func(p string, info os.FileInfo, e error) error {
		if e != nil {
			return e
		}
		if !info.IsDir() {
			relative, _ := filepath.Rel(s.root, p)
			out = append(out, filepath.ToSlash(relative))
		}
		return nil
	})
	if os.IsNotExist(err) {
		return out, nil
	}
	return out, err
}

type contextReader struct {
	ctx context.Context
	r   io.Reader
}

func (c *contextReader) Read(p []byte) (int, error) {
	select {
	case <-c.ctx.Done():
		return 0, c.ctx.Err()
	default:
		return c.r.Read(p)
	}
}

type limitedReadCloser struct {
	io.Reader
	io.Closer
}
