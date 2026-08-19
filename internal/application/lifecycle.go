package application

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"artifact-registry/internal/domain"
)

type LifecycleService struct {
	Manifests    ManifestStore
	Tags         TagStore
	Repositories RepositoryStore
	Blobs        BlobStore
	Policies     PolicyStore
	Scans        ScanStore
	Replications ReplicationStore
	Scanner      Scanner
	Replicator   Replicator
	Clock        Clock
	Audits       AuditStore
}

type GCRequest struct {
	IdempotencyKey string `json:"-"`
	Confirm        string `json:"-"`
	DryRun         bool   `json:"dryRun"`
	Repository     string `json:"repository,omitempty"`
	Force          bool   `json:"force"`
}
type GCReport struct {
	ID                 string            `json:"id"`
	StartedAt          time.Time         `json:"startedAt"`
	FinishedAt         time.Time         `json:"finishedAt"`
	DryRun             bool              `json:"dryRun"`
	CandidateManifests []string          `json:"candidateManifests"`
	CandidateBlobs     []string          `json:"candidateBlobs"`
	DeletedManifests   []string          `json:"deletedManifests"`
	DeletedBlobs       []string          `json:"deletedBlobs"`
	Protected          map[string]string `json:"protected"`
}

func (s *LifecycleService) SubmitScan(ctx context.Context, repo, reference string) (domain.ScanReport, error) {
	r, err := s.Repositories.Get(ctx, mustRepo(repo))
	if err != nil {
		return domain.ScanReport{}, err
	}
	m, err := s.Manifests.GetManifest(ctx, r.Name, reference)
	if err != nil {
		return domain.ScanReport{}, err
	}
	if s.Scanner == nil {
		return domain.ScanReport{}, domain.ErrNotImplemented
	}
	report, err := s.Scanner.Submit(ctx, r.Name, m)
	if err != nil {
		return domain.ScanReport{}, err
	}
	if err := s.Scans.CreateScan(ctx, report); err != nil {
		return domain.ScanReport{}, err
	}
	now := s.now()
	next := domain.LifecycleScanning
	if report.Status == domain.ScanPassed {
		next = domain.LifecycleAvailable
	}
	if report.Status == domain.ScanFailed {
		next = domain.LifecycleBlocked
	}
	if err := m.Transition(next, now); err != nil {
		return domain.ScanReport{}, err
	}
	if err := s.Manifests.PutManifest(ctx, r.Name, m); err != nil {
		return domain.ScanReport{}, err
	}
	return report, nil
}

func (s *LifecycleService) ListScans(ctx context.Context, repo string, p domain.PageRequest) (domain.Page[domain.ScanReport], error) {
	var r domain.RepositoryName
	if repo != "" {
		r = mustRepo(repo)
	}
	return s.Scans.ListScans(ctx, r, p)
}

func (s *LifecycleService) CreateReplication(ctx context.Context, source, target, reference, id string) (domain.ReplicationTask, error) {
	r, err := domain.ParseRepositoryName(source)
	if err != nil {
		return domain.ReplicationTask{}, err
	}
	if _, err := s.Repositories.Get(ctx, r); err != nil {
		return domain.ReplicationTask{}, err
	}
	if strings.TrimSpace(target) == "" {
		return domain.ReplicationTask{}, domain.ValidationError{Field: "target", Reason: "required"}
	}
	if strings.TrimSpace(id) == "" {
		id = fmt.Sprintf("rep-%d", s.now().UnixNano())
	}
	now := s.now()
	task := domain.ReplicationTask{ID: id, Source: r, Target: target, Reference: reference, State: domain.ReplicationQueued, CreatedAt: now, UpdatedAt: now, Version: 1}
	if err := s.Replications.CreateReplication(ctx, task); err != nil {
		return domain.ReplicationTask{}, err
	}
	return task, nil
}
func (s *LifecycleService) RunReplication(ctx context.Context, id string) (domain.ReplicationTask, error) {
	task, err := s.Replications.GetReplication(ctx, id)
	if err != nil {
		return domain.ReplicationTask{}, err
	}
	old := task.Version
	now := s.now()
	if err := task.Claim(now.Add(5*time.Minute), now); err != nil {
		return domain.ReplicationTask{}, err
	}
	if err := s.Replications.UpdateReplication(ctx, task, old); err != nil {
		return domain.ReplicationTask{}, err
	}
	if s.Replicator == nil {
		task.Fail(domain.ErrNotImplemented.Error(), s.now())
	} else {
		n, copyErr := s.Replicator.Copy(ctx, task)
		task.BytesCopied = n
		if copyErr != nil {
			task.Fail(copyErr.Error(), s.now())
		} else {
			task.Succeed(s.now())
		}
	}
	if err := s.Replications.UpdateReplication(ctx, task, task.Version-1); err != nil {
		return domain.ReplicationTask{}, err
	}
	return task, nil
}
func (s *LifecycleService) ListReplications(ctx context.Context, p domain.PageRequest) (domain.Page[domain.ReplicationTask], error) {
	return s.Replications.ListReplications(ctx, p)
}

