# Graph Report - Huginn  (2026-10-06)

## Corpus Check
- 435 files · ~318,622 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 40 file(s) not represented in the graph (top: .xml 13, (none) 8, .css 4)

## Summary
- 5006 nodes · 17377 edges · 182 communities (124 shown, 58 thin omitted)
- Extraction: 96% EXTRACTED · 4% INFERRED · 0% AMBIGUOUS · INFERRED: 742 edges (avg confidence: 0.85)
- Token cost: 104,470 input · 0 output

## Community Hubs (Navigation)
- Import Pipeline & Muninn Store
- Dashboard Leads Pages & Auth
- Android Token Store & Tests
- Company Text Normalization
- Dashboard API Index & UI Kit
- Backend Entrypoints & Wiring
- Android Design Cards
- Dashboard App Shell & Test Setup
- Backend Leads Cache & Config Tests
- Android HuginnApi Retrofit
- LLM & Messaging Service Tests
- Android Session State
- Android Date/Time Utils
- Integration Test Harness
- Android Info Rows & Badges
- Android Mock Data
- Lead Detail Screen (Android)
- Android Repository Results
- Dashboard ApiEndpoints
- Backend Config & Cache Client
- Android External Launchers
- Next Actions Engine
- Lead Assignments Service
- HTTP Data Handlers
- Dashboard Forms & Layout
- Android ApiCaller Errors
- Android Speech Recognizer
- Teleprompter Service
- Leads Service & Repository
- Android Lead Models
- HTTP Lead Handler Helpers
- Phone Number Formatting
- Queue Worker Handler
- Android Work Queue
- Dashboard Mock API
- Android Navigation Routes
- Contacts & Duplicates Scan
- Android Buttons & Messaging Sheet
- Analytics Reports
- Dashboard API Types (Health/Timeline)
- Call Scenario Strategy Model
- Follow-ups Service
- Android Detail Models
- Android Voice Input UI
- Audit & Team Reports
- Android App UI Tests
- Qualification Repository (Android)
- Report Draft Validation
- Edit Lead ViewModel
- OpenAPI Spec
- Leads Database Queries
- Call Scenario Generation
- Campaigns Repository
- Android Home Screen
- Note Composer UI Tests
- Worker Error Classification
- Dashboard API Interface
- Dashboard Work Queue Page
- Qualification & Scoring Services
- Android LLM API
- Android Scaffold & Bottom Nav
- Fake Speech Recognizer
- JWT Auth Manager
- Android Refresh & Session Manager
- Messenger Deep Link Routing
- Android Reports ViewModel
- Env Config Loading Tests
- Dashboard HTTP Client
- Roles Repository
- Android Follow-ups ViewModel
- Auth Middleware & RBAC
- HTTP Lead Responses
- Work Queue Tab UI Tests
- Queue Delivery Tests
- Android UI Label Mappers
- Android Leads ViewModel
- Android Application & Localization
- HTTP Error Responses
- Users Repository
- Android Network Module
- Duplicates & Evidence Detection
- User Handlers & RBAC Checks
- Dashboard package.json
- Dashboard tsconfig.app
- Voice Input Controller
- HTTP Principal & Filters
- Projects & Notes Scanning
- Scenario Normalization Tests
- Deep Links & vCards
- Prompt Settings Storage
- Dashboard Dev Dependencies
- Mandatory Report UI Tests
- Android Room DAOs
- Lead Versioning & Diff
- Dashboard Charts
- Muninn HTTP Exporter
- Android Database Module
- Android Room Database
- Follow-ups ViewModel Tests
- Robolectric Test Config
- Dashboard tsconfig.node
- Raven Brand Icons
- Android Theme Settings
- Clipboard & Suggested Message
- Contact Reports Store
- Messaging Channels & Caps
- Dashboard Documents & Audit API
- Android Settings DataStore
- Android Notifications
- Score Classifier
- Fake Auth Test Doubles
- Docker Compose Services
- LLM Call Metrics
- LLM Runner Interface
- Scenario Prompt Builder
- Scoring Signals
- Language Resolver
- Android Permissions Manager
- RTL Text Utilities
- Android MainActivity
- Dashboard Docs & RBAC Roles
- Import Chunk Extraction Tests
- Scenario Strategy Synthesis
- Projects Service
- Edit Lead Screen
- Dashboard Analytics Service
- Message Prompt Builder
- Android Architecture Docs
- DB Migrator
- Backend Architecture Docs
- Android Auth Models
- Manifest Permissions Test
- Theme Tests
- LLM Retry & JSON Extraction
- Refresh Interceptor
- Token Storage
- Edit Lead Flow Tests
- Interactions Store
- Sender Perspective Checks
- Dashboard Runtime Dependencies
- Pending Report Fakes
- Lead Filter Sheet
- Lead Detail Mapping Tests
- Document Extraction (Tika/MinIO)
- OpenAI Provider
- Android Auth DI Module
- Locale Manager
- Mandatory Report Workflow Docs
- Mock LLM Server
- Dashboard Leads API
- Request Logging Interceptor
- Dashboard npm Scripts
- LLM Error Types
- Prompt Settings
- Dashboard Projects API
- Auth Interceptor
- Android Color Palette
- Android Auth Docs
- Campaign HTTP Handler
- Dashboard Campaigns API
- Personalization Similarity
- Dashboard Imports API
- Dashboard Lead Trend API
- Fake LLM Provider
- Dashboard Agent Performance API
- Dashboard Muninn Settings API
- Dashboard tsconfig Root
- Vite Config
- Fake LLM Runner (Integration)
- Slow LLM Runner (Test)
- Muninn Export Errors
- Fake HTTP Error (Queue)
- Graphify Project Rules
- No-Content Document Error
- Docker Bridge Network
- Backend Go Module
- Go Module Path

