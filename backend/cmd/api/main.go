package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tafakorpardazan/huginn/internal/analytics"
	"github.com/tafakorpardazan/huginn/internal/assignments"
	"github.com/tafakorpardazan/huginn/internal/audit"
	"github.com/tafakorpardazan/huginn/internal/auth"
	"github.com/tafakorpardazan/huginn/internal/cache"
	"github.com/tafakorpardazan/huginn/internal/callscenarios"
	"github.com/tafakorpardazan/huginn/internal/campaigns"
	"github.com/tafakorpardazan/huginn/internal/config"
	"github.com/tafakorpardazan/huginn/internal/contacts"
	"github.com/tafakorpardazan/huginn/internal/dashboard"
	"github.com/tafakorpardazan/huginn/internal/database"
	"github.com/tafakorpardazan/huginn/internal/documents"
	"github.com/tafakorpardazan/huginn/internal/duplicates"
	"github.com/tafakorpardazan/huginn/internal/evidence"
	"github.com/tafakorpardazan/huginn/internal/followups"
	"github.com/tafakorpardazan/huginn/internal/httpapi"
	"github.com/tafakorpardazan/huginn/internal/imports"
	"github.com/tafakorpardazan/huginn/internal/interactions"
	"github.com/tafakorpardazan/huginn/internal/leads"
	"github.com/tafakorpardazan/huginn/internal/leadversions"
	"github.com/tafakorpardazan/huginn/internal/llm"
	"github.com/tafakorpardazan/huginn/internal/logging"
	"github.com/tafakorpardazan/huginn/internal/messaging"
	"github.com/tafakorpardazan/huginn/internal/muninn"
	"github.com/tafakorpardazan/huginn/internal/nextactions"
	"github.com/tafakorpardazan/huginn/internal/notes"
	"github.com/tafakorpardazan/huginn/internal/projects"
	"github.com/tafakorpardazan/huginn/internal/qualification"
	"github.com/tafakorpardazan/huginn/internal/queue"
	"github.com/tafakorpardazan/huginn/internal/reports"
	"github.com/tafakorpardazan/huginn/internal/scoring"
	"github.com/tafakorpardazan/huginn/internal/storage"
	"github.com/tafakorpardazan/huginn/internal/timeline"
	"github.com/tafakorpardazan/huginn/internal/users"
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

	redisClient, err := cache.New(ctx, cfg.Redis)
	if err != nil {
		logger.Warn("redis unavailable", "error", err)
		redisClient = nil
	}
	if redisClient != nil {
		defer redisClient.Close()
	}

	store, err := storage.New(ctx, cfg.MinIO)
	if err != nil {
		logger.Warn("minio unavailable", "error", err)
		store = nil
	}

	userRepo := users.NewRepository(db)
	authManager := auth.NewManager(cfg.Auth)
	authMW := auth.NewMiddleware(authManager, userRepo)
	authSvc := auth.NewAuthService(authManager, userRepo)
	userSvc := users.NewService(userRepo)

	llmAuditor := llm.NewPostgresAuditor(db)
	auditLogger := audit.NewLogger(audit.NewPostgresStore(db))
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
	llmRunner := llm.NewRunner(provider, llmAuditor, provCfg, llm.WithRunnerLogger(logger))

	leadRepo := leads.NewRepository(db)
	if err := leadRepo.EnsureSchema(ctx); err != nil {
		logger.Warn("leads schema ensure failed", "error", err)
	}
	contactRepo := contacts.NewRepository(db)
	versionsRepo := leadversions.NewRepository(db)
	interRepo := interactions.NewRepository(db)
	reportRepo := reports.NewRepository(db)
	followRepo := followups.NewRepository(db)
	importRepo := imports.NewRepository(db)
	if err := importRepo.EnsureSchema(ctx); err != nil {
		logger.Warn("imports schema ensure failed", "error", err)
	}
	scoreRepo := scoring.NewRepository(db)
	noteRepo := notes.NewRepository(db)
	timelineRepo := timeline.NewRepository(db)
	dupsRepo := duplicates.NewRepository(db)
	qualRepo := qualification.NewRepository(db)
	assignRepo := assignments.NewRepository(db)

	leadSvc := leads.NewService(leadRepo, leads.Deps{Contacts: contactRepo, Versions: versionsRepo})
	projectRepo := projects.NewRepository(db)
	if err := projectRepo.EnsureSchema(ctx); err != nil {
		logger.Warn("projects schema ensure failed", "error", err)
	}
	projectSvc := projects.NewService(projectRepo)
	campaignRepo := campaigns.NewRepository(db)
	if err := campaignRepo.EnsureSchema(ctx); err != nil {
		logger.Warn("campaigns schema ensure failed", "error", err)
	}
	campaignSvc := campaigns.NewService(campaignRepo)
	reportSvc := reports.NewService(reportRepo, interRepo)
	followSvc := followups.NewService(followRepo)
	qualSvc := qualification.NewService(qualRepo, leadSvc, scoreRepo, qualification.Rules{
		ScoreThreshold: cfg.Qualification.ScoreThreshold,
	})
	dupDetector := duplicates.NewDetector(leadRepo, dupsRepo, db)

	exportRepo := muninn.NewRepository(db)
	assembler := muninn.NewAssembler(db, leadRepo, contactRepo, noteRepo, interRepo, reportRepo, importRepo, followRepo, leadSvc)
	exportSvc := muninn.NewService(exportRepo, muninn.NewHTTPExporter(cfg.Muninn), assembler)

	q, err := queue.New(ctx, cfg.RabbitMQ)
	if err != nil {
		logger.Warn("rabbitmq unavailable", "error", err)
	}
	defer func() {
		if q != nil {
			q.Close()
		}
	}()

	importSvc := imports.NewService(importRepo, store, q, cfg.Storage)
	extractor := documents.NewExtractor(cfg.Tika, store)
	extractionSvc := imports.NewExtractionService(llmRunner, leadRepo, contactRepo,
		evidence.NewRepository(db), dupDetector, scoreRepo, importRepo)
	docSvc := documents.NewService(extractor, importSvc, extractionSvc)

	senderProfile := &messaging.SenderProfile{
		CompanyName: cfg.Messaging.CompanyName,
		Description: cfg.Messaging.CompanyDescription,
		SenderName:  cfg.Messaging.SenderName,
		SenderTitle: cfg.Messaging.SenderTitle,
	}
	promptStore := messaging.NewPostgresPromptSettings(db)
	gen := messaging.NewGenerator(llmRunner, senderProfile, promptStore)
	scenarioGen := messaging.NewCallScenarioGenerator(llmRunner, senderProfile, promptStore, messaging.WithLogger(logger), messaging.WithOperationTimeout(cfg.CallScenarioOperationTimeout))
	scenarioRepo := callscenarios.NewRepository(db)
	if err := scenarioRepo.EnsureSchema(ctx); err != nil {
		logger.Warn("call scenarios schema ensure failed", "error", err)
	}
	actionSvc := messaging.NewContactActionService(interRepo, contactRepo, leadRepo, noteRepo, reportRepo, projectRepo, gen, scenarioGen, scenarioRepo, messaging.ContactActionDeps{Campaigns: campaignRepo, Logger: logger})

	dashSvc := dashboard.NewService(leadRepo, interRepo, followRepo, reportRepo, qualRepo, exportRepo)
	nextActionsSvc := nextactions.NewService(db)

	services := httpapi.Services{
		Config: cfg, Logger: logger,
		AuthMW: authMW, AuthSvc: authSvc, Users: userSvc,
		Leads: leadSvc, Imports: importSvc, Documents: docSvc,
		Messaging: actionSvc, CallScenarios: scenarioRepo, Reports: reportSvc, ReportRepo: reportRepo, FollowUps: followSvc,
		Qualify: qualSvc, Scoring: scoreRepo,
		LeadScorer: scoring.NewLeadScorer(db, leadRepo, contactRepo, reportRepo, followRepo, scoreRepo),
		Analytics:  analytics.NewService(db),
		Duplicates: dupDetector,
		DupsRepo:   dupsRepo, Dashboard: dashSvc, Muninn: exportSvc,
		Notes: noteRepo, Timeline: timelineRepo, Audit: auditLogger,
		Assignments: assignRepo, LeadRepo: leadRepo, ContactRepo: contactRepo,
		Interactions: interRepo, LLMRunner: llmRunner, Store: store,
		DB: db, Cache: redisClient, Projects: projectSvc,
		Campaigns: campaignSvc,
		NextActions: nextActionsSvc,
	}

	router := httpapi.NewRouter(ctx, services)
	srv := &http.Server{
		Addr:              ":" + cfg.APIPort,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		// ReadTimeout bounds reading the request BODY, not generating the
		// response. The call-scenario body is a few hundred bytes, so 30s here
		// never cancels generation.
		ReadTimeout: 30 * time.Second,
		// Deliberate timeout hierarchy for LLM-backed endpoints:
		//   LLM per-attempt timeout        = LLM_TIMEOUT_SECONDS (180s)
		//   call-scenario operation budget = CALL_SCENARIO_OP_TIMEOUT_SECONDS (185s)
		//   Go WriteTimeout                = 240s  (must exceed the op budget)
		//   Android/PWA call timeout       = 250s  (outermost)
		// The outermost layer always lives longer than the inner one, so a
		// 180s LLM attempt is never killed by the server or by a client that
		// expects quick responses.
		WriteTimeout: 240 * time.Second,
	}

	// Durable outbox dispatcher: uploads enqueue processing triggers that this
	// loop publishes to RabbitMQ (retrying on failure). Worker also runs one.
	if q != nil {
		go imports.RunOutboxDispatcher(ctx, logger, importSvc, q.Publish, 3*time.Second)
	}

	go func() {
		logger.Info("huginn-api listening", "port", cfg.APIPort, "env", cfg.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server error", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down api")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}
