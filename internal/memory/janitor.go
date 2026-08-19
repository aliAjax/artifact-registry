package memory

import (
	"artifact-registry/internal/domain"
	"context"
	"sort"
	"strings"
	"time"
)

type Janitor struct {
	Catalog *Catalog
	Clock   func() time.Time
}
type CleanupReport struct {
	StartedAt      time.Time `json:"startedAt"`
	FinishedAt     time.Time `json:"finishedAt"`
	UploadsRemoved []string  `json:"uploadsRemoved"`
	BlobsRemoved   []string  `json:"blobsRemoved"`
	Errors         []string  `json:"errors,omitempty"`
}

func (j *Janitor) Run(ctx context.Context) CleanupReport {
	start := j.now()
	report := CleanupReport{StartedAt: start}
	uploads, err := j.Catalog.ListExpiredUploads(ctx, start)
	if err != nil {
		report.Errors = append(report.Errors, err.Error())
	} else {
		for _, u := range uploads {
			if err := j.Catalog.DeleteUpload(ctx, u.ID); err != nil {
				report.Errors = append(report.Errors, u.ID+": "+err.Error())
			} else {
				report.UploadsRemoved = append(report.UploadsRemoved, u.ID)
			}
		}
	}
	report.FinishedAt = j.now()
	sort.Strings(report.UploadsRemoved)
	return report
}
func (j *Janitor) DryRun(ctx context.Context) CleanupReport {
	now := j.now()
	report := CleanupReport{StartedAt: now, FinishedAt: now}
	items, err := j.Catalog.ListExpiredUploads(ctx, now)
	if err != nil {
		report.Errors = append(report.Errors, err.Error())
		return report
	}
	for _, u := range items {
		report.UploadsRemoved = append(report.UploadsRemoved, u.ID)
	}
	sort.Strings(report.UploadsRemoved)
	return report
}
func (j *Janitor) Estimate(ctx context.Context) (int64, error) {
	blobs, err := j.Catalog.ListBlobs(ctx)
	if err != nil {
		return 0, err
	}
	var total int64
	for _, b := range blobs {
		total += b.Size
	}
	return total, nil
}
func (j *Janitor) FindOrphans(ctx context.Context) []domain.Digest {
	all, _ := j.Catalog.ListBlobs(ctx)
	manifests, _ := j.Catalog.ListAllManifests(ctx)
	referenced := map[domain.Digest]bool{}
	for _, items := range manifests {
		for _, m := range items {
			referenced[m.Digest] = true
			for _, d := range m.References() {
				referenced[d] = true
			}
		}
	}
	out := []domain.Digest{}
	for _, b := range all {
		if !referenced[b.Digest] {
			out = append(out, b.Digest)
		}
	}
	sort.Slice(out, func(i, k int) bool { return out[i] < out[k] })
	return out
}
func (j *Janitor) FilterRepository(names []domain.RepositoryName, prefix string) []domain.RepositoryName {
	out := make([]domain.RepositoryName, 0, len(names))
	for _, n := range names {
		if strings.HasPrefix(n.String(), prefix) {
			out = append(out, n)
		}
	}
	sort.Slice(out, func(i, k int) bool { return out[i] < out[k] })
	return out
}
func (j *Janitor) now() time.Time {
	if j.Clock != nil {
		return j.Clock().UTC()
	}
	return time.Now().UTC()
}
