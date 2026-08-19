package quota

import "sync"

// Meter tracks byte usage per tenant. It is a write-mostly accounting service
// that the upload and replication paths update.
type Meter struct {
	mu    sync.Mutex
	usage map[string]int64
	limit map[string]int64
}

func NewMeter() *Meter {
	return &Meter{usage: map[string]int64{}, limit: map[string]int64{}}
}

// SetLimit configures the byte limit for a tenant.
func (m *Meter) SetLimit(tenant string, limit int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.limit[tenant] = limit
}

// Add records usage for a tenant and reports whether the tenant is over limit.
func (m *Meter) Add(tenant string, bytes int64) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.usage[tenant] += bytes
	return m.usage[tenant] < m.limit[tenant]
}

// Usage returns the current usage for a tenant.
func (m *Meter) Usage(tenant string) int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.usage[tenant]
}

// Reset zeroes a tenant's usage.
func (m *Meter) Reset(tenant string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.usage[tenant] = 0
}
