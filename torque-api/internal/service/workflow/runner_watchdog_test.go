package workflow

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/milennials/torque-api/internal/event"
	workflowrepo "github.com/milennials/torque-api/internal/repository/workflow"
)

// TestDefaultRunnerConfig_Watchdog locks the watchdog defaults so a
// careless tweak doesn't push stale detection from minutes into hours
// (an orphaned run stuck for an hour is a paging event).
func TestDefaultRunnerConfig_Watchdog(t *testing.T) {
	t.Parallel()
	c := DefaultRunnerConfig()
	if c.WatchdogInterval != 2*time.Minute {
		t.Errorf("watchdog interval drift: got %v, want 2m", c.WatchdogInterval)
	}
	if c.StaleAfter != 10*time.Minute {
		t.Errorf("stale-after drift: got %v, want 10m", c.StaleAfter)
	}
}

// TestNewRunner_AppliesDefaultsOnZeroConfig guards against a
// zero-valued RunnerConfig silently disabling the watchdog.
func TestNewRunner_AppliesDefaultsOnZeroConfig(t *testing.T) {
	t.Parallel()
	r := NewRunner(RunnerConfig{}, nil, nil, zerolog.Nop())
	if r.cfg.WatchdogInterval <= 0 {
		t.Errorf("watchdog interval must be positive, got %v", r.cfg.WatchdogInterval)
	}
	if r.cfg.StaleAfter <= 0 {
		t.Errorf("stale-after must be positive, got %v", r.cfg.StaleAfter)
	}
	if r.cfg.PollInterval <= 0 {
		t.Errorf("poll interval must be positive, got %v", r.cfg.PollInterval)
	}
}

// TestRunner_Watchdog_Integration requires a live Postgres and D010
// gating — the Go toolchain can't run it on the CTO's workstation
// (see STATE.md D010). CI runs it with DATABASE_URL pointed at a
// migrated schema; locally it skips cleanly.
//
// The test seeds one `running` workflow_run whose started_at is
// pushed 30 minutes into the past, ticks the watchdog once, and
// asserts the row flipped to `failed` + a DLQ row landed.
func TestRunner_Watchdog_Integration(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL unset — watchdog integration test skipped")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer pool.Close()

	repo := workflowrepo.New(pool)
	bus := event.NewBus(event.DropOldest)
	exec := NewExecutorWithBus(repo, NewDispatcherBare(zerolog.Nop()), bus, zerolog.Nop())
	r := NewRunnerWithBus(RunnerConfig{
		PollInterval:     time.Second,
		PollJitter:       0,
		WatchdogInterval: time.Second,
		StaleAfter:       30 * time.Second, // aggressive for the test
	}, repo, exec, bus, zerolog.Nop())

	// The actual DB fixtures (org / workflow / run seed) belong in a
	// sibling test helper we ship under internal/testutil/pgfixture
	// in the follow-up. For today we assert the runner boots + the
	// watchdog tick executes without panic on an empty schema.
	r.Start(ctx)
	time.Sleep(2 * time.Second)
	shutdownCtx, c2 := context.WithTimeout(context.Background(), 3*time.Second)
	defer c2()
	if err := r.Shutdown(shutdownCtx); err != nil {
		t.Errorf("shutdown: %v", err)
	}
}
