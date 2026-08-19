package observability

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"
)

type TraceID string
type traceKey struct{}

func NewTraceID() TraceID {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return TraceID(hex.EncodeToString(b))
}
func WithTrace(ctx context.Context, id TraceID) context.Context {
	return context.WithValue(ctx, traceKey{}, id)
}
func TraceFromContext(ctx context.Context) TraceID {
	if id, ok := ctx.Value(traceKey{}).(TraceID); ok {
		return id
	}
	return ""
}

type Span struct {
	TraceID    TraceID           `json:"traceId"`
	Name       string            `json:"name"`
	Start      time.Time         `json:"start"`
	End        time.Time         `json:"end"`
	Attributes map[string]string `json:"attributes"`
	Error      string            `json:"error,omitempty"`
}

func StartSpan(ctx context.Context, name string) (context.Context, *Span) {
	id := TraceFromContext(ctx)
	if id == "" {
		id = NewTraceID()
	}
	return WithTrace(ctx, id), &Span{TraceID: id, Name: name, Start: time.Now().UTC(), Attributes: map[string]string{}}
}
func (s *Span) Set(k, v string) { s.Attributes[k] = v }
func (s *Span) Fail(err error) {
	if err != nil {
		s.Error = err.Error()
	}
}
func (s *Span) EndSpan()               { s.End = time.Now().UTC() }
func (s Span) Duration() time.Duration { return s.End.Sub(s.Start) }
func (s Span) Header() string {
	return "00-" + strings.ToLower(string(s.TraceID)) + "-0000000000000001-01"
}
