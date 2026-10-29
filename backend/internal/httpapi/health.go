package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/tafakorpardazan/huginn/internal/cache"
	"github.com/tafakorpardazan/huginn/internal/config"
	"github.com/tafakorpardazan/huginn/internal/database"
)

type pinger interface {
	Ping(ctx context.Context) error
}

func handleHealth(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		RespondJSON(w, http.StatusOK, map[string]any{
			"status": "ok",
			"time":   time.Now().UTC(),
		})
	}
}

func handleReady(db *database.DB, cache *cache.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		checks := map[string]bool{}
		ready := true
		if db != nil {
			if err := db.Ping(ctx); err != nil {
				ready = false
			} else {
				checks["postgres"] = true
			}
		}
		if cache != nil {
			if err := cache.Ping(ctx); err != nil {
				ready = false
			} else {
				checks["redis"] = true
			}
		}
		status := "ready"
		code := http.StatusOK
		if !ready {
			status = "not_ready"
			code = http.StatusServiceUnavailable
		}
		RespondJSON(w, code, map[string]any{"status": status, "checks": checks})
	}
}

func handleMetrics(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !cfg.Observability.MetricsEnabled {
			RespondError(w, http.StatusNotFound, &APIError{Code: CodeNotFound, Message: "metrics disabled"})
			return
		}
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		fmt.Fprintf(w, "# HELP huginn_goroutines Current number of goroutines\n")
		fmt.Fprintf(w, "# TYPE huginn_goroutines gauge\n")
		fmt.Fprintf(w, "huginn_goroutines %d\n", runtime.NumGoroutine())
		fmt.Fprintf(w, "# TYPE huginn_memory_alloc_bytes gauge\n")
		fmt.Fprintf(w, "huginn_memory_alloc_bytes %d\n", m.Alloc)
		fmt.Fprintf(w, "# TYPE huginn_uptime_seconds gauge\n")
		fmt.Fprintf(w, "huginn_uptime_seconds %d\n", int64(time.Since(startTime).Seconds()))
	}
}

var startTime = time.Now()
