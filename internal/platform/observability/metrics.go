package observability

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

type Counter struct{ value atomic.Int64 }

func (c *Counter) Inc()         { c.value.Add(1) }
func (c *Counter) Add(v int64)  { c.value.Add(v) }
func (c *Counter) Value() int64 { return c.value.Load() }

type Gauge struct{ value atomic.Int64 }

func (g *Gauge) Set(v int64)  { g.value.Store(v) }
func (g *Gauge) Add(v int64)  { g.value.Add(v) }
func (g *Gauge) Value() int64 { return g.value.Load() }

type Histogram struct {
	mu     sync.Mutex
	values []float64
	count  Counter
	sum    atomic.Uint64
}

func (h *Histogram) Observe(v float64) {
	h.mu.Lock()
	if len(h.values) < 10000 {
		h.values = append(h.values, v)
	}
	h.mu.Unlock()
	h.count.Inc()
	h.sum.Store(math.Float64bits(h.Sum() + v))
}
func (h *Histogram) Count() int64 { return h.count.Value() }

// Values returns a copy of the recorded observations.
func (h *Histogram) Values() []float64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]float64(nil), h.values...)
}
func (h *Histogram) Sum() float64 { return math.Float64frombits(h.sum.Load()) }
func (h *Histogram) Quantile(q float64) float64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.values) == 0 {
		return 0
	}
	copyv := append([]float64(nil), h.values...)
	sort.Float64s(copyv)
	if q < 0 {
		q = 0
	}
	if q > 1 {
		q = 1
	}
	return copyv[int(float64(len(copyv)-1)*q)]
}

type Registry struct {
	Counters   map[string]*Counter
	Gauges     map[string]*Gauge
	Histograms map[string]*Histogram
	mu         sync.RWMutex
}

func NewRegistry() *Registry {
	return &Registry{Counters: map[string]*Counter{}, Gauges: map[string]*Gauge{}, Histograms: map[string]*Histogram{}}
}
func (r *Registry) Counter(name string) *Counter {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Counters[name] == nil {
		r.Counters[name] = &Counter{}
	}
	return r.Counters[name]
}
func (r *Registry) Gauge(name string) *Gauge {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Gauges[name] == nil {
		r.Gauges[name] = &Gauge{}
	}
	return r.Gauges[name]
}
func (r *Registry) Histogram(name string) *Histogram {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Histograms[name] == nil {
		r.Histograms[name] = &Histogram{}
	}
	return r.Histograms[name]
}
func (r *Registry) Snapshot() map[string]any {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := map[string]any{}
	for n, c := range r.Counters {
		out["counter."+n] = c.Value()
	}
	for n, g := range r.Gauges {
		out["gauge."+n] = g.Value()
	}
	for n, h := range r.Histograms {
		out["histogram."+n] = map[string]any{"count": h.Count(), "sum": h.Sum(), "p50": h.Quantile(.5), "p95": h.Quantile(.95)}
	}
	return out
}
func (r *Registry) Text() string {
	snap := r.Snapshot()
	result := ""
	for k, v := range snap {
		result += fmt.Sprintf("artifact_registry_%s %v\n", k, v)
	}
	return result
}

type Timer struct {
	start time.Time
	hist  *Histogram
}

func StartTimer(h *Histogram) Timer { return Timer{start: time.Now(), hist: h} }
func (t Timer) Stop() {
	if t.hist != nil {
		t.hist.Observe(time.Since(t.start).Seconds())
	}
}
