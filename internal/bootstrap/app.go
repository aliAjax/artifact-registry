package bootstrap

import (
	"artifact-registry/internal/application"
	"artifact-registry/internal/memory"
	transport "artifact-registry/internal/distribution/transport/http"
	"artifact-registry/internal/platform/config"
	"context"
	"net/http"
	"time"
)

type App struct {
	Config  config.Config
	Handler http.Handler
	catalog *memory.Catalog
}

func NewRegistryApp() (*App, error) {
	cfg := config.Load()
	object, err := memory.NewLocalObjectStore(cfg.DataDir)
	if err != nil {
		return nil, err
	}
	catalog := memory.NewCatalog(object)
	clock := application.SystemClock{}
	registry := &application.RegistryService{Repositories: catalog, Blobs: catalog, Manifests: catalog, Tags: catalog, Uploads: catalog, Quotas: catalog, Audits: catalog, ObjectStore: object, Events: application.NoopPublisher{}, Clock: clock, UploadTTL: cfg.UploadTTL, MaxBlobSize: cfg.MaxBlobSize, DefaultQuota: cfg.DefaultQuota}
	life := &application.LifecycleService{Manifests: catalog, Tags: catalog, Repositories: catalog, Blobs: catalog, Policies: catalog, Scans: catalog, Replications: catalog, Scanner: application.DevelopmentScanner{Clock: clock}, Replicator: application.StaticReplicator{}, Clock: clock, Audits: catalog}
	return &App{Config: cfg, Handler: transport.NewRouter(registry, life), catalog: catalog}, nil
}
func (a *App) Close() {}

type WorkerApp struct{ stop chan struct{} }

func NewWorkerApp() (*WorkerApp, error) { return &WorkerApp{stop: make(chan struct{})}, nil }
func (w *WorkerApp) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stop:
			return
		case <-ticker.C:
		}
	}
}
func (w *WorkerApp) Stop(_ time.Duration) {
	select {
	case <-w.stop:
	default:
		close(w.stop)
	}
}
