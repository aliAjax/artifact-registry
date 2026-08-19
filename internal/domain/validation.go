package domain

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

type ValidationReport struct {
	Valid         bool              `json:"valid"`
	Errors        []ValidationError `json:"errors,omitempty"`
	Warnings      []string          `json:"warnings,omitempty"`
	CheckedFields int               `json:"checkedFields"`
}

func (r *ValidationReport) Add(field, reason string) {
	r.Valid = false
	r.Errors = append(r.Errors, ValidationError{Field: field, Reason: reason})
}
func (r *ValidationReport) Warn(message string) { r.Warnings = append(r.Warnings, message) }
func (r ValidationReport) Error() string {
	if r.Valid {
		return ""
	}
	parts := make([]string, len(r.Errors))
	for i, e := range r.Errors {
		parts[i] = e.Error()
	}
	return strings.Join(parts, "; ")
}

type ManifestValidator struct {
	MaxLayers         int
	MaxDescriptors    int
	AllowedMediaTypes map[string]bool
	RequireConfig     bool
	RequireSize       bool
	DigestPattern     *regexp.Regexp
}

func NewManifestValidator() *ManifestValidator {
	return &ManifestValidator{MaxLayers: 128, MaxDescriptors: 1024, AllowedMediaTypes: map[string]bool{MediaOCIManifest: true, MediaOCIIndex: true, MediaOCIArtifact: true, MediaDockerManifest: true}, RequireConfig: false, RequireSize: true, DigestPattern: regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)}
}
func (v *ManifestValidator) Validate(raw []byte, mediaType string) ValidationReport {
	report := ValidationReport{Valid: true}
	if len(raw) == 0 {
		report.Add("manifest", "empty payload")
		return report
	}
	if !json.Valid(raw) {
		report.Add("manifest", "invalid JSON")
		return report
	}
	if !v.AllowedMediaTypes[NormalizeMediaType(mediaType)] {
		report.Add("mediaType", "unsupported")
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		report.Add("manifest", err.Error())
		return report
	}
	report.CheckedFields = len(body)
	if v.RequireConfig {
		if _, ok := body["config"]; !ok {
			report.Add("config", "required")
		}
	}
	for _, field := range []string{"layers", "manifests"} {
		if list, ok := body[field].([]any); ok {
			if len(list) > v.MaxLayers {
				report.Add(field, fmt.Sprintf("more than %d descriptors", v.MaxLayers))
			}
			for i, item := range list {
				report.CheckedFields++
				m, ok := item.(map[string]any)
				if !ok {
					report.Add(fmt.Sprintf("%s[%d]", field, i), "must be object")
					continue
				}
				v.validateDescriptor(&report, fmt.Sprintf("%s[%d]", field, i), m)
			}
		}
	}
	return report
}
func (v *ManifestValidator) validateDescriptor(r *ValidationReport, path string, m map[string]any) {
	d, ok := m["digest"].(string)
	if !ok || !v.DigestPattern.MatchString(d) {
		r.Add(path+".digest", "must be a sha256 digest")
	}
	if v.RequireSize {
		switch x := m["size"].(type) {
		case float64:
			if x < 0 {
				r.Add(path+".size", "must not be negative")
			}
		default:
			r.Add(path+".size", "required number")
		}
	}
	if mt, ok := m["mediaType"].(string); !ok || strings.TrimSpace(mt) == "" {
		r.Add(path+".mediaType", "required")
	}
}
func (v *ManifestValidator) ValidateDescriptor(d Descriptor) error {
	if err := d.Validate(); err != nil {
		return err
	}
	if v.DigestPattern != nil && !v.DigestPattern.MatchString(d.Digest.String()) {
		return ErrInvalidDigest
	}
	return nil
}

type ReferenceGraph struct {
	Edges   map[Digest][]Digest
	Reverse map[Digest][]Digest
}

func NewReferenceGraph() *ReferenceGraph {
	return &ReferenceGraph{Edges: map[Digest][]Digest{}, Reverse: map[Digest][]Digest{}}
}
func (g *ReferenceGraph) AddManifest(m *Manifest) {
	refs := m.References()
	g.Edges[m.Digest] = append([]Digest(nil), refs...)
	for _, d := range refs {
		g.Reverse[d] = append(g.Reverse[d], m.Digest)
	}
}
func (g *ReferenceGraph) Reachable(roots []Digest) map[Digest]bool {
	seen := map[Digest]bool{}
	queue := append([]Digest(nil), roots...)
	for len(queue) > 0 {
		d := queue[0]
		queue = queue[1:]
		if seen[d] {
			continue
		}
		seen[d] = true
		queue = append(queue, g.Edges[d]...)
	}
	return seen
}
func (g *ReferenceGraph) Dependents(d Digest) []Digest { return append([]Digest(nil), g.Reverse[d]...) }
func (g *ReferenceGraph) Remove(d Digest) {
	for _, ref := range g.Edges[d] {
		list := g.Reverse[ref]
		out := list[:0]
		for _, x := range list {
			if x != d {
				out = append(out, x)
			}
		}
		g.Reverse[ref] = out
	}
	delete(g.Edges, d)
}
func (g *ReferenceGraph) Validate() error {
	for source, refs := range g.Edges {
		if source.Validate() != nil {
			return ErrInvalidDigest
		}
		for _, ref := range refs {
			if ref.Validate() != nil {
				return ErrInvalidDigest
			}
		}
	}
	return nil
}
