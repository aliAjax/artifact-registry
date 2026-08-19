package httpx

import (
	"artifact-registry/internal/domain"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func WriteError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	code := "INTERNAL_ERROR"
	switch {
	case domain.IsNotFound(err):
		status = http.StatusNotFound
		code = "NOT_FOUND"
	case errors.Is(err, domain.ErrInvalidDigest), errors.Is(err, domain.ErrInvalidRepository), errors.Is(err, domain.ErrInvalidRange), errors.Is(err, domain.ErrInvalidMediaType):
		status = http.StatusBadRequest
		code = "INVALID_REQUEST"
	case errors.Is(err, domain.ErrDigestMismatch):
		status = http.StatusBadRequest
		code = "DIGEST_INVALID"
	case errors.Is(err, domain.ErrQuotaExceeded):
		status = http.StatusRequestEntityTooLarge
		code = "DENIED"
	case errors.Is(err, domain.ErrPreconditionFailed):
		status = http.StatusPreconditionFailed
		code = "PRECONDITION_FAILED"
	case errors.Is(err, domain.ErrTagImmutable), errors.Is(err, domain.ErrUploadConflict), errors.Is(err, domain.ErrUploadOutOfOrder), errors.Is(err, domain.ErrIdempotencyConflict):
		status = http.StatusConflict
		code = "CONFLICT"
	case errors.Is(err, domain.ErrManifestBlocked):
		status = http.StatusForbidden
		code = "DENIED"
	case errors.Is(err, domain.ErrNotImplemented):
		status = http.StatusNotImplemented
		code = "UNIMPLEMENTED"
	}
	var validation domain.ValidationError
	if errors.As(err, &validation) {
		status = http.StatusBadRequest
		code = "INVALID_REQUEST"
	}
	WriteJSON(w, status, Error{Code: code, Message: err.Error()})
}
func ParsePage(r *http.Request) domain.PageRequest {
	limit, _ := strconv.Atoi(r.URL.Query().Get("n"))
	return domain.PageRequest{Limit: limit, Last: r.URL.Query().Get("last")}
}
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = strconv.FormatInt(time.Now().UnixNano(), 36)
		}
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey{}, id)))
	})
}

type requestIDKey struct{}

func Log(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		slog.Info("http request", "method", r.Method, "path", r.URL.Path, "duration", time.Since(start).String())
	})
}
