package application

import (
	"artifact-registry/internal/domain"
	"context"
	"fmt"
	"sync"
	"time"
)

type QuotaService struct {
	Store QuotaStore
	Clock Clock
	mu    sync.Mutex
}
type Reservation struct {
	ID        string    `json:"id"`
	Tenant    string    `json:"tenant"`
	Bytes     int64     `json:"bytes"`
	State     string    `json:"state"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
}

func (q *QuotaService) Reserve(ctx context.Context, tenant string, bytes int64, ttl time.Duration) (Reservation, error) {
	if bytes < 0 {
		return Reservation{}, domain.ValidationError{Field: "bytes", Reason: "negative"}
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	quota, err := q.Store.GetQuota(ctx, tenant)
	if err != nil {
		return Reservation{}, err
	}
	old := quota.Version
	if err = quota.Reserve(bytes, q.now()); err != nil {
		return Reservation{}, err
	}
	if err = q.Store.UpdateQuota(ctx, quota, old); err != nil {
		return Reservation{}, err
	}
	return Reservation{ID: fmt.Sprintf("res-%d", q.now().UnixNano()), Tenant: tenant, Bytes: bytes, State: "reserved", CreatedAt: q.now(), ExpiresAt: q.now().Add(ttl)}, nil
}
func (q *QuotaService) Commit(ctx context.Context, r Reservation, actual int64) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if r.State != "reserved" {
		return domain.ErrQuotaExceeded
	}
	quota, err := q.Store.GetQuota(ctx, r.Tenant)
	if err != nil {
		return err
	}
	old := quota.Version
	if err = quota.Commit(r.Bytes, actual, q.now()); err != nil {
		return err
	}
	return q.Store.UpdateQuota(ctx, quota, old)
}
func (q *QuotaService) Release(ctx context.Context, r Reservation) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	quota, err := q.Store.GetQuota(ctx, r.Tenant)
	if err != nil {
		return err
	}
	old := quota.Version
	if err = quota.Release(r.Bytes, q.now()); err != nil {
		return err
	}
	return q.Store.UpdateQuota(ctx, quota, old)
}
func (q *QuotaService) Snapshot(ctx context.Context, tenant string) (domain.Quota, error) {
	v, err := q.Store.GetQuota(ctx, tenant)
	if err != nil {
		return domain.Quota{}, err
	}
	return *v, nil
}
func (q *QuotaService) now() time.Time {
	if q.Clock != nil {
		return q.Clock.Now().UTC()
	}
	return time.Now().UTC()
}
