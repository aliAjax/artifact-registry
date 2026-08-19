// Package http implements the OCI Distribution HTTP surface and management API.
package http

import (
	"artifact-registry/internal/application"
	"artifact-registry/internal/domain"
	"artifact-registry/internal/platform/httpx"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

type Router struct {
	registry  *application.RegistryService
	lifecycle *application.LifecycleService
	mux       *http.ServeMux
}

func NewRouter(registry *application.RegistryService, lifecycle *application.LifecycleService) http.Handler {
	r := &Router{registry: registry, lifecycle: lifecycle, mux: http.NewServeMux()}
	r.routes()
	return httpx.RequestID(httpx.Log(r.mux))
}
func (r *Router) routes() {
	r.mux.HandleFunc("GET /healthz", r.health)
	r.mux.HandleFunc("GET /readyz", r.health)
	r.mux.HandleFunc("GET /v2/", r.v2)
	r.mux.HandleFunc("/v2/", r.dispatchV2)
	r.mux.HandleFunc("POST /api/v1/repositories", r.createRepository)
	r.mux.HandleFunc("GET /api/v1/repositories", r.listRepositories)
	r.mux.HandleFunc("PATCH /api/v1/repositories/", r.dispatchManagement)
	r.mux.HandleFunc("GET /api/v1/scan-reports", r.listScans)
	r.mux.HandleFunc("POST /api/v1/repositories/", r.dispatchManagement)
	r.mux.HandleFunc("POST /api/v1/retention-policies", r.createPolicy)
	r.mux.HandleFunc("GET /api/v1/retention-policies", r.listPolicies)
	r.mux.HandleFunc("DELETE /api/v1/retention-policies/{id}", r.deletePolicy)
	r.mux.HandleFunc("POST /api/v1/replications", r.createReplication)
	r.mux.HandleFunc("GET /api/v1/replications", r.listReplications)
	r.mux.HandleFunc("POST /api/v1/replications/{id}/run", r.runReplication)
	r.mux.HandleFunc("POST /api/v1/gc/dry-run", r.gcDryRun)
	r.mux.HandleFunc("POST /api/v1/gc/execute", r.gcExecute)
}

// dispatchV2 keeps multi-segment repository names compatible with Go ServeMux.
func (r *Router) dispatchV2(w http.ResponseWriter, req *http.Request) {
	parts := strings.Split(strings.TrimPrefix(strings.TrimPrefix(req.URL.Path, "/v2/"), "/"), "/")
	for i := range parts {
		if parts[i] == "blobs" || parts[i] == "manifests" || parts[i] == "tags" || parts[i] == "referrers" {
			repo := strings.Join(parts[:i], "/")
			req.SetPathValue("repository", repo)
			switch {
			case parts[i] == "blobs" && i+1 < len(parts) && parts[i+1] == "uploads":
				if i+2 == len(parts) {
					if req.Method == http.MethodPost {
						r.startUpload(w, req)
						return
					}
				}
				if i+2 < len(parts) {
					req.SetPathValue("uuid", parts[i+2])
					switch req.Method {
					case http.MethodPatch:
						r.patchUpload(w, req)
					case http.MethodPut:
						r.completeUpload(w, req)
					case http.MethodGet:
						r.uploadStatus(w, req)
					case http.MethodDelete:
						r.abortUpload(w, req)
					default:
						http.NotFound(w, req)
					}
					return
				}
			case parts[i] == "blobs" && i+1 < len(parts):
				req.SetPathValue("digest", strings.Join(parts[i+1:], "/"))
				if req.Method == http.MethodGet {
					r.getBlob(w, req)
				} else if req.Method == http.MethodHead {
					r.headBlob(w, req)
				} else {
					http.NotFound(w, req)
				}
				return
			case parts[i] == "manifests" && i+1 < len(parts):
				req.SetPathValue("reference", strings.Join(parts[i+1:], "/"))
				switch req.Method {
				case http.MethodPut:
					r.putManifest(w, req)
				case http.MethodGet:
					r.getManifest(w, req)
				case http.MethodHead:
					r.headManifest(w, req)
				case http.MethodDelete:
					req.SetPathValue("digest", strings.Join(parts[i+1:], "/"))
					r.deleteManifest(w, req)
				default:
					http.NotFound(w, req)
				}
				return
			case parts[i] == "tags" && i+1 < len(parts) && parts[i+1] == "list":
				r.tags(w, req)
				return
			case parts[i] == "referrers" && i+1 < len(parts):
				req.SetPathValue("digest", strings.Join(parts[i+1:], "/"))
				r.referrers(w, req)
				return
			}
		}
	}
	http.NotFound(w, req)
}

func (r *Router) dispatchManagement(w http.ResponseWriter, req *http.Request) {
	path := strings.TrimPrefix(req.URL.Path, "/api/v1/repositories/")
	if strings.HasSuffix(path, "/scan") {
		req.SetPathValue("repository", strings.TrimSuffix(path, "/scan"))
		r.submitScan(w, req)
		return
	}
	if req.Method == http.MethodPatch {
		req.SetPathValue("repository", path)
		r.updateRepository(w, req)
		return
	}
	http.NotFound(w, req)
}
func (r *Router) health(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"status": "ok", "service": "artifact-registry"})
}
func (r *Router) v2(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Docker-Distribution-API-Version", "registry/2.0")
	w.WriteHeader(http.StatusOK)
}
func (r *Router) startUpload(w http.ResponseWriter, req *http.Request) {
	repo := req.PathValue("repository")
	if mount, from := req.URL.Query().Get("mount"), req.URL.Query().Get("from"); mount != "" && from != "" {
		b, err := r.registry.MountBlob(req.Context(), repo, from, mount)
		if err == nil {
			w.Header().Set("Location", "/v2/"+repo+"/blobs/"+b.Digest.String())
			w.Header().Set("Docker-Content-Digest", b.Digest.String())
			w.WriteHeader(http.StatusCreated)
			return
		}
		if !errors.Is(err, domain.ErrBlobNotFound) {
			httpx.WriteError(w, err)
			return
		}
	}
	var expected *int64
	if raw := req.Header.Get("X-Expected-Size"); raw != "" {
		v, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			httpx.WriteError(w, domain.ValidationError{Field: "X-Expected-Size", Reason: "must be an integer"})
			return
		}
		expected = &v
	}
	session, err := r.registry.StartUpload(req.Context(), repo, expected)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	r.writeUploadHeaders(w, repo, session)
	w.WriteHeader(http.StatusAccepted)
}
func (r *Router) patchUpload(w http.ResponseWriter, req *http.Request) {
	repo, id := req.PathValue("repository"), req.PathValue("uuid")
	rng, err := parseContentRange(req.Header.Get("Content-Range"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	session, err := r.registry.UploadChunk(req.Context(), repo, id, req.Header.Get("X-Upload-Token"), rng, io.LimitReader(req.Body, rng.Length()+1))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	r.writeUploadHeaders(w, repo, session)
	w.WriteHeader(http.StatusAccepted)
}
func (r *Router) completeUpload(w http.ResponseWriter, req *http.Request) {
	repo, id := req.PathValue("repository"), req.PathValue("uuid")
	digest := req.URL.Query().Get("digest")
	if digest == "" {
		httpx.WriteError(w, domain.ValidationError{Field: "digest", Reason: "query parameter is required"})
		return
	}
	b, err := r.registry.CompleteUpload(req.Context(), repo, id, req.Header.Get("X-Upload-Token"), digest, req.Header.Get("Content-Type"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.Header().Set("Location", "/v2/"+repo+"/blobs/"+b.Digest.String())
	w.Header().Set("Docker-Content-Digest", b.Digest.String())
	w.WriteHeader(http.StatusCreated)
}
func (r *Router) uploadStatus(w http.ResponseWriter, req *http.Request) {
	u, err := r.registry.Uploads.GetUpload(req.Context(), req.PathValue("uuid"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	if u.Repository.String() != req.PathValue("repository") {
		httpx.WriteError(w, domain.ErrUploadNotFound)
		return
	}
	r.writeUploadHeaders(w, u.Repository.String(), u)
	w.WriteHeader(http.StatusNoContent)
}
func (r *Router) abortUpload(w http.ResponseWriter, req *http.Request) {
	err := r.registry.AbortUpload(req.Context(), req.PathValue("repository"), req.PathValue("uuid"), req.Header.Get("X-Upload-Token"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (r *Router) writeUploadHeaders(w http.ResponseWriter, repo string, u *domain.UploadSession) {
	w.Header().Set("Location", "/v2/"+repo+"/blobs/uploads/"+u.ID)
	w.Header().Set("Docker-Upload-UUID", u.ID)
	w.Header().Set("X-Upload-Token", u.Token)
	if end := u.ContiguousEnd(); end >= 0 {
		w.Header().Set("Range", fmt.Sprintf("0-%d", end))
	}
}
func (r *Router) getBlob(w http.ResponseWriter, req *http.Request)  { r.serveBlob(w, req, false) }
func (r *Router) headBlob(w http.ResponseWriter, req *http.Request) { r.serveBlob(w, req, true) }
func (r *Router) serveBlob(w http.ResponseWriter, req *http.Request, head bool) {
	digest := req.PathValue("digest")
	var rng *domain.ByteRange
	rangeHeader := req.Header.Get("Range")
	if rangeHeader != "" {
		v, err := parseHTTPRange(rangeHeader)
		if err != nil {
			httpx.WriteError(w, err)
			return
		}
		rng = &v
	}
	stream, b, err := r.registry.GetBlob(req.Context(), req.PathValue("repository"), digest, rng)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	if stream != nil {
		defer stream.Close()
	}
	w.Header().Set("Docker-Content-Digest", b.Digest.String())
	w.Header().Set("ETag", `"`+b.Digest.String()+`"`)
	w.Header().Set("Content-Type", b.MediaType)
	w.Header().Set("Accept-Ranges", "bytes")
	start, end := int64(0), b.Size-1
	if rng != nil {
		start, end = rng.Start, rng.End
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, b.Size))
		w.Header().Set("Content-Length", strconv.FormatInt(rng.Length(), 10))
		w.WriteHeader(http.StatusPartialContent)
	} else {
		w.Header().Set("Content-Length", strconv.FormatInt(b.Size, 10))
		w.WriteHeader(http.StatusOK)
	}
	_ = start
	if !head && stream != nil {
		_, _ = io.Copy(w, stream)
	}
}
func (r *Router) putManifest(w http.ResponseWriter, req *http.Request) {
	data, err := io.ReadAll(io.LimitReader(req.Body, 16<<20+1))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	m, err := r.registry.PutManifest(req.Context(), req.PathValue("repository"), req.PathValue("reference"), req.Header.Get("Content-Type"), data)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.Header().Set("Docker-Content-Digest", m.Digest.String())
	w.Header().Set("Location", "/v2/"+req.PathValue("repository")+"/manifests/"+m.Digest.String())
	w.WriteHeader(http.StatusCreated)
}
func (r *Router) getManifest(w http.ResponseWriter, req *http.Request) {
	r.serveManifest(w, req, false)
}
func (r *Router) headManifest(w http.ResponseWriter, req *http.Request) {
	r.serveManifest(w, req, true)
}
func (r *Router) serveManifest(w http.ResponseWriter, req *http.Request, head bool) {
	m, err := r.registry.GetManifest(req.Context(), req.PathValue("repository"), req.PathValue("reference"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	if req.Header.Get("If-None-Match") == m.ETag() {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Docker-Content-Digest", m.Digest.String())
	w.Header().Set("ETag", m.ETag())
	w.Header().Set("Content-Type", m.MediaType)
	w.Header().Set("Content-Length", strconv.FormatInt(m.Size, 10))
	if head {
		w.WriteHeader(http.StatusOK)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(m.Raw)
}
func (r *Router) deleteManifest(w http.ResponseWriter, req *http.Request) {
	err := r.registry.DeleteManifest(req.Context(), req.PathValue("repository"), req.PathValue("digest"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}
func (r *Router) tags(w http.ResponseWriter, req *http.Request) {
	p, err := r.registry.ListTags(req.Context(), req.PathValue("repository"), httpx.ParsePage(req))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	tags := make([]string, 0, len(p.Items))
	for _, t := range p.Items {
		tags = append(tags, t.Name)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"name": req.PathValue("repository"), "tags": tags, "next": p.Next})
}
func (r *Router) referrers(w http.ResponseWriter, req *http.Request) {
	items, err := r.registry.Referrers(req.Context(), req.PathValue("repository"), req.PathValue("digest"), req.URL.Query().Get("artifactType"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	descriptors := make([]domain.Descriptor, 0, len(items))
	for _, m := range items {
		descriptors = append(descriptors, m.Descriptor())
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"schemaVersion": 2, "mediaType": "application/vnd.oci.image.index.v1+json", "manifests": descriptors})
}