## God Nodes (most connected - your core abstractions)
1. `useI18n()` - 122 edges
2. `ApiEndpoints` - 112 edges
3. `huginnColors()` - 103 edges
4. `RespondOK()` - 96 edges
5. `MockHuginnApi` - 93 edges
6. `DB` - 86 edges
7. `RespondError()` - 83 edges
8. `LeadDetailViewModel` - 73 edges
9. `Icon()` - 73 edges
10. `internalErr()` - 72 edges

## Surprising Connections (you probably didn't know these)
- `RefreshInterceptor (single-flight 401 refresh)` --semantically_similar_to--> `Single-flight 401 -> refresh via X-Refresh-Token`  [INFERRED] [semantically similar]
  android/README.md → dashboard/README.md
- `MockHuginnApi (test-only)` --semantically_similar_to--> `Explicit dev mock layer (VITE_USE_MOCK_API, src/mocks)`  [INFERRED] [semantically similar]
  android/README.md → dashboard/README.md
- `Dashboard Raven Logo (512px)` --semantically_similar_to--> `Play Store Launcher Icon (512px raven)`  [INFERRED] [semantically similar]
  dashboard/public/raven.png → android/app/src/main/ic_launcher-playstore.png
- `Huginn Android App` --conceptually_related_to--> `Huginn pre-CRM sales intelligence platform`  [INFERRED]
  android/README.md → backend/README.md
- `Mandatory contact report workflow (Android)` --references--> `Mandatory report enforcement (MANDATORY_REPORT_PENDING, manager override)`  [INFERRED]
  android/README.md → backend/README.md

## Import Cycles
- None detected.

## Hyperedges (group relationships)
- **Clients consuming Huginn backend API** — android_readme_huginn_android_app, dashboard_readme_huginn_admin_dashboard, backend_api_openapi_huginn_backend_api, backend_readme_huginn_backend [EXTRACTED 1.00]
- **Mandatory contact report flow across client and backend** — android_readme_pendingreportgate, android_readme_mandatory_report_workflow, backend_api_openapi_contact_actions_endpoint, backend_api_openapi_interaction_report_endpoint, backend_readme_mandatory_report_enforcement, android_readme_offline_pending_report_queue [INFERRED 0.85]
- **Backend infrastructure stack** — backend_docker_compose_postgres, backend_docker_compose_redis, backend_docker_compose_rabbitmq, backend_docker_compose_minio, backend_docker_compose_tika, backend_docker_compose_huginn_api, backend_docker_compose_huginn_worker [EXTRACTED 1.00]
- **Android Launcher Icon Density Set** — android_app_src_main_res_mipmap_hdpi_ic_launcher_icon, android_app_src_main_res_mipmap_hdpi_ic_launcher_round_icon, android_app_src_main_res_mipmap_mdpi_ic_launcher_icon, android_app_src_main_res_mipmap_mdpi_ic_launcher_round_icon, android_app_src_main_res_mipmap_xhdpi_ic_launcher_icon, android_app_src_main_res_mipmap_xhdpi_ic_launcher_round_icon, android_app_src_main_res_mipmap_xxhdpi_ic_launcher_icon, android_app_src_main_res_mipmap_xxhdpi_ic_launcher_round_icon, android_app_src_main_res_mipmap_xxxhdpi_ic_launcher_icon, android_app_src_main_res_mipmap_xxxhdpi_ic_launcher_round_icon, android_app_src_main_ic_launcher_playstore_icon [INFERRED 0.95]
- **Cross-platform Raven Branding (Android + Dashboard)** — android_app_src_main_res_drawable_nodpi_ic_logo_icon, android_app_src_main_ic_launcher_playstore_icon, dashboard_public_raven_icon, dashboard_public_raven_icon_icon [INFERRED 0.85]

## Communities (182 total, 58 thin omitted)

### Community 0 - "Import Pipeline & Muninn Store"
Cohesion: 0.03
Nodes (30): PostgresStore, Entry, RequestID(), StorageConfig, Service, newImportHandler(), Batch, Document (+22 more)

### Community 1 - "Dashboard Leads Pages & Auth"
Cohesion: 0.09
Nodes (99): useAuth(), ProtectedRoute(), CreateFollowUpDialog(), Badge(), ConfidenceBadge(), ScoreBadge(), StatusBadge(), Button() (+91 more)

### Community 2 - "Android Token Store & Tests"
Cohesion: 0.05
Nodes (24): TokenStore, CallLauncher, ContactSaveRequired, EmailLauncher, Launched, LaunchResult, MessengerLauncher, NotInstalled (+16 more)

### Community 3 - "Company Text Normalization"
Cohesion: 0.04
Nodes (89): indexOf(), main(), backfillLeadContacts(), companyLineMatches(), findCompanyBlock(), ExtractionService, TestBackfillDoesNotOverwriteManualValue(), TestBackfillDropsInvalidRecovered() (+81 more)

### Community 4 - "Dashboard API Index & UI Kit"
Cohesion: 0.07
Nodes (66): api, isMockApi, queryKeys, PaginationMeta, PERMISSIONS, ImportStatusBadge(), LeadStatusBadge(), ReportStatusBadge() (+58 more)

