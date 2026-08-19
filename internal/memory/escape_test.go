package memory

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"artifact-registry/internal/domain"
)

func newTestCatalogMem(t *testing.T) *Catalog {
	t.Helper()
	o, err := NewLocalObjectStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return NewCatalog(o)
}

func TestGetDoesNotAlias(t *testing.T) {
	c := newTestCatalogMem(t)
	now := time.Now()
	_ = c.Create(context.Background(), &domain.Repository{Name: domain.RepositoryName("repo-a"), Tenant: "t", Version: 1, CreatedAt: now, UpdatedAt: now})
	got, _ := c.Get(context.Background(), domain.RepositoryName("repo-a"))
	got.Version = 99
	again, _ := c.Get(context.Background(), domain.RepositoryName("repo-a"))
	if again.Version == 99 {
		t.Fatalf("Get returned an aliased repository")
	}
}

func TestGetBlobDoesNotAlias(t *testing.T) {
	c := newTestCatalogMem(t)
	_ = c.PutBlob(context.Background(), &domain.Blob{Digest: domain.DigestBytes([]byte("x")), Size: 1, StorageKey: "k"}, bytes.NewReader([]byte("x")))
	got, _ := c.GetBlob(context.Background(), domain.DigestBytes([]byte("x")))
	got.Size = 99
	again, _ := c.GetBlob(context.Background(), domain.DigestBytes([]byte("x")))
	if again.Size == 99 {
		t.Fatalf("GetBlob returned an aliased blob")
	}
}

func TestGetTagDoesNotAlias(t *testing.T) {
	c := newTestCatalogMem(t)
	_ = c.Create(context.Background(), &domain.Repository{Name: domain.RepositoryName("repo-a"), Tenant: "t", Version: 1})
	_ = c.PutTag(context.Background(), domain.RepositoryName("repo-a"), &domain.Tag{Name: "v1", Digest: domain.DigestBytes([]byte("m")), Version: 1})
	got, _ := c.GetTag(context.Background(), domain.RepositoryName("repo-a"), "v1")
	got.Version = 99
	again, _ := c.GetTag(context.Background(), domain.RepositoryName("repo-a"), "v1")
	if again.Version == 99 {
		t.Fatalf("GetTag returned an aliased tag")
	}
}

func TestGetUploadDoesNotAlias(t *testing.T) {
	c := newTestCatalogMem(t)
	_ = c.CreateUpload(context.Background(), &domain.UploadSession{ID: "u1", Version: 1})
	got, _ := c.GetUpload(context.Background(), "u1")
	got.Version = 99
	again, _ := c.GetUpload(context.Background(), "u1")
	if again.Version == 99 {
		t.Fatalf("GetUpload returned an aliased upload")
	}
}

func TestCatalogConcurrentAccess(t *testing.T) {
	c := newTestCatalogMem(t)
	_ = c.Create(context.Background(), &domain.Repository{Name: domain.RepositoryName("repo-a"), Tenant: "t", Version: 1})
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(4)
	for g := 0; g < 4; g++ {
		go func() {
			defer wg.Done()
			<-start
			for i := 0; i < 300; i++ {
				if i%2 == 0 {
					_, _ = c.Get(context.Background(), domain.RepositoryName("repo-a"))
				} else {
					_ = c.Create(context.Background(), &domain.Repository{Name: domain.RepositoryName(fmt.Sprintf("repo-%d", i)), Tenant: "t", Version: 1})
				}
			}
		}()
	}
	close(start)
	wg.Wait()
}
