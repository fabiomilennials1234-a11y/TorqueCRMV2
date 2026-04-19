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
	authhandler "github.com/milennials/torque-api/internal/handler/auth"
	"github.com/milennials/torque-api/internal/handler/bootstrap"
	"github.com/milennials/torque-api/internal/handler/health"
	preferenceshandler "github.com/milennials/torque-api/internal/handler/preferences"
	mw "github.com/milennials/torque-api/internal/httpx/middleware"
	refreshrepo "github.com/milennials/torque-api/internal/repository/refresh"
	userrepo "github.com/milennials/torque-api/internal/repository/user"
	jwtsvc "github.com/milennials/torque-api/internal/service/jwt"
	"github.com/milennials/torque-api/internal/service/permission"
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

	pool, err := db.Open(ctx, cfg.DatabaseURL, logger)
	if err != nil {
		return err
	}
	defer pool.Close()

	router := newRouter(cfg, logger, pool)

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

func newRouter(cfg config.Config, logger zerolog.Logger, pool *db.Pool) http.Handler {
	r := chi.NewRouter()

	// Middleware order is deliberate:
	//  RequestID → AccessLog  — so every access line carries the id.
	//  Recover                — catches panics from everything below.
	//  SecurityHeaders        — set early; cheap; applies to every response.
	//  CORS                   — must run before StripOrganizationID so preflights pass.
	//  StripOrganizationID    — enforces multi-tenancy invariant on mutations.
	r.Use(mw.RequestID)
	r.Use(mw.AccessLog(logger))
	r.Use(mw.Recover(logger))
	r.Use(mw.SecurityHeaders)

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
		SentryDSN:    cfg.SentryDSN,
		FeatureFlags: map[string]bool{},
	}).Routes(r)

	// --- API v1: auth entry points (pre-session) -----------------------
	r.Route("/api/v1", func(v1 chi.Router) {
		// Authenticator is transparent — attaches session when cookie is
		// valid, does not 401 when absent. Login/refresh/logout all need this.
		v1.Use(mw.Authenticator(jsvc))

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
				// Feature handlers land here in S04+ (leads, pipes, etc).
			})
		})
	})

	return r
}
