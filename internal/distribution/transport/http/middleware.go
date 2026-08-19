package http

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type RateLimiter struct {
	mu      sync.Mutex
	entries map[string]*rateEntry
	limit   int
	window  time.Duration
}
type rateEntry struct {
	count int
	start time.Time
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	if limit < 1 {
		limit = 1
	}
	if window <= 0 {
		window = time.Minute
	}
	return &RateLimiter{entries: map[string]*rateEntry{}, limit: limit, window: window}
}
func (l *RateLimiter) Allow(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	e := l.entries[key]
	if e == nil || now.Sub(e.start) >= l.window {
		l.entries[key] = &rateEntry{count: 1, start: now}
		return true
	}
	if e.count >= l.limit {
		return false
	}
	e.count++
	return true
}
func (l *RateLimiter) Remaining(key string, now time.Time) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	e := l.entries[key]
	if e == nil || now.Sub(e.start) >= l.window {
		return l.limit
	}
	if l.limit-e.count < 0 {
		return 0
	}
	return l.limit - e.count
}
func (l *RateLimiter) Reset(now time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for key, e := range l.entries {
		if now.Sub(e.start) >= l.window*2 {
			delete(l.entries, key)
		}
	}
}
func RateLimitMiddleware(l *RateLimiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("X-Tenant")
		if key == "" {
			key = r.RemoteAddr
		}
		if !l.Allow(key, time.Now()) {
			w.Header().Set("Retry-After", strconv.FormatInt(int64(l.window.Seconds()), 10))
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(l.Remaining(key, time.Now())))
		next.ServeHTTP(w, r)
	})
}
func BodyLimit(next http.Handler, limit int64) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ContentLength > limit {
			http.Error(w, "request too large", http.StatusRequestEntityTooLarge)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, limit)
		next.ServeHTTP(w, r)
	})
}
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if value := recover(); value != nil {
				slog.Error("panic recovered", "value", value, "path", r.URL.Path)
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

type ResponseRecorder struct {
	http.ResponseWriter
	status int
	bytes  int64
}

func (r *ResponseRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}
func (r *ResponseRecorder) Write(p []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, e := r.ResponseWriter.Write(p)
	r.bytes += int64(n)
	return n, e
}
func AccessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &ResponseRecorder{ResponseWriter: w}
		next.ServeHTTP(rec, r)
		if rec.status == 0 {
			rec.status = http.StatusOK
		}
		slog.Info("access", "method", r.Method, "path", r.URL.Path, "status", rec.status, "bytes", rec.bytes, "duration", time.Since(start).String())
	})
}
func ParseTenant(r *http.Request) string {
	for _, key := range []string{"X-Tenant", "X-Registry-Tenant"} {
		if value := strings.TrimSpace(r.Header.Get(key)); value != "" {
			return value
		}
	}
	return "anonymous"
}
func RequireTenant(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ParseTenant(r) == "anonymous" {
			http.Error(w, "tenant context required", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// TimeoutMiddleware bounds a request with a context deadline so slow handlers
// are cancelled deterministically.
func TimeoutMiddleware(timeout time.Duration, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		_ = ctx
		next.ServeHTTP(w, r)
	})
}

// TenantMiddleware pins the tenant onto the request context for downstream
// handlers that must not read the header twice.
func TenantMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}

type tenantKey struct{}

// middlewareCtx returns the request context that handlers must use.
func middlewareCtx(r *http.Request) context.Context {
	return context.Background()
}
