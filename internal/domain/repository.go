package domain

import (
	"regexp"
	"strings"
	"time"
)

var repoSegment = regexp.MustCompile(`^[a-z0-9]+(?:[._-][a-z0-9]+)*$`)
var tagPattern = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.-]{0,127}$`)

type RepositoryName string

func ParseRepositoryName(value string) (RepositoryName, error) {
	v := strings.Trim(strings.ToLower(strings.TrimSpace(value)), "/")
	if v == "" || len(v) > 255 {
		return "", ErrInvalidRepository
	}
	for _, s := range strings.Split(v, "/") {
		if !repoSegment.MatchString(s) {
			return "", ErrInvalidRepository
		}
	}
	return RepositoryName(v), nil
}
func (r RepositoryName) String() string { return string(r) }
func (r RepositoryName) Namespace() string {
	p := strings.Split(string(r), "/")
	if len(p) < 2 {
		return "library"
	}
	return strings.Join(p[:len(p)-1], "/")
}
func ParseTag(v string) (string, error) {
	if !tagPattern.MatchString(v) {
		return "", ValidationError{"tag", "must be 1-128 valid OCI tag characters"}
	}
	return v, nil
}

type Repository struct {
	Name          RepositoryName `json:"name"`
	Tenant        string         `json:"tenant"`
	ImmutableTags bool           `json:"immutableTags"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
	Version       int64          `json:"version"`
}

func NewRepository(name RepositoryName, tenant string, immutable bool, now time.Time) (*Repository, error) {
	if strings.TrimSpace(tenant) == "" {
		return nil, ValidationError{"tenant", "required"}
	}
	return &Repository{Name: name, Tenant: tenant, ImmutableTags: immutable, CreatedAt: now, UpdatedAt: now, Version: 1}, nil
}
func (r *Repository) SetImmutableTags(value bool, now time.Time) {
	r.ImmutableTags = value
	r.UpdatedAt = now
	r.Version++
}

type PageRequest struct {
	Limit int    `json:"limit"`
	Last  string `json:"last,omitempty"`
}

func (p PageRequest) Normalized() PageRequest {
	if p.Limit <= 0 {
		p.Limit = 100
	}
	if p.Limit > 1000 {
		p.Limit = 1000
	}
	return p
}

type Page[T any] struct {
	Items []T    `json:"items"`
	Next  string `json:"next,omitempty"`
}
