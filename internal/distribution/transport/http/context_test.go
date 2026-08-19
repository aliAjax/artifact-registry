package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTimeoutMiddlewareCancels(t *testing.T) {
	cancelled := make(chan struct{}, 1)
	slow := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
			cancelled <- struct{}{}
		case <-time.After(200 * time.Millisecond):
		}
	})
	h := TimeoutMiddleware(10*time.Millisecond, slow)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	h.ServeHTTP(httptest.NewRecorder(), req)
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("request context was not cancelled")
	}
}

func TestTenantMiddlewarePropagates(t *testing.T) {
	got := make(chan string, 1)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		v, _ := r.Context().Value(tenantKey{}).(string)
		got <- v
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Tenant", "acme")
	TenantMiddleware(next).ServeHTTP(httptest.NewRecorder(), req)
	if v := <-got; v != "acme" {
		t.Fatalf("tenant not propagated, got %q", v)
	}
}

func TestWaitForReadyHonorsCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if waitForReady(ctx, 100) {
		t.Fatal("cancelled context should stop waitForReady")
	}
}

func TestActorFromContext(t *testing.T) {
	ctx := context.WithValue(context.Background(), actorKey{}, "operator")
	if v := actorFromContext(ctx); v != "operator" {
		t.Fatalf("expected operator, got %q", v)
	}
}

func TestRequestActor(t *testing.T) {
	ctx := context.WithValue(context.Background(), actorKey{}, "operator")
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	if v := requestActor(req); v != "operator" {
		t.Fatalf("expected operator, got %q", v)
	}
}

func TestMiddlewareCtx(t *testing.T) {
	ctx := context.WithValue(context.Background(), tenantKey{}, "acme")
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	if v, _ := middlewareCtx(req).Value(tenantKey{}).(string); v != "acme" {
		t.Fatalf("expected tenant acme, got %q", v)
	}
}
