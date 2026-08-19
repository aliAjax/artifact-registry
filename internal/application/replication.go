package application

import (
	"artifact-registry/internal/domain"
	"context"
	"fmt"
	"io"
	"sync"
	"time"
)

type ReplicationCoordinator struct {
	Tasks ReplicationStore
	Blobs BlobStore
	Clock Clock
	mu    sync.Mutex
}
type CopyProgress struct {
	TaskID    string        `json:"taskId"`
	Digest    domain.Digest `json:"digest"`
	Total     int64         `json:"total"`
	Copied    int64         `json:"copied"`
	Attempts  int           `json:"attempts"`
	StartedAt time.Time     `json:"startedAt"`
	UpdatedAt time.Time     `json:"updatedAt"`
	State     string        `json:"state"`
}

func (c *ReplicationCoordinator) Begin(ctx context.Context, id string) (CopyProgress, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	task, err := c.Tasks.GetReplication(ctx, id)
	if err != nil {
		return CopyProgress{}, err
	}
	if err = task.Claim(c.now().Add(10*time.Minute), c.now()); err != nil {
		return CopyProgress{}, err
	}
	if err = c.Tasks.UpdateReplication(ctx, task, task.Version-1); err != nil {
		return CopyProgress{}, err
	}
	return CopyProgress{TaskID: id, StartedAt: c.now(), UpdatedAt: c.now(), State: string(task.State), Attempts: task.Attempts}, nil
}
func (c *ReplicationCoordinator) Checkpoint(ctx context.Context, id string, copied int64) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	task, err := c.Tasks.GetReplication(ctx, id)
	if err != nil {
		return err
	}
	task.BytesCopied = copied
	task.UpdatedAt = c.now()
	old := task.Version
	task.Version++
	return c.Tasks.UpdateReplication(ctx, task, old)
}
func (c *ReplicationCoordinator) Finish(ctx context.Context, id string, copyErr error) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	task, err := c.Tasks.GetReplication(ctx, id)
	if err != nil {
		return err
	}
	old := task.Version
	if copyErr != nil {
		task.Fail(copyErr.Error(), c.now())
	} else {
		task.Succeed(c.now())
	}
	return c.Tasks.UpdateReplication(ctx, task, old)
}
func StreamCopy(ctx context.Context, dst io.Writer, src io.Reader, limit int64) (int64, error) {
	if limit < 0 {
		return 0, domain.ValidationError{Field: "limit", Reason: "negative"}
	}
	reader := io.Reader(src)
	if limit > 0 {
		reader = io.LimitReader(src, limit)
	}
	return io.Copy(dst, &ctxReader{ctx: ctx, r: reader})
}

type ctxReader struct {
	ctx context.Context
	r   io.Reader
}

func (r *ctxReader) Read(p []byte) (int, error) {
	select {
	case <-r.ctx.Done():
		return 0, r.ctx.Err()
	default:
		return r.r.Read(p)
	}
}
func (c *ReplicationCoordinator) Describe(p CopyProgress) string {
	return fmt.Sprintf("%s %d/%d bytes (%s)", p.TaskID, p.Copied, p.Total, p.State)
}
func (c *ReplicationCoordinator) now() time.Time {
	if c.Clock != nil {
		return c.Clock.Now().UTC()
	}
	return time.Now().UTC()
}
