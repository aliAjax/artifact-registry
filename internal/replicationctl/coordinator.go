package replicationctl

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ErrClosed is returned when a task is enqueued after the coordinator is closed.
var ErrClosed = errors.New("replicationctl: coordinator closed")

// Replicator copies an artifact to a remote target and returns copied bytes.
type Replicator interface {
	Copy(context.Context, Task) (int64, error)
}

// Coordinator drains the queue with a fixed worker pool. It stops cleanly when
// the queue is closed and all buffered tasks have been processed.
type Coordinator struct {
	queue   *Queue
	repl    Replicator
	workers int

	jobs    chan Task
	wg      sync.WaitGroup
	closeMu sync.RWMutex
	closed  bool

	mu     sync.Mutex
	failed []string
	copied int64
}

func NewCoordinator(queue *Queue, repl Replicator, workers int) *Coordinator {
	if workers < 1 {
		workers = 1
	}
	return &Coordinator{queue: queue, repl: repl, workers: workers, jobs: make(chan Task, 64)}
}

func (c *Coordinator) Start(ctx context.Context) {
	c.wg.Add(c.workers)
	for i := 0; i < c.workers; i++ {
		go c.worker(ctx)
	}
}

func (c *Coordinator) worker(ctx context.Context) {
	defer c.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case task, ok := <-c.jobs:
			if !ok {
				return
			}
			n, err := c.repl.Copy(ctx, task)
			c.mu.Lock()
			if err != nil {
				c.failed = append(c.failed, task.ID)
			} else {
				c.copied += n
			}
			c.mu.Unlock()
		}
	}
}

// Enqueue pushes one task onto the shared channel. It returns ErrClosed if the
// coordinator has already been closed, instead of panicking on a send to a
// closed channel.
func (c *Coordinator) Enqueue(task Task) error {
	c.closeMu.RLock()
	defer c.closeMu.RUnlock()
	if c.closed {
		return ErrClosed
	}
	c.jobs <- task
	return nil
}

// Close marks the channel closed so workers drain and exit.
func (c *Coordinator) Close() {
	c.closeMu.Lock()
	defer c.closeMu.Unlock()
	if c.closed {
		return
	}
	c.closed = true
	close(c.jobs)
}

// Wait blocks until all workers exit.
func (c *Coordinator) Wait() { c.wg.Wait() }

// Results returns the number of failed task ids and copied bytes.
func (c *Coordinator) Results() ([]string, int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := append([]string(nil), c.failed...)
	return out, c.copied
}

// Drain pulls every queued task into the worker channel. It returns the number
// of tasks handed off.
func (c *Coordinator) Drain() int {
	n := 0
	for {
		task, ok := c.queue.Pop()
		if !ok {
			break
		}
		_ = c.Enqueue(task)
		n++
	}
	return n
}

var _ = time.Second
