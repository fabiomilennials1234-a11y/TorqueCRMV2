// Package db exposes a pgx connection pool pre-tuned for Torque.
//
// Rationale: pgx native pool, not database/sql, because we want statement
// pipelining, typed scanning, and no prepared-statement cache fights.
package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// Pool is a thin alias over pgxpool.Pool so call sites stay import-clean.
type Pool = pgxpool.Pool

// Open creates and pings a pgx pool.
//
// Pool sizing defaults:
//   - MaxConns: 20 (per instance; scale horizontally)
//   - MinConns: 2  (keep connections warm; fast cold-start for health probes)
//   - MaxConnLifetime: 1h   (rotate connections to keep listeners fresh)
//   - MaxConnIdleTime: 10m  (release unused connections back to Postgres)
//   - HealthCheckPeriod: 30s
//
// These are defensive defaults. Tune via PGX_POOL_* envs later when load tests
// justify it; resist tuning without data.
func Open(ctx context.Context, dsn string, logger zerolog.Logger) (*Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}

	cfg.MaxConns = 20
	cfg.MinConns = 2
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnIdleTime = 10 * time.Minute
	cfg.HealthCheckPeriod = 30 * time.Second
	cfg.ConnConfig.RuntimeParams["application_name"] = "torque-api"

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("open pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}

	logger.Info().
		Int32("max_conns", cfg.MaxConns).
		Int32("min_conns", cfg.MinConns).
		Msg("db pool opened")

	return pool, nil
}
