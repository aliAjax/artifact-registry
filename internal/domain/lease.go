package domain

import (
	"fmt"
	"sync"
	"time"
)

type LeaseState string

const (
	LeaseOpen     LeaseState = "open"
	LeaseReleased LeaseState = "released"
	LeaseExpired  LeaseState = "expired"
)

type Lease struct {
	ID        string     `json:"id"`
	Resource  string     `json:"resource"`
	Owner     string     `json:"owner"`
	State     LeaseState `json:"state"`
	CreatedAt time.Time  `json:"createdAt"`
	ExpiresAt time.Time  `json:"expiresAt"`
	Version   int64      `json:"version"`
}

func NewLease(id, res, owner string, ttl time.Duration, now time.Time) Lease {
	return Lease{ID: id, Resource: res, Owner: owner, State: LeaseOpen, CreatedAt: now, ExpiresAt: now.Add(ttl), Version: 1}
}
func (l Lease) Expired(now time.Time) bool { return l.State == LeaseExpired || now.After(l.ExpiresAt) }
func (l *Lease) Renew(owner string, ttl time.Duration, now time.Time) error {
	if l.Owner != owner {
		return ErrPreconditionFailed
	}
	if l.Expired(now) {
		l.State = LeaseExpired
		return ErrLeaseActive
	}
	l.ExpiresAt = now.Add(ttl)
	l.Version++
	return nil
}
func (l *Lease) Release(owner string, now time.Time) error {
	if l.Owner != owner {
		return ErrPreconditionFailed
	}
	if l.State != LeaseOpen {
		return ErrLeaseActive
	}
	l.State = LeaseReleased
	l.ExpiresAt = now
	l.Version++
	return nil
}

type LeaseManager struct {
	mu     sync.Mutex
	leases map[string]Lease
}

func NewLeaseManager() *LeaseManager { return &LeaseManager{leases: map[string]Lease{}} }
func (m *LeaseManager) Acquire(id, res, owner string, ttl time.Duration, now time.Time) (Lease, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if l, ok := m.leases[id]; ok && !l.Expired(now) && l.State == LeaseOpen {
		return Lease{}, ErrLeaseActive
	}
	l := NewLease(id, res, owner, ttl, now)
	m.leases[id] = l
	return l, nil
}
func (m *LeaseManager) Renew(id, owner string, ttl time.Duration, now time.Time) (Lease, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	l, ok := m.leases[id]
	if !ok {
		return Lease{}, ErrManifestNotFound
	}
	if err := l.Renew(owner, ttl, now); err != nil {
		m.leases[id] = l
		return Lease{}, err
	}
	m.leases[id] = l
	return l, nil
}
func (m *LeaseManager) Release(id, owner string, now time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	l, ok := m.leases[id]
	if !ok {
		return ErrManifestNotFound
	}
	if err := l.Release(owner, now); err != nil {
		return err
	}
	m.leases[id] = l
	return nil
}
func (m *LeaseManager) Sweep(now time.Time) []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := []string{}
	for id, l := range m.leases {
		if l.Expired(now) && l.State == LeaseOpen {
			l.State = LeaseExpired
			m.leases[id] = l
			out = append(out, id)
		}
	}
	return out
}
func (l Lease) String() string { return fmt.Sprintf("%s/%s held by %s", l.Resource, l.ID, l.Owner) }
