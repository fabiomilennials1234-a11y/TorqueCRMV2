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
	authhandler "github.com/milennials/torque-api/internal/handler/auth"
	"github.com/milennials/torque-api/internal/handler/bootstrap"
	confirmationshandler "github.com/milennials/torque-api/internal/handler/confirmations"
	"github.com/milennials/torque-api/internal/handler/health"
	inboxhandler "github.com/milennials/torque-api/internal/handler/inbox"
	leadshandler "github.com/milennials/torque-api/internal/handler/leads"
	"github.com/milennials/torque-api/internal/handler/openapi"
	operationshandler "github.com/milennials/torque-api/internal/handler/operations"
	pipeshandler "github.com/milennials/torque-api/internal/handler/pipes"
	preferenceshandler "github.com/milennials/torque-api/internal/handler/preferences"
	proposalshandler "github.com/milennials/torque-api/internal/handler/proposals"
	mw "github.com/milennials/torque-api/internal/httpx/middleware"
	"github.com/milennials/torque-api/internal/observability/sentry"
	confirmationrepo "github.com/milennials/torque-api/internal/repository/confirmation"
	inboxrepo "github.com/milennials/torque-api/internal/repository/inbox"
	leadrepo "github.com/milennials/torque-api/internal/repository/lead"
	operationrepo "github.com/milennials/torque-api/internal/repository/operation"
	piperepo "github.com/milennials/torque-api/internal/repository/pipe"
	proposalrepo "github.com/milennials/torque-api/internal/repository/proposal"
	refreshrepo "github.com/milennials/torque-api/internal/repository/refresh"
	userrepo "github.com/milennials/torque-api/internal/repository/user"
	jwtsvc "github.com/milennials/torque-api/internal/service/jwt"
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
				preferenceshandler.New(users).Routes(t)
				operationshandler.New(operations).Routes(t)
				leadshandler.New(leadrepo.New(pool), bus).Routes(t)
				pipeshandler.New(piperepo.New(pool), bus).Routes(t)
				confirmationshandler.New(confirmationrepo.New(pool), bus).Routes(t)
				proposalshandler.New(proposalrepo.New(pool), bus).Routes(t)
				inboxhandler.New(inboxrepo.New(pool), bus).Routes(t)
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
