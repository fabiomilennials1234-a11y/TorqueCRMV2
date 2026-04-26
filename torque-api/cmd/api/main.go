// Command torque-api is the Torque CRM HTTP entrypoint.
//
// Boot order:
//  1. Load config from env (fail fast on missing secrets).
//  2. Initialize structured logger.
//  3. Open pgx connection pool and ping.
//  4. Mount middleware stack and routes.
//  5. Start HTTP server.
//  6. Wait for SIGINT/SIGTERM, then drain with a bounded timeout.
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/milennials/torque-api/internal/config"
	"github.com/milennials/torque-api/internal/db"
	"github.com/milennials/torque-api/internal/event"
	agentshandler "github.com/milennials/torque-api/internal/handler/agents"
	analyticshandler "github.com/milennials/torque-api/internal/handler/analytics"
	authhandler "github.com/milennials/torque-api/internal/handler/auth"
	billinghandler "github.com/milennials/torque-api/internal/handler/billing"
	"github.com/milennials/torque-api/internal/handler/bootstrap"
	campaignshandler "github.com/milennials/torque-api/internal/handler/campaigns"
	confirmationshandler "github.com/milennials/torque-api/internal/handler/confirmations"
	"github.com/milennials/torque-api/internal/handler/health"
	inboxhandler "github.com/milennials/torque-api/internal/handler/inbox"
	integrationshandler "github.com/milennials/torque-api/internal/handler/integrations"
	leadwebhookhandler "github.com/milennials/torque-api/internal/handler/leadwebhook"
	leadshandler "github.com/milennials/torque-api/internal/handler/leads"
	masterhandler "github.com/milennials/torque-api/internal/handler/master"
	quotashandler "github.com/milennials/torque-api/internal/handler/quotas"
	meetingshandler "github.com/milennials/torque-api/internal/handler/meetings"
	membershandler "github.com/milennials/torque-api/internal/handler/members"
	onboardinghandler "github.com/milennials/torque-api/internal/handler/onboarding"
	"github.com/milennials/torque-api/internal/handler/openapi"
	operationshandler "github.com/milennials/torque-api/internal/handler/operations"
	performancehandler "github.com/milennials/torque-api/internal/handler/performance"
	pipeshandler "github.com/milennials/torque-api/internal/handler/pipes"
	preferenceshandler "github.com/milennials/torque-api/internal/handler/preferences"
	productshandler "github.com/milennials/torque-api/internal/handler/products"
	proposalshandler "github.com/milennials/torque-api/internal/handler/proposals"
	settingshandler "github.com/milennials/torque-api/internal/handler/settings"
	tasksahandler "github.com/milennials/torque-api/internal/handler/tasks"
	templateshandler "github.com/milennials/torque-api/internal/handler/templates"
	workflowshandler "github.com/milennials/torque-api/internal/handler/workflows"
	mw "github.com/milennials/torque-api/internal/httpx/middleware"
	"github.com/milennials/torque-api/internal/domain"
	"github.com/milennials/torque-api/internal/observability/sentry"
	agentrepo "github.com/milennials/torque-api/internal/repository/agent"
	analyticsrepo "github.com/milennials/torque-api/internal/repository/analytics"
	campaignrepo "github.com/milennials/torque-api/internal/repository/campaign"
	confirmationrepo "github.com/milennials/torque-api/internal/repository/confirmation"
	inboxrepo "github.com/milennials/torque-api/internal/repository/inbox"
	integrationrepo "github.com/milennials/torque-api/internal/repository/integration"
	leadwebhookrepo "github.com/milennials/torque-api/internal/repository/leadwebhook"
	metacacherepo "github.com/milennials/torque-api/internal/repository/metainsights"
	auditrepo "github.com/milennials/torque-api/internal/repository/audit"
	leadrepo "github.com/milennials/torque-api/internal/repository/lead"
	masterrepo "github.com/milennials/torque-api/internal/repository/master"
	meetingrepo "github.com/milennials/torque-api/internal/repository/meeting"
	memberrepo "github.com/milennials/torque-api/internal/repository/member"
	onboardingrepo "github.com/milennials/torque-api/internal/repository/onboarding"
	operationrepo "github.com/milennials/torque-api/internal/repository/operation"
	performancerepo "github.com/milennials/torque-api/internal/repository/performance"
	piperepo "github.com/milennials/torque-api/internal/repository/pipe"
	productrepo "github.com/milennials/torque-api/internal/repository/product"
	proposalrepo "github.com/milennials/torque-api/internal/repository/proposal"
	quotarepo "github.com/milennials/torque-api/internal/repository/quota"
	refreshrepo "github.com/milennials/torque-api/internal/repository/refresh"
	settingsrepo "github.com/milennials/torque-api/internal/repository/settings"
	subscriptionrepo "github.com/milennials/torque-api/internal/repository/subscription"
	templaterepo "github.com/milennials/torque-api/internal/repository/template"
	taskrepo "github.com/milennials/torque-api/internal/repository/task"
	userrepo "github.com/milennials/torque-api/internal/repository/user"
	workflowrepo "github.com/milennials/torque-api/internal/repository/workflow"
	"github.com/milennials/torque-api/internal/service/ai"
	aitrigger "github.com/milennials/torque-api/internal/service/ai/trigger"
	"github.com/milennials/torque-api/internal/service/billing"
	cryptosvc "github.com/milennials/torque-api/internal/service/crypto"
	"github.com/milennials/torque-api/internal/service/integration/gcal"
	"github.com/milennials/torque-api/internal/service/integration/meta"
	"github.com/milennials/torque-api/internal/service/integration/szchat"
	"github.com/milennials/torque-api/internal/service/integration/tinyerp"
	jwtsvc "github.com/milennials/torque-api/internal/service/jwt"
	knowledgesvc "github.com/milennials/torque-api/internal/service/knowledge"
	"github.com/milennials/torque-api/internal/service/permission"
	workflowsvc "github.com/milennials/torque-api/internal/service/workflow"
	"github.com/milennials/torque-api/internal/worker"
	"github.com/milennials/torque-api/internal/ws"
)

