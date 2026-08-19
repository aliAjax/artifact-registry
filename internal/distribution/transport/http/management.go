package http

import (
	"artifact-registry/internal/application"
	"artifact-registry/internal/domain"
	"artifact-registry/internal/platform/httpx"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func decode(req *http.Request, target any) error {
	d := json.NewDecoder(http.MaxBytesReader(nil, req.Body, 1<<20))
	d.DisallowUnknownFields()
	return d.Decode(target)
}
func (r *Router) createRepository(w http.ResponseWriter, req *http.Request) {
	var cmd application.CreateRepositoryCommand
	if err := decode(req, &cmd); err != nil {
		httpx.WriteError(w, domain.ValidationError{Field: "body", Reason: err.Error()})
		return
	}
	repo, err := r.registry.CreateRepository(req.Context(), cmd)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, repo)
}
func (r *Router) listRepositories(w http.ResponseWriter, req *http.Request) {
	p, err := r.registry.ListRepositories(req.Context(), req.URL.Query().Get("prefix"), httpx.ParsePage(req))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, p)
}
func (r *Router) updateRepository(w http.ResponseWriter, req *http.Request) {
	var body struct {
		ImmutableTags *bool `json:"immutableTags"`
	}
	if err := decode(req, &body); err != nil || body.ImmutableTags == nil {
		httpx.WriteError(w, domain.ValidationError{Field: "immutableTags", Reason: "required boolean"})
		return
	}
	version, _ := strconv.ParseInt(req.Header.Get("If-Match-Version"), 10, 64)
	repo, err := r.registry.SetTagImmutability(req.Context(), req.PathValue("repository"), *body.ImmutableTags, version)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, repo)
}
func (r *Router) submitScan(w http.ResponseWriter, req *http.Request) {
	var body struct {
		Reference string `json:"reference"`
	}
	if err := decode(req, &body); err != nil || body.Reference == "" {
		httpx.WriteError(w, domain.ValidationError{Field: "reference", Reason: "required"})
		return
	}
	report, err := r.lifecycle.SubmitScan(req.Context(), req.PathValue("repository"), body.Reference)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, report)
}
func (r *Router) listScans(w http.ResponseWriter, req *http.Request) {
	p, err := r.lifecycle.ListScans(req.Context(), req.URL.Query().Get("repository"), httpx.ParsePage(req))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, p)
}
func (r *Router) createPolicy(w http.ResponseWriter, req *http.Request) {
	var body struct {
		ID               string `json:"id"`
		RepositoryPrefix string `json:"repositoryPrefix"`
		KeepLast         int    `json:"keepLast"`
		MaxAge           string `json:"maxAge"`
		KeepTagged       bool   `json:"keepTagged"`
		Enabled          bool   `json:"enabled"`
	}
	if err := decode(req, &body); err != nil {
		httpx.WriteError(w, domain.ValidationError{Field: "body", Reason: err.Error()})
		return
	}
	var d time.Duration
	var err error
	if body.MaxAge != "" {
		d, err = time.ParseDuration(body.MaxAge)
		if err != nil {
			httpx.WriteError(w, domain.ValidationError{Field: "maxAge", Reason: "must be duration"})
			return
		}
	}
	p := domain.RetentionPolicy{ID: body.ID, RepositoryPrefix: body.RepositoryPrefix, KeepLast: body.KeepLast, MaxAge: d, KeepTagged: body.KeepTagged, Enabled: body.Enabled, CreatedAt: time.Now().UTC()}
	if err = r.lifecycle.Policies.PutPolicy(req.Context(), p); err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, p)
}
func (r *Router) listPolicies(w http.ResponseWriter, req *http.Request) {
	p, err := r.lifecycle.Policies.ListPolicies(req.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": p})
}
func (r *Router) deletePolicy(w http.ResponseWriter, req *http.Request) {
	if err := r.lifecycle.Policies.DeletePolicy(req.Context(), req.PathValue("id")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (r *Router) createReplication(w http.ResponseWriter, req *http.Request) {
	var body struct {
		ID        string `json:"id"`
		Source    string `json:"source"`
		Target    string `json:"target"`
		Reference string `json:"reference"`
	}
	if err := decode(req, &body); err != nil {
		httpx.WriteError(w, domain.ValidationError{Field: "body", Reason: err.Error()})
		return
	}
	task, err := r.lifecycle.CreateReplication(req.Context(), body.Source, body.Target, body.Reference, body.ID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, task)
}
func (r *Router) listReplications(w http.ResponseWriter, req *http.Request) {
	p, err := r.lifecycle.ListReplications(req.Context(), httpx.ParsePage(req))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, p)
}
func (r *Router) runReplication(w http.ResponseWriter, req *http.Request) {
	task, err := r.lifecycle.RunReplication(req.Context(), req.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, task)
}
func (r *Router) gcDryRun(w http.ResponseWriter, req *http.Request)  { r.runGC(w, req, true) }
func (r *Router) gcExecute(w http.ResponseWriter, req *http.Request) { r.runGC(w, req, false) }
func (r *Router) runGC(w http.ResponseWriter, req *http.Request, dry bool) {
	var body struct {
		Repository string `json:"repository"`
		Force      bool   `json:"force"`
	}
	if req.ContentLength > 0 {
		if err := decode(req, &body); err != nil {
			httpx.WriteError(w, domain.ValidationError{Field: "body", Reason: err.Error()})
			return
		}
	}
	report, err := r.lifecycle.GarbageCollect(req.Context(), application.GCRequest{IdempotencyKey: req.Header.Get("Idempotency-Key"), Confirm: req.Header.Get("X-Confirm-GC"), DryRun: dry, Repository: body.Repository, Force: body.Force})
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, report)
}
func parseContentRange(v string) (domain.ByteRange, error) {
	v = strings.TrimSpace(v)
	if !strings.HasPrefix(v, "bytes ") {
		return domain.ByteRange{}, domain.ValidationError{Field: "Content-Range", Reason: "expected bytes start-end"}
	}
	part := strings.TrimPrefix(v, "bytes ")
	if i := strings.Index(part, "/"); i >= 0 {
		part = part[:i]
	}
	pieces := strings.Split(part, "-")
	if len(pieces) != 2 {
		return domain.ByteRange{}, domain.ErrInvalidRange
	}
	a, e1 := strconv.ParseInt(pieces[0], 10, 64)
	b, e2 := strconv.ParseInt(pieces[1], 10, 64)
	if e1 != nil || e2 != nil {
		return domain.ByteRange{}, domain.ErrInvalidRange
	}
	return domain.NewByteRange(a, b)
}
func parseHTTPRange(v string) (domain.ByteRange, error) {
	if !strings.HasPrefix(v, "bytes=") {
		return domain.ByteRange{}, domain.ErrInvalidRange
	}
	part := strings.TrimPrefix(v, "bytes=")
	if strings.Contains(part, ",") || strings.HasPrefix(part, "-") || strings.HasSuffix(part, "-") {
		return domain.ByteRange{}, domain.ErrInvalidRange
	}
	pieces := strings.Split(part, "-")
	if len(pieces) != 2 {
		return domain.ByteRange{}, domain.ErrInvalidRange
	}
	a, e1 := strconv.ParseInt(pieces[0], 10, 64)
	b, e2 := strconv.ParseInt(pieces[1], 10, 64)
	if e1 != nil || e2 != nil {
		return domain.ByteRange{}, domain.ErrInvalidRange
	}
	return domain.NewByteRange(a, b)
}
