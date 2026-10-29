// Package integration contains tests that run against real infrastructure
// (PostgreSQL, Redis, RabbitMQ, MinIO). They are skipped unless
// HUGINN_INTEGRATION=1 is set, to keep `go test ./...` green without Docker.
package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/tafakorpardazan/huginn/internal/cache"
	"github.com/tafakorpardazan/huginn/internal/config"
	"github.com/tafakorpardazan/huginn/internal/database"
	"github.com/tafakorpardazan/huginn/internal/storage"
)

func enabled() bool {
	return os.Getenv("HUGINN_INTEGRATION") == "1"
}

func testContext() context.Context {
	return context.Background()
}

func setupDB(t *testing.T) *database.DB {
	t.Helper()
	if !enabled() {
		t.Skip("integration tests disabled; set HUGINN_INTEGRATION=1")
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	db, err := database.New(testContext(), cfg.Postgres)
	if err != nil {
		t.Fatalf("connect db: %v", err)
	}
	t.Cleanup(db.Close)
	return db
}

func setupRedis(t *testing.T) *cache.Client {
	t.Helper()
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	c, err := cache.New(testContext(), cfg.Redis)
	if err != nil {
		t.Fatalf("connect redis: %v", err)
	}
	t.Cleanup(func() { c.Close() })
	return c
}

func setupMinIO(t *testing.T) *storage.Store {
	t.Helper()
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	s, err := storage.New(testContext(), cfg.MinIO)
	if err != nil {
		t.Fatalf("connect minio: %v", err)
	}
	return s
}

var _ = time.Now
