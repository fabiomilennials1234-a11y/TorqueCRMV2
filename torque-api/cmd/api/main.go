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
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
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
	leadshandler "github.com/milennials/torque-api/internal/handler/leads"
	masterhandler "github.com/milennials/torque-api/internal/handler/master"
	membershandler "github.com/milennials/torque-api/internal/handler/members"
	onboardinghandler "github.com/milennials/torque-api/internal/handler/onboarding"
	"github.com/milennials/torque-api/internal/handler/openapi"
	operationshandler "github.com/milennials/torque-api/internal/handler/operations"
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
	auditrepo "github.com/milennials/torque-api/internal/repository/audit"
	leadrepo "github.com/milennials/torque-api/internal/repository/lead"
	masterrepo "github.com/milennials/torque-api/internal/repository/master"
	memberrepo "github.com/milennials/torque-api/internal/repository/member"
	onboardingrepo "github.com/milennials/torque-api/internal/repository/onboarding"
	operationrepo "github.com/milennials/torque-api/internal/repository/operation"
	piperepo "github.com/milennials/torque-api/internal/repository/pipe"
	productrepo "github.com/milennials/torque-api/internal/repository/product"
	proposalrepo "github.com/milennials/torque-api/internal/repository/proposal"
	refreshrepo "github.com/milennials/torque-api/internal/repository/refresh"
	settingsrepo "github.com/milennials/torque-api/internal/repository/settings"
	subscriptionrepo "github.com/milennials/torque-api/internal/repository/subscription"
	templaterepo "github.com/milennials/torque-api/internal/repository/template"
	taskrepo "github.com/milennials/torque-api/internal/repository/task"
	userrepo "github.com/milennials/torque-api/internal/repository/user"
	workflowrepo "github.com/milennials/torque-api/internal/repository/workflow"
	"github.com/milennials/torque-api/internal/service/ai"
	"github.com/milennials/torque-api/internal/service/billing"
	jwtsvc "github.com/milennials/torque-api/internal/service/jwt"
	knowledgesvc "github.com/milennials/torque-api/internal/service/knowledge"
	"github.com/milennials/torque-api/internal/service/permission"
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

	router, err := newRouter(cfg, logger, pool, rl, hub, operations, bus)
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

	// --- Public billing webhook ----------------------------------------
	// Sits OUTSIDE /api/v1 so it doesn't require a session; auth is via
	// the X-Torque-Billing-Secret header.
	subRepo := subscriptionrepo.New(pool)
	r.Route("/webhooks", func(wh chi.Router) {
		billinghandler.NewWebhook(subRepo, bus, cfg.BillingWebhookSecret).Routes(wh)
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
				leadshandler.New(leadrepo.New(pool), bus).Routes(t)
				pipeshandler.New(piperepo.New(pool), bus).Routes(t)
				confirmationshandler.New(confirmationrepo.New(pool), bus).Routes(t)
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
					agentBase := agentshandler.New(agentRepo, bus)
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
					agentshandler.NewPlayground(agentBase, aiProvider, embedder).Routes(admin)
					proposalshandler.New(proposalrepo.New(pool), bus).Routes(admin)
					workflowshandler.New(workflowrepo.New(pool), bus).Routes(admin)
					campaignshandler.New(campaignrepo.New(pool), bus).Routes(admin)
					membershandler.NewAdmin(memberRepo, bus).Routes(admin)
					productshandler.NewAdmin(productRepo, bus).Routes(admin)
					pipeshandler.NewAdmin(piperepo.New(pool), bus).Routes(admin)

					// Billing checkout/cancel (admin-only). Provider is
					// pluggable; S24 wires the mock, Asaas lands after
					// credentials + dual review.
					var provider billing.Provider
					switch cfg.BillingProvider {
					case "asaas":
						// Placeholder — real Asaas client lands in a
						// follow-up sprint gated by credentials.
						logger.Warn().Msg("asaas provider not yet implemented; falling back to mock")
						provider = billing.NewMock()
					default:
						provider = billing.NewMock()
					}
					billinghandler.NewAdmin(subRepo, provider, bus).Routes(admin)
					settingshandler.NewAdmin(settingsRepo).Routes(admin)
					templateshandler.NewAdmin(templateRepo).Routes(admin)
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
				})
			})

			// WebSocket upgrade — authenticated + session carries org_id.
			// Not under TenantScope because the hub reads org from the session
			// directly and does not need the chi-level scope.
			wsHandler := ws.NewHandler(hub, logger, cfg.CORSOrigins)
			priv.Handle("/ws", wsHandler)
		})
	})

	return r, nil
}
