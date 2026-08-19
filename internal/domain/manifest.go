package domain

import (
	"encoding/json"
	"sort"
	"strings"
	"time"
)

type Descriptor struct {
	MediaType    string            `json:"mediaType"`
	Digest       Digest            `json:"digest"`
	Size         int64             `json:"size"`
	URLs         []string          `json:"urls,omitempty"`
	Annotations  map[string]string `json:"annotations,omitempty"`
	ArtifactType string            `json:"artifactType,omitempty"`
	Platform     *Platform         `json:"platform,omitempty"`
}
type Platform struct {
	Architecture string `json:"architecture"`
	OS           string `json:"os"`
	Variant      string `json:"variant,omitempty"`
}

func (d Descriptor) Validate() error {
	if d.Digest.Validate() != nil {
		return ErrInvalidDigest
	}
	if d.Size < 0 {
		return ValidationError{"descriptor.size", "negative"}
	}
	if strings.TrimSpace(d.MediaType) == "" {
		return ErrInvalidMediaType
	}
	return nil
}
func (d Descriptor) CanonicalKey() string { return d.Digest.String() + ":" + d.MediaType }

type Lifecycle string

const (
	LifecycleQuarantined Lifecycle = "quarantined"
	LifecycleScanning    Lifecycle = "scanning"
	LifecycleAvailable   Lifecycle = "available"
	LifecycleBlocked     Lifecycle = "blocked"
	LifecycleRetained    Lifecycle = "retained"
	LifecycleExpired     Lifecycle = "expired"
)

func (l Lifecycle) AllowsPull() bool { return l == LifecycleAvailable || l == LifecycleRetained }
func (l Lifecycle) CanTransition(to Lifecycle) bool {
	if l == to {
		return true
	}
	allowed := map[Lifecycle]map[Lifecycle]bool{LifecycleQuarantined: {LifecycleScanning: true, LifecycleBlocked: true, LifecycleExpired: true}, LifecycleScanning: {LifecycleAvailable: true, LifecycleBlocked: true, LifecycleExpired: true}, LifecycleAvailable: {LifecycleBlocked: true, LifecycleRetained: true, LifecycleExpired: true}, LifecycleBlocked: {LifecycleScanning: true, LifecycleExpired: true}, LifecycleRetained: {LifecycleAvailable: true, LifecycleExpired: true}, LifecycleExpired: {}}
	return allowed[l][to]
}

type Manifest struct {
	Digest       Digest            `json:"digest"`
	MediaType    string            `json:"mediaType"`
	Size         int64             `json:"size"`
	Config       *Descriptor       `json:"config,omitempty"`
	Layers       []Descriptor      `json:"layers,omitempty"`
	Manifests    []Descriptor      `json:"manifests,omitempty"`
	Subject      *Descriptor       `json:"subject,omitempty"`
	ArtifactType string            `json:"artifactType,omitempty"`
	Annotations  map[string]string `json:"annotations,omitempty"`
	State        Lifecycle         `json:"state"`
	CreatedAt    time.Time         `json:"createdAt"`
	UpdatedAt    time.Time         `json:"updatedAt"`
	Version      int64             `json:"version"`
	Raw          []byte            `json:"-"`
}

func NewManifest(raw []byte, mediaType string, now time.Time) (*Manifest, error) {
	if err := ValidateManifestMediaType(mediaType); err != nil {
		return nil, err
	}
	if !json.Valid(raw) {
		return nil, ValidationError{"manifest", "invalid JSON"}
	}
	m := &Manifest{Digest: DigestBytes(raw), MediaType: NormalizeMediaType(mediaType), Size: int64(len(raw)), State: LifecycleQuarantined, CreatedAt: now, UpdatedAt: now, Version: 1, Raw: append([]byte(nil), raw...)}
	var wire struct {
		Config       *Descriptor       `json:"config"`
		Layers       []Descriptor      `json:"layers"`
		Manifests    []Descriptor      `json:"manifests"`
		Subject      *Descriptor       `json:"subject"`
		ArtifactType string            `json:"artifactType"`
		Annotations  map[string]string `json:"annotations"`
	}
	if err := json.Unmarshal(raw, &wire); err != nil {
		return nil, err
	}
	m.Config, m.Layers, m.Manifests, m.Subject, m.ArtifactType, m.Annotations = wire.Config, wire.Layers, wire.Manifests, wire.Subject, wire.ArtifactType, wire.Annotations
	if err := m.ValidateReferences(); err != nil {
		return nil, err
	}
	return m, nil
}
func (m *Manifest) ValidateReferences() error {
	if m.Config != nil {
		if err := m.Config.Validate(); err != nil {
			return err
		}
	}
	for _, d := range append(append([]Descriptor{}, m.Layers...), m.Manifests...) {
		if err := d.Validate(); err != nil {
			return err
		}
	}
	if m.Subject != nil {
		return m.Subject.Validate()
	}
	return nil
}
func (m *Manifest) References() []Digest {
	refs := make([]Digest, 0, len(m.Layers)+len(m.Manifests)+2)
	if m.Config != nil {
		refs = append(refs, m.Config.Digest)
	}
	for _, d := range m.Layers {
		refs = append(refs, d.Digest)
	}
	for _, d := range m.Manifests {
		refs = append(refs, d.Digest)
	}
	if m.Subject != nil {
		refs = append(refs, m.Subject.Digest)
	}
	return refs
}
func (m *Manifest) Transition(to Lifecycle, now time.Time) error {
	if !m.State.CanTransition(to) {
		return ErrInvalidStateTransition
	}
	m.State = to
	m.UpdatedAt = now
	m.Version++
	return nil
}
func (m *Manifest) ETag() string {
	return `"` + m.Digest.String() + `-` + string(rune(m.Version)) + `"`
}
func (m *Manifest) IsIndex() bool    { return m.MediaType == MediaOCIIndex }
func (m *Manifest) IsArtifact() bool { return m.MediaType == MediaOCIArtifact }
func (m *Manifest) Descriptor() Descriptor {
	return Descriptor{MediaType: m.MediaType, Digest: m.Digest, Size: m.Size, ArtifactType: m.ArtifactType, Annotations: m.Annotations}
}
func (m *Manifest) SortedAnnotations() []string {
	out := make([]string, 0, len(m.Annotations))
	for k := range m.Annotations {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