### Community 5 - "Backend Entrypoints & Wiring"
Cohesion: 0.11
Nodes (27): formatInt(), itoa(), TestChunkText_SingleChunkWhenSmall(), TestChunkText_SplitsWithoutLosingContent(), ChunkText(), prevRuneBoundary(), networkStage(), TestNetworkStage() (+19 more)

### Community 6 - "Android Design Cards"
Cohesion: 0.08
Nodes (3): HuginnTopBar(), ColorDot(), HuginnDimens

### Community 7 - "Dashboard App Shell & Test Setup"
Cohesion: 0.07
Nodes (45): setOnUnauthorized(), setTokensForClient(), Role, App(), queryClient, AuthContext, AuthContextValue, AuthProvider() (+37 more)

### Community 8 - "Backend Leads Cache & Config Tests"
Cohesion: 0.14
Nodes (15): CanTransition(), Status, IsValidStatus(), TestInvalidTransitions(), TestIsValidStatus(), TestSelfTransitionAllowed(), TestValidTransitions(), AllPermissions() (+7 more)

### Community 9 - "Android HuginnApi Retrofit"
Cohesion: 0.06
Nodes (16): HuginnApi, DisqualifyRequest, DisqualifyResponse, FollowUp, FollowUpCreateRequest, FollowUpPriority, FollowUpRescheduleRequest, FollowUpsResponse (+8 more)

### Community 10 - "LLM & Messaging Service Tests"
Cohesion: 0.04
Nodes (71): TestCallScenarioDefaults(), TestDuplicatePairKeyStable(), TestQualificationReportConstants(), callCreate(), TestCreateAllowsEmptyProject(), TestCreateAllowsNilProject(), attemptContext(), TestAttemptContextDropsParentDeadline() (+63 more)

### Community 11 - "Android Session State"
Cohesion: 0.04
Nodes (25): Anonymous, Authenticated, SessionState, Unknown, AlreadyExported, AlreadyQualified, AppError, Conflict (+17 more)

### Community 12 - "Android Date/Time Utils"
Cohesion: 0.07
Nodes (7): DateTimeUtil, DueStateText(), applyPreset(), combineDateAndTime(), FollowUpTimeField(), presetSelected(), DateTimeUtilTest

### Community 13 - "Integration Test Harness"
Cohesion: 0.09
Nodes (72): main(), main(), usage(), main(), NewPostgresStore(), NewManager(), NewAuthService(), NewRepository() (+64 more)

### Community 14 - "Android Info Rows & Badges"
Cohesion: 0.09
Nodes (43): InfoRow(), InfoSpacer(), SectionHeader(), TPCard(), EmptyState(), ErrorState(), LeadScoreBadge(), LeadStatusBadge() (+35 more)

### Community 15 - "Android Mock Data"
Cohesion: 0.07
Nodes (3): MockHuginnApi, MockLead, MockNext

### Community 16 - "Lead Detail Screen (Android)"
Cohesion: 0.06
Nodes (4): CallScenarioUi, LeadDetailUiState, LeadDetailViewModel, MessagingDraft

### Community 17 - "Android Repository Results"
Cohesion: 0.09
Nodes (10): AppResult, Failure, getOrNull(), runCatchingAppError(), Success, FollowUpRepository, InteractionRepository, LeadPatch (+2 more)

### Community 18 - "Dashboard ApiEndpoints"
Cohesion: 0.04
Nodes (10): ApiEndpoints, Contact, ContactReport, FollowUp, FollowUpCreateData, FollowUpCreateGlobalData, FollowUpUpdateData, LLMSettings (+2 more)

### Community 19 - "Backend Config & Cache Client"
Cohesion: 0.05
Nodes (40): Tokens, Client, New(), AuthConfig, Config, MinIOConfig, MuninnConfig, PostgresConfig (+32 more)

### Community 20 - "Android External Launchers"
Cohesion: 0.07
Nodes (13): AppInfo, ContactInsertLauncher, DeepLinkValidator, MessengerChannels, startSafely(), SystemContactInsertLauncher, SystemEmailLauncher, SystemIntentCallLauncher (+5 more)

### Community 21 - "Next Actions Engine"
Cohesion: 0.08
Nodes (46): ParseTime(), ScoreSnapshot, Service, reasonSummary(), sourceForInferred(), strVal(), actionTypeForChannel(), ctaLabelFor() (+38 more)

### Community 22 - "Lead Assignments Service"
Cohesion: 0.09
Nodes (48): Assignment, Service, Store, Service, NewService(), Repository, NewRepository(), NewService() (+40 more)

### Community 23 - "HTTP Data Handlers"
Cohesion: 0.10
Nodes (12): dataHandler, intQuery(), joinAnd(), internalErr(), dataHandler, leadHandler, intParam(), RespondOK() (+4 more)

### Community 24 - "Dashboard Forms & Layout"
Cohesion: 0.09
Nodes (40): FOLLOWUP_TYPE_OPTIONS, FollowUpForm(), FollowUpFormValue, Icon(), IconProps, paths, AppShell(), pageTitleFor() (+32 more)

### Community 25 - "Android ApiCaller Errors"
Cohesion: 0.06
Nodes (14): ApiCaller, ApiErrorCodes, ApiException, NetworkException, toAppError(), SafeLog, FollowUpStatus, Failed (+6 more)