// version is stamped at build time via -ldflags. Defaults to "dev" in local runs.
var version = "dev"

func main() {
	if err := run(); err != nil {
		// Logger may not exist yet (config failure). Use stderr.
		_, _ = os.Stderr.WriteString("fatal: " + err.Error() + "\n")
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := newLogger(cfg)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Sentry is initialized early so panics during DB open still get captured.
	// Empty DSN is a no-op, so dev without SENTRY_DSN just works.
	sentryEnv := cfg.SentryEnvironment
	if sentryEnv == "" {
		sentryEnv = cfg.Env
	}
	if err := sentry.Init(sentry.Config{
		DSN:              cfg.SentryDSN,
		Environment:      sentryEnv,
		Release:          cfg.AppVersion,
		SampleRate:       cfg.SentrySampleRate,
		TracesSampleRate: cfg.SentryTracesRate,
	}); err != nil {
		logger.Warn().Err(err).Msg("sentry init failed — continuing without reporting")
	}
	defer sentry.Flush(2 * time.Second)

	pool, err := db.Open(ctx, cfg.DatabaseURL, logger)
	if err != nil {
		return err
	}
	defer pool.Close()

	rl := mw.NewRateLimiter(mw.RateLimitConfig{
		AnonRPS:           cfg.RateLimitAnonRPS,
		AnonBurst:         cfg.RateLimitAnonBurst,
		UserRPS:           cfg.RateLimitUserRPS,
		UserBurst:         cfg.RateLimitUserBurst,
		TrustForwardedFor: cfg.RateLimitTrustProxy,
	})
	defer rl.Close()

	// Event bus + WebSocket hub. The bus is the only way domain services
	// publish real-time patches; the hub is the only sink that reaches the
	// browser. Bridging is a single goroutine that drains the bus and hands
	// each event to hub.Broadcast.
	bus := event.NewBus(event.DropOldest)
	hub := ws.NewHub(ws.DefaultHubConfig(), logger)
	busSub, busUnsub := bus.Subscribe(256)
	defer busUnsub()
	go func() {
		for evt := range busSub {
			hub.Broadcast(evt)
		}
	}()

	// S52 — in-proc registry of live LLM streams so kill-switch +
	// agent disable can cancel them mid-turn. Shared between the
	// playground handler (registers on dial) and the agents handler
	// (cancels on flip).
	aiRegistry := ai.NewRegistry()

	// Worker pool. S04 ships the infrastructure; handlers per-kind are wired
	// by future sprints (leads import, bulk workflows, etc).
	operations := operationrepo.New(pool)
	workerPool := worker.New(worker.Config{
		Concurrency:  4,
		PollInterval: 2 * time.Second,
		PollJitter:   500 * time.Millisecond,
	}, operations, bus, logger, []worker.Handler{
		// Intentionally empty here — handlers are registered as features ship.
	})
	workerPool.Start(ctx)
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := workerPool.Shutdown(shutdownCtx); err != nil {
			logger.Warn().Err(err).Msg("worker shutdown timed out")
		}
	}()

	// S44/S45/S52 — workflow executor runner + event-bus subscriber.
	// Runner claims pending workflow_runs via SKIP LOCKED and walks the
	// DAG; subscriber listens for `lead.created` / `lead.stage_changed`
	// / `message.received` and fans out one enqueue per active workflow.
	//
	// S52 turns the 7 action handlers into real side-effects:
	//   - send_message wires to the messaging provider (nil today; the
	//     Evolution adapter is instantiated per-tenant at send time in
	//     a follow-up; handlers degrade to logged no-ops until then).
	//   - update_lead / create_task / call_agent plug into their repos.
	//   - http calls out with https-only + SSRF allowlist.
	//   - wait suspends the run via workflow_runs.next_retry_at.
	//
	// The executor + runner carry retry/DLQ + watchdog (see migration
	// 0028). Transient failures backoff-retry up to max_attempts; all
	// attempts (including retries) land rows in workflow_run_failures.
	// The watchdog reclaims `running` rows whose process died mid-step.
	wfRepo := workflowrepo.New(pool)
	dispatcher := workflowsvc.NewDispatcher(logger, workflowsvc.Deps{
		Pool:   pool,
		Bus:    bus,
		Leads:  leadrepo.New(pool),
		Tasks:  taskrepo.New(pool),
		Agents: agentrepo.New(pool),
		Inbox:  inboxrepo.New(pool),
		// Messaging: wired in a follow-up once the per-tenant
		// Evolution/SZ.Chat provider selector lands. Handlers already
		// log "wired:false" and pass through when nil.
		Messaging: nil,
	})
	executor := workflowsvc.NewExecutorWithBus(wfRepo, dispatcher, bus, logger)
	wfRunner := workflowsvc.NewRunnerWithBus(
		workflowsvc.DefaultRunnerConfig(), wfRepo, executor, bus, logger)
	wfRunner.Start(ctx)
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := wfRunner.Shutdown(shutdownCtx); err != nil {
			logger.Warn().Err(err).Msg("workflow runner shutdown timed out")
		}
	}()
	wfSub := workflowsvc.NewBusSubscriber(bus, wfRepo, logger)
	wfSub.Start(ctx)
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := wfSub.Shutdown(shutdownCtx); err != nil {
			logger.Warn().Err(err).Msg("workflow bus subscriber shutdown timed out")
		}
	}()

	// S52 — agent trigger dispatcher. Subscribes to the same bus and
	// translates message.received / conversation.created events into
	// AssignAgent calls when a tenant's active triggers match. Closes
	// the S40 gap where triggers had a matcher but no runtime.
	agentRepoForSub := agentrepo.New(pool)
	// S63 (D074-k): LeadResolver hidrata LeadFacts (Origin/Segment/UTMs/
	// Rating) a partir do conversation_id pre-match. Sem isso, triggers
	// com filtros em atributos do lead silenciosamente falham.
	triggerLeadResolver := &poolLeadResolver{pool: pool}
	triggerSub := aitrigger.New(bus, agentRepoForSub, logger).WithLeads(triggerLeadResolver)
	triggerSub.Start(ctx)
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := triggerSub.Shutdown(shutdownCtx); err != nil {
			logger.Warn().Err(err).Msg("agent trigger subscriber shutdown timed out")
		}
	}()

	router, err := newRouter(cfg, logger, pool, rl, hub, operations, bus, aiRegistry)
	if err != nil {
		return err
	}


	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadTimeout:       cfg.ReadTimeout,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
	}

	// Serve in a goroutine so we can wait on a signal in the main routine.
	srvErr := make(chan error, 1)
	go func() {
		logger.Info().
			Str("addr", cfg.HTTPAddr).
			Str("env", cfg.Env).
			Str("version", version).
			Msg("http server starting")
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			srvErr <- err
			return
		}
		srvErr <- nil
	}()

	// Wait for shutdown signal or an unrecoverable serve error.
	select {
	case <-ctx.Done():
		logger.Info().Msg("shutdown signal received")
	case err := <-srvErr:
		if err != nil {
			return err
		}
	}

	// Graceful shutdown — bounded to cfg.ShutdownTimeout so a hung connection
	// does not block pod rolling forever.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("graceful shutdown failed")
		return err
	}
	logger.Info().Msg("shutdown complete")
	return nil
}

