package application

import (
	"artifact-registry/internal/domain"
	"context"
	"sort"
	"strings"
	"time"
)

type RetentionService struct {
	Policies  PolicyStore
	Manifests ManifestStore
	Tags      TagStore
	Clock     Clock
}
type RetentionDecision struct {
	Digest      domain.Digest `json:"digest"`
	Repository  string        `json:"repository"`
	Keep        bool          `json:"keep"`
	Reasons     []string      `json:"reasons"`
	EvaluatedAt time.Time     `json:"evaluatedAt"`
}

func (s *RetentionService) Evaluate(ctx context.Context, repo string) ([]RetentionDecision, error) {
	r, err := domain.ParseRepositoryName(repo)
	if err != nil {
		return nil, err
	}
	policies, err := s.Policies.ListPolicies(ctx)
	if err != nil {
		return nil, err
	}
	digests, err := s.Manifests.ListManifestDigests(ctx, r)
	if err != nil {
		return nil, err
	}
	tags, err := s.listTags(ctx, r)
	if err != nil {
		return nil, err
	}
	tagged := map[domain.Digest]bool{}
	for _, t := range tags {
		tagged[t.Digest] = true
	}
	out := make([]RetentionDecision, 0, len(digests))
	now := s.now()
	for i, d := range digests {
		m, e := s.Manifests.GetManifest(ctx, r, d.String())
		if e != nil {
			continue
		}
		decision := RetentionDecision{Digest: d, Repository: repo, EvaluatedAt: now}
		if tagged[d] {
			decision.Keep = true
			decision.Reasons = append(decision.Reasons, "tagged")
		}
		for _, p := range policies {
			if !p.Applies(r) {
				continue
			}
			if p.KeepLast > i {
				decision.Keep = true
				decision.Reasons = append(decision.Reasons, "keep-last")
			}
			if p.MaxAge > 0 && m.CreatedAt.After(now.Add(-p.MaxAge)) {
				decision.Keep = true
				decision.Reasons = append(decision.Reasons, "max-age")
			}
		}
		if len(decision.Reasons) == 0 {
			decision.Reasons = []string{"unreferenced"}
		}
		out = append(out, decision)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Digest < out[j].Digest })
	return out, nil
}
func (s *RetentionService) listTags(ctx context.Context, r domain.RepositoryName) ([]*domain.Tag, error) {
	out := []*domain.Tag{}
	last := ""
	for {
		p, e := s.Tags.ListTags(ctx, r, domain.PageRequest{Limit: 1000, Last: last})
		if e != nil {
			return nil, e
		}
		out = append(out, p.Items...)
		if p.Next == "" {
			return out, nil
		}
		last = p.Next
	}
}
func (s *RetentionService) Compile(prefix string, keep int, age time.Duration) domain.RetentionPolicy {
	return domain.RetentionPolicy{ID: "compiled-" + strings.ReplaceAll(prefix, "/", "-"), RepositoryPrefix: prefix, KeepLast: keep, MaxAge: age, Enabled: true, CreatedAt: s.now()}
}
func (s *RetentionService) now() time.Time {
	if s.Clock != nil {
		return s.Clock.Now().UTC()
	}
	return time.Now().UTC()
}
