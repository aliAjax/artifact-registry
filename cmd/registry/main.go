package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"artifact-registry/internal/bootstrap"
)

func main() {
	app, err := bootstrap.NewRegistryApp()
	if err != nil {
		slog.Error("registry initialization failed", "error", err)
		os.Exit(1)
	}
	server := &http.Server{Addr: app.Config.HTTPAddr, Handler: app.Handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 0, IdleTimeout: 60 * time.Second}
	go func() {
		slog.Info("registry listening", "address", app.Config.HTTPAddr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("registry server stopped unexpectedly", "error", err)
			os.Exit(1)
		}
	}()
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	<-signals
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		slog.Error("graceful shutdown failed", "error", err)
	}
	app.Close()
}
