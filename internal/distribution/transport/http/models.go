package http

import (
	"artifact-registry/internal/domain"
	"encoding/json"
	"net/http"
	"strings"
)

type OCIErrorCode string

const (
	OCINameUnknown     OCIErrorCode = "NAME_UNKNOWN"
	OCIDigestInvalid   OCIErrorCode = "DIGEST_INVALID"
	OCIManifestUnknown OCIErrorCode = "MANIFEST_UNKNOWN"
	OCIBlobUnknown     OCIErrorCode = "BLOB_UNKNOWN"
	OCIUploadUnknown   OCIErrorCode = "BLOB_UPLOAD_UNKNOWN"
	OCIUploadInvalid   OCIErrorCode = "BLOB_UPLOAD_INVALID"
	OCIDenied          OCIErrorCode = "DENIED"
	OCIUnsupported     OCIErrorCode = "UNSUPPORTED"
)

type OCIError struct {
	Code    OCIErrorCode `json:"code"`
	Message string       `json:"message"`
	Detail  any          `json:"detail,omitempty"`
}
type OCIErrorResponse struct {
	Errors []OCIError `json:"errors"`
}

func ociError(code OCIErrorCode, message string) OCIErrorResponse {
	return OCIErrorResponse{Errors: []OCIError{{Code: code, Message: message}}}
}
func writeOCIError(w http.ResponseWriter, status int, code OCIErrorCode, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = jsonEncode(w, ociError(code, err.Error()))
}
func jsonEncode(w http.ResponseWriter, v any) error { return json.NewEncoder(w).Encode(v) }

type ManifestHeaders struct {
	Digest    string
	MediaType string
	Size      int64
	ETag      string
}

func readManifestHeaders(r *http.Request) ManifestHeaders {
	return ManifestHeaders{Digest: r.Header.Get("Docker-Content-Digest"), MediaType: domain.NormalizeMediaType(r.Header.Get("Content-Type")), Size: r.ContentLength, ETag: r.Header.Get("If-Match")}
}
func parseAcceptHeader(r *http.Request) []string {
	raw := r.Header.Get("Accept")
	if raw == "" {
		return []string{domain.MediaOCIManifest, domain.MediaOCIIndex, domain.MediaDockerManifest}
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(strings.SplitN(p, ";", 2)[0])
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
func accepts(r *http.Request, mediaType string) bool {
	for _, v := range parseAcceptHeader(r) {
		if v == "*/*" || v == mediaType {
			return true
		}
	}
	return false
}

type RangeSpec struct {
	Start          int64
	End            int64
	CompleteLength int64
}

func (r RangeSpec) ContentRange() string {
	return "bytes " + itoa(r.Start) + "-" + itoa(r.End) + "/" + itoa(r.CompleteLength)
}
func itoa(v int64) string {
	if v == 0 {
		return "0"
	}
	negative := v < 0
	if negative {
		v = -v
	}
	digits := ""
	for v > 0 {
		digits = string(byte('0'+v%10)) + digits
		v /= 10
	}
	if negative {
		return "-" + digits
	}
	return digits
}

type UploadResponse struct {
	ID       string `json:"id"`
	Location string `json:"location"`
	Range    string `json:"range"`
	State    string `json:"state"`
	Received int64  `json:"received"`
	Expires  string `json:"expires"`
}

func uploadResponse(u domain.UploadSession, location string) UploadResponse {
	return UploadResponse{ID: u.ID, Location: location, Range: rangeString(u.ContiguousEnd()), State: string(u.State), Received: u.ContiguousEnd() + 1, Expires: u.ExpiresAt.UTC().Format("2006-01-02T15:04:05Z")}
}
func rangeString(end int64) string {
	if end < 0 {
		return ""
	}
	return "0-" + itoa(end)
}