### Community 26 - "Android Speech Recognizer"
Cohesion: 0.06
Nodes (18): Error, Listening, Partial, Processing, RealSpeechRecognizer, Result, SpeechErrorKind, FAILURE (+10 more)

### Community 27 - "Teleprompter Service"
Cohesion: 0.05
Nodes (3): TeleprompterOverlayContent(), TeleprompterSection(), TeleprompterService

### Community 28 - "Leads Service & Repository"
Cohesion: 0.09
Nodes (13): IsNoRows(), Service, nilVal(), strptr(), timeOrNil(), RecentItem, Version, Repository (+5 more)

### Community 29 - "Android Lead Models"
Cohesion: 0.11
Nodes (13): AgentMetrics, Lead, LeadListResponse, MeStats, PendingReport, PendingReportsResponse, WorkItem, WorkQueueResponse (+5 more)

### Community 30 - "HTTP Lead Handler Helpers"
Cohesion: 0.15
Nodes (12): FromContext(), actorEntry(), strptrPtr(), atoi(), bodyNote(), leadHandler, stringOrEmpty(), updateStatusFromReport() (+4 more)

### Community 31 - "Phone Number Formatting"
Cohesion: 0.08
Nodes (13): PhoneFormatter, ChannelAvailability, CONTACT_SAVE_REQUIRED, DISABLED_FORBIDDEN, DISABLED_NO_EMAIL, DISABLED_NO_NUMBER, DISABLED_NO_TARGET, DISABLED_NOT_INSTALLED (+5 more)

### Community 32 - "Queue Worker Handler"
Cohesion: 0.07
Nodes (20): ProcessOutcome, orEmpty(), stringOrEmpty(), NewPermanentf(), Client, Handler, Job, New() (+12 more)

### Community 33 - "Android Work Queue"
Cohesion: 0.09
Nodes (25): NextActionItem, NextActionPriority, NextActionSource, NextActionType, actionIcon(), SnoozeDialog(), SnoozeOption(), tabLabelRes() (+17 more)

### Community 34 - "Dashboard Mock API"
Cohesion: 0.07
Nodes (36): attachLeadMemberships(), attachMembership(), auditEntries, batches, businessTypes, cities, companies, daysAgo() (+28 more)

### Community 35 - "Android Navigation Routes"
Cohesion: 0.07
Nodes (16): disqualifyReasonLabel(), Routes, HuginnApp(), MainNavHost(), ProfileScreen(), roleLabel(), disqualifyError(), DisqualifyScreen() (+8 more)

### Community 36 - "Contacts & Duplicates Scan"
Cohesion: 0.07
Nodes (17): Contact, scan(), nullStr(), evidenceValue(), nextActionFor(), populateOwnerNames(), Lead, contactName() (+9 more)

### Community 37 - "Android Buttons & Messaging Sheet"
Cohesion: 0.10
Nodes (18): DangerButton(), FullWidthSpacer(), LoadingBox(), PrimaryButton(), SecondaryButton(), channelLabel(), CallScenarioPromptSheet(), CallScenarioSheet() (+10 more)

### Community 38 - "Analytics Reports"
Cohesion: 0.10
Nodes (18): ActivityPoint, AgentPerformanceRow, condition, DecisionMakerReport, FunnelStep, InterestCount, Objection, OutcomeCount (+10 more)

### Community 39 - "Dashboard API Types (Health/Timeline)"
Cohesion: 0.05
Nodes (34): ActionStatus, AgentPerformanceRow, ApiErrorShape, Assignment, BackendHealth, BatchStatus, ChannelType, DashboardStats (+26 more)

### Community 40 - "Call Scenario Strategy Model"
Cohesion: 0.08
Nodes (22): CallScenarioStrategy, ContextUsed, ConversationBranch, DiscoveryQuestion, jsonOrNull(), nullGenerationError(), nullID(), scanScenario() (+14 more)

### Community 41 - "Follow-ups Service"
Cohesion: 0.10
Nodes (13): ListItem, Repository, Service, queryOne(), scan(), scanListItem(), CreateParams, FollowUp (+5 more)

### Community 42 - "Android Detail Models"
Cohesion: 0.18
Nodes (23): Agent, AssignRequest, Contact, ContactUpdateRequest, CreateNoteRequest, EvidenceItem, EvidenceItemDto, EvidenceResponse (+15 more)

### Community 43 - "Android Voice Input UI"
Cohesion: 0.10
Nodes (19): rememberVoiceInput(), VoiceInputButton(), VoiceInputHandlers, VoiceInputStatus(), VoiceTextField(), Idle, Listening, Processing (+11 more)

### Community 44 - "Audit & Team Reports"
Cohesion: 0.16
Nodes (5): FollowUpReport, FollowUpStatusCount, ctxKey, loginRequest, mwKey

### Community 45 - "Android App UI Tests"
Cohesion: 0.10
Nodes (10): AppUiTest, HuginnTheme(), toMaterialScheme(), errorMessage(), LoginScreen(), FollowUpCard(), FollowUpsContent(), FollowUpsScreen() (+2 more)

### Community 46 - "Qualification Repository (Android)"
Cohesion: 0.10
Nodes (9): buildQualifyRequest(), ChecklistKey, QualificationRepository, DisqualifyUiState, DisqualifyViewModel, QualificationChecklist, QualifyUiState, QualifyViewModel (+1 more)

### Community 47 - "Report Draft Validation"
Cohesion: 0.12
Nodes (10): ReportDraft, ReportValidationError, FOLLOW_UP_TIME_REQUIRED, INTEREST_REQUIRED, NEXT_ACTION_REQUIRED, OUTCOME_REQUIRED, SUMMARY_REQUIRED, ReportValidator (+2 more)

