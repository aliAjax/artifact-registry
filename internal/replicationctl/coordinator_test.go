package replicationctl

import (
	"context"
	"sync"
	"testing"
	"time"
)

type noopReplicator struct{}

func (noopReplicator) Copy(context.Context, Task) (int64, error) { return 1, nil }

func TestEnqueueAfterCloseErrors(t *testing.T) {
	q := NewQueue()
	c := NewCoordinator(q, noopReplicator{}, 2)
	c.Start(context.Background())
	c.Close()
	c.Wait()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Enqueue after Close panicked: %v", r)
		}
	}()
	if err := c.Enqueue(Task{ID: "t"}); err == nil {
		t.Fatal("expected error when enqueueing after Close")
	}
}

func TestDrainCountsTasks(t *testing.T) {
	q := NewQueue()
	c := NewCoordinator(q, noopReplicator{}, 2)
	for i := 0; i < 5; i++ {
		q.Push(Task{ID: string(rune('a' + i))})
	}
	if n := c.Drain(); n != 5 {
		t.Fatalf("expected 5 drained tasks, got %d", n)
	}
}

func TestQueuePopAllowsRepush(t *testing.T) {
	q := NewQueue()
	if !q.Push(Task{ID: "t"}) {
		t.Fatal("first push should succeed")
	}
	if _, ok := q.Pop(); !ok {
		t.Fatal("pop should succeed")
	}
	if !q.Push(Task{ID: "t"}) {
		t.Fatal("re-push after pop should succeed")
	}
}

func TestQueueLen(t *testing.T) {
	q := NewQueue()
	q.Push(Task{ID: "a"})
	q.Push(Task{ID: "b"})
	if l := q.Len(); l != 2 {
		t.Fatalf("expected len 2, got %d", l)
	}
}

func TestCoordinatorConcurrentDrain(t *testing.T) {
	q := NewQueue()
	c := NewCoordinator(q, noopReplicator{}, 4)
	c.Start(context.Background())
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 200; i++ {
			q.Push(Task{ID: string(rune(i))})
			_ = c.Enqueue(Task{ID: string(rune(i))})
		}
	}()
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 200; i++ {
			c.Drain()
		}
	}()
	go func() {
		defer wg.Done()
		<-start
		c.Close()
	}()
	close(start)
	wg.Wait()
	time.Sleep(10 * time.Millisecond)
	c.Wait()
}
