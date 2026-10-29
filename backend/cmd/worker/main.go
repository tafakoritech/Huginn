package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/tafakorpardazan/huginn/internal/audit"
	"github.com/tafakorpardazan/huginn/internal/config"
	"github.com/tafakorpardazan/huginn/internal/contacts"
	"github.com/tafakorpardazan/huginn/internal/database"
	"github.com/tafakorpardazan/huginn/internal/documents"
	"github.com/tafakorpardazan/huginn/internal/duplicates"
	"github.com/tafakorpardazan/huginn/internal/evidence"
	"github.com/tafakorpardazan/huginn/internal/followups"
	"github.com/tafakorpardazan/huginn/internal/imports"
	"github.com/tafakorpardazan/huginn/internal/interactions"
	"github.com/tafakorpardazan/huginn/internal/leads"
	"github.com/tafakorpardazan/huginn/internal/llm"
	"github.com/tafakorpardazan/huginn/internal/logging"
	"github.com/tafakorpardazan/huginn/internal/muninn"
	"github.com/tafakorpardazan/huginn/internal/notes"
	"github.com/tafakorpardazan/huginn/internal/queue"
	"github.com/tafakorpardazan/huginn/internal/reports"
	"github.com/tafakorpardazan/huginn/internal/scoring"
	"github.com/tafakorpardazan/huginn/internal/storage"
	"github.com/tafakorpardazan/huginn/internal/worker"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	logger := logging.New(cfg.Env)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := database.New(ctx, cfg.Postgres)
	if err != nil {
		logger.Error("postgres connection failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	store, err := storage.New(ctx, cfg.MinIO)
	if err != nil {
		logger.Warn("minio unavailable", "error", err)
		store = nil
	}

	provCfg := llm.ProviderConfig{
		BaseURL: cfg.LLM.BaseURL, APIKey: cfg.LLM.APIKey, Model: cfg.LLM.Model,
		Timeout: cfg.LLM.Timeout, MaxRetries: cfg.LLM.MaxRetries, RetryBaseDelay: cfg.LLM.RetryBaseDelay,
		StructuredMode: cfg.LLM.StructuredMode,
		SOCKS5Proxy:    cfg.LLM.SOCKS5Proxy,
	}
	provider, err := llm.NewOpenAICompatibleProvider(provCfg, llm.WithLogger(logger))
	if err != nil {
		logger.Error("llm provider configuration error", "error", err)
		os.Exit(1)
	}
	llm.LogTransportConfig(logger, provCfg)
	llmRunner := llm.NewRunner(provider, llm.NewPostgresAuditor(db), provCfg, llm.WithRunnerLogger(logger))

	leadRepo := leads.NewRepository(db)
	if err := leadRepo.EnsureSchema(ctx); err != nil {
		logger.Warn("leads schema ensure failed", "error", err)
	}
	contactRepo := contacts.NewRepository(db)
	importRepo := imports.NewRepository(db)
	if err := importRepo.EnsureSchema(ctx); err != nil {
		logger.Warn("imports schema ensure failed", "error", err)
	}
	scoreRepo := scoring.NewRepository(db)
	dupsRepo := duplicates.NewRepository(db)
	interRepo := interactions.NewRepository(db)
	reportRepo := reports.NewRepository(db)
	followRepo := followups.NewRepository(db)
	noteRepo := notes.NewRepository(db)
	leadSvc := leads.NewService(leadRepo)

	q, err := queue.New(ctx, cfg.RabbitMQ)
	if err != nil {
		logger.Error("rabbitmq connection failed", "error", err)
		os.Exit(1)
	}
	q.SetLogger(logger)
	defer q.Close()

	importSvc := imports.NewService(importRepo, store, q, cfg.Storage)
	extractor := documents.NewExtractor(cfg.Tika, store)
	extractionSvc := imports.NewExtractionService(llmRunner, leadRepo, contactRepo,
		evidence.NewRepository(db), duplicates.NewDetector(leadRepo, dupsRepo, db), scoreRepo, importRepo)
	docSvc := documents.NewService(extractor, importSvc, extractionSvc)
	extractionSvc.SetLogger(logger)
	docSvc.SetLogger(logger)

	assembler := muninn.NewAssembler(db, leadRepo, contactRepo, noteRepo, interRepo, reportRepo, importRepo, followRepo, leadSvc)
	exportRepo := muninn.NewRepository(db)
	exportSvc := muninn.NewService(exportRepo, muninn.NewHTTPExporter(cfg.Muninn), assembler)

	svc := worker.NewService(logger, docSvc, importSvc, exportSvc, q)

	auditLogger := audit.NewLogger(audit.NewPostgresStore(db))
	_ = auditLogger

	var wg sync.WaitGroup
	for jobType, handler := range svc.Handlers() {
		wg.Add(1)
		go func(jt string, h queue.Handler) {
			defer wg.Done()
			if err := q.Consume(ctx, jt, h); err != nil {
				logger.Error("consume failed", "job", jt, "error", err)
			}
		}(jobType, handler)
	}

	logger.Info("huginn-worker started", "worker", cfg.WorkerName, "env", cfg.Env)

	// Durable outbox dispatcher: publishes pending import processing triggers
	// to RabbitMQ, retrying failures so uploads always reach the pipeline.
	go imports.RunOutboxDispatcher(ctx, logger, importSvc, q.Publish, 3*time.Second)

	// NOTE: overdue follow-ups are now computed on read (status='PENDING' AND
	// due_at < now()), so no periodic status-flip job is needed here.

	<-ctx.Done()
	logger.Info("shutting down worker")
	wg.Wait()
}