### Community 49 - "OpenAPI Spec"
Cohesion: 0.08
Nodes (31): AgentMetrics schema, Auth endpoints (/auth/login, /auth/refresh, /auth/logout), Call scenario endpoints (/leads/{leadID}/call-scenario(s)), CallScenario schema, Campaign schema, Campaigns endpoints, /change-history (Audit), Contact schema (+23 more)

### Community 50 - "Leads Database Queries"
Cohesion: 0.08
Nodes (6): Querier, nullStr(), scanLead(), Repository, ListParams, ListResult

### Community 51 - "Call Scenario Generation"
Cohesion: 0.12
Nodes (25): WithoutDeadline(), applyGenerationMeta(), compactRejection(), errString(), CallScenario, GenerationOptions, interactionCount(), projectID() (+17 more)

### Community 52 - "Campaigns Repository"
Cohesion: 0.08
Nodes (10): Repository, joinStrings(), scanCampaign(), Service, CreateParams, ListParams, ListResult, ProjectRef (+2 more)

### Community 53 - "Android Home Screen"
Cohesion: 0.13
Nodes (16): FollowUpDateTimeDialog(), actionCompletable(), actionIcon(), actionSnoozable(), HomeContent(), HomeScreen(), NextActionCard(), NextActionsSection() (+8 more)

### Community 55 - "Worker Error Classification"
Cohesion: 0.13
Nodes (28): IsPermanent(), IsRetryable(), NewPermanent(), PermanentCode(), TestHTTPStatusClassification(), TestIsPermanent(), TestIsRetryableDefaultsTransient(), TestIsRetryablePermanent() (+20 more)

### Community 56 - "Dashboard API Interface"
Cohesion: 0.07
Nodes (15): AuditQuery, ImportCreateResult, LeadsQuery, ReportsQuery, AuthUser, CrmExport, DocumentDetail, GlobalSearchResults (+7 more)

### Community 57 - "Dashboard Work Queue Page"
Cohesion: 0.17
Nodes (27): filterItems(), FilterKey, FILTERS, priorityLabel(), priorityTone(), WorkQueuePage(), addDaysJalali(), BREAKS (+19 more)

### Community 58 - "Qualification & Scoring Services"
Cohesion: 0.10
Nodes (21): Repository, Service, IsValidDisqualifyReason(), NewRepository(), NewService(), ShouldQualify(), TestDisqualifyReasons(), TestShouldQualifyAllRules() (+13 more)

### Community 59 - "Android LLM API"
Cohesion: 0.16
Nodes (7): LlmHuginnApi, CallScenario, CallScenarioRequest, CallScenarioResponse, ContactActionRequest, ContactActionResponse, WorkQueueMappingTest

### Community 60 - "Android Scaffold & Bottom Nav"
Cohesion: 0.08
Nodes (8): HuginnBottomNavigation(), HuginnDestination, FOLLOWUPS, HOME, LEADS, PROFILE, REPORTS, HuginnScaffold()

### Community 62 - "JWT Auth Manager"
Cohesion: 0.10
Nodes (13): LoginParams, Manager, TokenClaims, HasPermission(), NewMiddleware(), LoginResult, Service, strptr() (+5 more)

### Community 64 - "Messenger Deep Link Routing"
Cohesion: 0.18
Nodes (4): MessengerDeepLinks, MessengerPolicy, DeepLink, MessengerRoutingTest

### Community 65 - "Android Reports ViewModel"
Cohesion: 0.15
Nodes (7): Empty, Error, Loaded, Loading, ReportsUiState, ReportsViewModel, ReportsViewModelTest

### Community 66 - "Env Config Loading Tests"
Cohesion: 0.13
Nodes (25): TestDebugOverview(), TestPersistenceRoundTripGenerationStatus(), getEnv(), getEnvBool(), getEnvInt(), Load(), New(), integrationEnabled() (+17 more)

### Community 67 - "Dashboard HTTP Client"
Cohesion: 0.14
Nodes (18): apiDelete(), ApiError, ApiErrorShape, apiGet(), apiPatch(), apiPost(), apiPut(), buildQuery() (+10 more)

### Community 68 - "Roles Repository"
Cohesion: 0.15
Nodes (10): dedupe(), NewRepository(), NewService(), scan(), validKey(), CreateParams, Repository, Role (+2 more)

### Community 69 - "Android Follow-ups ViewModel"
Cohesion: 0.16
Nodes (11): Empty, Error, FollowUpsUiState, FollowUpsViewModel, FollowUpTab, COMPLETED, OVERDUE, TODAY (+3 more)

### Community 70 - "Auth Middleware & RBAC"
Cohesion: 0.12
Nodes (11): writeAuthError(), Middleware, respondLogin(), clientIP(), NewMiddleware(), newRequestID(), RequestIDFromContext(), splitComma() (+3 more)

### Community 71 - "HTTP Lead Responses"
Cohesion: 0.16
Nodes (5): beforeMap(), leadHandler, contactFullName(), RespondNoContent(), userHandler

### Community 73 - "Queue Delivery Tests"
Cohesion: 0.21
Nodes (12): delivery(), jobBody(), newTestClient(), TestHandleDeliveryMalformedBodyDeadLetters(), TestHandleDeliveryPermanentDeadLettersImmediately(), TestHandleDeliveryRetryLimitExhausted(), TestHandleDeliverySuccessAcks(), TestHandleDeliveryTransientRecovers() (+4 more)

