// Package repository holds tests that prove database-level invariants for the
// tasks table (ADR-007). These are integration tests — they require a real
// PostgreSQL with the full migration set applied.
//
// Run:
//
//	DATABASE_URL="postgres://torque:torque@localhost:5432/torque?sslmode=disable" \
//	  go test -race -run TestTasksConcurrentInProgressSingleton ./internal/repository/...
//
// Or via the Makefile:
//
//	make test-integration
//
// The test is gated by DATABASE_URL and by `-short`: plain `go test ./...`
// skips it so the default workflow stays fast.
package repository_test

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// TestTasksConcurrentInProgressSingleton proves the invariant
// "at most one task in_progress per (organization_id, assigned_to)" by racing
// many concurrent updates and asserting that exactly one row ends in
// in_progress — Postgres must serialize the rest via the partial unique index
// `uq_tasks_one_in_progress_per_assignee`.
//
// This is the database's last line of defense: even if the Go repository has
// a race bug, the partial unique index must reject the duplicate.
func TestTasksConcurrentInProgressSingleton(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err, "open pool")
	defer pool.Close()

	// --- Arrange fixtures --------------------------------------------------
	// Each run is isolated by its own organization + user + team_member so the
	// test is safe to run repeatedly against the same database without reset.
	orgID := uuid.New()
	userID := uuid.New()
	memberID := uuid.New()
	runID := uuid.New().String()[:8]

	mustExec(t, ctx, pool,
		`INSERT INTO organizations (id, slug, name, plan_id)
		 VALUES ($1, $2, $3, 'free')`,
		orgID, "it-"+runID, "Integration Test "+runID,
	)
	mustExec(t, ctx, pool,
		`INSERT INTO users (id, email, password_hash, display_name)
		 VALUES ($1, $2, 'x', 'Tester')`,
		userID, "it-"+runID+"@example.test",
	)
	mustExec(t, ctx, pool,
		`INSERT INTO team_members (id, organization_id, user_id, role, display_name)
		 VALUES ($1, $2, $3, 'admin', 'Tester')`,
		memberID, orgID, userID,
	)

	// Pre-create N candidate tasks in pending state. Each will attempt to
	// transition to in_progress concurrently.
	const n = 32
	taskIDs := make([]uuid.UUID, n)
	for i := 0; i < n; i++ {
		taskIDs[i] = uuid.New()
		mustExec(t, ctx, pool,
			`INSERT INTO tasks (id, organization_id, assigned_to, kind, title, status)
			 VALUES ($1, $2, $3, 'generic', 'Candidate', 'pending')`,
			taskIDs[i], orgID, memberID,
		)
	}

	// Cleanup always — even on failure.
	t.Cleanup(func() {
		cleanupCtx, cancelCleanup := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelCleanup()
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM tasks WHERE organization_id = $1`, orgID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM team_members WHERE id = $1`, memberID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM users WHERE id = $1`, userID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM organizations WHERE id = $1`, orgID)
	})

	// --- Act: race all N transitions ---------------------------------------
	var wg sync.WaitGroup
	results := make([]error, n)
	start := make(chan struct{})

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-start // barrier — maximize contention
			_, err := pool.Exec(ctx, `
				UPDATE tasks
				   SET status = 'in_progress',
				       started_at = now()
				 WHERE id = $1
				   AND organization_id = $2
				   AND status = 'pending'
			`, taskIDs[idx], orgID)
			results[idx] = err
		}(i)
	}
	close(start)
	wg.Wait()

	// --- Assert ------------------------------------------------------------
	var uniqueViolations, otherErrors int
	for _, err := range results {
		switch {
		case err == nil:
			// statement ran; the row may or may not have actually flipped —
			// the count query below is authoritative.
		case isUniqueViolation(err):
			uniqueViolations++
		default:
			otherErrors++
			t.Errorf("unexpected error: %v", err)
		}
	}
	require.Zero(t, otherErrors, "only unique-violation errors are acceptable here")

	var inProgress int
	err = pool.QueryRow(ctx, `
		SELECT count(*) FROM tasks
		WHERE organization_id = $1 AND assigned_to = $2 AND status = 'in_progress'
	`, orgID, memberID).Scan(&inProgress)
	require.NoError(t, err, "count in_progress")
	require.Equalf(t, 1, inProgress,
		"expected exactly one task in_progress, got %d (unique_violations=%d)",
		inProgress, uniqueViolations,
	)

	// Sanity: contention must have materialized at least once. If every
	// concurrent update succeeded serially without a collision, the test is
	// not exercising what we think.
	require.Greaterf(t, uniqueViolations, 0,
		"expected at least one unique-violation from uq_tasks_one_in_progress_per_assignee",
	)
}

func mustExec(t *testing.T, ctx context.Context, pool *pgxpool.Pool, sql string, args ...any) {
	t.Helper()
	_, err := pool.Exec(ctx, sql, args...)
	require.NoErrorf(t, err, "exec: %s", sql)
}

// isUniqueViolation returns true if err is a Postgres unique_violation (SQLSTATE 23505).
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
