package domain

import "time"

type ReplicationState string

const (
	ReplicationQueued    ReplicationState = "queued"
	ReplicationRunning   ReplicationState = "running"
	ReplicationSucceeded ReplicationState = "succeeded"
	ReplicationFailed    ReplicationState = "failed"
	ReplicationCancelled ReplicationState = "cancelled"
)

type ReplicationTask struct {
	ID          string           `json:"id"`
	Source      RepositoryName   `json:"source"`
	Target      string           `json:"target"`
	Reference   string           `json:"reference"`
	State       ReplicationState `json:"state"`
	BytesCopied int64            `json:"bytesCopied"`
	Attempts    int              `json:"attempts"`
	LastError   string           `json:"lastError,omitempty"`
	LeaseUntil  time.Time        `json:"leaseUntil"`
	CreatedAt   time.Time        `json:"createdAt"`
	UpdatedAt   time.Time        `json:"updatedAt"`
	Version     int64            `json:"version"`
}

func (t *ReplicationTask) Claim(until, timeNow time.Time) error {
	if t.State == ReplicationSucceeded || t.State == ReplicationCancelled {
		return ErrInvalidStateTransition
	}
	if t.LeaseUntil.After(timeNow) {
		return ErrLeaseActive
	}
	t.State = ReplicationRunning
	t.LeaseUntil = until
	t.Attempts++
	t.UpdatedAt = timeNow
	t.Version++
	return nil
}
func (t *ReplicationTask) Succeed(now time.Time) {
	t.State = ReplicationSucceeded
	t.LeaseUntil = time.Time{}
	t.UpdatedAt = now
	t.Version++
}
func (t *ReplicationTask) Fail(err string, now time.Time) {
	t.State = ReplicationFailed
	t.LastError = err
	t.LeaseUntil = time.Time{}
	t.UpdatedAt = now
	t.Version++
}
