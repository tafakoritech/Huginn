package main

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/tafakorpardazan/huginn/internal/config"
	"github.com/tafakorpardazan/huginn/internal/contacts"
	"github.com/tafakorpardazan/huginn/internal/database"
	"github.com/tafakorpardazan/huginn/internal/evidence"
	"github.com/tafakorpardazan/huginn/internal/imports"
	"github.com/tafakorpardazan/huginn/internal/leads"
	"github.com/tafakorpardazan/huginn/internal/llm"
	"github.com/tafakorpardazan/huginn/internal/logging"
	"github.com/tafakorpardazan/huginn/internal/seed"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		return
	}
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	ctx := context.Background()
	logger := logging.New(cfg.Env)

	// Migration files load from disk. In production builds these can be embedded.
	files, err := database.LoadMigrationFiles(os.Getenv("MIGRATIONS_DIR"))
	if err != nil || len(files) == 0 {
		logger.Error("failed to load migrations", "error", err, "dir", os.Getenv("MIGRATIONS_DIR"))
		os.Exit(1)
	}

	m, err := database.NewMigrator(ctx, cfg.Postgres.PlainDSN(), files)
	if err != nil {
		logger.Error("migrator connect failed", "error", err)
		os.Exit(1)
	}
	defer m.Close()

	switch os.Args[1] {
	case "up":
		if err := m.Up(ctx); err != nil {
			logger.Error("migrate up failed", "error", err)
			os.Exit(1)
		}
	case "down":
		if err := m.Down(ctx); err != nil {
			logger.Error("migrate down failed", "error", err)
			os.Exit(1)
		}
	case "status":
		if err := m.Status(ctx); err != nil {
			logger.Error("migrate status failed", "error", err)
			os.Exit(1)
		}
	case "seed":
		db, err := database.New(ctx, cfg.Postgres)
		if err != nil {
			logger.Error("db connect failed", "error", err)
			os.Exit(1)
		}
		defer db.Close()
		if err := seed.Run(ctx, db, cfg, logger); err != nil {
			logger.Error("seed failed", "error", err)
			os.Exit(1)
		}
	case "backfill-contacts":
		// LLM-driven recovery: re-extract contact data via the LLM for leads
		// created without contact info. Only empty fields are filled from the
		// LLM output; never overwrites manual corrections; never duplicates.
		limit := 500
		if len(os.Args) > 2 {
			if n, err := strconv.Atoi(os.Args[2]); err == nil && n > 0 {
				limit = n
			}
		}
		db, err := database.New(ctx, cfg.Postgres)
		if err != nil {
			logger.Error("db connect failed", "error", err)
			os.Exit(1)
		}
		defer db.Close()

		provCfg := llm.ProviderConfig{
			BaseURL: cfg.LLM.BaseURL, APIKey: cfg.LLM.APIKey, Model: cfg.LLM.Model,
			Timeout: cfg.LLM.Timeout, MaxRetries: cfg.LLM.MaxRetries, RetryBaseDelay: cfg.LLM.RetryBaseDelay,
			SOCKS5Proxy: cfg.LLM.SOCKS5Proxy,
		}
		provider, err := llm.NewOpenAICompatibleProvider(provCfg, llm.WithLogger(logger))
		if err != nil {
			logger.Error("llm provider configuration error", "error", err)
			os.Exit(1)
		}
		llm.LogTransportConfig(logger, provCfg)
		runner := llm.NewRunner(provider, llm.NewPostgresAuditor(db), provCfg)

		importRepo := imports.NewRepository(db)
		extractSvc := imports.NewExtractionService(runner, leads.NewRepository(db), contacts.NewRepository(db),
			evidence.NewRepository(db), nil, nil, importRepo)
		extractSvc.SetLogger(logger)
		processed, updated, failed, err := extractSvc.BackfillMissingContacts(ctx, limit)
		if err != nil {
			logger.Error("backfill failed", "error", err)
			os.Exit(1)
		}
		logger.Info("backfill complete", "processed", processed, "updated", updated, "failed", failed)
	default:
		usage()
	}
}

func usage() {
	fmt.Println("usage: huginn-cli <up|down|status|seed|backfill-contacts [limit]>")
	fmt.Println("  MIGRATIONS_DIR must point to the migrations directory")
}