### Community 75 - "Android Leads ViewModel"
Cohesion: 0.15
Nodes (4): LeadFilters, LeadsFiltersUi, LeadsUiState, LeadsViewModel

### Community 77 - "HTTP Error Responses"
Cohesion: 0.17
Nodes (10): followUpListItem(), valueOrEmpty(), strptr(), ErrorFrom(), statusForCode(), Project, joinNonEmpty(), APIError (+2 more)

### Community 78 - "Users Repository"
Cohesion: 0.15
Nodes (6): Repository, User, itoa(), scanSession(), scanUser(), Session

### Community 80 - "Duplicates & Evidence Detection"
Cohesion: 0.14
Nodes (13): Service, Detector, Repository, NewDetector(), Repository, ExtractionService, NewExtractionService(), Runner (+5 more)

### Community 81 - "User Handlers & RBAC Checks"
Cohesion: 0.13
Nodes (9): newUserHandler(), Role, HasPermission(), IsRole(), Service, NewService(), validateRole(), CreateParams (+1 more)

### Community 82 - "Dashboard package.json"
Cohesion: 0.11
Nodes (20): description, name, private, type, version, eslint, @eslint/js, eslint-plugin-react-hooks (+12 more)

### Community 83 - "Dashboard tsconfig.app"
Cohesion: 0.09
Nodes (21): compilerOptions, allowImportingTsExtensions, baseUrl, isolatedModules, jsx, lib, module, moduleDetection (+13 more)

### Community 85 - "HTTP Principal & Filters"
Cohesion: 0.35
Nodes (7): Principal, WithPrincipal(), idPtr(), optStr(), parseFilters(), unauthorized(), reportsHandler

### Community 86 - "Projects & Notes Scanning"
Cohesion: 0.13
Nodes (9): RowScanner, scanNote(), joinStrings(), scanProject(), scanProjectRow(), Note, ListParams, ListResult (+1 more)

### Community 87 - "Scenario Normalization Tests"
Cohesion: 0.12
Nodes (17): discoverLike(), TestConsistencyValidationFlags(), TestNormalizeStrategyEvidenceDefaults(), TestSparseLeadCoverage(), nextContractLike(), normalizeScenario(), normalizeStrategyEvidence(), pitchLike() (+9 more)

### Community 88 - "Deep Links & vCards"
Cohesion: 0.20
Nodes (19): BuildDeepLink(), BuildVCard(), emailTarget(), escapeVCard(), smsTarget(), telTarget(), strPtr(), TestBuildDeepLinkCall() (+11 more)

### Community 89 - "Prompt Settings Storage"
Cohesion: 0.22
Nodes (7): ensureSettingsTable(), getSetting(), dataHandler, hostOf(), setSetting(), NewPostgresPromptSettings(), Weights()

### Community 90 - "Dashboard Dev Dependencies"
Cohesion: 0.10
Nodes (20): devDependencies, eslint, @eslint/js, eslint-plugin-react-hooks, eslint-plugin-react-refresh, globals, jsdom, @testing-library/jest-dom (+12 more)

### Community 92 - "Android Room DAOs"
Cohesion: 0.15
Nodes (3): CachedDetailDao, CachedLeadDao, PendingReportDao

### Community 93 - "Lead Versioning & Diff"
Cohesion: 0.25
Nodes (16): Diff(), Snapshot, HasChanges(), indexContacts(), jsonString(), contact(), leadMap(), TestDiffContactAddedAndRemoved() (+8 more)

### Community 94 - "Dashboard Charts"
Cohesion: 0.22
Nodes (14): BarChart(), DonutChart(), FunnelChart(), LineChart(), PALETTE, QualifiedBarChart(), IconName, CardSkeletonGrid() (+6 more)

### Community 95 - "Muninn HTTP Exporter"
Cohesion: 0.14
Nodes (11): HTTPExporter, Service, trimSlash(), truncate(), Bundle, CompanyInfo, ContactInfo, ExportPayload (+3 more)

### Community 97 - "Android Room Database"
Cohesion: 0.23
Nodes (4): CachedLeadDetailEntity, CachedLeadEntity, CachedWorkDao, CachedWorkItemEntity

### Community 100 - "Dashboard tsconfig.node"
Cohesion: 0.12
Nodes (16): compilerOptions, allowImportingTsExtensions, baseUrl, isolatedModules, jsx, lib, module, moduleDetection (+8 more)

### Community 101 - "Raven Brand Icons"
Cohesion: 0.23
Nodes (16): Play Store Launcher Icon (512px raven), Huginn Raven Brand Identity, Huginn Raven Logo (full-res, transparent), Radar + Circuit Trace Motif, Launcher Icon hdpi, Round Launcher Icon hdpi, Launcher Icon mdpi, Round Launcher Icon mdpi (+8 more)

### Community 102 - "Android Theme Settings"
Cohesion: 0.17
Nodes (5): ThemeMode, DARK, LIGHT, SYSTEM, SettingsViewModel

### Community 103 - "Clipboard & Suggested Message"
Cohesion: 0.22
Nodes (3): copyToClipboard(), fullSuggestedMessage(), SuggestedMessageCopyTest

### Community 104 - "Contact Reports Store"
Cohesion: 0.21
Nodes (10): interestPtr(), nextActionPtr(), Interest, NextAction, Report, scan(), Validate(), Outcome (+2 more)

