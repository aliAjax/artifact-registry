package replicationctl

import "sync"

// Task describes one remote copy that must be performed.
type Task struct {
	ID       string
	Artifact string
	Target   string
	Attempts int
}

// Queue is a bounded in-memory task queue with de-duplication by task id.
type Queue struct {
	mu    sync.Mutex
	items []Task
	seen  map[string]bool
}

func NewQueue() *Queue {
	return &Queue{items: []Task{}, seen: map[string]bool{}}
}

// Push appends a task unless an identical task id is already queued.
func (q *Queue) Push(t Task) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.seen[t.ID] {
		return false
	}
	q.seen[t.ID] = true
	q.items = append(q.items, t)
	return true
}

// Pop removes and returns the next task, if any.
func (q *Queue) Pop() (Task, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.items) == 0 {
		return Task{}, false
	}
	t := q.items[0]
	q.items = q.items[1:]
	delete(q.seen, t.ID)
	return t, true
}

// Len returns the number of queued tasks.
func (q *Queue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.items)
}