func (s *LifecycleService) GarbageCollect(ctx context.Context, request GCRequest) (GCReport, error) {
	if request.IdempotencyKey == "" {
		return GCReport{}, domain.ValidationError{Field: "Idempotency-Key", Reason: "required"}
	}
	if !request.DryRun && request.Confirm != "delete-unreferenced-artifacts" {
		return GCReport{}, domain.ErrPreconditionFailed
	}
	report := GCReport{ID: "gc-" + request.IdempotencyKey, StartedAt: s.now(), DryRun: request.DryRun, Protected: map[string]string{}}
	all, err := s.Manifests.ListAllManifests(ctx)
	if err != nil {
		return report, err
	}
	policies, err := s.Policies.ListPolicies(ctx)
	if err != nil {
		return report, err
	}
	reachable := map[domain.Digest]string{}
	candidates := map[domain.RepositoryName][]*domain.Manifest{}
	for repo, manifests := range all {
		if request.Repository != "" && repo.String() != request.Repository {
			continue
		}
		tags, err := s.listAllTags(ctx, repo)
		if err != nil {
			return report, err
		}
		tagged := map[domain.Digest]bool{}
		for _, t := range tags {
			tagged[t.Digest] = true
			reachable[t.Digest] = "tag:" + t.Name
		}
		sort.Slice(manifests, func(i, j int) bool { return manifests[i].CreatedAt.After(manifests[j].CreatedAt) })
		for index, m := range manifests {
			if tagged[m.Digest] {
				continue
			}
			keep := false
			for _, p := range policies {
				if p.Applies(repo) && (p.KeepLast > index || (p.MaxAge > 0 && m.CreatedAt.After(s.now().Add(-p.MaxAge)))) {
					keep = true
				}
			}
			if keep {
				reachable[m.Digest] = "retention policy"
			} else {
				candidates[repo] = append(candidates[repo], m)
				report.CandidateManifests = append(report.CandidateManifests, repo.String()+"@"+m.Digest.String())
			}
		}
	}
	for _, manifests := range all {
		for _, m := range manifests {
			if _, removed := candidateContains(candidates, m); removed {
				continue
			}
			reachable[m.Digest] = "live manifest"
			for _, d := range m.References() {
				reachable[d] = "manifest reference"
			}
		}
	}
	for d, reason := range reachable {
		report.Protected[d.String()] = reason
	}
	blobs, err := s.Blobs.ListBlobs(ctx)
	if err != nil {
		return report, err
	}
	for _, b := range blobs {
		if _, ok := reachable[b.Digest]; !ok {
			report.CandidateBlobs = append(report.CandidateBlobs, b.Digest.String())
		}
	}
	sort.Strings(report.CandidateManifests)
	sort.Strings(report.CandidateBlobs)
	if !request.DryRun {
		for repo, manifests := range candidates {
			for _, m := range manifests {
				if err := s.Manifests.DeleteManifest(ctx, repo, m.Digest); err != nil {
					return report, err
				}
				report.DeletedManifests = append(report.DeletedManifests, repo.String()+"@"+m.Digest.String())
			}
		}
		for _, dText := range report.CandidateBlobs {
			d, _ := domain.ParseDigest(dText)
			if err := s.Blobs.DeleteBlob(ctx, d); err == nil {
				report.DeletedBlobs = append(report.DeletedBlobs, dText)
			}
		}
	}
	report.FinishedAt = s.now()
	return report, nil
}
func (s *LifecycleService) listAllTags(ctx context.Context, repo domain.RepositoryName) ([]*domain.Tag, error) {
	out := []*domain.Tag{}
	last := ""
	for {
		p, err := s.Tags.ListTags(ctx, repo, domain.PageRequest{Limit: 1000, Last: last})
		if err != nil {
			return nil, err
		}
		out = append(out, p.Items...)
		if p.Next == "" {
			return out, nil
		}
		last = p.Next
	}
}
func candidateContains(c map[domain.RepositoryName][]*domain.Manifest, m *domain.Manifest) (domain.RepositoryName, bool) {
	for repo, items := range c {
		for _, x := range items {
			if x.Digest == m.Digest {
				return repo, true
			}
		}
	}
	return "", false
}
func (s *LifecycleService) now() time.Time {
	if s.Clock != nil {
		return s.Clock.Now().UTC()
	}
	return time.Now().UTC()
}
func mustRepo(v string) domain.RepositoryName {
	r, err := domain.ParseRepositoryName(v)
	if err != nil {
		return ""
	}
	return r
}
