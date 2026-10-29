package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/tafakorpardazan/huginn/internal/assignments"
	"github.com/tafakorpardazan/huginn/internal/audit"
	"github.com/tafakorpardazan/huginn/internal/analytics"
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
	"github.com/tafakorpardazan/huginn/internal/followups"
	"github.com/tafakorpardazan/huginn/internal/imports"
	"github.com/tafakorpardazan/huginn/internal/interactions"
	"github.com/tafakorpardazan/huginn/internal/leads"
	"github.com/tafakorpardazan/huginn/internal/llm"
	"github.com/tafakorpardazan/huginn/internal/messaging"
	"github.com/tafakorpardazan/huginn/internal/muninn"
	"github.com/tafakorpardazan/huginn/internal/nextactions"
	"github.com/tafakorpardazan/huginn/internal/notes"
	"github.com/tafakorpardazan/huginn/internal/projects"
	"github.com/tafakorpardazan/huginn/internal/qualification"
	"github.com/tafakorpardazan/huginn/internal/rbac"
	"github.com/tafakorpardazan/huginn/internal/reports"
	"github.com/tafakorpardazan/huginn/internal/scoring"
	"github.com/tafakorpardazan/huginn/internal/storage"
	"github.com/tafakorpardazan/huginn/internal/timeline"
	"github.com/tafakorpardazan/huginn/internal/users"
)

// Services bundles the service layer for handlers.
type Services struct {
	Config        *config.Config
	Logger        *slog.Logger
	AuthMW        *auth.Middleware
	AuthSvc       *auth.Service
	Users         *users.Service
	Leads         *leads.Service
	Imports       *imports.Service
	Documents     *documents.Service
	Messaging     *messaging.ContactActionService
	CallScenarios *callscenarios.Repository
	Reports       *reports.Service
	ReportRepo    *reports.Repository
	FollowUps     *followups.Service
	Qualify       *qualification.Service
	Scoring        *scoring.Repository
	LeadScorer     *scoring.LeadScorer
	Analytics      *analytics.Service
	Duplicates     *duplicates.Detector
	DupsRepo      *duplicates.Repository
	Dashboard     *dashboard.Service
	Muninn        *muninn.Service
	Notes         *notes.Repository
	Timeline      *timeline.Repository
	Audit         *audit.Logger
	Assignments   *assignments.Repository
	LeadRepo      *leads.Repository
	ContactRepo   *contacts.Repository
	Projects      *projects.Service
	Campaigns     *campaigns.Service
	Interactions  *interactions.Repository
	NextActions   *nextactions.Service
	LLMRunner     llm.Runner
	Store         *storage.Store
	DB            *database.DB
	Cache         *cache.Client
}