### Community 105 - "Messaging Channels & Caps"
Cohesion: 0.17
Nodes (11): Channel, IsValidChannel(), TestMessageCharCap(), defaultMessage(), runeCount(), trimRunes(), withTone(), DeepLink (+3 more)

### Community 106 - "Dashboard Documents & Audit API"
Cohesion: 0.13
Nodes (7): AuditEntry, ChangeHistoryItem, DuplicatePair, LeadVersion, Paginated, ProjectLeadRow, SourceDocument

### Community 109 - "Score Classifier"
Cohesion: 0.25
Nodes (7): ScoreClassifier, ScoreLevel, HIGH, LOW, MEDIUM, ScoreThresholds, ScoreClassifierTest

### Community 110 - "Fake Auth Test Doubles"
Cohesion: 0.28
Nodes (3): FakeRefreshApi, FakeTokenStore, SessionManagerTest

### Community 111 - "Docker Compose Services"
Cohesion: 0.18
Nodes (12): grafana service (observability profile), huginn-api service, huginn-worker service, minio service, mock-llm dev service, postgres service (postgres:16), prometheus service (observability profile), rabbitmq service (+4 more)

### Community 112 - "LLM Call Metrics"
Cohesion: 0.19
Nodes (10): inputTokensOf(), latencyOf(), modelOf(), outputTokensOf(), responseCharsOf(), statusOf(), nullIfEmpty(), Run (+2 more)

### Community 113 - "LLM Runner Interface"
Cohesion: 0.18
Nodes (5): Request, Response, fakeRunner, captureRunner, failingRunner

### Community 114 - "Scenario Prompt Builder"
Cohesion: 0.16
Nodes (13): buildScenarioPrompt(), fallbackScenario(), TestBuildScenarioPromptIncludesPreviousAngleForGenerateAgain(), TestBuildScenarioPromptIncludesUniqueEnvelopeAndContext(), TestFallbackScenarioIncludesStrategy(), TestFallbackScenarioNoCompany(), TestFallbackScenarioUsesProjectContext(), TestFallbackScenarioUsesRealCompany() (+5 more)

### Community 115 - "Scoring Signals"
Cohesion: 0.15
Nodes (14): BuildSignals(), boolp(), purchaseTimingSoon(), strVal(), strp(), TestBuildSignalsAccumulatesAcrossReports(), TestBuildSignalsDoesNotClearFactsFromEmptyLaterReport(), TestBuildSignalsLeadFactPowersAuthority() (+6 more)

### Community 120 - "Dashboard Docs & RBAC Roles"
Cohesion: 0.16
Nodes (9): MockHuginnApi (test-only), RBAC roles (super_admin, sales_development_manager, sales_agent), Dashboard index.html (RTL fa shell, Vazirmatn font, /src/main.tsx), ApiEndpoints interface (src/api/interface.ts), Huginn Admin Dashboard, nginx static deployment with /api proxy (BACKEND_UPSTREAM), src/api/real.ts centralized REST client, React + TypeScript + Vite + TanStack + React Hook Form/Zod (+1 more)

### Community 121 - "Import Chunk Extraction Tests"
Cohesion: 0.26
Nodes (13): sadaf2CandidatesJSON(), TestChunkExtractionLLMSeesCleanContacts(), candidateJSON(), doc(), fixtureText(), newService(), normalizedMobile(), strPtr() (+5 more)

### Community 122 - "Scenario Strategy Synthesis"
Cohesion: 0.21
Nodes (10): ensureStrategy(), TestEnsureStrategySynthesizesWhenMissing(), LeadContext, joinNonEmpty(), projectProduct(), projectTitle(), projectValue(), TestProjectUpdateAffectsFutureScenarios() (+2 more)

### Community 123 - "Projects Service"
Cohesion: 0.18
Nodes (3): Service, CreateParams, UpdateParams

### Community 124 - "Edit Lead Screen"
Cohesion: 0.26
Nodes (7): errorMessage(), ContactEditCard(), EditLeadForm(), EditLeadScreen(), EditSection(), LabeledField(), ToggleRow()

### Community 125 - "Dashboard Analytics Service"
Cohesion: 0.15
Nodes (7): Service, AgentPerformanceRow, CityCount, FunnelStep, QualificationBySource, RejectionReason, TrendPoint

### Community 126 - "Message Prompt Builder"
Cohesion: 0.38
Nodes (12): buildMessagePrompt(), leadContextOf(), sadafLead(), TestBuildMessagePromptIncludesRichContext(), TestBuildMessagePromptNoSenderProfile(), TestBuildMessagePromptSeparatesSenderAndRecipient(), testSender(), TestValidateMessagePerspectiveAcceptsCorrectPerspective() (+4 more)

### Community 127 - "Android Architecture Docs"
Cohesion: 0.20
Nodes (8): ApiCaller (error envelope -> AppError), core/external Intent launchers (CallLauncher, SmsLauncher, MessengerLauncher, EmailLauncher, ContactInsertLauncher), Huginn Android App, HUGINN_API_BASE_URL Gradle property, HuginnApi (Retrofit contract), NotificationHelper, ErrorResponse schema, Machine-readable error envelope {error:{code,message,details}}

### Community 128 - "DB Migrator"
Cohesion: 0.32
Nodes (4): LoadMigrationFiles(), NewMigrator(), MigrationFile, Migrator

