package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr     string
	DataDir      string
	UploadTTL    time.Duration
	MaxBlobSize  int64
	DefaultQuota int64
}

func Load() Config {
	return Config{HTTPAddr: env("REGISTRY_HTTP_ADDR", ":8083"), DataDir: env("REGISTRY_DATA_DIR", "./data"), UploadTTL: duration("REGISTRY_UPLOAD_TTL", 24*time.Hour), MaxBlobSize: int64Env("REGISTRY_MAX_BLOB_SIZE", 10<<30), DefaultQuota: int64Env("REGISTRY_DEFAULT_QUOTA", 10<<30)}
}
func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
func duration(key string, fallback time.Duration) time.Duration {
	if v, err := time.ParseDuration(os.Getenv(key)); err == nil && v > 0 {
		return v
	}
	return fallback
}
func int64Env(key string, fallback int64) int64 {
	if v, err := strconv.ParseInt(os.Getenv(key), 10, 64); err == nil && v > 0 {
		return v
	}
	return fallback
}