func newLogger(cfg config.Config) zerolog.Logger {
	level, _ := zerolog.ParseLevel(cfg.LogLevel)
	zerolog.SetGlobalLevel(level)
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs

	// In dev, emit human-friendly console output. In prod/staging, emit JSON
	// so log aggregators (Loki, ELK, CloudWatch) can parse fields.
	if cfg.Env == "dev" {
		return zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).
			With().Timestamp().Str("service", "torque-api").Logger()
	}
	return zerolog.New(os.Stdout).With().Timestamp().Str("service", "torque-api").Logger()
}

func newRouter(
	cfg config.Config,
	logger zerolog.Logger,
	pool *db.Pool,
	rl *mw.RateLimiter,
	hub *ws.Hub,
	operations *operationrepo.Repository,
	bus *event.Bus,
	aiRegistry *ai.Registry,
) (http.Handler, error) {
	r := chi.NewRouter()

	// Middleware order is deliberate:
	//  RequestID → AccessLog           — so every access line carries the id.
	//  sentry.Recovery                 — catches panics + reports to Sentry.
	//  SecurityHeaders                 — set early; cheap; applies to every response.
	//  CORS                            — must run before StripOrganizationID so preflights pass.
	//  StripOrganizationID             — enforces multi-tenancy invariant on mutations.
	//  Authenticator (inside /api/v1)  — attaches session when cookie is valid.
	//  RateLimit (inside /api/v1)      — uses session when present, else client IP.
	r.Use(mw.RequestID)
	r.Use(mw.AccessLog(logger))
	r.Use(sentry.Recovery(logger))

	// HSTS is disabled when the process runs over plain http in dev. In
	// staging/prod the browser must never fall back to http.
	r.Use(mw.SecurityHeadersWith(mw.SecurityHeadersConfig{
		EnableHSTS:  cfg.Env != "dev",
		HSTSMaxAge:  2 * 365 * 24 * time.Hour,
		HSTSPreload: cfg.Env == "prod",
	}))

	if len(cfg.CORSOrigins) > 0 {
		r.Use(cors.Handler(cors.Options{
			AllowedOrigins:   cfg.CORSOrigins,
			AllowedMethods:   []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", mw.CSRFHeaderName, mw.RequestIDHeader},
			ExposedHeaders:   []string{mw.RequestIDHeader},
			AllowCredentials: true,
			MaxAge:           300,
		}))
	}

	r.Use(mw.StripOrganizationID)

	// --- Wiring --------------------------------------------------------
	jsvc, err := jwtsvc.New(cfg.JWTSecret, cfg.AccessTTL)
	if err != nil {
		// Config.Load already guarantees length; this is defensive.
		panic("jwt service: " + err.Error())
	}
	users := userrepo.New(pool)
	refresh := refreshrepo.New(pool)
	permResolver := permission.New(users)

	// --- S49 / Fase F.1 — integrations wiring ------------------------
	cipher, err := cryptosvc.New(cfg.IntegrationEncKey)
	if err != nil {
		return nil, fmt.Errorf("integration cipher: %w", err)
	}
	credStore := integrationrepo.NewStore(pool, cipher)

	// Google Calendar adapter — built unconditionally so Disconnect /
	// callback error paths work even when OAuth is not configured. The
	// /connect endpoint short-circuits with 503 when ClientID is empty.
	gcalProvider := gcal.New(gcal.Config{
		ClientID:     cfg.GoogleOAuthClientID,
		ClientSecret: cfg.GoogleOAuthClientSecret,
		RedirectURL:  cfg.GoogleOAuthRedirectURL,
	}, credStore, nil, nil)

	// TinyERP adapter — per-tenant API keys live in the credential
	// store so no global key is needed at boot.
	tinyProvider := tinyerp.New(tinyerp.Config{BaseURL: cfg.TinyERPBaseURL},
		credStore, nil, nil)

	// --- S50 / Fase F.2 — Meta + SZ.Chat + lead webhook wiring -------
	metaProvider := meta.New(meta.Config{
		BaseURL:   cfg.MetaGraphBaseURL,
		AppSecret: cfg.MetaAppSecret,
	}, credStore, nil, nil)

	szchatProvider := szchat.New(szchat.Config{BaseURL: cfg.SZChatBaseURL},
		credStore, nil, nil)

	metaCache := metacacherepo.New(pool)
	leadWebhookEvents := leadwebhookrepo.New(pool)

	productRepoForSync := productrepo.New(pool)
	productSink := productSyncSink{repo: productRepoForSync}

	integrationsHandler := integrationshandler.New(integrationshandler.Options{
		Store:        credStore,
		GCal:         gcalProvider,
		Tiny:         tinyProvider,
		Meta:         metaProvider,
		SZChat:       szchatProvider,
		MetaCache:    metaCache,
		ProductSink:  productSink,
		StateSecret:  cfg.IntegrationStateSecret,
		Logger:       logger,
		FrontendBase: "", // same-origin redirects; configurable later
	})

	auth := authhandler.New(authhandler.Options{
		Pool:         pool,
		Users:        users,
		Refresh:      refresh,
		JWT:          jsvc,
		Permission:   permResolver,
		RefreshTTL:   cfg.RefreshTTL,
		CookieSecure: cfg.CookieSecure,
		CookieDomain: cfg.CookieDomain,
	})

	// --- Public routes -------------------------------------------------
	// Probes intentionally unauthenticated — orchestrators must be able to
	// poll without a token.
	healthHandler := health.New(pool, version)
	healthHandler.Routes(r)

	bootstrap.New(bootstrap.Config{
		AppVersion:   cfg.AppVersion,
		Env:          cfg.Env,
		WSURL:        cfg.WSURL,
		SentryDSN:    cfg.SentryPublicDSN, // PUBLIC DSN only — never the server one
		FeatureFlags: cfg.FeatureFlags,
	}).Routes(r)

	// OpenAPI spec served from disk (api/openapi.yaml). Fails fast if missing.
	specHandler, err := openapi.New("api/openapi.yaml")
	if err != nil {
		return nil, err
	}
	specHandler.Routes(r)

	// --- Public webhooks -----------------------------------------------
	// Sit OUTSIDE /api/v1 so they don't require a session; auth is
	// via shared secret headers (billing) or HMAC signature (lead).
	subRepo := subscriptionrepo.New(pool)
	leadRepoForWebhook := leadrepo.New(pool)
	// S51 — quota repo powers RequireQuota middleware + GET /quotas
	// observability + billing webhook seed of plan_quotas→org_quotas.
	quotaRepo := quotarepo.New(pool)
	r.Route("/webhooks", func(wh chi.Router) {
		billinghandler.NewWebhook(subRepo, bus, cfg.BillingWebhookSecret).
			WithQuotaRepo(quotaRepo).Routes(wh)
		leadwebhookhandler.New(leadwebhookhandler.Options{
			Leads:  leadRepoForWebhook,
			Events: leadWebhookEvents,
			Bus:    bus,
			Secret: cfg.LeadWebhookSecret,
			Logger: logger,
		}).Routes(wh)
	})

	// --- API v1: auth entry points (pre-session) -----------------------
	r.Route("/api/v1", func(v1 chi.Router) {
		// Authenticator is transparent — attaches session when cookie is
		// valid, does not 401 when absent. Login/refresh/logout all need this.
		v1.Use(mw.Authenticator(jsvc))
		// Rate-limit sits AFTER Authenticator so the key uses user_id when
		// present and falls back to IP for anonymous traffic.
		v1.Use(rl.Middleware)

		auth.Routes(v1) // /auth/login, /auth/refresh, /auth/logout

		// Authenticated subrouter: gated by RequireAuth + CSRF on mutations.
		v1.Group(func(priv chi.Router) {
			priv.Use(mw.RequireAuth)
			priv.Use(mw.CSRF)

			auth.MeRoute(priv) // GET /auth/me

			// Routes that need tenant scope (org_id in context).
			priv.Group(func(t chi.Router) {
				t.Use(mw.TenantScope)

				// Member-accessible surfaces. Tenant isolation + per-field
				// guards inside each repo are enough; RBAC can tighten per
				// endpoint in future sprints once member view keys settle.
				preferenceshandler.New(users).Routes(t)
				operationshandler.New(operations).Routes(t)
				// S51 — leads POST enforces the tenant quota via
				// RequireQuota; other verbs stay unrestricted. Decrement
				// on soft-delete lives in the handler.
				leadshandler.New(leadrepo.New(pool), bus).WithQuota(quotaRepo).Routes(t)
				pipeshandler.New(piperepo.New(pool), bus).Routes(t)
				// S50 — confirmations now carry the GCal push on
				// /confirm, matching the pattern set by meetings in
				// S49.
				confirmationshandler.New(confirmationrepo.New(pool), bus).
					WithIntegrations(gcalProvider, credStore, logger).Routes(t)
				inboxhandler.New(inboxrepo.New(pool), bus).
					WithAudit(auditrepo.New(pool)).Routes(t)
				tasksahandler.New(taskrepo.New(pool), bus).Routes(t)
				analyticshandler.New(analyticsrepo.New(pool)).Routes(t)

				// F10/F11 read surfaces are member-accessible; mutations sit
				// in the admin group below.
				memberRepo := memberrepo.New(pool)
				productRepo := productrepo.New(pool)
				membershandler.NewRead(memberRepo).Routes(t)
				productshandler.NewRead(productRepo).Routes(t)

				// S47 — F09 Performance: reads are member-accessible,
				// writes land on the admin group.
				performanceHandler := performancehandler.New(performancerepo.New(pool), bus)
				performanceHandler.Routes(t)

				// S48 — F13 Agenda (meetings). CRUD member-accessible.
				// S49 — attach GCal + credential store so the handler
				// can push/cancel events asynchronously.
				meetingshandler.New(meetingrepo.New(pool), bus).
					WithIntegrations(gcalProvider, credStore, logger).Routes(t)

				// S49 — /integrations list (member-accessible).
				integrationsHandler.MemberRoutes(t)

				// S51 — /quotas observability (member-accessible read;
				// master-only write lands in the master subgroup below).
				quotasHandler := quotashandler.New(quotaRepo)
				quotasHandler.MemberRoutes(t)
				onboardinghandler.New(onboardingrepo.New(pool)).Routes(t)
				billinghandler.NewRead(subRepo).Routes(t)
				settingsRepo := settingsrepo.New(pool)
				settingshandler.NewRead(settingsRepo).Routes(t)
				settingshandler.NewMe(settingsRepo).Routes(t)

				// S35: message templates (ComposerBar do Inbox).
				templateRepo := templaterepo.New(pool)
				templateshandler.NewRead(templateRepo).Routes(t)

				// --- Admin-only surfaces (Copilot kill-switch + KB +
				// proposal money flow). Any authenticated member could
				// previously derail these; sprint/remediation gates them
				// to admin+master, matching the catalog rows seeded in
				// migration 0010. RequireRole also lets master through.
				t.Group(func(admin chi.Router) {
					admin.Use(mw.RequireRole(domain.RoleAdmin))
					// S37 — F06 Copilot: base handler + optional LLM provider
					// for the /playground/message SSE stream + PATCH /agents/:id.
					// Empty OPENROUTER_API_KEY leaves provider=nil and the
					// playground endpoint returns 503 PROVIDER_UNAVAILABLE.
					agentRepo := agentrepo.New(pool)
					// S52 — registry is shared so kill_switch + disable
					// cancel in-flight streams. WithQuota lights up the
					// RequireQuota gate on POST /agents (resource=agents
					// slot count) + IncrementUsage on create /
					// decrement on disable. Same *quotarepo.Repository
					// instance is reused by PlaygroundHandler for the
					// variable-cost `ai_tokens` meter below.
					agentBase := agentshandler.New(agentRepo, bus).
						WithRegistry(aiRegistry).
						WithQuota(quotaRepo)
					var aiProvider ai.Provider
					if cfg.OpenRouterAPIKey != "" {
						p, err := ai.NewOpenRouter(ai.Config{
							BaseURL: cfg.OpenRouterBaseURL,
							APIKey:  cfg.OpenRouterAPIKey,
							Referer: cfg.OpenRouterReferer,
							Title:   cfg.OpenRouterTitle,
						})
						if err != nil {
							logger.Warn().Err(err).Msg("openrouter init failed — playground disabled")
						} else {
							aiProvider = p
						}
					}
					// S39 — embedder selection. Gemini is the production
					// embedder; mock is the test + empty-key fallback so
					// ingest + retrieval don't 500 in dev. Prod MUST set
					// GEMINI_API_KEY or the retrieval quality collapses to
					// "random chunk of matching hash".
					var embedder ai.Embedder
					if cfg.GeminiAPIKey != "" {
						em, err := ai.NewGeminiEmbedder(ai.GeminiConfig{
							BaseURL: cfg.GeminiBaseURL,
							APIKey:  cfg.GeminiAPIKey,
							Model:   cfg.GeminiEmbeddingModel,
						})
						if err != nil {
							logger.Warn().Err(err).Msg("gemini embedder init failed — falling back to mock")
							embedder = ai.NewMockEmbedder()
						} else {
							embedder = em
						}
					} else {
						logger.Warn().Msg("GEMINI_API_KEY empty — using MockEmbedder (retrieval quality is meaningless in prod)")
						embedder = ai.NewMockEmbedder()
					}
					ingestSvc := knowledgesvc.New(agentRepo, embedder, logger)
					agentBase = agentBase.WithIngest(ingestSvc)
					// S41 — TTS adapter selection. ElevenLabs when key
					// is set; otherwise MockTTS so the preview endpoint
					// stays functional in dev. A missing key in prod
					// should fail loud — the config validation layer
					// is the right place to enforce that when we're
					// ready to promote TTS to required.
					var tts ai.TTS
					if cfg.ElevenLabsAPIKey != "" {
						t, err := ai.NewElevenLabsTTS(ai.ElevenLabsConfig{
							BaseURL: cfg.ElevenLabsBaseURL,
							APIKey:  cfg.ElevenLabsAPIKey,
							ModelID: cfg.ElevenLabsModelID,
						})
						if err != nil {
							logger.Warn().Err(err).Msg("elevenlabs init failed — falling back to mock tts")
							tts = ai.NewMockTTS()
						} else {
							tts = t
						}
					} else {
						logger.Warn().Msg("ELEVENLABS_API_KEY empty — using MockTTS (synthesis quality is fake)")
						tts = ai.NewMockTTS()
					}
					// S52 — playground carries budget middleware (pre-stream
					// 402 at cap) + WithQuota (post-stream IncrementUsage)
					// + WithRegistry (register stream cancel) + WithLogger
					// (ai.pii_scrub + quota_increment fields). The gate
					// is applied per-route via Routes(r, aiBudget, ttsBudget)
					// so only the paid endpoints 402 — PATCH /agents/:id
					// stays ungated.
					agentshandler.NewPlayground(agentBase, aiProvider, embedder, tts).
						WithQuota(quotaRepo).
						WithRegistry(aiRegistry).
						WithLogger(logger).
						Routes(admin, mw.RequireAITokenBudget(quotaRepo), mw.RequireTTSBudget(quotaRepo))
					proposalshandler.New(proposalrepo.New(pool), bus).Routes(admin)
					// S52 — workflows + team_members carry the same RequireQuota
					// drop-in as leads (S51). Builder pattern keeps nil-quota
					// call sites working for fixture tests.
					workflowshandler.New(workflowrepo.New(pool), bus).WithQuota(quotaRepo).Routes(admin)
					campaignshandler.New(campaignrepo.New(pool), bus).Routes(admin)
					membershandler.NewAdmin(memberRepo, bus).WithQuota(quotaRepo).Routes(admin)
					productshandler.NewAdmin(productRepo, bus).Routes(admin)
					pipeshandler.NewAdmin(piperepo.New(pool), bus).Routes(admin)
					performanceHandler.AdminRoutes(admin)

					// Billing checkout/cancel (admin-only). Provider is
					// pluggable; S24 wired the mock, S51 activates
					// Asaas when ASAAS_API_KEY is set (dual-review
					// tripwire — empty key falls back to mock even
					// when BILLING_PROVIDER=asaas).
					var provider billing.Provider
					switch cfg.BillingProvider {
					case "asaas":
						asaasProv, aerr := billing.NewAsaas(billing.AsaasConfig{
							APIKey:  cfg.AsaasAPIKey,
							BaseURL: cfg.AsaasBaseURL,
						}, nil)
						if aerr != nil {
							logger.Warn().Err(aerr).Msg("asaas provider not configured; falling back to mock")
							provider = billing.NewMock()
						} else {
							logger.Info().Str("base_url", cfg.AsaasBaseURL).
								Msg("asaas provider active")
							provider = asaasProv
						}
					default:
						provider = billing.NewMock()
					}
					billinghandler.NewAdmin(subRepo, provider, bus).Routes(admin)
					settingshandler.NewAdmin(settingsRepo).Routes(admin)
					templateshandler.NewAdmin(templateRepo).Routes(admin)

					// S49 — /integrations admin surfaces (connect,
					// disconnect, push-order). OAuth callback mounted
					// separately below because it cannot carry a CSRF
					// token on the full-page redirect from Google.
					integrationsHandler.AdminRoutes(admin)
				})

				// --- Master-only surfaces (cross-org) -----------------
				t.Group(func(mst chi.Router) {
					mst.Use(mw.RequireMaster)
					masterhandler.New(masterhandler.Options{
						Repo:         masterrepo.New(pool),
						Users:        users,
						Audit:        auditrepo.New(pool),
						JWT:          jsvc,
						CookieSecure: cfg.CookieSecure,
						CookieDomain: cfg.CookieDomain,
					}).Routes(mst)
					// S51 — PATCH /quotas/:resource (master-only admin
					// adjustment + purchased_addons override).
					quotasHandler.MasterRoutes(mst)
				})
			})

			// WebSocket upgrade — authenticated + session carries org_id.
			// Not under TenantScope because the hub reads org from the session
			// directly and does not need the chi-level scope.
			wsHandler := ws.NewHandler(hub, logger, cfg.CORSOrigins)
			priv.Handle("/ws", wsHandler)
		})

		// --- S49 OAuth callback (CSRF-exempt) -------------------------
		// The browser arrives here via a full-page 302 from Google and
		// cannot carry the X-CSRF-Token header a double-submit enforcer
		// requires. The state HMAC (minted on /connect, verified here)
		// fills the same anti-CSRF role with a stronger guarantee: it
		// binds the flow to the originating org and expires after
		// 10 minutes. Auth + TenantScope still run — only CSRF is
		// skipped.
		v1.Group(func(cb chi.Router) {
			cb.Use(mw.RequireAuth)
			cb.Use(mw.TenantScope)
			integrationsHandler.CallbackRoute(cb)
		})
	})

	return r, nil
}

