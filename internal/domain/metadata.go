package domain

import (
	"sort"
	"strings"
	"time"
)

type ArtifactMetadata struct {
	Labels        map[string]string `json:"labels"`
	Annotations   map[string]string `json:"annotations"`
	CreatedBy     string            `json:"createdBy"`
	Source        string            `json:"source,omitempty"`
	BuildID       string            `json:"buildId,omitempty"`
	BuildCacheKey string            `json:"buildCacheKey,omitempty"`
	CreatedAt     time.Time         `json:"createdAt"`
}

func (m ArtifactMetadata) Label(key string) string { return m.Labels[key] }
func (m *ArtifactMetadata) SetLabel(key, value string) {
	if m.Labels == nil {
		m.Labels = map[string]string{}
	}
	m.Labels[key] = value
}
func (m *ArtifactMetadata) DeleteLabel(key string) { delete(m.Labels, key) }
func (m ArtifactMetadata) Keys() []string {
	out := make([]string, 0, len(m.Labels))
	for k := range m.Labels {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
func (m ArtifactMetadata) Matches(filters map[string]string) bool {
	for k, v := range filters {
		if m.Labels[k] != v {
			return false
		}
	}
	return true
}
func (m ArtifactMetadata) Canonical() string {
	keys := m.Keys()
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+m.Labels[k])
	}
	return strings.Join(parts, ";")
}

type CacheKey struct {
	Repository      string   `json:"repository"`
	BuildDefinition string   `json:"buildDefinition"`
	Inputs          []Digest `json:"inputs"`
	Toolchain       string   `json:"toolchain"`
	Platform        string   `json:"platform"`
}

func (k CacheKey) String() string {
	parts := []string{k.Repository, k.BuildDefinition, k.Toolchain, k.Platform}
	for _, d := range k.Inputs {
		parts = append(parts, d.String())
	}
	return strings.Join(parts, "|")
}
func (k CacheKey) Validate() error {
	if strings.TrimSpace(k.BuildDefinition) == "" {
		return ValidationError{"buildDefinition", "required"}
	}
	for _, d := range k.Inputs {
		if err := d.Validate(); err != nil {
			return err
		}
	}
	return nil
}

type CacheEntry struct {
	Key        string     `json:"key"`
	Manifest   Digest     `json:"manifest"`
	CreatedAt  time.Time  `json:"createdAt"`
	LastUsedAt time.Time  `json:"lastUsedAt"`
	Hits       int64      `json:"hits"`
	Size       int64      `json:"size"`
	ExpiresAt  *time.Time `json:"expiresAt,omitempty"`
}

func (e *CacheEntry) Hit(now time.Time)         { e.Hits++; e.LastUsedAt = now }
func (e CacheEntry) Expired(now time.Time) bool { return e.ExpiresAt != nil && now.After(*e.ExpiresAt) }

type BuildResult struct {
	ID         string     `json:"id"`
	CacheKey   string     `json:"cacheKey"`
	Manifest   Digest     `json:"manifest"`
	Status     string     `json:"status"`
	StartedAt  time.Time  `json:"startedAt"`
	FinishedAt *time.Time `json:"finishedAt,omitempty"`
	LogsDigest Digest     `json:"logsDigest,omitempty"`
	Inputs     []Digest   `json:"inputs"`
	Warnings   []string   `json:"warnings,omitempty"`
}

func (b *BuildResult) Finish(status string, now time.Time) { b.Status = status; b.FinishedAt = &now }
func (b *BuildResult) AddWarning(v string) {
	if strings.TrimSpace(v) != "" {
		b.Warnings = append(b.Warnings, v)
	}
}
func (b BuildResult) Successful() bool { return b.Status == "succeeded" }

type ReferrerFilter struct {
	ArtifactType string
	Annotations  map[string]string
	Platform     *Platform
}

func (f ReferrerFilter) Match(m *Manifest) bool {
	if f.ArtifactType != "" && m.ArtifactType != f.ArtifactType {
		return false
	}
	for k, v := range f.Annotations {
		if m.Annotations[k] != v {
			return false
		}
	}
	if f.Platform != nil {
		for _, d := range append(m.Layers, m.Manifests...) {
			if d.Platform != nil && d.Platform.OS == f.Platform.OS && d.Platform.Architecture == f.Platform.Architecture {
				return true
			}
		}
		return false
	}
	return true
}

type DigestPair struct {
	Source     Digest     `json:"source"`
	Target     Digest     `json:"target"`
	Verified   bool       `json:"verified"`
	VerifiedAt *time.Time `json:"verifiedAt,omitempty"`
}

func (p *DigestPair) Verify(now time.Time) { p.Verified = true; p.VerifiedAt = &now }
func (p DigestPair) Equal() bool           { return p.Source == p.Target }
