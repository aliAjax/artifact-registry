package domain

import "time"

type EventType string

const (
	EventRepositoryCreated    EventType = "repository.created"
	EventUploadStarted        EventType = "upload.started"
	EventUploadChunkReceived  EventType = "upload.chunk.received"
	EventUploadCompleted      EventType = "upload.completed"
	EventManifestPushed       EventType = "manifest.pushed"
	EventManifestDeleted      EventType = "manifest.deleted"
	EventScanSubmitted        EventType = "scan.submitted"
	EventScanCompleted        EventType = "scan.completed"
	EventGCStarted            EventType = "gc.started"
	EventGCCompleted          EventType = "gc.completed"
	EventReplicationQueued    EventType = "replication.queued"
	EventReplicationCompleted EventType = "replication.completed"
)

type DomainEvent struct {
	ID         string            `json:"id"`
	Type       EventType         `json:"type"`
	Tenant     string            `json:"tenant"`
	Repository RepositoryName    `json:"repository"`
	Subject    string            `json:"subject"`
	OccurredAt time.Time         `json:"occurredAt"`
	Sequence   int64             `json:"sequence"`
	Data       map[string]string `json:"data,omitempty"`
	TraceID    string            `json:"traceId,omitempty"`
}

func (e DomainEvent) Validate() error {
	if e.ID == "" {
		return ValidationError{"event.id", "required"}
	}
	if e.Type == "" {
		return ValidationError{"event.type", "required"}
	}
	if e.OccurredAt.IsZero() {
		return ValidationError{"event.occurredAt", "required"}
	}
	return nil
}

type EventCursor struct {
	Sequence  int64     `json:"sequence"`
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
}

func (c EventCursor) String() string { return c.ID }
func (c EventCursor) After(other EventCursor) bool {
	return c.Sequence > other.Sequence || (c.Sequence == other.Sequence && c.ID > other.ID)
}

type OutboxRecord struct {
	ID            string      `json:"id"`
	Event         DomainEvent `json:"event"`
	State         string      `json:"state"`
	Attempts      int         `json:"attempts"`
	NextAttemptAt time.Time   `json:"nextAttemptAt"`
	CreatedAt     time.Time   `json:"createdAt"`
	PublishedAt   *time.Time  `json:"publishedAt,omitempty"`
	LastError     string      `json:"lastError,omitempty"`
}

func (o *OutboxRecord) Claim(now time.Time) error {
	if o.State == "published" {
		return ErrConflictPublished
	}
	if o.NextAttemptAt.After(now) {
		return ErrRetryNotReady
	}
	o.State = "publishing"
	o.Attempts++
	return nil
}
func (o *OutboxRecord) MarkPublished(now time.Time) {
	o.State = "published"
	o.PublishedAt = &now
	o.LastError = ""
}
func (o *OutboxRecord) MarkFailed(err error, now time.Time) {
	o.State = "pending"
	o.LastError = err.Error()
	delay := time.Second << min(o.Attempts, 8)
	o.NextAttemptAt = now.Add(delay)
}

var ErrConflictPublished = ConflictError{Resource: "outbox", Detail: "already published"}
var ErrRetryNotReady = ConflictError{Resource: "outbox", Detail: "retry is not ready"}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