// productSyncSink adapts *productrepo.Repository to the narrow
// tinyerp.SyncProductsSink interface. Living here keeps the provider
// package free of a direct repo import.
type productSyncSink struct {
	repo *productrepo.Repository
}

func (s productSyncSink) UpsertBySKU(
	ctx context.Context, orgID uuid.UUID,
	name, sku string, description *string, priceCents int64, currency string,
) (bool, error) {
	_, inserted, err := s.repo.UpsertBySKU(ctx, orgID, productrepo.UpsertBySKUInput{
		Name: name, SKU: sku, Description: description,
		PriceCents: priceCents, Currency: currency,
	})
	return inserted, err
}

// poolLeadResolver implementa aitrigger.LeadResolver via SQL direto.
// S63 (D074-k): JOIN conversations -> leads na ida pra evitar 2
// roundtrips. Mantemos a query inline aqui (cmd/api e o composition
// root) ao inves de poluir leadrepo com um metodo cross-aggregate.
type poolLeadResolver struct {
	pool *pgxpool.Pool
}

func (r *poolLeadResolver) LeadFactsByConversation(
	ctx context.Context, orgID, conversationID uuid.UUID,
) (aitrigger.LeadAttrs, error) {
	const q = `
		SELECT l.origin, l.segment, l.utm_source, l.utm_medium, l.utm_campaign, l.rating
		FROM conversations c
		INNER JOIN leads l ON l.id = c.lead_id AND l.organization_id = c.organization_id
		WHERE c.organization_id = $1 AND c.id = $2 AND l.deleted_at IS NULL
	`
	var attrs aitrigger.LeadAttrs
	if err := r.pool.QueryRow(ctx, q, orgID, conversationID).Scan(
		&attrs.Origin, &attrs.Segment, &attrs.UTMSource, &attrs.UTMMedium, &attrs.UTMCampaign, &attrs.Rating,
	); err != nil {
		return aitrigger.LeadAttrs{}, err
	}
	return attrs, nil
}