// NewRouter builds the full HTTP router.
func NewRouter(ctx context.Context, s Services) http.Handler {
	r := chi.NewRouter()

	mw := NewMiddleware(s.Logger, s.Cache, s.Config.Security.AllowedOrigins)
	rateLimit := mw.RateLimit(s.Config.Security.RateLimitPerMin, time.Minute)

	r.Use(mw.RequestID)
	r.Use(mw.Logging)
	r.Use(mw.Recover)
	r.Use(mw.CORS)
	r.Use(mw.SecurityHeaders)

	r.Get("/health", handleHealth(s.DB))
	r.Get("/ready", handleReady(s.DB, s.Cache))
	r.Get("/metrics", handleMetrics(s.Config))

	authH := newAuthHandler(s.AuthSvc, s.Config.Auth.RefreshCookieName)

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(rateLimit)

		r.Route("/auth", func(r chi.Router) {
			r.Post("/login", authH.Login)
			r.Post("/refresh", authH.Refresh)
			r.Post("/logout", authH.Logout)
		})

		r.Group(func(r chi.Router) {
			r.Use(s.AuthMW.Authenticate)

			meH := newMeHandler(s.Users, s.Leads, s.Interactions, s.ContactRepo, s.ReportRepo, s.Logger)
			r.Get("/me", meH.GetMe)
			r.Get("/me/pending-reports", meH.PendingReports)
			r.Get("/me/reports", meH.MyReports)

			nextActionH := newNextActionHandler(s.NextActions, s.FollowUps, s.Audit)
			r.Get("/me/next-actions", nextActionH.MyActions)
			r.Get("/me/work-queue", nextActionH.MyActions)
			r.Post("/me/next-actions/{actionID}/snooze", nextActionH.Snooze)
			r.Post("/me/next-actions/{actionID}/complete", nextActionH.Complete)
			r.With(s.AuthMW.RequirePermission(rbac.PermDashboardView)).Get("/team/next-actions", nextActionH.TeamActions)

			userH := newUserHandler(s.Users, s.AuthMW, s.Audit)
			r.Route("/users", func(r chi.Router) {
				r.With(s.AuthMW.RequirePermission(rbac.PermUsersManage)).Post("/", userH.Create)
				r.With(s.AuthMW.RequirePermission(rbac.PermUsersManage)).Get("/", userH.List)
				r.Route("/{userID}", func(r chi.Router) {
					r.With(s.AuthMW.RequirePermission(rbac.PermUsersManage)).Get("/", userH.Get)
					r.With(s.AuthMW.RequirePermission(rbac.PermUsersManage)).Patch("/", userH.Update)
					r.With(s.AuthMW.RequirePermission(rbac.PermUsersManage)).Post("/deactivate", userH.Deactivate)
					r.With(s.AuthMW.RequirePermission(rbac.PermUsersManage)).Post("/reactivate", userH.Reactivate)
					r.With(s.AuthMW.RequirePermission(rbac.PermUsersManage)).Post("/reset-password", userH.ResetPassword)
					r.With(s.AuthMW.RequirePermission(rbac.PermUsersManage)).Get("/sessions", userH.ListSessions)
					r.With(s.AuthMW.RequirePermission(rbac.PermUsersManage)).Delete("/sessions/{sessionID}", userH.RevokeSession)
				})
			})

			importH := newImportHandler(s.Imports, s.Documents)
			r.Route("/imports", func(r chi.Router) {
				r.With(s.AuthMW.RequirePermission(rbac.PermImportsCreate)).Post("/", importH.Create)
				r.Get("/", importH.List)
				r.Get("/{batchID}", importH.Get)
				r.With(s.AuthMW.RequirePermission(rbac.PermImportsCreate)).Post("/{batchID}/queue", importH.Queue)
				r.With(s.AuthMW.RequirePermission(rbac.PermImportsCreate)).Post("/{batchID}/documents", importH.AddDocument)
				r.With(s.AuthMW.RequirePermission(rbac.PermImportsCreate)).Post("/{batchID}/raw-text", importH.AddRawText)
			})
			r.Get("/documents/{docID}", importH.GetDocument)

			leadH := newLeadHandler(&s)
			r.Route("/leads", func(r chi.Router) {
				r.Get("/", leadH.List)
				r.With(s.AuthMW.RequirePermission(rbac.PermLeadsCreate)).Post("/", leadH.Create)
				r.Route("/{leadID}", func(r chi.Router) {
					r.Get("/", leadH.Get)
					r.With(s.AuthMW.RequirePermission(rbac.PermLeadsEdit)).Patch("/", leadH.Update)
					r.Get("/timeline", leadH.Timeline)
					r.Get("/duplicates", leadH.Duplicates)
					r.With(s.AuthMW.RequirePermission(rbac.PermLeadsHistory)).Get("/versions", leadH.Versions)
					r.With(s.AuthMW.RequirePermission(rbac.PermLeadsHistory)).Get("/versions/{versionID}", leadH.VersionDetail)
					r.Get("/notes", leadH.Notes)
					r.Post("/notes", leadH.AddNote)
					r.Get("/contacts", leadH.Contacts)
					r.With(s.AuthMW.RequirePermission(rbac.PermLeadsEdit)).Post("/contacts", leadH.AddContact)
					r.Get("/contacts/{contactID}/vcard", leadH.VCard)
					r.With(s.AuthMW.RequirePermission(rbac.PermLeadsEdit)).Patch("/contacts/{contactID}", leadH.UpdateContact)
					r.With(s.AuthMW.RequirePermission(rbac.PermLeadsEdit)).Delete("/contacts/{contactID}", leadH.DeleteContact)
					r.Get("/evidence", leadH.Evidence)
					r.Get("/score", leadH.Score)
					r.Get("/interactions", leadH.Interactions)
					r.Get("/reports", leadH.Reports)
					r.Get("/followups", leadH.LeadFollowUps)
					r.Get("/export", leadH.ExportGet)
					r.With(s.AuthMW.RequirePermission(rbac.PermLeadsAssign)).Post("/assign", leadH.Assign)
					r.With(s.AuthMW.RequirePermission(rbac.PermLeadsAssign)).Post("/assign-bulk", leadH.AssignBulk)
					r.With(s.AuthMW.RequirePermission(rbac.PermLeadsMerge)).Post("/merge", leadH.Merge)
					r.With(s.AuthMW.RequirePermission(rbac.PermLeadsContact)).Post("/contact-actions", leadH.ContactAction)
					r.With(s.AuthMW.RequirePermission(rbac.PermLeadsContact)).Post("/call-scenario", leadH.CallScenario)
					r.Get("/call-scenarios", leadH.ListCallScenarios)
					r.With(s.AuthMW.RequirePermission(rbac.PermLeadsContact)).Post("/call-scenarios", leadH.CreateCallScenario)
					r.Get("/call-scenarios/{scenarioID}", leadH.GetCallScenario)
					r.With(s.AuthMW.RequirePermission(rbac.PermLeadsContact)).Put("/call-scenarios/{scenarioID}", leadH.UpdateCallScenario)
					r.With(s.AuthMW.RequirePermission(rbac.PermLeadsContact)).Patch("/call-scenarios/{scenarioID}", leadH.UpdateCallScenario)
					r.With(s.AuthMW.RequirePermission(rbac.PermLeadsContact)).Delete("/call-scenarios/{scenarioID}", leadH.DeleteCallScenario)
					r.Post("/followups", leadH.CreateFollowUp)
					r.With(s.AuthMW.RequirePermission(rbac.PermLeadsQualify)).Post("/qualify", leadH.Qualify)
					r.With(s.AuthMW.RequirePermission(rbac.PermLeadsDisqualify)).Post("/disqualify", leadH.Disqualify)
					r.With(s.AuthMW.RequirePermission(rbac.PermLeadsExport)).Post("/export/muninn", leadH.ExportToMuninn)
					r.With(s.AuthMW.RequirePermission(rbac.PermLeadsEdit)).Post("/tags", leadH.AddTag)
					r.With(s.AuthMW.RequirePermission(rbac.PermLeadsEdit)).Delete("/tags", leadH.RemoveTag)
					r.With(s.AuthMW.RequirePermission(rbac.PermLeadsEdit)).Post("/archive", leadH.Archive)
				})
			})

			r.Route("/interactions", func(r chi.Router) {
				r.Post("/{interactionID}/report", leadH.SubmitReport)
				r.Post("/{interactionID}/open", leadH.MarkOpened)
				r.With(s.AuthMW.RequirePermission(rbac.PermReportsOverride)).Post("/{interactionID}/void", leadH.VoidReport)
			})

			dashH := newDashboardHandler(s.Dashboard)
			r.With(s.AuthMW.RequirePermission(rbac.PermDashboardMine)).Get("/dashboard/me", dashH.Agent)
			r.With(s.AuthMW.RequirePermission(rbac.PermDashboardView)).Get("/dashboard", dashH.Manager)

			// Project management — the business context for leads & AI generation.
			projectH := newProjectHandler(&s)
			r.Route("/projects", func(r chi.Router) {
				r.With(s.AuthMW.RequirePermission(rbac.PermProjectsManage)).Post("/", projectH.Create)
				r.Get("/", projectH.List)
				r.Route("/{projectID}", func(r chi.Router) {
					r.Get("/", projectH.Get)
					r.With(s.AuthMW.RequirePermission(rbac.PermProjectsManage)).Put("/", projectH.Update)
					r.With(s.AuthMW.RequirePermission(rbac.PermProjectsManage)).Delete("/", projectH.Delete)
					r.With(s.AuthMW.RequirePermission(rbac.PermProjectsManage)).Post("/leads/{leadID}", projectH.AttachLead)
					r.With(s.AuthMW.RequirePermission(rbac.PermProjectsManage)).Delete("/leads/{leadID}", projectH.DetachLead)
				})
			})

			// Campaigns — the higher-level strategic grouping above Projects.
			campaignH := newCampaignHandler(&s)
			r.Route("/campaigns", func(r chi.Router) {
				r.With(s.AuthMW.RequirePermission(rbac.PermProjectsManage)).Post("/", campaignH.Create)
				r.Get("/", campaignH.List)
				r.Route("/{campaignID}", func(r chi.Router) {
					r.Get("/", campaignH.Get)
					r.With(s.AuthMW.RequirePermission(rbac.PermProjectsManage)).Put("/", campaignH.Update)
					r.With(s.AuthMW.RequirePermission(rbac.PermProjectsManage)).Delete("/", campaignH.Delete)
				})
			})

			r.With(s.AuthMW.RequirePermission(rbac.PermDashboardView)).Get("/dashboard/funnel", dashH.Funnel)
			r.With(s.AuthMW.RequirePermission(rbac.PermDashboardView)).Get("/dashboard/qualification-by-source", dashH.QualificationBySource)
			r.With(s.AuthMW.RequirePermission(rbac.PermDashboardView)).Get("/dashboard/agent-performance", dashH.AgentPerformance)
			r.With(s.AuthMW.RequirePermission(rbac.PermDashboardView)).Get("/dashboard/lead-trend", dashH.LeadTrend)
			r.With(s.AuthMW.RequirePermission(rbac.PermDashboardView)).Get("/dashboard/rejection-reasons", dashH.RejectionReasons)
			r.With(s.AuthMW.RequirePermission(rbac.PermDashboardView)).Get("/dashboard/leads-by-city", dashH.LeadsByCity)

			sysH := newSystemHandler(s.DB, s.Cache)
			r.Get("/jobs/pending", sysH.PendingJobs)
			r.Get("/notifications", sysH.Notifications)
			r.Post("/notifications/read", sysH.MarkNotificationsRead)

			dataH := newDataHandler(&s)
			reportsH := newReportsHandler(s.Analytics, s.Logger)
			r.Route("/reports", func(r chi.Router) {
				r.Get("/overview", reportsH.Overview)
				r.Get("/funnel", reportsH.Funnel)
				r.Get("/activity", reportsH.Activity)
				r.Get("/score-distribution", reportsH.ScoreDistribution)
				r.Get("/objections", reportsH.Objections)
				r.Get("/outcomes", reportsH.Outcomes)
				r.Get("/interest", reportsH.Interest)
				r.Get("/agents", reportsH.Agents)
				r.Get("/projects", reportsH.Projects)
				r.Get("/followups", reportsH.FollowUps)
				r.Get("/decision-makers", reportsH.DecisionMakers)
			})
			r.Get("/documents", dataH.DocumentsList)
			r.Get("/documents/{docID}/candidates", dataH.DocumentCandidates)
			r.Get("/duplicates", dataH.DuplicatesList)
			r.With(s.AuthMW.RequirePermission(rbac.PermLeadsMerge)).Post("/duplicates/{pairID}/resolve", dataH.DuplicateResolve)
			r.With(s.AuthMW.RequirePermission(rbac.PermDashboardView)).Get("/reports", dataH.ReportsList)
			r.With(s.AuthMW.RequirePermission(rbac.PermDashboardView)).Get("/reports/{reportID}", dataH.ReportGet)
			r.With(s.AuthMW.RequirePermission(rbac.PermReportsOverride)).Post("/reports/{reportID}/override", dataH.ReportOverride)
			r.Get("/followups", dataH.FollowUpsList)
			r.Post("/followups", leadH.FollowUpCreateGlobal)
			r.Get("/followups/{followupID}", leadH.FollowUpGet)
			r.Patch("/followups/{followupID}", leadH.FollowUpUpdate)
			r.Post("/followups/{followupID}/complete", dataH.FollowUpComplete)
			r.Post("/followups/{followupID}/reopen", leadH.FollowUpReopen)
			r.Post("/followups/{followupID}/reschedule", leadH.FollowUpReschedule)
			r.Delete("/followups/{followupID}", leadH.FollowUpCancel)
			r.Get("/agents", dataH.AgentsList)
			r.Get("/agents/{userID}", dataH.AgentGet)
			r.Get("/search", dataH.GlobalSearch)
			r.With(s.AuthMW.RequirePermission(rbac.PermAuditView)).Get("/audit", dataH.AuditList)
			r.With(s.AuthMW.RequirePermission(rbac.PermLeadsHistory)).Get("/change-history", dataH.ChangeHistory)

			r.Route("/settings", func(r chi.Router) {
				r.Group(func(r chi.Router) {
					r.Use(s.AuthMW.RequirePermission(rbac.PermUsersManage))
					r.Get("/scoring", dataH.ScoringGet)
					r.Put("/scoring", dataH.ScoringUpdate)
					r.Get("/qualification", dataH.QualificationGet)
					r.Patch("/qualification", dataH.QualificationUpdate)
					r.Get("/llm", dataH.LLMGet)
					r.Patch("/llm", dataH.LLMUpdate)
					r.Post("/llm/test", dataH.LLMTest)
					r.Get("/muninn", dataH.MuninnGet)
					r.Patch("/muninn", dataH.MuninnUpdate)
					r.Get("/system", dataH.SystemGet)
					r.Get("/messaging", dataH.MessagingGet)
					r.Patch("/messaging", dataH.MessagingUpdate)
				})
			})
		})
	})

	return r
}