### Community 129 - "Backend Architecture Docs"
Cohesion: 0.26
Nodes (9): Deterministic explainable scoring, Huginn pre-CRM sales intelligence platform, Huginn Backend, Ingestion data flow (Import -> MinIO -> RabbitMQ -> Worker -> Tika -> LLM -> Dedup -> Scoring -> Assignment -> Qualification -> Outbox -> Muninn), Strict lead state machine, LLM-driven contact backfill (make backfill-contacts), Muninn CRM, Deterministic + pg_trgm fuzzy deduplication (+1 more)

### Community 130 - "Android Auth Models"
Cohesion: 0.40
Nodes (5): AuthResponse, ErrorBody, ErrorEnvelope, Tokens, User

### Community 133 - "LLM Retry & JSON Extraction"
Cohesion: 0.24
Nodes (8): asErrProvider(), retryable(), truncate(), extractJSON(), repairPrompt(), requestHash(), sleepCtx(), TestSOCKSUnreachableIsRetryable()

### Community 137 - "Interactions Store"
Cohesion: 0.29
Nodes (5): pendingID(), Interaction, scan(), ActionStatus, Direction

### Community 138 - "Sender Perspective Checks"
Cohesion: 0.44
Nodes (8): claimsSenderFromDescriptors(), claimsSenderFromRecipientBusiness(), containsCompanyName(), normalizeCompanyName(), recipientKeywords(), sentenceClaimsSenderBelongsToRecipient(), splitSentences(), validateMessagePerspective()

### Community 139 - "Dashboard Runtime Dependencies"
Cohesion: 0.20
Nodes (10): dependencies, @hookform/resolvers, react, react-dom, react-hook-form, react-router-dom, @tanstack/react-query, @tanstack/react-table (+2 more)

### Community 141 - "Lead Filter Sheet"
Cohesion: 0.42
Nodes (6): FilterGroup(), FilterSheet(), LeadList(), LeadsListContent(), LeadsScreen(), RadioRow()

### Community 144 - "OpenAI Provider"
Cohesion: 0.33
Nodes (4): formatLabel(), formatUnsupported(), parseProviderError(), openAIProvider

### Community 147 - "Mandatory Report Workflow Docs"
Cohesion: 0.25
Nodes (5): PendingReportGate / AppGateViewModel, ActionResult schema, POST /leads/{leadID}/contact-actions, Me endpoints (/me, /me/work-queue, /me/pending-reports, /me/reports), WorkQueueItem schema

### Community 151 - "Dashboard npm Scripts"
Cohesion: 0.29
Nodes (7): scripts, build, dev, lint, preview, test, test:watch

### Community 152 - "LLM Error Types"
Cohesion: 0.29
Nodes (3): ErrProvider, ErrValidation, Purpose

### Community 153 - "Prompt Settings"
Cohesion: 0.47
Nodes (3): Defaults(), PostgresPromptSettings, PromptSettings

### Community 157 - "Android Auth Docs"
Cohesion: 0.50
Nodes (5): RefreshInterceptor (single-flight 401 refresh), SessionManager, TokenStorage (EncryptedSharedPreferences/Keystore), JWT access + rotating hashed refresh tokens, Single-flight 401 -> refresh via X-Refresh-Token

### Community 161 - "Personalization Similarity"
Cohesion: 0.50
Nodes (3): distinctTokens(), jaccard(), personalizationSimilarity()

## Knowledge Gaps
- **312 isolated node(s):** `Unknown`, `Anonymous`, `HOME`, `LEADS`, `FOLLOWUPS` (+307 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 795 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **58 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `MockHuginnApi` connect `Android Mock Data` to `Messenger Deep Link Routing`, `Android Reports ViewModel`, `Android Token Store & Tests`, `Android Auth Models`, `Follow-ups ViewModel Tests`, `Robolectric Test Config`, `Edit Lead Flow Tests`, `Android HuginnApi Retrofit`, `Android Detail Models`, `Qualification Repository (Android)`, `Edit Lead ViewModel`, `Android Repository Results`, `Android LLM API`, `Android Lead Models`?**
  _High betweenness centrality (0.028) - this node is a cross-community bridge._
- **What connects `Unknown`, `Anonymous`, `HOME` to the rest of the system?**
  _312 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Import Pipeline & Muninn Store` be split into smaller, more focused modules?**
  _Cohesion score 0.026185816792470413 - nodes in this community are weakly interconnected._
- **Why does `LeadDetailViewModel` connect `Lead Detail Screen (Android)` to `Android Token Store & Tests`, `Android Navigation Routes`, `Robolectric Test Config`, `Android Info Rows & Badges`, `Qualification Repository (Android)`, `Android Repository Results`, `Android External Launchers`?**
  _High betweenness centrality (0.012) - this node is a cross-community bridge._
- **Should `Dashboard Leads Pages & Auth` be split into smaller, more focused modules?**
  _Cohesion score 0.08590492076730609 - nodes in this community are weakly interconnected._
- **Why does `HuginnApi` connect `Android HuginnApi Retrofit` to `Android Auth Models`, `Android Token Store & Tests`, `Follow-ups ViewModel Tests`, `Android Detail Models`, `Android Session State`, `Qualification Repository (Android)`, `Android Network Module`, `Android Mock Data`, `Android Repository Results`, `Android ApiCaller Errors`, `Android LLM API`?**
  _High betweenness centrality (0.010) - this node is a cross-community bridge._
- **Should `Android Token Store & Tests` be split into smaller, more focused modules?**
  _Cohesion score 0.046381243628950054 - nodes in this community are weakly interconnected._